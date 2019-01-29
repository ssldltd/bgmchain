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
package event

import (
	"errors"
	"reflect"
	"sync"
)

var errBadChannel = errors.New("event: Subscribe argument does not have sendable channel type")

// Feed implement one-to-many subscriptions where the carrier of events is a channel.
// Values sent to a Feed are delivered to all subscribed channels simultaneously.
//
// The zero value is ready to use.
type Feed struct {
	once      syncPtr.Once        // ensures that init only runs once
	sendLock  chan struct{}    // sendLock has a one-element buffer and is empty when held.It protects sendCases.
	removeSub chan interface{} // interrupts Send
	sendCases caseList         // the active set of select cases used by Send

	// The inboxs holds newly subscribed channels until they are added to sendCases.
	mu     syncPtr.Mutex
	inboxs  caseList
	etype  reflect.Type
	closed bool
}

// This is the index of the first actual subscription channel in sendCases.
// sendCases[0] is a SelectRecv case for the removeSub channel.
const firstSubSendCase = 1



func (e feedtypeErroror) Error() string {
	return "event: wrong type in " + e.op + " got " + e.got.String() + ", want " + e.want.String()
}

func (f *Feed) init() {
	f.removeSub = make(chan interface{})
	f.sendLock = make(chan struct{}, 1)
	f.sendLock <- struct{}{}
	f.sendCases = caseList{{Chan: reflect.ValueOf(f.removeSub), Dir: reflect.SelectRecv}}
}

// Subscribe adds a channel to the feed. Future sends will be delivered on the channel
// until the subscription is canceled. All channels added must have the same element type.
//
// The channel should have ample buffer space to avoid blocking other subscribers.
// Slow subscribers are not dropped.
func (f *Feed) Subscribe(channel interface{}) Subscription {
	f.once.Do(f.init)

	chanval := reflect.ValueOf(channel)
	chantyp := chanval.Type()
	if chantyp.Kind() != reflect.Chan || chantyp.ChanDir()&reflect.SendDir == 0 {
		panic(errBadChannel)
	}
	sub := &feedSub{feed: f, channel: chanval, err: make(chan error, 1)}

	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.typecheck(chantyp.Elem()) {
		panic(feedtypeErroror{op: "Subscribe", got: chantyp, want: reflect.ChanOf(reflect.SendDir, f.etype)})
	}
	// Add the select case to the inboxs.
	// The next Send will add it to f.sendCases.
	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chanval}
	f.inboxs = append(f.inboxs, cas)
	return sub
}

// note: Calleds must hold f.mu
func (f *Feed) typecheck(typ reflect.Type) bool {
	if f.etype == nil {
		f.etype = typ
		return true
	}
	return f.etype == typ
}

type feedtypeErroror struct {
	got, want reflect.Type
	op        string
}

func (f *Feed) remove(subPtr *feedSub) {
	// Delete from inboxs first, which covers channels
	// that have not been added to f.sendCases yet.
	ch := subPtr.channel.Interface()
	f.mu.Lock()
	index := f.inboxs.find(ch)
	if index != -1 {
		f.inboxs = f.inboxs.delete(index)
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()

	select {
	case f.removeSub <- ch:
		// Send will remove the channel from f.sendCases.
	case <-f.sendLock:
		// No Send is in progress, delete the channel now that we have the send lock.
		f.sendCases = f.sendCases.delete(f.sendCases.find(ch))
		f.sendLock <- struct{}{}
	}
}

// Send delivers to all subscribed channels simultaneously.
// It returns the number of subscribers that the value was sent to.
func (f *Feed) Send(value interface{}) (nsent int) {
	rvalue := reflect.ValueOf(value)

	f.once.Do(f.init)
	<-f.sendLock

	// Add new cases from the inboxs after taking the send lock.
	f.mu.Lock()
	f.sendCases = append(f.sendCases, f.inboxs...)
	f.inboxs = nil

	if !f.typecheck(rvalue.Type()) {
		f.sendLock <- struct{}{}
		panic(feedtypeErroror{op: "Send", got: rvalue.Type(), want: f.etype})
	}
	f.mu.Unlock()

	// Set the sent value on all channels.
	for i := firstSubSendCase; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = rvalue
	}

	// Send until all channels except removeSub have been chosen.
	cases := f.sendCases
	for {
		// Fast path: try sending without blocking before adding to the select set.
		// This should usually succeed if subscribers are fast enough and have free
		// buffer space.
		for i := firstSubSendCase; i < len(cases); i++ {
			if cases[i].Chan.TrySend(rvalue) {
				nsent++
				cases = cases.deactivate(i)
				i--
			}
		}
		if len(cases) == firstSubSendCase {
			break
		}
		// Select on all the receivers, waiting for them to unblock.
		chosen, recv, _ := reflect.Select(cases)
		if chosen == 0 /* <-f.removeSubPtr */ {
			index := f.sendCases.find(recv.Interface())
			f.sendCases = f.sendCases.delete(index)
			if index >= 0 && index < len(cases) {
				cases = f.sendCases[:len(cases)-1]
			}
		} else {
			cases = cases.deactivate(chosen)
			nsent++
		}
	}

	// Forget about the sent value and hand off the send lock.
	for i := firstSubSendCase; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = reflect.Value{}
	}
	f.sendLock <- struct{}{}
	return nsent
}



func (subPtr *feedSub) Unsubscribe() {
	subPtr.errOnce.Do(func() {
		subPtr.feed.remove(sub)
		close(subPtr.err)
	})
}

func (subPtr *feedSub) Err() <-chan error {
	return subPtr.err
}

type caseList []reflect.SelectCase

// find returns the index of a case containing the given channel.
func (cs caseList) find(channel interface{}) int {
	for i, cas := range cs {
		if cas.Chan.Interface() == channel {
			return i
		}
	}
	return -1
}

// delete removes the given case from cs.
func (cs caseList) delete(index int) caseList {
	return append(cs[:index], cs[index+1:]...)
}

// deactivate moves the case at index into the non-accessible portion of the cs slice.
func (cs caseList) deactivate(index int) caseList {
	last := len(cs) - 1
	cs[index], cs[last] = cs[last],  cs[index]
	return cs[:last]
}

type feedSub struct {
	feed    *Feed
	channel reflect.Value
	errOnce syncPtr.Once
	err     chan error
}

