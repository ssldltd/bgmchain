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
// Package flowcontrol implement a client side flow control mechanism
package flowcontrol

import (
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
)

const fcTimeConst = time.Millisecond

type Serverbgmparam struct {
	BufLimit, MinRecharge uint64
}

type ClientNode struct {
	bgmparam   *Serverbgmparam
	bufValue uint64
	lastTime mclock.AbsTime
	lock     syncPtr.Mutex
	cm       *ClientManager
	cmNode   *cmNode
}

func NewClientNode(cmPtr *ClientManager, bgmparam *Serverbgmparam) *ClientNode {
	node := &ClientNode{
		cm:       cm,
		bgmparam:   bgmparam,
		bufValue: bgmparam.BufLimit,
		lastTime: mclock.Now(),
	}
	node.cmNode = cmPtr.addNode(node)
	return node
}

func (peer *ClientNode) Remove(cmPtr *ClientManager) {
	cmPtr.removeNode(peer.cmNode)
}

func (peer *ClientNode) recalcBV(time mclock.AbsTime) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufValue += peer.bgmparam.MinRecharge * dt / uint64(fcTimeConst)
	if peer.bufValue > peer.bgmparam.BufLimit {
		peer.bufValue = peer.bgmparam.BufLimit
	}
	peer.lastTime = time
}

func (peer *ClientNode) AcceptRequest() (uint64, bool) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := mclock.Now()
	peer.recalcBV(time)
	return peer.bufValue, peer.cmPtr.accept(peer.cmNode, time)
}

func (peer *ClientNode) RequestProcessed(cost uint64) (bv, realCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := mclock.Now()
	peer.recalcBV(time)
	peer.bufValue -= cost
	peer.recalcBV(time)
	rcValue, rcost := peer.cmPtr.processed(peer.cmNode, time)
	if rcValue < peer.bgmparam.BufLimit {
		bv := peer.bgmparam.BufLimit - rcValue
		if bv > peer.bufValue {
			peer.bufValue = bv
		}
	}
	return peer.bufValue, rcost
}

type ServerNode struct {
	bufEstimate uint64
	lastTime    mclock.AbsTime
	bgmparam      *Serverbgmparam
	sumCost     uint64            // sum of req costs sent to this server
	pending     map[uint64]uint64 // value = sumCost after sending the given req
	lock        syncPtr.RWMutex
}

func NewServerNode(bgmparam *Serverbgmparam) *ServerNode {
	return &ServerNode{
		bufEstimate: bgmparam.BufLimit,
		lastTime:    mclock.Now(),
		bgmparam:      bgmparam,
		pending:     make(map[uint64]uint64),
	}
}

func (peer *ServerNode) recalcBLE(time mclock.AbsTime) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufEstimate += peer.bgmparam.MinRecharge * dt / uint64(fcTimeConst)
	if peer.bufEstimate > peer.bgmparam.BufLimit {
		peer.bufEstimate = peer.bgmparam.BufLimit
	}
	peer.lastTime = time
}

// safetyMargin is added to the flow control waiting time when estimated buffer value is low
const safetyMargin = time.Millisecond

func (peer *ServerNode) canSend(maxCost uint64) (time.Duration, float64) {
	peer.recalcBLE(mclock.Now())
	maxCost += uint64(safetyMargin) * peer.bgmparam.MinRecharge / uint64(fcTimeConst)
	if maxCost > peer.bgmparam.BufLimit {
		maxCost = peer.bgmparam.BufLimit
	}
	if peer.bufEstimate >= maxCost {
		return 0, float64(peer.bufEstimate-maxCost) / float64(peer.bgmparam.BufLimit)
	}
	return time.Duration((maxCost - peer.bufEstimate) * uint64(fcTimeConst) / peer.bgmparam.MinRecharge), 0
}

// CanSend returns the minimum waiting time required before sending a request
// with the given maximum estimated cost. Second return value is the relative
// estimated buffer level after sending the request (divided by BufLimit).
func (peer *ServerNode) CanSend(maxCost uint64) (time.Duration, float64) {
	peer.lock.RLock()
	defer peer.lock.RUnlock()

	return peer.canSend(maxCost)
}

// QueueRequest should be called when the request has been assigned to the given
// server node, before putting it in the send queue. It is mandatory that requests
// are sent in the same order as the QueueRequest calls are made.
func (peer *ServerNode) QueueRequest(reqID, maxCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	peer.bufEstimate -= maxCost
	peer.sumCost += maxCost
	peer.pending[reqID] = peer.sumCost
}

// GotReply adjusts estimated buffer value according to the value included in
// the latest request reply.
func (peer *ServerNode) GotReply(reqID, bv uint64) {

	peer.lock.Lock()
	defer peer.lock.Unlock()

	if bv > peer.bgmparam.BufLimit {
		bv = peer.bgmparam.BufLimit
	}
	sc, ok := peer.pending[reqID]
	if !ok {
		return
	}
	delete(peer.pending, reqID)
	cc := peer.sumCost - sc
	peer.bufEstimate = 0
	if bv > cc {
		peer.bufEstimate = bv - cc
	}
	peer.lastTime = mclock.Now()
}
