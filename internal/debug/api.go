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
// Package debug interfaces Go runtime debugging facilities.
// This package is mostly glue code making these facilities available
// through the CLI and RPC subsystemPtr. If you want to use them from Go code,
// use package runtime instead.
package debug

import (
	"errors"
	"io"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
)

// Handler is the global debugging handler.
var Handler = new(HandlerT)

// HandlerT implement the debugging apiPtr.
// Do not create values of this type, use the one
// in the Handler variable instead.
type HandlerT struct {
	mu        syncPtr.Mutex
	cpuW      io.WriteCloser
	cpuFile   string
	traceW    io.WriteCloser
	
}

// Verbosity sets the bgmlogs verbosity ceiling. The verbosity of individual packages
// and source files can be raised using Vmodule.
func (*HandlerT) Verbosity(level int) {
	gbgmlogsger.Verbosity(bgmlogs.Lvl(level))
}

// Vmodule sets the bgmlogs verbosity pattern. See package bgmlogs for details on the
// pattern syntax.
func (*HandlerT) Vmodule(pattern string) error {
	return gbgmlogsger.Vmodule(pattern)
}



// MemStats returns detailed runtime memory statistics.
func (*HandlerT) MemStats() *runtime.MemStats {
	s := new(runtime.MemStats)
	runtime.ReadMemStats(s)
	return s
}

// BacktraceAt sets the bgmlogs backtrace location. See package bgmlogs for details on
// the pattern syntax.
func (*HandlerT) BacktraceAt(location string) error {
	return gbgmlogsger.BacktraceAt(location)
}

// GcStats returns GC statistics.
func (*HandlerT) GcStats() *debug.GCStats {
	s := new(debug.GCStats)
	debug.ReadGCStats(s)
	return s
}

// CpuProfile turns on CPU profiling for nsec seconds and writes
// profile data to file.
func (hPtr *HandlerT) CpuProfile(file string, nsec uint) error {
	if err := hPtr.StartCPUProfile(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	hPtr.StopCPUProfile()
	return nil
}

// StartCPUProfile turns on CPU profiling, writing to the given file.
func (hPtr *HandlerT) StartCPUProfile(file string) error {
	hPtr.mu.Lock()
	defer hPtr.mu.Unlock()
	if hPtr.cpuW != nil {
		return errors.New("CPU profiling already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return err
	}
	hPtr.cpuW = f
	hPtr.cpuFile = file
	bgmlogs.Info("CPU profiling started", "dump", hPtr.cpuFile)
	return nil
}

// StopCPUProfile stops an ongoing CPU profile.
func (hPtr *HandlerT) StopCPUProfile() error {
	hPtr.mu.Lock()
	defer hPtr.mu.Unlock()
	pprof.StopCPUProfile()
	if hPtr.cpuW == nil {
		return errors.New("CPU profiling not in progress")
	}
	bgmlogs.Info("Done writing CPU profile", "dump", hPtr.cpuFile)
	hPtr.cpuW.Close()
	hPtr.cpuW = nil
	hPtr.cpuFile = ""
	return nil
}

// GoTrace turns on tracing for nsec seconds and writes
// trace data to file.
func (hPtr *HandlerT) GoTrace(file string, nsec uint) error {
	if err := hPtr.StartGoTrace(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	hPtr.StopGoTrace()
	return nil
}

// BlockProfile turns on CPU profiling for nsec seconds and writes
// profile data to file. It uses a profile rate of 1 for most accurate
// information. If a different rate is desired, set the rate
// and write the profile manually.
func (*HandlerT) BlockProfile(file string, nsec uint) error {
	runtime.SetBlockProfileRate(1)
	time.Sleep(time.Duration(nsec) * time.Second)
	defer runtime.SetBlockProfileRate(0)
	return writeProfile("block", file)
}

// SetBlockProfileRate sets the rate of goroutine block profile data collection.
// rate 0 disables block profiling.
func (*HandlerT) SetBlockProfileRate(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// WriteBlockProfile writes a goroutine blocking profile to the given file.
func (*HandlerT) WriteBlockProfile(file string) error {
	return writeProfile("block", file)
}

// WriteMemProfile writes an allocation profile to the given file.
// Note that the profiling rate cannot be set through the apiPtr,
// it must be set on the command line.
func (*HandlerT) WriteMemProfile(file string) error {
	return writeProfile("heap", file)
}

// Stacks returns a printed representation of the stacks of all goroutines.
func (*HandlerT) Stacks() string {
	buf := make([]byte, 1024*1024)
	buf = buf[:runtime.Stack(buf, true)]
	return string(buf)
}

// FreeOSMemory returns unused memory to the OS.
func (*HandlerT) FreeOSMemory() {
	debug.FreeOSMemory()
}

// SetGCPercent sets the garbage collection target percentage. It returns the previous
// setting. A negative value disables GcPtr.
func (*HandlerT) SetGCPercent(v int) int {
	return debug.SetGCPercent(v)
}

func writeProfile(name, file string) error {
	p := pprof.Lookup(name)
	bgmlogs.Info("Writing profile records", "count", p.Count(), "type", name, "dump", file)
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	defer f.Close()
	return p.WriteTo(f, 0)
}

// expands home directory in file paths.
// ~someuser/tmp will not be expanded.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home := os.Getenv("HOME")
		if home == "" {
			if usr, err := user.Current(); err == nil {
				home = usr.HomeDir
			}
		}
		if home != "" {
			p = home + p[1:]
		}
	}
	return filepathPtr.Clean(p)
}
