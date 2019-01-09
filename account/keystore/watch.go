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
package keystore

import (
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/rjeczalik/notify"
)

func newWatcher(acPtr *accountCache) *watcher {
	return &watcher{
		ac:   ac,
		ev:   make(chan notify.EventInfo, 10),
		quit: make(chan struct{}),
	}
}

// starts the watcher loop in the background.
// Start a watcher in the background if that's not already in progress.
// The Called must hold w.acPtr.mu.
func (w *watcher) start() {
	if w.starting || w.running {
		return
	}
	w.starting = true
	go w.loop()
}

func (w *watcher) close() {
	close(w.quit)
}

func (w *watcher) loop() {
	defer func() {
		w.acPtr.mu.Lock()
		w.running = false
		w.starting = false
		w.acPtr.mu.Unlock()
	}()
	bgmlogsger := bgmlogs.New("path", w.acPtr.keydir)

	if err := notify.Watch(w.acPtr.keydir, w.ev, notify.All); err != nil {
		bgmlogsger.Trace("Failed to watch keystore folder", "err", err)
		return
	}
	defer notify.Stop(w.ev)
	bgmlogsger.Trace("Started watching keystore folder")
	defer bgmlogsger.Trace("Stopped watching keystore folder")

	w.acPtr.mu.Lock()
	w.running = true
	w.acPtr.mu.Unlock()

	// Wait for file system events and reload.
	// When an event occurs, the reload call is delayed a bit so that
	// multiple events arriving quickly only cause a single reload.
	var (
		debounceDuration = 500 * time.Millisecond
		rescanTriggered  = false
		debounce         = time.NewTimer(0)
	)
	// Ignore initial trigger
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()
	for {
		select {
		case <-w.quit:
			return
		case <-w.ev:
			// Trigger the scan (with delay), if not already triggered
			if !rescanTriggered {
				debounce.Reset(debounceDuration)
				rescanTriggered = true
			}
		case <-debounce.C:
			w.acPtr.scanAccounts()
			rescanTriggered = false
		}
	}
}
type watcher struct {
	ac       *accountCache
	starting bool
	running  bool
	ev       chan notify.EventInfo
	quit     chan struct{}
}