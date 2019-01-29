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
	"bytes"
	"bgmcrypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
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

func (e1 rpcEndpoint) equal(e2 rpcEndpoint) bool {
	return e1.UDP == e2.UDP && e1.TCP == e2.TCP && e1.IP.Equal(e2.IP)
}

func NodesFromRPC(sender *net.UDPAddr, rn rpcNodes) (*Nodes, error) {
	if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
		return nil, err
	}
	n := NewNodes(rn.ID, rn.IP, rn.UDP, rn.TCP)
	err := n.validateComplete()
	return n, err
}

func NodesToRPC(n *Nodes) rpcNodes {
	return rpcNodes{ID: n.ID, IP: n.IP, UDP: n.UDP, TCP: n.TCP}
}

type ingressPacket struct {
	remoteID   NodesID
	remoteAddr *net.UDPAddr
	ev         NodesEvent
	hash       []byte
	data       interface{} // one of the RPC structs
	rawData    []byte
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
	priv        *ecdsa.PrivateKey
	ourEndpoint rpcEndpoint
	nat         nat.Interface
	net         *Network
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv *ecdsa.PrivateKey, laddr string, natm nat.Interface, NodessDBPath string, netrestrict *netutil.Netlist) (*Network, error) {
	transport, err := listenUDP(priv, laddr)
	if err != nil {
		return nil, err
	}
	net, err := newbgmNetwork(transport, priv.PublicKey, natm, NodessDBPath, netrestrict)
	if err != nil {
		return nil, err
	}
	transport.net = net
	go transport.readLoop()
	return net, nil
}

func listenUDP(priv *ecdsa.PrivateKey, laddr string) (*udp, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &udp{conn: conn, priv: priv, ourEndpoint: makeEndpoint(addr, uint16(addr.Port))}, nil
}

func (tPtr *udp) localAddr() *net.UDPAddr {
	return tPtr.conn.LocalAddr().(*net.UDPAddr)
}

func (tPtr *udp) Close() {
	tPtr.conn.Close()
}

func (tPtr *udp) send(remote *Nodes, ptype NodesEvent, data interface{}) (hash []byte) {
	hash, _ = tPtr.sendPacket(remote.ID, remote.addr(), byte(ptype), data)
	return hash
}

func (tPtr *udp) sendPing(remote *Nodes, toaddr *net.UDPAddr, topics []Topic) (hash []byte) {
	hash, _ = tPtr.sendPacket(remote.ID, toaddr, byte(pingPacket), ping{
		Version:    Version,
		From:       tPtr.ourEndpoint,
		To:         makeEndpoint(toaddr, uint16(toaddr.Port)), // TODO: maybe use known TCP port from DB
		Expiration: Uint64(time.Now().Add(expiration).Unix()),
		Topics:     topics,
	})
	return hash
}

func (tPtr *udp) sendFindNodes(remote *Nodes, target NodesID) {
	tPtr.sendPacket(remote.ID, remote.addr(), byte(findNodesPacket), findNodes{
		Target:     target,
		Expiration: Uint64(time.Now().Add(expiration).Unix()),
	})
}

func (tPtr *udp) sendNeighbours(remote *Nodes, results []*Nodes) {
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	p := neighbors{Expiration: Uint64(time.Now().Add(expiration).Unix())}
	for i, result := range results {
		ptr.Nodess = append(ptr.Nodess, NodesToRPC(result))
		if len(ptr.Nodess) == maxNeighbors || i == len(results)-1 {
			tPtr.sendPacket(remote.ID, remote.addr(), byte(neighborsPacket), p)
			ptr.Nodess = ptr.Nodess[:0]
		}
	}
}

func (tPtr *udp) sendFindNodesHash(remote *Nodes, target bgmcommon.Hash) {
	tPtr.sendPacket(remote.ID, remote.addr(), byte(findNodesHashPacket), findNodesHash{
		Target:     target,
		Expiration: Uint64(time.Now().Add(expiration).Unix()),
	})
}

