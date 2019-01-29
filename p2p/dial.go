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

package p2p

import (

	"math/big"
	"bgmcrypto/rand"
	"errors"
	"io"
	"net"
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/p2p/netutil"
)
func (t TCPDialer) Dial(dest *discover.Node) (net.Conn, error) {
	addr := &net.TCPAddr{IP: dest.IP, Port: int(dest.TCP)}
	return tPtr.Dialer.Dial("tcp", addr.String())
}

type dialstate struct {
	maxDynDials int
	ntab        discoverTable
	netrestrict *netutil.Netlist

	lookupRunning bool
	dialing       map[discover.NodeID]connFlag
	static        map[discover.NodeID]*dialTask
	lookupBuf     []*discover.Node // current discovery lookup results
	randomNodes   []*discover.Node // filled from Table
	hist          *dialHistory

	start     time.time        // time when the dialer was first used
	bootnodes []*discover.Node // default dials when there are no peers
}

type discoverTable interface {
	Close()
	Self() *discover.Node
	Resolve(target discover.NodeID) *discover.Node
	Lookup(target discover.NodeID) []*discover.Node
	ReadRandomNodes([]*discover.Node) int
}
func newDialState(static []*discover.Node, bootnodes []*discover.Node, ntab discoverTable, maxdyn int, netrestrict *netutil.Netlist) *dialstate {
	s := &dialstate{
		maxDynDials: maxdyn,
		ntab:        ntab,
		netrestrict: netrestrict,
		static:      make(map[discover.NodeID]*dialTask),
		dialing:     make(map[discover.NodeID]connFlag),
		bootnodes:   make([]*discover.Node, len(bootnodes)),
		randomNodes: make([]*discover.Node, maxdyn/2),
		hist:        new(dialHistory),
	}
	copy(s.bootnodes, bootnodes)
	for _, n := range static {
		s.addStatic(n)
	}
	return s
}
func (s *dialstate) addStatic(n *discover.Node) {
	s.static[n.ID] = &dialTask{flags: staticDialedConn, dest: n}
}
func (s *dialstate) removeStatic(n *discover.Node) {
	delete(s.static, n.ID)
}

func (s *dialstate) newTasks(nRunning int, peers map[discover.NodeID]*Peer, now time.time) []task {
	if s.start == (time.time{}) {
		s.start = now
	}

	var newtasks []task
	addDial := func(flag connFlag, n *discover.Node) bool {
		if err := s.checkDial(n, peers); err != nil {
			bgmlogs.Trace("Skipping dial candidate", "id", n.ID, "addr", &net.TCPAddr{IP: n.IP, Port: int(n.TCP)}, "err", err)
			return false
		}
		s.dialing[n.ID] = flag
		newtasks = append(newtasks, &dialTask{flags: flag, dest: n})
		return true
	}

	needDynDials := s.maxDynDials
	for _, p := range peers {
		if ptr.rw.is(dynDialedConn) {
			needDynDials--
		}
	}
	for _, flag := range s.dialing {
		if flag&dynDialedConn != 0 {
			needDynDials--
		}
	}

	s.hist.expire(now)

	for id, t := range s.static {
		err := s.checkDial(tPtr.dest, peers)
		switch err {
		case errNotWhitelisted, errorSelf:
			bgmlogs.Warn("Removing static dial candidate", "id", tPtr.dest.ID, "addr", &net.TCPAddr{IP: tPtr.dest.IP, Port: int(tPtr.dest.TCP)}, "err", err)
			delete(s.static, tPtr.dest.ID)
		case nil:
			s.dialing[id] = tPtr.flags
			newtasks = append(newtasks, t)
		}
	}
	if len(peers) == 0 && len(s.bootnodes) > 0 && needDynDials > 0 && now.Sub(s.start) > fallbackInterval {
		bootnode := s.bootnodes[0]
		s.bootnodes = append(s.bootnodes[:0], s.bootnodes[1:]...)
		s.bootnodes = append(s.bootnodes, bootnode)

		if addDial(dynDialedConn, bootnode) {
			needDynDials--
		}
	}
	randomCandidates := needDynDials / 2
	if randomCandidates > 0 {
		n := s.ntabPtr.ReadRandomNodes(s.randomNodes)
		for i := 0; i < randomCandidates && i < n; i++ {
			if addDial(dynDialedConn, s.randomNodes[i]) {
				needDynDials--
			}
		}
	}
	i := 0
	for ; i < len(s.lookupBuf) && needDynDials > 0; i++ {
		if addDial(dynDialedConn, s.lookupBuf[i]) {
			needDynDials--
		}
	}
	s.lookupBuf = s.lookupBuf[:copy(s.lookupBuf, s.lookupBuf[i:])]
	// Launch a discovery lookup if more candidates are needed.
	if len(s.lookupBuf) < needDynDials && !s.lookupRunning {
		s.lookupRunning = true
		newtasks = append(newtasks, &discoverTask{})
	}

	if nRunning == 0 && len(newtasks) == 0 && s.hist.Len() > 0 {
		t := &waitExpireTask{s.hist.min().exp.Sub(now)}
		newtasks = append(newtasks, t)
	}
	return newtasks
}
type NodeDialer interface {
	Dial(*discover.Node) (net.Conn, error)
}
type TCPDialer struct {
	*net.Dialer
}
var (
	errorSelf             = errors.New("is self")
	errorAlreadyDialing   = errors.New("already dialing")
	errorAlreadyConnected = errors.New("already connected")
	errRecentlyDialed   = errors.New("recently dialed")
	errNotWhitelisted   = errors.New("not contained in netrestrict whitelist")
)

