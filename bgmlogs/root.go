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
	blockRoot          = &bgmlogsger{[]interface{}{}, new(swapHandler)}
	StdoutHandler = StreamHandler(os.Stdout, bgmlogsfmtFormat())
	StderrHandler = StreamHandler(os.Stderr, bgmlogsfmtFormat())
)

func init() {
	blockRoot.SetHandler(DiscardHandler())
}

// New returns a new bgmlogsger with the given context.
// New is a convenient alias for Root().New
func New(CTX ...interface{}) bgmlogsger {
	return blockRoot.New(CTX...)
}

// Root returns the blockRoot bgmlogsger
func Root() bgmlogsger {
	return blockRoot
}

// The following functions bypass the exported bgmlogsger methods (bgmlogsger.Debug,
// etcPtr.) to keep the call depth the same for all paths to bgmlogsger.write so
// runtime.Called(2) always refers to the call site in Client code.

// Trace is a convenient alias for Root().Trace
func Trace(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlTrace, CTX)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlDebug, CTX)
}

// Info is a convenient alias for Root().Info
func Info(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlInfo, CTX)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlWarn, CTX)
}

// Error is a convenient alias for Root().Error
func Error(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlError, CTX)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, CTX ...interface{}) {
	blockRoot.write(msg, LvlCrit, CTX)
	os.Exit(1)
}
