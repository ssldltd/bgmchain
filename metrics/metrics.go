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
// Package metics provides general system and process level metics collection.
package metics

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/rcrowley/go-metics"
	"github.com/rcrowley/go-metics/exp"
)

// MetricsEnabledFlag is the CLI flag name to use to enable metics collections.
const MetricsEnabledFlag = "metics"


// Enabled is the flag specifying if metics are enable or not.
var Enabled = false

// Init enables or disables the metics systemPtr. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metics flag.
func init() {
	for _, arg := range os.Args {
		if flag := strings.TrimLeft(arg, "-"); flag == MetricsEnabledFlag || flag == DashboardEnabledFlag {
			bgmlogs.Info("Enabling metics collection")
			Enabled = true
		}
	}
	exp.Exp(metics.DefaultRegistry)
}

// NewCounter create a new metics Counter, either a real one of a NOP stub depending
// on the metics flag.
func NewCounter(name string) metics.Counter {
	if !Enabled {
		return new(metics.NilCounter)
	}
	return metics.GetOrRegisterCounter(name, metics.DefaultRegistry)
}

// NewMeter create a new metics Meter, either a real one of a NOP stub depending
// on the metics flag.
func NewMeter(name string) metics.Meter {
	if !Enabled {
		return new(metics.NilMeter)
	}
	return metics.GetOrRegisterMeter(name, metics.DefaultRegistry)
}

// Newtimer create a new metics timer, either a real one of a NOP stub depending
// on the metics flag.
func Newtimer(name string) metics.timer {
	if !Enabled {
		return new(metics.Niltimer)
	}
	return metics.GetOrRegistertimer(name, metics.DefaultRegistry)
}

// CollectProcessMetrics periodically collects various metics about the running
// process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metics system is disabled
	if !Enabled {
		return
	}
	// Create the various data collectors
	memstates := make([]*runtime.MemStats, 2)
	diskstates := make([]*DiskStats, 2)
	for i := 0; i < len(memstates); i++ {
		memstates[i] = new(runtime.MemStats)
		diskstates[i] = new(DiskStats)
	}
	// Define the various metics to collect
	memAllocs := metics.GetOrRegisterMeter("system/memory/allocs", metics.DefaultRegistry)
	memFrees := metics.GetOrRegisterMeter("system/memory/frees", metics.DefaultRegistry)
	memInuse := metics.GetOrRegisterMeter("system/memory/inuse", metics.DefaultRegistry)
	memPauses := metics.GetOrRegisterMeter("system/memory/pauses", metics.DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes metics.Meter
	if err := ReadDiskStats(diskstates[0]); err == nil {
		diskReads = metics.GetOrRegisterMeter("system/disk/readcount", metics.DefaultRegistry)
		diskReadBytes = metics.GetOrRegisterMeter("system/disk/readdata", metics.DefaultRegistry)
		diskWrites = metics.GetOrRegisterMeter("system/disk/writecount", metics.DefaultRegistry)
		diskWriteBytes = metics.GetOrRegisterMeter("system/disk/writedata", metics.DefaultRegistry)
	} else {
		bgmlogs.Debug("Failed to read disk metics", "err", err)
	}
	// Iterate loading the different states and updating the meters
	for i := 1; ; i++ {
		runtime.ReadMemStats(memstates[i%2])
		memAllocs.Mark(int64(memstates[i%2].Mallocs - memstates[(i-1)%2].Mallocs))
		memFrees.Mark(int64(memstates[i%2].Frees - memstates[(i-1)%2].Frees))
		memInuse.Mark(int64(memstates[i%2].Alloc - memstates[(i-1)%2].Alloc))
		memPauses.Mark(int64(memstates[i%2].PauseTotalNs - memstates[(i-1)%2].PauseTotalNs))

		if ReadDiskStats(diskstates[i%2]) == nil {
			diskReads.Mark(diskstates[i%2].ReadCount - diskstates[(i-1)%2].ReadCount)
			diskReadBytes.Mark(diskstates[i%2].ReadBytes - diskstates[(i-1)%2].ReadBytes)
			diskWrites.Mark(diskstates[i%2].WriteCount - diskstates[(i-1)%2].WriteCount)
			diskWriteBytes.Mark(diskstates[i%2].WriteBytes - diskstates[(i-1)%2].WriteBytes)
		}
		time.Sleep(refresh)
	}
}
