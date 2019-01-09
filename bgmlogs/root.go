// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.
package bgmlogs

import (
	"os"
)

var (
	root          = &bgmlogsger{[]interface{}{}, new(swapHandler)}
	StdoutHandler = StreamHandler(os.Stdout, bgmlogsfmtFormat())
	StderrHandler = StreamHandler(os.Stderr, bgmlogsfmtFormat())
)

func init() {
	root.SetHandler(DiscardHandler())
}

// New returns a new bgmlogsger with the given context.
// New is a convenient alias for Root().New
func New(ctx ...interface{}) bgmlogsger {
	return root.New(ctx...)
}

// Root returns the root bgmlogsger
func Root() bgmlogsger {
	return root
}

// The following functions bypass the exported bgmlogsger methods (bgmlogsger.Debug,
// etcPtr.) to keep the call depth the same for all paths to bgmlogsger.write so
// runtime.Called(2) always refers to the call site in client code.

// Trace is a convenient alias for Root().Trace
func Trace(msg string, ctx ...interface{}) {
	root.write(msg, LvlTrace, ctx)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, ctx ...interface{}) {
	root.write(msg, LvlDebug, ctx)
}

// Info is a convenient alias for Root().Info
func Info(msg string, ctx ...interface{}) {
	root.write(msg, LvlInfo, ctx)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, ctx ...interface{}) {
	root.write(msg, LvlWarn, ctx)
}

// Error is a convenient alias for Root().Error
func Error(msg string, ctx ...interface{}) {
	root.write(msg, LvlError, ctx)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, ctx ...interface{}) {
	root.write(msg, LvlCrit, ctx)
	os.Exit(1)
}
