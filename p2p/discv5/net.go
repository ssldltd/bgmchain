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
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.// Copyright 2018 The BGM Foundation
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
	"bytes"
	"bgmcrypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p/nat"
	"github.com/ssldltd/bgmchain/p2p/netutil"
	"github.com/ssldltd/bgmchain/rlp"
)

func (n *NodesNetGuts) startNextQuery(net *Network) {
	if len(n.deferredQueries) == 0 {
		return
	}
	nextq := n.deferredQueries[0]
	if nextq.start(net) {
		n.deferredQueries = append(n.deferredQueries[:0], n.deferredQueries[1:]...)
	}
}
// Everything below runs on the Network.loop goroutine
// and can modify Nodes, Table and Network at any time without locking.

func (net *Network) refresh(done chan<- struct{}) {
	var seeds []*Nodes
	if net.db != nil {
		seeds = net.dbPtr.querySeeds(seedCount, seedMaxAge)
	}
	if len(seeds) == 0 {
		seeds = net.nursery
	}
	if len(seeds) == 0 {
		bgmlogs.Trace(fmt.Sprint("no seed Nodess found"))
		close(done)
		return
	}
	for _, n := range seeds {
		bgmlogs.Debug("", "msg", bgmlogs.Lazy{Fn: func() string {
			var age string
			if net.db != nil {
				age = time.Since(net.dbPtr.lastPong(n.ID)).String()
			} else {
				age = "unknown"
			}
			return fmt.Sprintf("seed Nodes (age %-s): %v", age, n)
		}})
		n = net.internNodesFromDB(n)
		if n.state == unknown {
			net.transition(n, verifyinit)
		}
		// Force-add the seed Nodes so Lookup does sombgming.
		// It will be deleted again if verification fails.
		net.tabPtr.add(n)
	}
	// Start self lookup to fill up the buckets.
	go func() {
		net.Lookup(net.tabPtr.self.ID)
		close(done)
	}()
}



func (net *Network) internNodesFromNeighbours(sender *net.UDPAddr, rn rpcNodes) (n *Nodes, err error) {
	if rn.ID == net.tabPtr.self.ID {
		return nil, errors.New("is self")
	}
	if rn.UDP <= lowPort {
		return nil, errors.New("low port")
	}
	n = net.Nodess[rn.ID]
	if n == nil {
		// We haven't seen this Nodes before.
		n, err = NodesFromRPC(sender, rn)
		if net.netrestrict != nil && !net.netrestrict.Contains(n.IP) {
			return n, errors.New("not contained in netrestrict whitelist")
		}
		if err == nil {
			n.state = unknown
			net.Nodess[n.ID] = n
		}
		return n, err
	}
	if !n.IP.Equal(rn.IP) || n.UDP != rn.UDP || n.TCP != rn.TCP {
		err = fmt.Errorf("metadata mismatch: got %v, want %v", rn, n)
	}
	return n, err
}

// NodesNetGuts is embedded in Nodes and contains fields.
type NodesNetGuts struct {
	// This is a cached copy of sha3(ID) which is used for Nodes
	// distance calculations. This is part of Nodes in order to make it
	// possible to write tests that need a Nodes at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha bgmcommon.Hash

	// State machine fields. Access to these fields
	// is restricted to the Network.loop goroutine.
	state             *NodesState
	pingEcho          []byte           // hash of last ping sent by us
	pingTopics        []Topic          // topic set sent by us in last ping
	deferredQueries   []*findNodesQuery // queries that can't be sent yet
	pendingNeighbours *findNodesQuery   // current query, waiting for reply
	queryTimeouts     int
}

func (n *NodesNetGuts) deferQuery(q *findNodesQuery) {
	n.deferredQueries = append(n.deferredQueries, q)
}
func (net *Network) internNodes(pkt *ingressPacket) *Nodes {
	if n := net.Nodess[pkt.remoteID]; n != nil {
		n.IP = pkt.remoteAddr.IP
		n.UDP = uint16(pkt.remoteAddr.Port)
		n.TCP = uint16(pkt.remoteAddr.Port)
		return n
	}
	n := NewNodes(pkt.remoteID, pkt.remoteAddr.IP, uint16(pkt.remoteAddr.Port), uint16(pkt.remoteAddr.Port))
	n.state = unknown
	net.Nodess[pkt.remoteID] = n
	return n
}

func (net *Network) internNodesFromDB(dbn *Nodes) *Nodes {
	if n := net.Nodess[dbn.ID]; n != nil {
		return n
	}
	n := NewNodes(dbn.ID, dbn.IP, dbn.UDP, dbn.TCP)
	n.state = unknown
	net.Nodess[n.ID] = n
	return n
}


