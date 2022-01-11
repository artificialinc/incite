// Copyright 2022 The incite Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package incite

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type starter struct {
	worker
}

func newStarter(m *mgr) *starter {
	s := &starter{
		worker: worker{
			m:         m,
			regulator: makeRegulator(m.close, m.RPS[StopQuery], RPSDefaults[StopQuery]),
			in:        m.start,
			out:       m.update,
			name:      "starter",
			maxTry:    10, // TODO: put real constant here.
		},
	}
	s.manipulator = s
	return s
}

func (s *starter) context(c *chunk) context.Context {
	return c.ctx
}

func (s *starter) manipulate(c *chunk) bool {
	// Discard chunk if the owning stream is dead.
	if !c.stream.alive() {
		return true
	}

	// Get the chunk time range in Insights' format.
	starts := c.start.Unix()
	ends := c.end.Add(-time.Second).Unix() // CWL uses inclusive time ranges at 1 second granularity, we use exclusive ranges.

	// Start the chunk.
	input := cloudwatchlogs.StartQueryInput{
		QueryString:   &c.stream.Text,
		StartTime:     &starts,
		EndTime:       &ends,
		LogGroupNames: c.stream.groups,
		Limit:         &c.stream.Limit,
	}
	output, err := s.m.Actions.StartQueryWithContext(c.ctx, &input)
	s.lastReq = time.Now()
	if err != nil && isTemporary(err) {
		s.m.logChunk(c, "temporary failure to start", err.Error())
		return false
	} else if err != nil {
		c.err = &StartQueryError{c.stream.Text, c.start, c.end, err}
		s.m.logChunk(c, "permanent failure to start", "fatal error from CloudWatch Logs: "+err.Error())
		return true
	}

	// Save the current query ID into the chunk.
	queryID := output.QueryId
	if queryID == nil {
		c.err = &StartQueryError{c.stream.Text, c.start, c.end, errors.New(outputMissingQueryIDMsg)}
		s.m.logChunk(c, "nil query ID from CloudWatch Logs for", "")
		return true
	}
	c.queryID = *queryID

	// Chunk is started successfully.
	c.state = started
	s.m.logChunk(c, "started", "")
	return true
}

func (s *starter) release(c *chunk) {
	s.m.logChunk(c, "releasing startable", "")
}
