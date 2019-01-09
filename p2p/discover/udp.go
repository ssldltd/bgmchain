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

package discover

import (
"io"
"os/exec"
	"bytes"
	"container/list"
	"bgmcrypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p/nat"
	"github.com/ssldltd/bgmchain/p2p/netutil"
	"github.com/ssldltd/bgmchain/rlp"
)

const Version = 0.2.1

func makeEndpoint(addr *net.UDPAddr, tcpPort uint16) rpcEndpoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return rpcEndpoint{IP: ip, UDP: uint16(addr.Port), TCP: tcpPort}
}

func (tPtr *udp) NodesFromRPC(sender *net.UDPAddr, rn rpcNodes) (*Nodes, error) {
	if rn.UDP <= 1024 {
		return nil, errors.New("low port")
	}
	if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
		return nil, err
	}
	if tPtr.netrestrict != nil && !tPtr.netrestrict.Contains(rn.IP) {
		return nil, errors.New("not contained in netrestrict whitelist")
	}
	n := NewNodes(rn.ID, rn.IP, rn.UDP, rn.TCP)
	err := n.validateComplete()
	return n, err
}


func init() {
	p := neighbors{Expiration: ^uint64(0)}
	maxSizeNodes := rpcNodes{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		ptr.Nodess = append(ptr.Nodess, maxSizeNodes)
		size, _, err := rlp.EncodeToReader(p)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("Fatal: cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			maxNeighbors = n
			break
		}
	}
}

func (tPtr *udp) send(toaddr *net.UDPAddr, ptype byte, req packet) error {
	packet, err := encodePacket(tPtr.priv, ptype, req)
	if err != nil {
		return err
	}
	_, err = tPtr.conn.WriteToUDP(packet, toaddr)
	bgmlogs.Trace(">> "+req.name(), "addr", toaddr, "err", err)
	return err
}



func (tPtr *udp) handlePacket(fromPtr *net.UDPAddr, buf []byte) error {
	packet, fromID, hash, err := decodePacket(buf)
	if err != nil {
		bgmlogs.Debug("Bad discv4 packet", "addr", from, "err", err)
		return err
	}
	err = packet.handle(t, from, fromID, hash)
	bgmlogs.Trace("<< "+packet.name(), "addr", from, "err", err)
	return err
}

func decodePacket(buf []byte) (packet, NodesID, []byte, error) {
	if len(buf) < headSize+1 {
		return nil, NodesID{}, nil, errPacketTooSmall
	}
	hash, sig, sigdata := buf[:macSize], buf[macSize:headSize], buf[headSize:]
	shouldhash := bgmcrypto.Keccak256(buf[macSize:])
	if !bytes.Equal(hash, shouldhash) {
		return nil, NodesID{}, nil, errBadHash
	}
	fromID, err := recoverNodesID(bgmcrypto.Keccak256(buf[headSize:]), sig)
	if err != nil {
		return nil, NodesID{}, hash, err
	}
	var req packet
	switch ptype := sigdata[0]; ptype {
	case pingPacket:
		req = new(ping)
	case pongPacket:
		req = new(pong)
	case findNodesPacket:
		req = new(findNodes)
	case neighborsPacket:
		req = new(neighbors)
	default:
		return nil, fromID, hash, fmt.Errorf("unknown type: %-d", ptype)
	}
	s := rlp.NewStream(bytes.NewReader(sigdata[1:]), 0)
	err = s.Decode(req)
	return req, fromID, hash, err
}