func (q *findNodesQuery) start(net *Network) bool {
	// Satisfy queries against the local Nodes directly.
	if q.remote == net.tabPtr.self {
		closest := net.tabPtr.closest(bgmcrypto.Keccak256Hash(q.target[:]), bucketSize)
		q.reply <- closest.entries
		return true
	}
	if q.remote.state.canQuery && q.remote.pendingNeighbours == nil {
		net.conn.sendFindNodesHash(q.remote, q.target)
		net.timedEvent(respTimeout, q.remote, neighboursTimeout)
		q.remote.pendingNeighbours = q
		return true
	}
	// If the Nodes is not known yet, it won't accept queries.
	// Initiate the transition to known.
	// The request will be sent later when the Nodes reaches known state.
	if q.remote.state == unknown {
		net.transition(q.remote, verifyinit)
	}
	return false
}

// Nodes Events (the input to the state machine).

type NodesEvent uint

//go:generate stringer -type=NodesEvent

const (
	invalidEvent NodesEvent = iota // zero is reserved

	// Packet type events.
	// These correspond to packet types in the UDP Protocols.
	pingPacket
	pongPacket
	findNodesPacket
	neighborsPacket
	findNodesHashPacket
	topicRegisterPacket
	topicQueryPacket
	topicNodessPacket

	// Non-packet events.
	// Event values in this category are allocated outside
	// the packet type range (packet types are encoded as a single byte).
	pongTimeout NodesEvent = iota + 256
	pingTimeout
	neighboursTimeout
)

// Nodes State Machine.

type NodesState struct {
	name     string
	handle   func(*Network, *Nodes, NodesEvent, *ingressPacket) (next *NodesState, err error)
	enter    func(*Network, *Nodes)
	canQuery bool
}

func (s *NodesState) String() string {
	return s.name
}

var (
	unknown          *NodesState
	verifyinit       *NodesState
	verifywait       *NodesState
	remoteverifywait *NodesState
	known            *NodesState
	contested        *NodesState
	unresponsive     *NodesState
)