func (tPtr *udp) sendTopicRegister(remote *Nodes, topics []Topic, idx int, pong []byte) {
	tPtr.sendPacket(remote.ID, remote.addr(), byte(topicRegisterPacket), topicRegister{
		Topics: topics,
		Idx:    uint(idx),
		Pong:   pong,
	})
}

func (tPtr *udp) sendTopicNodess(remote *Nodes, queryHash bgmcommon.Hash, Nodess []*Nodes) {
	p := topicNodess{Echo: queryHash}
	if len(Nodess) == 0 {
		tPtr.sendPacket(remote.ID, remote.addr(), byte(topicNodessPacket), p)
		return
	}
	for i, result := range Nodess {
		if netutil.CheckRelayIP(remote.IP, result.IP) != nil {
			continue
		}
		ptr.Nodess = append(ptr.Nodess, NodesToRPC(result))
		if len(ptr.Nodess) == maxTopicNodess || i == len(Nodess)-1 {
			tPtr.sendPacket(remote.ID, remote.addr(), byte(topicNodessPacket), p)
			ptr.Nodess = ptr.Nodess[:0]
		}
	}
}

func (tPtr *udp) sendPacket(toid NodesID, toaddr *net.UDPAddr, ptype byte, req interface{}) (hash []byte, err error) {
	//fmt.Println("sendPacket", NodesEvent(ptype), toaddr.String(), toid.String())
	packet, hash, err := encodePacket(tPtr.priv, ptype, req)
	if err != nil {
		//fmt.Println(err)
		return hash, err
	}
	bgmlogs.Trace(fmt.Sprintf(">>> %v to %x@%v", NodesEvent(ptype), toid[:8], toaddr))
	if _, err = tPtr.conn.WriteToUDP(packet, toaddr); err != nil {
		bgmlogs.Trace(fmt.Sprint("UDP send failed:", err))
	}
	//fmt.Println(err)
	return hash, err
}

// zeroed padding space for encodePacket.
var headSpace = make([]byte, headSize)

func encodePacket(priv *ecdsa.PrivateKey, ptype byte, req interface{}) (p, hash []byte, err error) {
	b := new(bytes.Buffer)
	bPtr.Write(headSpace)
	bPtr.WriteByte(ptype)
	if err := rlp.Encode(b, req); err != nil {
		bgmlogs.Error(fmt.Sprint("error encoding packet:", err))
		return nil, nil, err
	}
	packet := bPtr.Bytes()
	sig, err := bgmcrypto.Sign(bgmcrypto.Keccak256(packet[headSize:]), priv)
	if err != nil {
		bgmlogs.Error(fmt.Sprint("could not sign packet:", err))
		return nil, nil, err
	}
	copy(packet[macSize:], sig)
	// add the hash to the front. Note: this doesn't protect the
	// packet in any way.
	hash = bgmcrypto.Keccak256(packet[macSize:])
	copy(packet, hash)
	return packet, hash, nil
}

// readLoop runs in its own goroutine. it injects ingress UDP packets
// into the network loop.
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
			bgmlogs.Debug(fmt.Sprintf("Temporary read error: %v", err))
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			bgmlogs.Debug(fmt.Sprintf("Read error: %v", err))
			return
		}
		tPtr.handlePacket(from, buf[:nbytes])
	}
}

func (tPtr *udp) handlePacket(fromPtr *net.UDPAddr, buf []byte) error {
	pkt := ingressPacket{remoteAddr: from}
	if err := decodePacket(buf, &pkt); err != nil {
		bgmlogs.Debug(fmt.Sprintf("Bad packet from %v: %v", from, err))
		//fmt.Println("bad packet", err)
		return err
	}
	tPtr.net.reqReadPacket(pkt)
	return nil
}
var (
	errPacketTooSmall   = errors.New("too small")
	errBadHash          = errors.New("bad hash")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNodes      = errors.New("unknown Nodes")
	errtimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
)

