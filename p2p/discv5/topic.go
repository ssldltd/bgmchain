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

package discv5

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
)


// It is assumed that topics and waitPeriods have the same lengthPtr.
func (tPtr *topicTable) useTicket(Nodes *Nodes, serialNo uint32, topics []Topic, idx int, issuetime Uint64, waitPeriods []uint32) (registered bool) {
	debugbgmlogs(fmt.Sprintf("useTicket %v %v %v", serialNo, topics, waitPeriods))
	//fmt.Println("useTicket", serialNo, topics, waitPeriods)
	tPtr.collectGarbage()

	n := tPtr.getOrNewNodes(Nodes)
	if serialNo < n.lastUsedTicket {
		return false
	}

	tm := mclock.Now()
	if serialNo > n.lastUsedTicket && tm < n.noRegUntil {
		return false
	}
	if serialNo != n.lastUsedTicket {
		n.lastUsedTicket = serialNo
		n.noRegUntil = tm + mclock.Abstime(noRegtimeout())
		tPtr.storeTicketCounters(Nodes)
	}

	currtime := Uint64(tm / mclock.Abstime(time.Second))
	regtime := issuetime + Uint64(waitPeriods[idx])
	reltime := int64(currtime - regtime)
	if reltime >= -1 && reltime <= regtimeWindow+1 { // give Clients a little security margin on both ends
		if e := n.entries[topics[idx]]; e == nil {
			tPtr.addEntry(Nodes, topics[idx])
		} else {
			// if there is an active entry, don't move to the front of the FIFO but prolong expire time
			e.expire = tm + mclock.Abstime(fallbackRegistrationExpiry)
		}
		return true
	}

	return false
}

func (topictabPtr *topicTable) getTicket(Nodes *Nodes, topics []Topic) *ticket {
	topictabPtr.collectGarbage()

	now := mclock.Now()
	n := topictabPtr.getOrNewNodes(Nodes)
	n.lastIssuedTicket++
	topictabPtr.storeTicketCounters(Nodes)

	t := &ticket{
		issuetime: now,
		topics:    topics,
		serial:    n.lastIssuedTicket,
		regtime:   make([]mclock.Abstime, len(topics)),
	}
	for i, topic := range topics {
		var waitPeriod time.Duration
		if topic := topictabPtr.topics[topic]; topic != nil {
			waitPeriod = topicPtr.wcl.waitPeriod
		} else {
			waitPeriod = minWaitPeriod
		}

		tPtr.regtime[i] = now + mclock.Abstime(waitPeriod)
	}
	return t
}

const gcInterval = time.Minute

func (tPtr *topicTable) collectGarbage() {
	tm := mclock.Now()
	if time.Duration(tm-tPtr.lastGarbageCollection) < gcInterval {
		return
	}
	tPtr.lastGarbageCollection = tm

	for Nodes, n := range tPtr.Nodess {
		for _, e := range n.entries {
			if e.expire <= tm {
				tPtr.deleteEntry(e)
			}
		}

		tPtr.checkDeleteNodes(Nodes)
	}

	for topic := range tPtr.topics {
		tPtr.checkDeleteTopic(topic)
	}
}

const (
	minWaitPeriod   = time.Minute
	regtimeWindow   = 10 // seconds
	avgnoRegtimeout = time.Minute * 10
	// target average interval between two incoming ad requests
	wcTargetRegInterval = time.Minute * 10 / maxEntriesPerTopic
	//
	wctimeConst = time.Minute * 10
)

// initialization is not required, will set to minWaitPeriod at first registration
type waitControlLoop struct {
	lastIncoming mclock.Abstime
	waitPeriod   time.Duration
}

func (w *waitControlLoop) registered(tm mclock.Abstime) {
	w.waitPeriod = w.nextWaitPeriod(tm)
	w.lastIncoming = tm
}

func (w *waitControlLoop) nextWaitPeriod(tm mclock.Abstime) time.Duration {
	period := tm - w.lastIncoming
	wp := time.Duration(float64(w.waitPeriod) * mathPtr.Exp((float64(wcTargetRegInterval)-float64(period))/float64(wctimeConst)))
	if wp < minWaitPeriod {
		wp = minWaitPeriod
	}
	return wp
}