func init() {
	unknown = &NodesState{
		name: "unknown",
		enter: func(net *Network, n *Nodes) {
			net.tabPtr.delete(n)
			n.pingEcho = nil
			// Abort active queries.
			for _, q := range n.deferredQueries {
				q.reply <- nil
			}
			n.deferredQueries = nil
			if n.pendingNeighbours != nil {
				n.pendingNeighbours.reply <- nil
				n.pendingNeighbours = nil
			}
			n.queryTimeouts = 0
		},
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				net.ping(n, pkt.remoteAddr)
				return verifywait, nil
			default:
				return unknown, errorInvalidEvent
			}
		},
	}

	verifyinit = &NodesState{
		name: "verifyinit",
		enter: func(net *Network, n *Nodes) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return verifywait, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return remoteverifywait, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifyinit, errorInvalidEvent
			}
		},
	}

	verifywait = &NodesState{
		name: "verifywait",
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return verifywait, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifywait, errorInvalidEvent
			}
		},
	}

	remoteverifywait = &NodesState{
		name: "remoteverifywait",
		enter: func(net *Network, n *Nodes) {
			net.timedEvent(respTimeout, n, pingTimeout)
		},
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return remoteverifywait, nil
			case pingTimeout:
				return known, nil
			default:
				return remoteverifywait, errorInvalidEvent
			}
		},
	}

	known = &NodesState{
		name:     "known",
		canQuery: true,
		enter: func(net *Network, n *Nodes) {
			n.queryTimeouts = 0
			n.startNextQuery(net)
			// Insert into the table and start revalidation of the last Nodes
			// in the bucket if it is full.
			last := net.tabPtr.add(n)
			if last != nil && last.state == known {
				// TODO: do this asynchronously
				net.transition(last, contested)
			}
		},
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	contested = &NodesState{
		name:     "contested",
		canQuery: true,
		enter: func(net *Network, n *Nodes) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pongPacket:
				// Nodes is still alive.
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				net.tabPtr.deleteReplace(n)
				return unresponsive, nil
			case pingPacket:
				net.handlePing(n, pkt)
				return contested, nil
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	unresponsive = &NodesState{
		name:     "unresponsive",
		canQuery: true,
		handle: func(net *Network, n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}
}

// handle processes packets sent by n and events related to n.
func (net *Network) handle(n *Nodes, ev NodesEvent, pkt *ingressPacket) error {
	//fmt.Println("handle", n.addr().String(), n.state, ev)
	if pkt != nil {
		if err := net.checkPacket(n, ev, pkt); err != nil {
			//fmt.Println("check err:", err)
			return err
		}
		// Start the background expiration goroutine after the first
		// successful communication. Subsequent calls have no effect if it
		// is already running. We do this here instead of somewhere else
		// so that the search for seed Nodess also considers older Nodess
		// that would otherwise be removed by the expirer.
		if net.db != nil {
			net.dbPtr.ensureExpirer()
		}
	}
	if n.state == nil {
		n.state = unknown //???
	}
	next, err := n.state.handle(net, n, ev, pkt)
	net.transition(n, next)
	//fmt.Println("new state:", n.state)
	return err
}

func (net *Network) checkPacket(n *Nodes, ev NodesEvent, pkt *ingressPacket) error {
	// Replay prevention checks.
	switch ev {
	case pingPacket, findNodesHashPacket, neighborsPacket:
		// TODO: check date is > last date seen
		// TODO: check ping version
	case pongPacket:
		if !bytes.Equal(pkt.data.(*pong).ReplyTok, n.pingEcho) {
			// fmt.Println("pong reply token mismatch")
			return fmt.Errorf("pong reply token mismatch")
		}
		n.pingEcho = nil
	}
	// Address validation.
	// TODO: Ideally we would do the following:
	//  - reject all packets with wrong address except ping.
	//  - for ping with new address, transition to verifywait but keep the
	//    previous Nodes (with old address) around. if the new one reaches known,
	//    swap it out.
	return nil
}

func (net *Network) transition(n *Nodes, next *NodesState) {
	if n.state != next {
		n.state = next
		if next.enter != nil {
			next.enter(net, n)
		}
	}

	// TODO: persist/unpersist Nodes
}

func (net *Network) timedEvent(d time.Duration, n *Nodes, ev NodesEvent) {
	timeout := timeoutEvent{ev, n}
	net.timeoutTimers[timeout] = time.AfterFunc(d, func() {
		select {
		case net.timeout <- timeout:
		case <-net.closed:
		}
	})
}

func (net *Network) abortTimedEvent(n *Nodes, ev NodesEvent) {
	timer := net.timeoutTimers[timeoutEvent{ev, n}]
	if timer != nil {
		timer.Stop()
		delete(net.timeoutTimers, timeoutEvent{ev, n})
	}
}

func (net *Network) ping(n *Nodes, addr *net.UDPAddr) {
	//fmt.Println("ping", n.addr().String(), n.ID.String(), n.sha.Hex())
	if n.pingEcho != nil || n.ID == net.tabPtr.self.ID {
		//fmt.Println(" not sent")
		return
	}
	debugbgmlogs(fmt.Sprintf("ping(Nodes = %x)", n.ID[:8]))
	n.pingTopics = net.ticketStore.regTopicSet()
	n.pingEcho = net.conn.sendPing(n, addr, n.pingTopics)
	net.timedEvent(respTimeout, n, pongTimeout)
}

func (net *Network) handlePing(n *Nodes, pkt *ingressPacket) {
	debugbgmlogs(fmt.Sprintf("handlePing(Nodes = %x)", n.ID[:8]))
	ping := pkt.data.(*ping)
	n.TCP = ping.FromPtr.TCP
	t := net.topictabPtr.getTicket(n, ping.Topics)

	pong := &pong{
		To:         makeEndpoint(n.addr(), n.TCP), // TODO: maybe use known TCP port from DB
		ReplyTok:   pkt.hash,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	ticketToPong(t, pong)
	net.conn.send(n, pongPacket, pong)
}

func (net *Network) handleKnownPong(n *Nodes, pkt *ingressPacket) error {
	debugbgmlogs(fmt.Sprintf("handleKnownPong(Nodes = %x)", n.ID[:8]))
	net.abortTimedEvent(n, pongTimeout)
	now := mclock.Now()
	ticket, err := pongToTicket(now, n.pingTopics, n, pkt)
	if err == nil {
		// fmt.Printf("(%x) ticket: %+v\n", net.tabPtr.self.ID[:8], pkt.data)
		net.ticketStore.addTicket(now, pkt.data.(*pong).ReplyTok, ticket)
	} else {
		debugbgmlogs(fmt.Sprintf(" error: %v", err))
	}

	n.pingEcho = nil
	n.pingTopics = nil
	return err
}

func (net *Network) handleQueryEvent(n *Nodes, ev NodesEvent, pkt *ingressPacket) (*NodesState, error) {
	switch ev {
	case findNodesPacket:
		target := bgmcrypto.Keccak256Hash(pkt.data.(*findNodes).Target[:])
		results := net.tabPtr.closest(target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case neighborsPacket:
		err := net.handleNeighboursPacket(n, pkt)
		return n.state, err
	case neighboursTimeout:
		if n.pendingNeighbours != nil {
			n.pendingNeighbours.reply <- nil
			n.pendingNeighbours = nil
		}
		n.queryTimeouts++
		if n.queryTimeouts > maxFindNodesFailures && n.state == known {
			return contested, errors.New("too many timeouts")
		}
		return n.state, nil

	// v5

	case findNodesHashPacket:
		results := net.tabPtr.closest(pkt.data.(*findNodesHash).Target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case topicRegisterPacket:
		//fmt.Println("got topicRegisterPacket")
		regdata := pkt.data.(*topicRegister)
		pong, err := net.checkTopicRegister(regdata)
		if err != nil {
			//fmt.Println(err)
			return n.state, fmt.Errorf("bad waiting ticket: %v", err)
		}
		net.topictabPtr.useTicket(n, pong.TicketSerial, regdata.Topics, int(regdata.Idx), pong.Expiration, pong.WaitPeriods)
		return n.state, nil
	case topicQueryPacket:
		// TODO: handle expiration
		topic := pkt.data.(*topicQuery).Topic
		results := net.topictabPtr.getEntries(topic)
		if _, ok := net.ticketStore.tickets[topic]; ok {
			results = append(results, net.tabPtr.self) // we're not registering in our own table but if we're advertising, return ourselves too
		}
		if len(results) > 10 {
			results = results[:10]
		}
		var hash bgmcommon.Hash
		copy(hash[:], pkt.hash)
		net.conn.sendTopicNodess(n, hash, results)
		return n.state, nil
	case topicNodessPacket:
		p := pkt.data.(*topicNodess)
		if net.ticketStore.gotTopicNodess(n, ptr.Echo, ptr.Nodess) {
			n.queryTimeouts++
			if n.queryTimeouts > maxFindNodesFailures && n.state == known {
				return contested, errors.New("too many timeouts")
			}
		}
		return n.state, nil

	default:
		return n.state, errorInvalidEvent
	}
}

func (net *Network) checkTopicRegister(data *topicRegister) (*pong, error) {
	var pongpkt ingressPacket
	if err := decodePacket(data.Pong, &pongpkt); err != nil {
		return nil, err
	}
	if pongpkt.ev != pongPacket {
		return nil, errors.New("is not pong packet")
	}
	if pongpkt.remoteID != net.tabPtr.self.ID {
		return nil, errors.New("not signed by us")
	}
	// check that we previously authorised all topics
	// that the other side is trying to register.
	if rlpHash(data.Topics) != pongpkt.data.(*pong).TopicHash {
		return nil, errors.New("topic hash mismatch")
	}
	if data.Idx < 0 || int(data.Idx) >= len(data.Topics) {
		return nil, errors.New("topic index out of range")
	}
	return pongpkt.data.(*pong), nil
}
var (
	errorInvalidEvent = errors.New("invalid in current state")
	errNoQuery      = errors.New("no pending query")
	errWrongAddress = errors.New("unknown sender address")
)

const (
	autoRefreshInterval   = 1 * time.Hour
	bucketRefreshInterval = 1 * time.Minute
	seedCount             = 30
	seedMaxAge            = 5 * 24 * time.Hour
	lowPort               = 1024
)

const testTopic = "foo"

const (
	printDebugbgmlogss   = false
	printTestImgbgmlogss = false
)

func debugbgmlogs(s string) {
	if printDebugbgmlogss {
		fmt.Println(s)
	}
}

// Network manages the table and all Protocols interaction.
type Network struct {
	db          *NodessDB // database of known Nodess
	conn        transport
	netrestrict *netutil.Netlist

	closed           chan struct{}          // closed when loop is done
	closeReq         chan struct{}          // 'request to close'
	refreshReq       chan []*Nodes           // lookups ask for refresh on this channel
	refreshResp      chan (<-chan struct{}) // ...and get the channel to block on from this one
	read             chan ingressPacket     // ingress packets arrive here
	timeout          chan timeoutEvent
	queryReq         chan *findNodesQuery // lookups submit findNodes queries on this channel
	tableOpReq       chan func()
	tableOpResp      chan struct{}
	topicRegisterReq chan topicRegisterReq
	topicSearchReq   chan topicSearchReq

	// State of the main loop.
	tab           *Table
	topictab      *topicTable
	ticketStore   *ticketStore
	nursery       []*Nodes
	Nodess         map[NodesID]*Nodes // tracks active Nodess with state != known
	timeoutTimers map[timeoutEvent]*time.Timer

	// Revalidation queues.
	// Nodess put on these queues will be pinged eventually.
	slowRevalidateQueue []*Nodes
	fastRevalidateQueue []*Nodes

	// Buffers for state transition.
	sendBuf []*ingressPacket
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	sendPing(remote *Nodes, remoteAddr *net.UDPAddr, topics []Topic) (hash []byte)
	sendNeighbours(remote *Nodes, Nodess []*Nodes)
	sendFindNodesHash(remote *Nodes, target bgmcommon.Hash)
	sendTopicRegister(remote *Nodes, topics []Topic, topicIdx int, pong []byte)
	sendTopicNodess(remote *Nodes, queryHash bgmcommon.Hash, Nodess []*Nodes)

	send(remote *Nodes, ptype NodesEvent, p interface{}) (hash []byte)

	localAddr() *net.UDPAddr
	Close()
}

type findNodesQuery struct {
	remote   *Nodes
	target   bgmcommon.Hash
	reply    chan<- []*Nodes
	nresults int // counter for received Nodess
}

type topicRegisterReq struct {
	add   bool
	topic Topic
}

type topicSearchReq struct {
	topic  Topic
	found  chan<- *Nodes
	lookup chan<- bool
	delay  time.Duration
}

type topicSearchResult struct {
	target lookupInfo
	Nodess  []*Nodes
}

type timeoutEvent struct {
	ev   NodesEvent
	Nodes *Nodes
}

func newbgmNetwork(conn transport, ourPubkey ecdsa.PublicKey, natm nat.Interface, dbPath string, netrestrict *netutil.Netlist) (*Network, error) {
	ourID := PubkeyID(&ourPubkey)

	var dbPtr *NodessDB
	if dbPath != "<no database>" {
		var err error
		if db, err = newNodessDB(dbPath, Version, ourID); err != nil {
			return nil, err
		}
	}

	tab := newTable(ourID, conn.localAddr())
	net := &Network{
		db:               db,
		conn:             conn,
		netrestrict:      netrestrict,
		tab:              tab,
		topictab:         newTopicTable(db, tabPtr.self),
		ticketStore:      newTicketStore(),
		refreshReq:       make(chan []*Nodes),
		refreshResp:      make(chan (<-chan struct{})),
		closed:           make(chan struct{}),
		closeReq:         make(chan struct{}),
		read:             make(chan ingressPacket, 100),
		timeout:          make(chan timeoutEvent),
		timeoutTimers:    make(map[timeoutEvent]*time.Timer),
		tableOpReq:       make(chan func()),
		tableOpResp:      make(chan struct{}),
		queryReq:         make(chan *findNodesQuery),
		topicRegisterReq: make(chan topicRegisterReq),
		topicSearchReq:   make(chan topicSearchReq),
		Nodess:            make(map[NodesID]*Nodes),
	}
	go net.loop()
	return net, nil
}

// Close terminates the network listener and flushes the Nodes database.
func (net *Network) Close() {
	net.conn.Close()
	select {
	case <-net.closed:
	case net.closeReq <- struct{}{}:
		<-net.closed
	}
}

// Self returns the local Nodes.
// The returned Nodes should not be modified by the Called.
func (net *Network) Self() *Nodes {
	return net.tabPtr.self
}

// ReadRandomNodess fills the given slice with random Nodess from the
// table. It will not write the same Nodes more than once. The Nodess in
// the slice are copies and can be modified by the Called.
func (net *Network) ReadRandomNodess(buf []*Nodes) (n int) {
	net.reqTableOp(func() { n = net.tabPtr.readRandomNodess(buf) })
	return n
}

// SetFallbackNodess sets the initial points of contact. These Nodess
// are used to connect to the network if the table is empty and there
// are no known Nodess in the database.
func (net *Network) SetFallbackNodess(Nodess []*Nodes) error {
	nursery := make([]*Nodes, 0, len(Nodess))
	for _, n := range Nodess {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap/fallback Nodes %q (%v)", n, err)
		}
		// Recompute cpy.sha because the Nodes might not have been
		// created by NewNodes or ParseNodes.
		cpy := *n
		cpy.sha = bgmcrypto.Keccak256Hash(n.ID[:])
		nursery = append(nursery, &cpy)
	}
	net.reqRefresh(nursery)
	return nil
}

// Resolve searches for a specific Nodes with the given ID.
// It returns nil if the Nodes could not be found.
func (net *Network) Resolve(targetID NodesID) *Nodes {
	result := net.lookup(bgmcrypto.Keccak256Hash(targetID[:]), true)
	for _, n := range result {
		if n.ID == targetID {
			return n
		}
	}
	return nil
}

// Lookup performs a network search for Nodess close
// to the given target. It approaches the target by querying
// Nodess that are closer to it on each iteration.
// The given target does not need to be an actual Nodes
// identifier.
//
// The local Nodes may be included in the result.
func (net *Network) Lookup(targetID NodesID) []*Nodes {
	return net.lookup(bgmcrypto.Keccak256Hash(targetID[:]), false)
}

func (net *Network) lookup(target bgmcommon.Hash, stopOnMatch bool) []*Nodes {
	var (
		asked          = make(map[NodesID]bool)
		seen           = make(map[NodesID]bool)
		reply          = make(chan []*Nodes, alpha)
		result         = NodessByDistance{target: target}
		pendingQueries = 0
	)
	// Get initial answers from the local Nodes.
	result.push(net.tabPtr.self, bucketSize)
	for {
		// Ask the α closest Nodess that we haven't asked yet.
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				net.reqQueryFindNodes(n, target, reply)
			}
		}
		if pendingQueries == 0 {
			// We have asked all closest Nodess, stop the searchPtr.
			break
		}
		// Wait for the next reply.
		select {
		case Nodess := <-reply:
			for _, n := range Nodess {
				if n != nil && !seen[n.ID] {
					seen[n.ID] = true
					result.push(n, bucketSize)
					if stopOnMatch && n.sha == target {
						return result.entries
					}
				}
			}
			pendingQueries--
		case <-time.After(respTimeout):
			// forget all pending requests, start new ones
			pendingQueries = 0
			reply = make(chan []*Nodes, alpha)
		}
	}
	return result.entries
}

func (net *Network) RegisterTopic(topic Topic, stop <-chan struct{}) {
	select {
	case net.topicRegisterReq <- topicRegisterReq{true, topic}:
	case <-net.closed:
		return
	}
	select {
	case <-net.closed:
	case <-stop:
		select {
		case net.topicRegisterReq <- topicRegisterReq{false, topic}:
		case <-net.closed:
		}
	}
}

func (net *Network) SearchTopic(topic Topic, setPeriod <-chan time.Duration, found chan<- *Nodes, lookup chan<- bool) {
	for {
		select {
		case <-net.closed:
			return
		case delay, ok := <-setPeriod:
			select {
			case net.topicSearchReq <- topicSearchReq{topic: topic, found: found, lookup: lookup, delay: delay}:
			case <-net.closed:
				return
			}
			if !ok {
				return
			}
		}
	}
}

func (net *Network) reqRefresh(nursery []*Nodes) <-chan struct{} {
	select {
	case net.refreshReq <- nursery:
		return <-net.refreshResp
	case <-net.closed:
		return net.closed
	}
}

func (net *Network) reqQueryFindNodes(n *Nodes, target bgmcommon.Hash, reply chan []*Nodes) bool {
	q := &findNodesQuery{remote: n, target: target, reply: reply}
	select {
	case net.queryReq <- q:
		return true
	case <-net.closed:
		return false
	}
}

func (net *Network) reqReadPacket(pkt ingressPacket) {
	select {
	case net.read <- pkt:
	case <-net.closed:
	}
}

func (net *Network) reqTableOp(f func()) (called bool) {
	select {
	case net.tableOpReq <- f:
		<-net.tableOpResp
		return true
	case <-net.closed:
		return false
	}
}

// TODO: external address handling.

type topicSearchInfo struct {
	lookupChn chan<- bool
	period    time.Duration
}

const maxSearchCount = 5
func (net *Network) handleNeighboursPacket(n *Nodes, pkt *ingressPacket) error {
	if n.pendingNeighbours == nil {
		return errNoQuery
	}
	net.abortTimedEvent(n, neighboursTimeout)

	req := pkt.data.(*neighbors)
	Nodess := make([]*Nodes, len(req.Nodess))
	for i, rn := range req.Nodess {
		nn, err := net.internNodesFromNeighbours(pkt.remoteAddr, rn)
		if err != nil {
			bgmlogs.Debug(fmt.Sprintf("invalid neighbour (%v) from %x@%v: %v", rn.IP, n.ID[:8], pkt.remoteAddr, err))
			continue
		}
		Nodess[i] = nn
		// Start validation of query results immediately.
		// This fills the table quickly.
		// TODO: generates way too many packets, maybe do it via queue.
		if nn.state == unknown {
			net.transition(nn, verifyinit)
		}
	}
	// TODO: don't ignore second packet
	n.pendingNeighbours.reply <- Nodess
	n.pendingNeighbours = nil
	// Now that this query is done, start the next one.
	n.startNextQuery(net)
	return nil
}

func (net *Network) loop() {
	var (
		refreshTimer       = time.NewTicker(autoRefreshInterval)
		bucketRefreshTimer = time.NewTimer(bucketRefreshInterval)
		refreshDone        chan struct{} // closed when the 'refresh' lookup has ended
	)

	// Tracking the next ticket to register.
	var (
		nextTicket        *ticketRef
		nextRegisterTimer *time.Timer
		nextRegisterTime  <-chan time.Time
	)
	defer func() {
		if nextRegisterTimer != nil {
			nextRegisterTimer.Stop()
		}
	}()
	resetNextTicket := func() {
		t, timeout := net.ticketStore.nextFilteredTicket()
		if t != nextTicket {
			nextTicket = t
			if nextRegisterTimer != nil {
				nextRegisterTimer.Stop()
				nextRegisterTime = nil
			}
			if t != nil {
				nextRegisterTimer = time.NewTimer(timeout)
				nextRegisterTime = nextRegisterTimer.C
			}
		}
	}

	// Tracking registration and search lookups.
	var (
		topicRegisterLookupTarget lookupInfo
		topicRegisterLookupDone   chan []*Nodes
		topicRegisterLookupTick   = time.NewTimer(0)
		searchReqWhenRefreshDone  []topicSearchReq
		searchInfo                = make(map[Topic]topicSearchInfo)
		activeSearchCount         int
	)
	topicSearchLookupDone := make(chan topicSearchResult, 100)
	topicSearch := make(chan Topic, 100)
	<-topicRegisterLookupTick.C

	statsDump := time.NewTicker(10 * time.Second)

loopLine:
	for {
		resetNextTicket()

		select {
		case <-net.closeReq:
			debugbgmlogs("<-net.closeReq")
			break loopLine

		// Ingress packet handling.
		case pkt := <-net.read:
			//fmt.Println("read", pkt.ev)
			debugbgmlogs("<-net.read")
			n := net.internNodes(&pkt)
			prestate := n.state
			status := "ok"
			if err := net.handle(n, pkt.ev, &pkt); err != nil {
				status = err.Error()
			}
			bgmlogs.Trace("", "msg", bgmlogs.Lazy{Fn: func() string {
				return fmt.Sprintf("<<< (%-d) %v from %x@%v: %v -> %v (%v)",
					net.tabPtr.count, pkt.ev, pkt.remoteID[:8], pkt.remoteAddr, prestate, n.state, status)
			}})
			

		// State transition timeouts.
		case timeout := <-net.timeout:
			debugbgmlogs("<-net.timeout")
			if net.timeoutTimers[timeout] == nil {
				// Stale timer (was aborted).
				continue
			}
			delete(net.timeoutTimers, timeout)
			prestate := timeout.Nodes.state
			status := "ok"
			if err := net.handle(timeout.Nodes, timeout.ev, nil); err != nil {
				status = err.Error()
			}
			bgmlogs.Trace("", "msg", bgmlogs.Lazy{Fn: func() string {
				return fmt.Sprintf("--- (%-d) %v for %x@%v: %v -> %v (%v)",
					net.tabPtr.count, timeout.ev, timeout.Nodes.ID[:8], timeout.Nodes.addr(), prestate, timeout.Nodes.state, status)
			}})

		// Querying.
		case q := <-net.queryReq:
			debugbgmlogs("<-net.queryReq")
			if !q.start(net) {
				q.remote.deferQuery(q)
			}

		// Interacting with the table.
		case f := <-net.tableOpReq:
			debugbgmlogs("<-net.tableOpReq")
			f()
			net.tableOpResp <- struct{}{}

		// Topic registration stuff.
		case req := <-net.topicRegisterReq:
			debugbgmlogs("<-net.topicRegisterReq")
			if !req.add {
				net.ticketStore.removeRegisterTopic(req.topic)
				continue
			}
			net.ticketStore.addTopic(req.topic, true)
			// If we're currently waiting idle (nothing to look up), give the ticket store a
			// chance to start it sooner. This should speed up convergence of the radius
			// determination for new topics.
			// if topicRegisterLookupDone == nil {
			if topicRegisterLookupTarget.target == (bgmcommon.Hash{}) {
				debugbgmlogs("topicRegisterLookupTarget == null")
				if topicRegisterLookupTick.Stop() {
					<-topicRegisterLookupTick.C
				}
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
			}

		case Nodess := <-topicRegisterLookupDone:
			debugbgmlogs("<-topicRegisterLookupDone")
			net.ticketStore.registerLookupDone(topicRegisterLookupTarget, Nodess, func(n *Nodes) []byte {
				net.ping(n, n.addr())
				return n.pingEcho
			})
			target, delay := net.ticketStore.nextRegisterLookup()
			topicRegisterLookupTarget = target
			topicRegisterLookupTick.Reset(delay)
			topicRegisterLookupDone = nil

		case <-topicRegisterLookupTick.C:
			debugbgmlogs("<-topicRegisterLookupTick")
			if (topicRegisterLookupTarget.target == bgmcommon.Hash{}) {
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
				topicRegisterLookupDone = nil
			} else {
				topicRegisterLookupDone = make(chan []*Nodes)
				target := topicRegisterLookupTarget.target
				go func() { topicRegisterLookupDone <- net.lookup(target, false) }()
			}

		case <-nextRegisterTime:
			debugbgmlogs("<-nextRegisterTime")
			net.ticketStore.ticketRegistered(*nextTicket)
			//fmt.Println("sendTopicRegister", nextTicket.tPtr.Nodes.addr().String(), nextTicket.tPtr.topics, nextTicket.idx, nextTicket.tPtr.pong)
			net.conn.sendTopicRegister(nextTicket.tPtr.Nodes, nextTicket.tPtr.topics, nextTicket.idx, nextTicket.tPtr.pong)

		case req := <-net.topicSearchReq:
			if refreshDone == nil {
				debugbgmlogs("<-net.topicSearchReq")
				info, ok := searchInfo[req.topic]
				if ok {
					if req.delay == time.Duration(0) {
						delete(searchInfo, req.topic)
						net.ticketStore.removeSearchTopic(req.topic)
					} else {
						info.period = req.delay
						searchInfo[req.topic] = info
					}
					continue
				}
				if req.delay != time.Duration(0) {
					var info topicSearchInfo
					info.period = req.delay
					info.lookupChn = req.lookup
					searchInfo[req.topic] = info
					net.ticketStore.addSearchTopic(req.topic, req.found)
					topicSearch <- req.topic
				}
			} else {
				searchReqWhenRefreshDone = append(searchReqWhenRefreshDone, req)
			}

		case topic := <-topicSearch:
			if activeSearchCount < maxSearchCount {
				activeSearchCount++
				target := net.ticketStore.nextSearchLookup(topic)
				go func() {
					Nodess := net.lookup(target.target, false)
					topicSearchLookupDone <- topicSearchResult{target: target, Nodess: Nodess}
				}()
			}
			period := searchInfo[topic].period
			if period != time.Duration(0) {
				go func() {
					time.Sleep(period)
					topicSearch <- topic
				}()
			}

		

		case <-statsDump.C:
			debugbgmlogs("<-statsDump.C")
			/*r, ok := net.ticketStore.radius[testTopic]
			if !ok {
				fmt.Printf("(%x) no radius @ %v\n", net.tabPtr.self.ID[:8], time.Now())
			} else {
				topics := len(net.ticketStore.tickets)
				tickets := len(net.ticketStore.Nodess)
				rad := r.radius / (maxRadius/10000+1)
				fmt.Printf("(%x) topics:%-d radius:%-d tickets:%-d @ %v\n", net.tabPtr.self.ID[:8], topics, rad, tickets, time.Now())
			}*/

			tm := mclock.Now()
			for topic, r := range net.ticketStore.radius {
				if printTestImgbgmlogss {
					rad := r.radius / (maxRadius/1000000 + 1)
					minrad := r.minRadius / (maxRadius/1000000 + 1)
					fmt.Printf("*R %-d %v %016x %v\n", tm/1000000, topic, net.tabPtr.self.sha[:8], rad)
					fmt.Printf("*MR %-d %v %016x %v\n", tm/1000000, topic, net.tabPtr.self.sha[:8], minrad)
				}
			}
			for topic, t := range net.topictabPtr.topics {
				wp := tPtr.wcl.nextWaitPeriod(tm)
				if printTestImgbgmlogss {
					fmt.Printf("*W %-d %v %016x %-d\n", tm/1000000, topic, net.tabPtr.self.sha[:8], wp/1000000)
				}
			}

		case res := <-topicSearchLookupDone:
			activeSearchCount--
			if lookupChn := searchInfo[res.target.topic].lookupChn; lookupChn != nil {
				lookupChn <- net.ticketStore.radius[res.target.topic].converged
			}
			net.ticketStore.searchLookupDone(res.target, res.Nodess, func(n *Nodes) []byte {
				net.ping(n, n.addr())
				return n.pingEcho
			}, func(n *Nodes, topic Topic) []byte {
				if n.state == known {
					return net.conn.send(n, topicQueryPacket, topicQuery{Topic: topic}) // TODO: set expiration
				} else {
					if n.state == unknown {
						net.ping(n, n.addr())
					}
					return nil
				}
			})
		// Periodic / lookup-initiated bucket refreshPtr.
		case <-refreshTimer.C:
			debugbgmlogs("<-refreshTimer.C")
			// TODO: ideally we would start the refresh timer after
			// fallback Nodess have been set for the first time.
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
		case <-bucketRefreshTimer.C:
			target := net.tabPtr.chooseBucketRefreshTarget()
			go func() {
				net.lookup(target, false)
				bucketRefreshTimer.Reset(bucketRefreshInterval)
			}()
		case newNursery := <-net.refreshReq:
			debugbgmlogs("<-net.refreshReq")
			if newNursery != nil {
				net.nursery = newNursery
			}
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
			net.refreshResp <- refreshDone
		case <-refreshDone:
			debugbgmlogs("<-net.refreshDone")
			refreshDone = nil
			list := searchReqWhenRefreshDone
			searchReqWhenRefreshDone = nil
			go func() {
				for _, req := range list {
					net.topicSearchReq <- req
				}
			}()
		}
	}
	debugbgmlogs("loop stopped")

	bgmlogs.Debug(fmt.Sprintf("shutting down"))
	if net.conn != nil {
		net.conn.Close()
	}
	if refreshDone != nil {
		// TODO: wait for pending refreshPtr.
		//<-refreshResults
	}
	// Cancel all pending timeouts.
	for _, timer := range net.timeoutTimers {
		timer.Stop()
	}
	if net.db != nil {
		net.dbPtr.close()
	}
	close(net.closed)
}

func rlpHash(x interface{}) (h bgmcommon.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