// timeouts
const (
	resptimeout = 500 * time.Millisecond
	sendtimeout = 500 * time.Millisecond
	expiration  = 20 * time.Second

	ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold      = 10 * time.Second // Allowed clock drift before warning user
)

// RPC request structures
type (
	ping struct {
		Version    uint
		From, To   rpcEndpoint
		Expiration Uint64

		// v5
		Topics []Topic

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
		Expiration Uint64 // Absolute timestamp at which the packet becomes invalid.

		// v5
		TopicHash    bgmcommon.Hash
		TicketSerial uint32
		WaitPeriods  []uint32

		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// findNodes is a query for Nodess close to the given target.
	findNodes struct {
		Target     NodesID // doesn't need to be an actual public key
		Expiration Uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// findNodes is a query for Nodess close to the given target.
	findNodesHash struct {
		Target     bgmcommon.Hash
		Expiration Uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	// reply to findNodes
	neighbors struct {
		Nodess      []rpcNodes
		Expiration Uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	topicRegister struct {
		Topics []Topic
		Idx    uint
		Pong   []byte
	}

	topicQuery struct {
		Topic      Topic
		Expiration Uint64
	}

	// reply to topicQuery
	topicNodess struct {
		Echo  bgmcommon.Hash
		Nodess []rpcNodes
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

const (
	macSize  = 256 / 8
	sigSize  = 520 / 8
	headSize = macSize + sigSize // space of packet frame data
)

// Neighbors replies are sent across multiple packets to
// stay below the 1280 byte limit. We compute the maximum number
// of entries by stuffing a packet until it grows too large.
var maxNeighbors = func() int {
	p := neighbors{Expiration: ^Uint64(0)}
	maxSizeNodes := rpcNodes{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		ptr.Nodess = append(ptr.Nodess, maxSizeNodes)
		size, _, err := rlp.EncodeToReader(p)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("Fatal: cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()
func decodePacket(buffer []byte, pkt *ingressPacket) error {
	if len(buffer) < headSize+1 {
		return errPacketTooSmall
	}
	buf := make([]byte, len(buffer))
	copy(buf, buffer)
	hash, sig, sigdata := buf[:macSize], buf[macSize:headSize], buf[headSize:]
	shouldhash := bgmcrypto.Keccak256(buf[macSize:])
	if !bytes.Equal(hash, shouldhash) {
		return errBadHash
	}
	fromID, err := recoverNodesID(bgmcrypto.Keccak256(buf[headSize:]), sig)
	if err != nil {
		return err
	}
	pkt.rawData = buf
	pkt.hash = hash
	pkt.remoteID = fromID
	switch pkt.ev = NodesEvent(sigdata[0]); pkt.ev {
	case pingPacket:
		pkt.data = new(ping)
	case pongPacket:
		pkt.data = new(pong)
	case findNodesPacket:
		pkt.data = new(findNodes)
	case neighborsPacket:
		pkt.data = new(neighbors)
	case findNodesHashPacket:
		pkt.data = new(findNodesHash)
	case topicRegisterPacket:
		pkt.data = new(topicRegister)
	case topicQueryPacket:
		pkt.data = new(topicQuery)
	case topicNodessPacket:
		pkt.data = new(topicNodess)
	default:
		return fmt.Errorf("unknown packet type: %-d", sigdata[0])
	}
	s := rlp.NewStream(bytes.NewReader(sigdata[1:]), 0)
	err = s.Decode(pkt.data)
	return err
}

var maxTopicNodess = func() int {
	p := topicNodess{}
	maxSizeNodes := rpcNodes{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		ptr.Nodess = append(ptr.Nodess, maxSizeNodes)
		size, _, err := rlp.EncodeToReader(p)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("Fatal: cannot encode: " + err.Error())
		}
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()

