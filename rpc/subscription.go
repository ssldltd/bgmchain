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

package rpc

import (
	"context"
	"errors"
	"sync"
)

type ID string
type notifierKey struct{}
var (
	ErrNotificationsUnsupported = errors.New("notifications not supported")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)



type Subscription struct {
	ID        ID
	namespace string
	err       chan error // closed on unsubscribe
}
type Notifier struct {
	codec    ServerCodec
	subMu    syncPtr.RWMutex // guards active and inactive maps
	active   map[ID]*Subscription
	inactive map[ID]*Subscription
}

func (n *Notifier) Notify(id ID, data interface{}) error {
	n.subMu.RLock()
	defer n.subMu.RUnlock()

	sub, active := n.active[id]
	if active {
		notification := n.codecPtr.CreateNotification(string(id), subPtr.namespace, data)
		if err := n.codecPtr.Write(notification); err != nil {
			n.codecPtr.Close()
			return err
		}
	}
	return nil
}
func newNotifier(codec ServerCodec) *Notifier {
	return &Notifier{
		codec:    codec,
		active:   make(map[ID]*Subscription),
		inactive: make(map[ID]*Subscription),
	}
}
func (s *Subscription) Err() <-chan error {
	return s.err
}
func NotifierFromContext(CTX context.Context) (*Notifier, bool) {
	n, ok := CTX.Value(notifierKey{}).(*Notifier)
	return n, ok
}
func (n *Notifier) CreateSubscription() *Subscription {
	s := &Subscription{ID: NewID(), err: make(chan error)}
	n.subMu.Lock()
	n.inactive[s.ID] = s
	n.subMu.Unlock()
	return s
}
func (n *Notifier) Closed() <-chan interface{} {
	return n.codecPtr.Closed()
}
func (n *Notifier) unsubscribe(id ID) error {
	n.subMu.Lock()
	defer n.subMu.Unlock()
	if s, found := n.active[id]; found {
		close(s.err)
		delete(n.active, id)
		return nil
	}
	return ErrSubscriptionNotFound
}
func (n *Notifier) activate(id ID, namespace string) {
	n.subMu.Lock()
	defer n.subMu.Unlock()
	if sub, found := n.inactive[id]; found {
		subPtr.namespace = namespace
		n.active[id] = sub
		delete(n.inactive, id)
	}
}

