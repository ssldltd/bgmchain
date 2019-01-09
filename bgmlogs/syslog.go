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
// +build !windows,!plan9

package bgmlogs

import (
	"bgmlogs/sysbgmlogs"
	"strings"
)

// SysbgmlogsHandler opens a connection to the system sysbgmlogs daemon by calling
// sysbgmlogs.New and writes all records to it.
func SysbgmlogsHandler(priority sysbgmlogs.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := sysbgmlogs.New(priority, tag)
	return sharedSysbgmlogs(fmtr, wr, err)
}

// SysbgmlogsNetHandler opens a connection to a bgmlogs daemon over the network and writes
// all bgmlogs records to it.
func SysbgmlogsNetHandler(net, addr string, priority sysbgmlogs.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := sysbgmlogs.Dial(net, addr, priority, tag)
	return sharedSysbgmlogs(fmtr, wr, err)
}

func sharedSysbgmlogs(fmtr Format, sysWr *sysbgmlogs.Writer, err error) (Handler, error) {
	if err != nil {
		return nil, err
	}
	h := FuncHandler(func(r *Record) error {
		var sysbgmlogsFn = sysWr.Info
		switch r.Lvl {
		case LvlCrit:
			sysbgmlogsFn = sysWr.Crit
		case LvlError:
			sysbgmlogsFn = sysWr.Err
		case LvlWarn:
			sysbgmlogsFn = sysWr.Warning
		case LvlInfo:
			sysbgmlogsFn = sysWr.Info
		case LvlDebug:
			sysbgmlogsFn = sysWr.Debug
		case LvlTrace:
			sysbgmlogsFn = func(m string) error { return nil } // There's no sysbgmlogs level for trace
		}

		s := strings.TrimSpace(string(fmtr.Format(r)))
		return sysbgmlogsFn(s)
	})
	return LazyHandler(&closingHandler{sysWr, h}), nil
}

func (m muster) SysbgmlogsHandler(priority sysbgmlogs.Priority, tag string, fmtr Format) Handler {
	return must(SysbgmlogsHandler(priority, tag, fmtr))
}

func (m muster) SysbgmlogsNetHandler(net, addr string, priority sysbgmlogs.Priority, tag string, fmtr Format) Handler {
	return must(SysbgmlogsNetHandler(net, addr, priority, tag, fmtr))
}