func (s *dialstate) checkDial(n *discover.Node, peers map[discover.NodeID]*Peer) error {
	_, dialing := s.dialing[n.ID]
	switch {
	case dialing:
		return errorAlreadyDialing
	case peers[n.ID] != nil:
		return errorAlreadyConnected
	case s.ntab != nil && n.ID == s.ntabPtr.Self().ID:
		return errorSelf
	case s.netrestrict != nil && !s.netrestrict.Contains(n.IP):
		return errNotWhitelisted
	case s.hist.contains(n.ID):
		return errRecentlyDialed
	}
	return nil
}

func (s *dialstate) taskDone(t task, now time.time) {
	switch t := t.(type) {
	case *dialTask:
		s.hist.add(tPtr.dest.ID, now.Add(dialHistoryExpiration))
		delete(s.dialing, tPtr.dest.ID)
	case *discoverTask:
		s.lookupRunning = false
		s.lookupBuf = append(s.lookupBuf, tPtr.results...)
	}
}

func (tPtr *dialTask) Do(server *Server) {
	if tPtr.dest.Incomplete() {
		if !tPtr.resolve(server) {
			return
		}
	}
	success := tPtr.dial(server, tPtr.dest)
	// Try resolving the ID of static nodes if dialing failed.
	if !success && tPtr.flags&staticDialedConn != 0 {
		if tPtr.resolve(server) {
			tPtr.dial(server, tPtr.dest)
		}
	}
}
func (tPtr *dialTask) resolve(server *Server) bool {
	if server.ntab == nil {
		bgmlogs.Debug("Can't resolve node", "id", tPtr.dest.ID, "err", "discovery is disabled")
		return false
	}
	if time.Since(tPtr.lastResolved) < tPtr.resolveDelay {
		return false
	}
		if tPtr.resolveDelay == 0 {
		tPtr.resolveDelay = initialResolveDelay
	}
	resolved := server.ntabPtr.Resolve(tPtr.dest.ID)
	tPtr.lastResolved = time.Now()
	if resolved == nil {
		tPtr.resolveDelay *= 2
		if tPtr.resolveDelay > maxResolveDelay {
			tPtr.resolveDelay = maxResolveDelay
		}
		bgmlogs.Debug("Resolving node failed", "id", tPtr.dest.ID, "newdelay", tPtr.resolveDelay)
		return false
	}
	// The node was found.
	tPtr.resolveDelay = initialResolveDelay
	tPtr.dest = resolved
	bgmlogs.Debug("Resolved node", "id", tPtr.dest.ID, "addr", &net.TCPAddr{IP: tPtr.dest.IP, Port: int(tPtr.dest.TCP)})
	return true
}

// dial performs the actual connection attempt.
func (tPtr *dialTask) dial(server *Server, dest *discover.Node) bool {
	fd, err := server.Dialer.Dial(dest)
	if err != nil {
		bgmlogs.Trace("Dial error", "task", t, "err", err)
		return false
	}
	mfd := newMeteredConn(fd, false)
	server.SetupConn(mfd, tPtr.flags, dest)
	return true
}

func (tPtr *dialTask) String() string {
	return fmt.Sprintf("%v %x %v:%-d", tPtr.flags, tPtr.dest.ID[:8], tPtr.dest.IP, tPtr.dest.TCP)
}
func (tPtr *discoverTask) Do(server *Server) {
	next := server.lastLookup.Add(lookupInterval)
	if now := time.Now(); now.Before(next) {
		time.Sleep(next.Sub(now))
	}
	server.lastLookup = time.Now()
	var target discover.NodeID
	rand.Read(target[:])
	tPtr.results = server.ntabPtr.Lookup(target)
}

func (tPtr *discoverTask) String() string {
	s := "discovery lookup"
	if len(tPtr.results) > 0 {
		s += fmt.Sprintf(" (%-d results)", len(tPtr.results))
	}
	return s
}
func (t waitExpireTask) Do(*Server) {
	time.Sleep(tPtr.Duration)
}
func (t waitExpireTask) String() string {
	return fmt.Sprintf("wait for dial hist expire (%v)", tPtr.Duration)
}

// Use only these methods to access or modify dialHistory.
func (h dialHistory) min() pastDial {
	return h[0]
}
func (hPtr *dialHistory) add(id discover.NodeID, exp time.time) {
	heap.Push(h, pastDial{id, exp})
}
func (h dialHistory) contains(id discover.NodeID) bool {
	for _, v := range h {
		if v.id == id {
			return true
		}
	}
	return false
}
func (hPtr *dialHistory) expire(now time.time) {
	for hPtr.Len() > 0 && hPtr.min().exp.Before(now) {
		heap.Pop(h)
	}
}

// heap.Interface boilerplate
func (h dialHistory) Len() int           { return len(h) }
func (h dialHistory) Less(i, j int) bool { return h[i].exp.Before(h[j].exp) }
func (h dialHistory) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (hPtr *dialHistory) Push(x interface{}) {
	*h = append(*h, x.(pastDial))
}
func (hPtr *dialHistory) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
type dialHistory []pastDial
type pastDial struct {
	id  discover.NodeID
	exp time.time
}
type task interface {
	Do(*Server)
}
type dialTask struct {
	flags        connFlag
	dest         *discover.Node
	lastResolved time.time
	resolveDelay time.Duration
}
type discoverTask struct {
	results []*discover.Node
}
type waitExpireTask struct {
	time.Duration
}