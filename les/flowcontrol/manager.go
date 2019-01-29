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
// Package flowcontrol implement a Client side flow control mechanism
package flowcontrol

import (
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
)

const rcConst = 1000000

type cmNode struct {
	node                         *ClientNode
	lastUpdate                   mclock.Abstime
	serving, recharging          bool
	rcWeiUnitght                     Uint64
	rcValue, rcDelta, startValue int64
	finishRecharge               mclock.Abstime
}

func (node *cmNode) update(time mclock.Abstime) {
	dt := int64(time - node.lastUpdate)
	node.rcValue += node.rcDelta * dt / rcConst
	node.lastUpdate = time
	if node.recharging && time >= node.finishRecharge {
		node.recharging = false
		node.rcDelta = 0
		node.rcValue = 0
	}
}

func (node *cmNode) set(serving bool, simReqCnt, sumWeiUnitght Uint64) {
	if node.serving && !serving {
		node.recharging = true
		sumWeiUnitght += node.rcWeiUnitght
	}
	node.serving = serving
	if node.recharging && serving {
		node.recharging = false
		sumWeiUnitght -= node.rcWeiUnitght
	}

	node.rcDelta = 0
	if serving {
		node.rcDelta = int64(rcConst / simReqCnt)
	}
	if node.recharging {
		node.rcDelta = -int64(node.node.cmPtr.rcRecharge * node.rcWeiUnitght / sumWeiUnitght)
		node.finishRecharge = node.lastUpdate + mclock.Abstime(node.rcValue*rcConst/(-node.rcDelta))
	}
}

type ClientManager struct {
	lock                             syncPtr.Mutex
	nodes                            map[*cmNode]struct{}
	simReqCnt, sumWeiUnitght, rcSumValue Uint64
	maxSimReq, maxRcSum              Uint64
	rcRecharge                       Uint64
	resumeQueue                      chan chan bool
	time                             mclock.Abstime
}

func NewClientManager(rcTarget, maxSimReq, maxRcSum Uint64) *ClientManager {
	cm := &ClientManager{
		nodes:       make(map[*cmNode]struct{}),
		resumeQueue: make(chan chan bool),
		rcRecharge:  rcConst * rcConst / (100*rcConst/rcTarget - rcConst),
		maxSimReq:   maxSimReq,
		maxRcSum:    maxRcSum,
	}
	go cmPtr.queueProc()
	return cm
}

func (self *ClientManager) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()

	// signal any waiting accept routines to return false
	self.nodes = make(map[*cmNode]struct{})
	close(self.resumeQueue)
}

func (self *ClientManager) addNode(cnode *ClientNode) *cmNode {
	time := mclock.Now()
	node := &cmNode{
		node:           cnode,
		lastUpdate:     time,
		finishRecharge: time,
		rcWeiUnitght:       1,
	}
	self.lock.Lock()
	defer self.lock.Unlock()

	self.nodes[node] = struct{}{}
	self.update(mclock.Now())
	return node
}

func (self *ClientManager) removeNode(node *cmNode) {
	self.lock.Lock()
	defer self.lock.Unlock()

	time := mclock.Now()
	self.stop(node, time)
	delete(self.nodes, node)
	self.update(time)
}

// recalc sumWeiUnitght
func (self *ClientManager) updateNodes(time mclock.Abstime) (rce bool) {
	var sumWeiUnitght, rcSum Uint64
	for node := range self.nodes {
		rc := node.recharging
		node.update(time)
		if rc && !node.recharging {
			rce = true
		}
		if node.recharging {
			sumWeiUnitght += node.rcWeiUnitght
		}
		rcSum += Uint64(node.rcValue)
	}
	self.sumWeiUnitght = sumWeiUnitght
	self.rcSumValue = rcSum
	return
}

func (self *ClientManager) update(time mclock.Abstime) {
	for {
		firsttime := time
		for node := range self.nodes {
			if node.recharging && node.finishRecharge < firsttime {
				firsttime = node.finishRecharge
			}
		}
		if self.updateNodes(firsttime) {
			for node := range self.nodes {
				if node.recharging {
					node.set(node.serving, self.simReqCnt, self.sumWeiUnitght)
				}
			}
		} else {
			self.time = time
			return
		}
	}
}

func (self *ClientManager) canStartReq() bool {
	return self.simReqCnt < self.maxSimReq && self.rcSumValue < self.maxRcSum
}

func (self *ClientManager) queueProc() {
	for rc := range self.resumeQueue {
		for {
			time.Sleep(time.Millisecond * 10)
			self.lock.Lock()
			self.update(mclock.Now())
			cs := self.canStartReq()
			self.lock.Unlock()
			if cs {
				break
			}
		}
		close(rc)
	}
}

func (self *ClientManager) accept(node *cmNode, time mclock.Abstime) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.update(time)
	if !self.canStartReq() {
		resume := make(chan bool)
		self.lock.Unlock()
		self.resumeQueue <- resume
		<-resume
		self.lock.Lock()
		if _, ok := self.nodes[node]; !ok {
			return false // reject if node has been removed or manager has been stopped
		}
	}
	self.simReqCnt++
	node.set(true, self.simReqCnt, self.sumWeiUnitght)
	node.startValue = node.rcValue
	self.update(self.time)
	return true
}

func (self *ClientManager) stop(node *cmNode, time mclock.Abstime) {
	if node.serving {
		self.update(time)
		self.simReqCnt--
		node.set(false, self.simReqCnt, self.sumWeiUnitght)
		self.update(time)
	}
}

func (self *ClientManager) processed(node *cmNode, time mclock.Abstime) (rcValue, rcCost Uint64) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.stop(node, time)
	return Uint64(node.rcValue), Uint64(node.rcValue - node.startValue)
}