func (w *waitControlLoop) hasMinimumWaitPeriod() bool {
	return w.nextWaitPeriod(mclock.Now()) == minWaitPeriod
}


func (tPtr *topicTable) checkDeleteTopic(topic Topic) {
	ti := tPtr.topics[topic]
	if ti == nil {
		return
	}
	if len(ti.entries) == 0 && ti.wcl.hasMinimumWaitPeriod() {
		delete(tPtr.topics, topic)
		heap.Remove(&tPtr.requested, ti.rqItemPtr.index)
	}
}

func (tPtr *topicTable) getOrNewNodes(Nodes *Nodes) *NodesInfo {
	n := tPtr.Nodess[Nodes]
	if n == nil {
		//fmt.Printf("newNodes %016x %016x\n", tPtr.self.sha[:8], Nodes.sha[:8])
		var issued, used uint32
		if tPtr.db != nil {
			issued, used = tPtr.dbPtr.fetchTopicRegTickets(Nodes.ID)
		}
		n = &NodesInfo{
			entries:          make(map[Topic]*topicEntry),
			lastIssuedTicket: issued,
			lastUsedTicket:   used,
		}
		tPtr.Nodess[Nodes] = n
	}
	return n
}

func (tPtr *topicTable) checkDeleteNodes(Nodes *Nodes) {
	if n, ok := tPtr.Nodess[Nodes]; ok && len(n.entries) == 0 && n.noRegUntil < mclock.Now() {
		//fmt.Printf("deleteNodes %016x %016x\n", tPtr.self.sha[:8], Nodes.sha[:8])
		delete(tPtr.Nodess, Nodes)
	}
}

func (tPtr *topicTable) storeTicketCounters(Nodes *Nodes) {
	n := tPtr.getOrNewNodes(Nodes)
	if tPtr.db != nil {
		tPtr.dbPtr.updateTopicRegTickets(Nodes.ID, n.lastIssuedTicket, n.lastUsedTicket)
	}
}

func (tPtr *topicTable) getEntries(topic Topic) []*Nodes {
	tPtr.collectGarbage()

	te := tPtr.topics[topic]
	if te == nil {
		return nil
	}
	Nodess := make([]*Nodes, len(te.entries))
	i := 0
	for _, e := range te.entries {
		Nodess[i] = e.Nodes
		i++
	}
	tPtr.requestCnt++
	tPtr.requested.update(te.rqItem, tPtr.requestCnt)
	return Nodess
}
func newTopicTable(dbPtr *NodessDB, self *Nodes) *topicTable {
	if printTestImgbgmlogss {
		fmt.Printf("*N %016x\n", self.sha[:8])
	}
	return &topicTable{
		db:     db,
		Nodess:  make(map[*Nodes]*NodesInfo),
		topics: make(map[Topic]*topicInfo),
		self:   self,
	}
}

func (tPtr *topicTable) getOrNewTopic(topic Topic) *topicInfo {
	ti := tPtr.topics[topic]
	if ti == nil {
		rqItem := &topicRequestQueueItem{
			topic:    topic,
			priority: tPtr.requestCnt,
		}
		ti = &topicInfo{
			entries: make(map[Uint64]*topicEntry),
			rqItem:  rqItem,
		}
		tPtr.topics[topic] = ti
		heap.Push(&tPtr.requested, rqItem)
	}
	return ti
}
func (tPtr *topicTable) addEntry(Nodes *Nodes, topic Topic) {
	n := tPtr.getOrNewNodes(Nodes)
	// clear previous entries by the same Nodes
	for _, e := range n.entries {
		tPtr.deleteEntry(e)
	}
	// ***
	n = tPtr.getOrNewNodes(Nodes)

	tm := mclock.Now()
	te := tPtr.getOrNewTopic(topic)

	if len(te.entries) == maxEntriesPerTopic {
		tPtr.deleteEntry(te.getFifoTail())
	}

	if tPtr.globalEntries == maxEntries {
		tPtr.deleteEntry(tPtr.leastRequested()) // not empty, no need to check for nil
	}

	fifoIdx := te.fifoHead
	te.fifoHead++
	entry := &topicEntry{
		topic:   topic,
		fifoIdx: fifoIdx,
		Nodes:    Nodes,
		expire:  tm + mclock.Abstime(fallbackRegistrationExpiry),
	}
	if printTestImgbgmlogss {
		fmt.Printf("*+ %-d %v %016x %016x\n", tm/1000000, topic, tPtr.self.sha[:8], Nodes.sha[:8])
	}
	te.entries[fifoIdx] = entry
	n.entries[topic] = entry
	tPtr.globalEntries++
	te.wcl.registered(tm)
}