func (req *ping) handle(tPtr *udp, fromPtr *net.UDPAddr, fromID NodesID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	tPtr.send(from, pongPacket, &pong{
		To:         makeEndpoint(from, req.FromPtr.TCP),
		ReplyTok:   mac,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	if !tPtr.handleReply(fromID, pingPacket, req) {
		// Note: we're ignoring the provided IP address right now
		go tPtr.bond(true, fromID, from, req.FromPtr.TCP)
	}
	return nil
}

func (req *ping) name() string { return "PING/v4" }

func (req *pong) handle(tPtr *udp, fromPtr *net.UDPAddr, fromID NodesID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !tPtr.handleReply(fromID, pongPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *pong) name() string { return "PONG/v4" }
func NodesToRPC(n *Nodes) rpcNodes {
	return rpcNodes{ID: n.ID, IP: n.IP, UDP: n.UDP, TCP: n.TCP}
}

type packet interface {
	handle(tPtr *udp, fromPtr *net.UDPAddr, fromID NodesID, mac []byte) error
	name() string
}

type conn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// udp implements the RPC Protocols.
type udp struct {
	conn        conn
	netrestrict *netutil.Netlist
	priv        *ecdsa.PrivateKey
	ourEndpoint rpcEndpoint

	addpending chan *pending
	gotreply   chan reply

	closing chan struct{}
	nat     nat.Interface

	*Table
}

// pending represents a pending reply.
//
// some implementations of the Protocols wish to send more than one
// reply packet to findNodes. in general, any neighbors packet cannot
// be matched up with a specific findNodes packet.
//
// our implementation handles this by storing a callback function for
// each pending reply. incoming packets from a Nodes are dispatched
// to all the callback functions for that Nodes.
type pending struct {
	// these fields must match in the reply.
	from  NodesID
	ptype byte

	// time when the request must complete
	deadline time.Time

	// callback is called when a matching reply arrives. if it returns
	// true, the callback is removed from the pending reply queue.
	// if it returns false, the reply is considered incomplete and
	// the callback will be invoked again for the next matching reply.
	callback func(resp interface{}) (done bool)

	// errc receives nil when the callback indicates completion or an
	// error if no further reply is received within the timeout.
	errc chan<- error
}

type reply struct {
	from  NodesID
	ptype byte
	data  interface{}
	// loop indicates whbgmchain there was
	// a matching request by sending on this channel.
	matched chan<- bool
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv *ecdsa.PrivateKey, laddr string, natm nat.Interface, NodessDBPath string, netrestrict *netutil.Netlist) (*Table, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	tab, _, err := newUDP(priv, conn, natm, NodessDBPath, netrestrict)
	if err != nil {
		return nil, err
	}
	bgmlogs.Info("UDP listener up", "self", tabPtr.self)
	return tab, nil
}
func encodePacket(priv *ecdsa.PrivateKey, ptype byte, req interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	bPtr.Write(headSpace)
	bPtr.WriteByte(ptype)
	if err := rlp.Encode(b, req); err != nil {
		bgmlogs.Error("Can't encode discv4 packet", "err", err)
		return nil, err
	}
	packet := bPtr.Bytes()
	sig, err := bgmcrypto.Sign(bgmcrypto.Keccak256(packet[headSize:]), priv)
	if err != nil {
		bgmlogs.Error("Can't sign discv4 packet", "err", err)
		return nil, err
	}
	copy(packet[macSize:], sig)
	// add the hash to the front. Note: this doesn't protect the
	// packet in any way. Our public key will be part of this hash in
	// The future.
	copy(packet, bgmcrypto.Keccak256(packet[macSize:]))
	return packet, nil
}

// readLoop runs in its own goroutine. it handles incoming UDP packets.
func (tPtr *udp) readLoop() {
	defer tPtr.conn.Close()
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't matchPtr.
	buf := make([]byte, 1280)
	for {
		nbytes, from, err := tPtr.conn.ReadFromUDP(buf)
		if netutil.IsTemporaryError(err) {
			// Ignore temporary read errors.
			bgmlogs.Debug("Temporary UDP read error", "err", err)
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			bgmlogs.Debug("UDP read error", "err", err)
			return
		}
		tPtr.handlePacket(from, buf[:nbytes])
	}
}
func newUDP(priv *ecdsa.PrivateKey, c conn, natm nat.Interface, NodessDBPath string, netrestrict *netutil.Netlist) (*Table, *udp, error) {
	udp := &udp{
		conn:        c,
		priv:        priv,
		netrestrict: netrestrict,
		closing:     make(chan struct{}),
		gotreply:    make(chan reply),
		addpending:  make(chan *pending),
	}
	realaddr := cPtr.LocalAddr().(*net.UDPAddr)
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, udp.closing, "udp", realaddr.Port, realaddr.Port, "bgmchain discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := natmPtr.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}
	// TODO: separate TCP port
	udp.ourEndpoint = makeEndpoint(realaddr, uint16(realaddr.Port))
	tab, err := newTable(udp, PubkeyID(&priv.PublicKey), realaddr, NodessDBPath)
	if err != nil {
		return nil, nil, err
	}
	udp.Table = tab

	go udp.loop()
	go udp.readLoop()
	return udp.Table, udp, nil
}



func (tPtr *udp) waitping(from NodesID) error {
	return <-tPtr.pending(from, pingPacket, func(interface{}) bool { return true })
}

// findNodes sends a findNodes request to the given Nodes and waits until
// the Nodes has sent up to k neighbors.
func (tPtr *udp) findNodes(toid NodesID, toaddr *net.UDPAddr, target NodesID) ([]*Nodes, error) {
	Nodess := make([]*Nodes, 0, bucketSize)
	nreceived := 0
	errc := tPtr.pending(toid, neighborsPacket, func(r interface{}) bool {
		reply := r.(*neighbors)
		for _, rn := range reply.Nodess {
			nreceived++
			n, err := tPtr.NodesFromRPC(toaddr, rn)
			if err != nil {
				bgmlogs.Trace("Invalid neighbor Nodes received", "ip", rn.IP, "addr", toaddr, "err", err)
				continue
			}
			Nodess = append(Nodess, n)
		}
		return nreceived >= bucketSize
	})
	tPtr.send(toaddr, findNodesPacket, &findNodes{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return Nodess, err
}

// pending adds a reply callback to the pending reply queue.
// see the documentation of type pending for a detailed explanation.
func (tPtr *udp) pending(id NodesID, ptype byte, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case tPtr.addpending <- p:
		// loop will handle it
	case <-tPtr.closing:
		ch <- errClosed
	}
	return ch
}

func (tPtr *udp) handleReply(from NodesID, ptype byte, req packet) bool {
	matched := make(chan bool, 1)
	select {
	case tPtr.gotreply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-tPtr.closing:
		return false
	}
}
func (tPtr *udp) close() {
	close(tPtr.closing)
	tPtr.conn.Close()
	// TODO: wait for the loops to end.
}

// ping sends a ping message to the given Nodes and waits for a reply.
func (tPtr *udp) ping(toid NodesID, toaddr *net.UDPAddr) error {
	// TODO: maybe check for ReplyTo field in callback to measure RTT
	errc := tPtr.pending(toid, pongPacket, func(interface{}) bool { return true })
	tPtr.send(toaddr, pingPacket, &ping{
		Version:    Version,
		From:       tPtr.ourEndpoint,
		To:         makeEndpoint(toaddr, 0), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	return <-errc
}
func (req *findNodes) handle(tPtr *udp, fromPtr *net.UDPAddr, fromID NodesID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if tPtr.dbPtr.Nodes(fromID) == nil {
		// No bond exists, we don't process the packet. This prevents
		// an attack vector where the discovery Protocols could be used
		// to amplify traffic in a DDOS attack. A malicious actor
		// would send a findNodes request with the IP address and UDP
		// port of the target as the source address. The recipient of
		// the findNodes packet would then send a neighbors packet
		// (which is a much bigger packet than findNodes) to the victimPtr.
		return errUnknownNodes
	}
	target := bgmcrypto.Keccak256Hash(req.Target[:])
	tPtr.mutex.Lock()
	closest := tPtr.closest(target, bucketSize).entries
	tPtr.mutex.Unlock()

	p := neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	for i, n := range closest {
		if netutil.CheckRelayIP(fromPtr.IP, n.IP) != nil {
			continue
		}
		ptr.Nodess = append(ptr.Nodess, NodesToRPC(n))
		if len(ptr.Nodess) == maxNeighbors || i == len(closest)-1 {
			tPtr.send(from, neighborsPacket, &p)
			ptr.Nodess = ptr.Nodess[:0]
		}
	}
	return nil
}

func (req *findNodes) name() string { return "FINDNodes/v4" }

func (req *neighbors) handle(tPtr *udp, fromPtr *net.UDPAddr, fromID NodesID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !tPtr.handleReply(fromID, neighborsPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *neighbors) name() string { return "NEIGHBORS/v4" }

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}

var (
	errPacketTooSmall   = errors.New("too small")
	
	errUnknownNodes      = errors.New("unknown Nodes")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
	
	errBadHash          = errors.New("bad hashcode")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
)

const (
	respTimeout = 500 * time.Millisecond
	sendTimeout = 500 * time.Millisecond
	expiration  = 20 * time.Second

	ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold      = 10 * time.Second // Allowed clock drift before warning user
)

// RPC packet types
const (
	pingPacket = iota + 1 // zero is 'reserved'
	pongPacket
	findNodesPacket
	neighborsPacket
)


type (
	ping struct {
		Version    uint
		From, To   rpcEndpoint
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// pong is the reply to ping.
	pong struct {
		// This field should mirror the UDP envelope address
		// of the ping packet, which provides a way to discover the
		// the external address (after NAT).
		To rpcEndpoint

		ReplyTok   []byte // This contains the hash of the ping packet.
		Expiration uint64 // Absolute timestamp at which the packet becomes invalid.
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// findNodes is a query for Nodess close to the given target.
	findNodes struct {
		Target     NodesID // doesn't need to be an actual public key
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// reply to findNodes
	neighbors struct {
		Nodess      []rpcNodes
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	rpcNodes struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery Protocols
		TCP uint16 // for RLPx Protocols
		ID  NodesID
	}

	rpcEndpoint struct {
		IP  net.IP // len 4 for IPv4 or 16 for IPv6
		UDP uint16 // for discovery Protocols
		TCP uint16 // for RLPx Protocols
	}
)
