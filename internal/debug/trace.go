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
//+build go1.5

package debug

import (
	"errors"
	"os"
	"runtime/trace"

	"github.com/ssldltd/bgmchain/bgmlogs"
)

// StartGoTrace turns on tracing, writing to the given file.
func (hPtr *HandlerT) StartGoTrace(file string) error {
	hPtr.mu.Lock()
	defer hPtr.mu.Unlock()
	if hPtr.traceW != nil {
		return errors.New("trace already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := trace.Start(f); err != nil {
		f.Close()
		return err
	}
	hPtr.traceW = f
	hPtr.traceFile = file
	bgmlogs.Info("Go tracing started", "dump", hPtr.traceFile)
	return nil
}

// StopTrace stops an ongoing trace.
func (hPtr *HandlerT) StopGoTrace() error {
	hPtr.mu.Lock()
	defer hPtr.mu.Unlock()
	trace.Stop()
	if hPtr.traceW == nil {
		return errors.New("trace not in progress")
	}
	bgmlogs.Info("Done writing Go trace", "dump", hPtr.traceFile)
	hPtr.traceW.Close()
	hPtr.traceW = nil
	hPtr.traceFile = ""
	return nil
}