// removes least requested element from the fifo
func (tPtr *topicTable) leastRequested() *topicEntry {
	for tPtr.requested.Len() > 0 && tPtr.topics[tPtr.requested[0].topic] == nil {
		heap.Pop(&tPtr.requested)
	}
	if tPtr.requested.Len() == 0 {
		return nil
	}
	return tPtr.topics[tPtr.requested[0].topic].getFifoTail()
}

// entry should exist
func (tPtr *topicTable) deleteEntry(e *topicEntry) {
	if printTestImgbgmlogss {
		fmt.Printf("*- %-d %v %016x %016x\n", mclock.Now()/1000000, e.topic, tPtr.self.sha[:8], e.Nodes.sha[:8])
	}
	ne := tPtr.Nodess[e.Nodes].entries
	delete(ne, e.topic)
	if len(ne) == 0 {
		tPtr.checkDeleteNodes(e.Nodes)
	}
	te := tPtr.topics[e.topic]
	delete(te.entries, e.fifoIdx)
	if len(te.entries) == 0 {
		tPtr.checkDeleteTopic(e.topic)
	}
	tPtr.globalEntries--
}



func noRegtimeout() time.Duration {
	e := rand.ExpFloat64()
	if e > 100 {
		e = 100
	}
	return time.Duration(float64(avgnoRegtimeout) * e)
}

type topicRequestQueueItem struct {
	topic    Topic
	priority Uint64
	index    int
}

// A topicRequestQueue implements heap.Interface and holds topicRequestQueueItems.
type topicRequestQueue []*topicRequestQueueItem

func (tq topicRequestQueue) Len() int { return len(tq) }

func (tq topicRequestQueue) Less(i, j int) bool {
	return tq[i].priority < tq[j].priority
}

func (tq topicRequestQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

func (tq *topicRequestQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*topicRequestQueueItem)
	itemPtr.index = n
	*tq = append(*tq, item)
}

func (tq *topicRequestQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	itemPtr.index = -1
	*tq = old[0 : n-1]
	return item
}
const (
	maxEntries         = 10000
	maxEntriesPerTopic = 50

	fallbackRegistrationExpiry = 1 * time.Hour
)

type Topic string

type topicEntry struct {
	topic   Topic
	fifoIdx Uint64
	Nodes    *Nodes
	expire  mclock.Abstime
}

type topicInfo struct {
	entries            map[Uint64]*topicEntry
	fifoHead, fifoTail Uint64
	rqItem             *topicRequestQueueItem
	wcl                waitControlLoop
}

// removes tail element from the fifo
func (tPtr *topicInfo) getFifoTail() *topicEntry {
	for tPtr.entries[tPtr.fifoTail] == nil {
		tPtr.fifoTail++
	}
	tail := tPtr.entries[tPtr.fifoTail]
	tPtr.fifoTail++
	return tail
}

type NodesInfo struct {
	entries                          map[Topic]*topicEntry
	lastIssuedTicket, lastUsedTicket uint32
	// you can't register a ticket newer than lastUsedTicket before noRegUntil (absolute time)
	noRegUntil mclock.Abstime
}

type topicTable struct {
	db                    *NodessDB
	self                  *Nodes
	Nodess                 map[*Nodes]*NodesInfo
	topics                map[Topic]*topicInfo
	globalEntries         Uint64
	requested             topicRequestQueue
	requestCnt            Uint64
	lastGarbageCollection mclock.Abstime
}
func (tq *topicRequestQueue) update(itemPtr *topicRequestQueueItem, priority Uint64) {
	itemPtr.priority = priority
	heap.Fix(tq, itemPtr.index)
}
