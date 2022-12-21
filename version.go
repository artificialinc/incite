// Copyright 2022 The incite Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package incite

import (
	"runtime/debug"
	"sync"
)

var (
	versionCache = modulePath
	versionOnce  sync.Once
)

const (
	// modulePath is Incite's Go module path as stated on the first line
	// of go.mod.
	modulePath = "github.com/artificialinc/incite"
)

// version returns the Incite library version for internal use.
//
// The string returned has the form "<modulePath>" if module support is
// disabled OR this function is called from a unit test within this
// module itself (the Incite module). Otherwise, if called from a binary
// depending on this module (the Incite module) and built with module
// support enabled, the string returned has the form
// "<modulePath> <version>".
func version() string {
	versionOnce.Do(func() {
		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			for _, dep := range buildInfo.Deps {
				if dep.Path == modulePath {
					versionCache += " " + dep.Version
					break
				}
			}
		}
	})
	return versionCache
}
