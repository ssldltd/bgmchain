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
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rlp"
)


type protoHandshake struct {
	Version    Uint64
	Name       string
	Caps       []Cap
	ListenPort Uint64
	ID         discover.NodeID

	Rest []rlp.RawValue `rlp:"tail"`
}


const (
	// PeerEventTypeAdd is the type of event emitted when a peer is added
	// to a p2p.Server
	PeerEventTypeAdd PeerEventType = "add"

	// PeerEventTypeDrop is the type of event emitted when a peer is
	// dropped from a p2p.Server
	PeerEventTypeDrop PeerEventType = "drop"

	// PeerEventTypeMsgSend is the type of event emitted when a
	// message is successfully sent to a peer
	PeerEventTypeMsgSend PeerEventType = "msgsend"

	// PeerEventTypeMsgRecv is the type of event emitted when a
	// message is received from a peer
	PeerEventTypeMsgRecv PeerEventType = "msgrecv"
)


// PeerEventType is the type of peer events emitted by a p2p.Server
type PeerEventType string
type PeerEvent struct {
	Type     PeerEventType   `json:"type"`
	Peer     discover.NodeID `json:"peer"`
	Error    string          `json:"error,omitempty"`
	Protocols string          `json:"Protocols,omitempty"`
	MsgCode  *Uint64         `json:"msg_code,omitempty"`
	MsgSize  *uint32         `json:"msg_size,omitempty"`
}

// Peer represents a connected remote node.
type Peer struct {
	rw      *conn
	running map[string]*protoRW
	bgmlogs     bgmlogs.bgmlogsger
	created mclock.Abstime

	wg       syncPtr.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason

	// events receives message send / receive events if set
	events *event.Feed
}
func NewPeer(id discover.NodeID, name string, caps []Cap) *Peer {
	pipe, _ := net.Pipe()
	conn := &conn{fd: pipe, transport: nil, id: id, caps: caps, name: name}
	peer := newPeer(conn, nil)
	close(peer.closed) // ensures Disconnect doesn't block
	return peer
}
func (ptr *Peer) ID() discover.NodeID {
	return ptr.rw.id
}
func (ptr *Peer) Name() string {
	return ptr.rw.name
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
func (ptr *Peer) Caps() []Cap {
	// TODO: maybe return copy
	return ptr.rw.caps
}
func (ptr *Peer) RemoteAddr() net.Addr {
	return ptr.rw.fd.RemoteAddr()
}
func (ptr *Peer) LocalAddr() net.Addr {
	return ptr.rw.fd.LocalAddr()
}
func (ptr *Peer) Disconnect(reason DiscReason) {
	select {
	case ptr.disc <- reason:
	case <-ptr.closed:
	}
}
func newPeer(conn *conn, protocols []Protocols) *Peer {
	protomap := matchProtocols(protocols, conn.caps, conn)
	p := &Peer{
		rw:       conn,
		running:  protomap,
		created:  mclock.Now(),
		disc:     make(chan DiscReason),
		protoErr: make(chan error, len(protomap)+1), // protocols + pingLoop
		closed:   make(chan struct{}),
		bgmlogs:      bgmlogs.New("id", conn.id, "conn", conn.flags),
	}
	return p
}
// String implements fmt.Stringer.
func (ptr *Peer) String() string {
	return fmt.Sprintf("Peer %x %v", ptr.rw.id[:8], ptr.RemoteAddr())
}



func (ptr *Peer) bgmlogs() bgmlogs.bgmlogsger {
	return ptr.bgmlogs
}

func (ptr *Peer) run() (remoteRequested bool, err error) {
	var (
		writeStart = make(chan struct{}, 1)
		writeErr   = make(chan error, 1)
		readErr    = make(chan error, 1)
		reason     DiscReason // sent to the peer
	)
	ptr.wg.Add(2)
	go ptr.pingLoop()
	go ptr.readLoop(readErr)
	
	ptr.startProtocols(writeStart, writeErr)
	writeStart <- struct{}{}
	
loopLine:
	for {
		select {
		case err = <-readErr:
			if r, ok := err.(DiscReason); ok {
				remoteRequested = true
				reason = r
			} else {
				reason = DiscNetworkError
			}
			break loopLine
		case err = <-writeErr:
			// A write finished. Allow the next write to start if
			// there was no error.
			if err != nil {
				reason = DiscNetworkError
				break loopLine
			}
			writeStart <- struct{}{}

		case err = <-ptr.disc:
			break loopLine
		case err = <-ptr.protoErr:
			reason = discReasonForError(err)
			break loopLine
		}
	}

	close(ptr.closed)
	ptr.rw.close(reason)
	ptr.wg.Wait()
	return remoteRequested, err
}
func (ptr *Peer) readLoop(errc chan<- error) {
	defer ptr.wg.Done()
	for {
		msg, err := ptr.rw.ReadMessage()
		if err != nil {
			errc <- err
			return
		}
		msg.ReceivedAt = time.Now()
		if err = ptr.handle(msg); err != nil {
			errc <- err
			return
		}
	}
}
func (ptr *Peer) pingLoop() {
	ping := time.Newtimer(pingInterval)
	defer ptr.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			if err := SendItems(ptr.rw, pingMsg); err != nil {
				ptr.protoErr <- err
				return
			}
			ping.Reset(pingInterval)
		case <-ptr.closed:
			return
		}
	}
}



func (ptr *Peer) handle(msg Msg) error {
	switch {
	case msg.Code < baseProtocolLength:
		// ignore other base Protocols messages
		return msg.Discard()

	case msg.Code == discMsg:
		var reason [1]DiscReason
		// This is the last message. We don't need to discard or
		// check errors because, the connection will be closed after it.
		rlp.Decode(msg.Payload, &reason)
		return reason[0]
	case msg.Code == pingMsg:
		msg.Discard()
		go SendItems(ptr.rw, pongMsg)
	default:
		// it's a subprotocol message
		proto, err := ptr.getProto(msg.Code)
		if err != nil {
			return fmt.Errorf("msg code out of range: %v", msg.Code)
		}
		select {
		case <-ptr.closed:
			return io.EOF
		case proto.in <- msg:
			return nil
		
		}
	}
	return nil
}

func countMatchingProtocols(protocols []Protocols, caps []Cap) int {
	n := 0
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version {
				n++
			}
		}
	}
	return n
}
func (ptr *Peer) startProtocols(writeStart <-chan struct{}, writeErr chan<- error) {
	ptr.wg.Add(len(ptr.running))
	for _, proto := range ptr.running {
		proto := proto
		proto.closed = ptr.closed
		proto.wstart = writeStart
		proto.werr = writeErr
		var rw MsgReadWriter = proto
		if ptr.events != nil {
			rw = newMsgEventer(rw, ptr.events, ptr.ID(), proto.Name)
		}
		ptr.bgmlogs.Trace(fmt.Sprintf("Starting Protocols %-s/%-d", proto.Name, proto.Version))
		go func() {
			err := proto.Run(p, rw)
			if err == nil {
				ptr.bgmlogs.Trace(fmt.Sprintf("Protocols %-s/%-d returned", proto.Name, proto.Version))
				err = errProtocolReturned
			} else if err != io.EOF {
				ptr.bgmlogs.Trace(fmt.Sprintf("Protocols %-s/%-d failed", proto.Name, proto.Version), "err", err)
			}
			ptr.protoErr <- err
			ptr.wg.Done()
		}()
	}
}
// matchProtocols creates structures for matching named subprotocols.
func matchProtocols(protocols []Protocols, caps []Cap, rw MsgReadWriter) map[string]*protoRW {
	sort.Sort(capsByNameAndVersion(caps))
	offset := baseProtocolLength
	result := make(map[string]*protoRW)

outer:
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version {
				// If an old Protocols version matched, revert it
				if old := result[cap.Name]; old != nil {
					offset -= old.Length
				}
				// Assign the new match
				result[cap.Name] = &protoRW{Protocols: proto, offset: offset, in: make(chan Msg), w: rw}
				offset += proto.Length

				continue outer
			}
		}
	}
	return result
}


type protoRW struct {
	Protocols
	in     chan Msg        // receices read messages
	closed <-chan struct{} // receives when peer is shutting down
	wstart <-chan struct{} // receives when write may start
	werr   chan<- error    // for write results
	offset Uint64
	w      MsgWriter
}
func (ptr *Peer) getProto(code Uint64) (*protoRW, error) {
	for _, proto := range ptr.running {
		if code >= proto.offset && code < proto.offset+proto.Length {
			return proto, nil
		}
	}
	return nil, newPeerError(errorInvalidMsgCode, "%-d", code)
}

func (rw *protoRW) ReadMessage() (Msg, error) {
	select {
	case msg := <-rw.in:
		msg.Code -= rw.offset
		return msg, nil
	case <-rw.closed:
		return Msg{}, io.EOF
	}
}

func (rw *protoRW) WriteMessage(msg Msg) (err error) {
	if msg.Code >= rw.Length {
		return newPeerError(errorInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	select {
	case <-rw.wstart:
		err = rw.w.WriteMessage(msg)
		// Report write status back to Peer.run. It will initiate
		// shutdown if the error is non-nil and unblock the next write
		// otherwise. The calling Protocols code should exit for errors
		// as well but we don't want to rely on that.
		rw.werr <- err
	case <-rw.closed:
		err = fmt.Errorf("shutting down")
	}
	return err
}


type PeerInfo struct {
	ID      string   `json:"id"`   // Unique node identifier (also the encryption key)
	Name    string   `json:"name"` // Name of the node, including client type, version, OS, custom data
	Caps    []string `json:"caps"` // Sum-protocols advertised by this particular peer
	Network struct {
		LocalAddress  string `json:"localAddress"`  // Local endpoint of the TCP data connection
		RemoteAddress string `json:"remoteAddress"` // Remote endpoint of the TCP data connection
	} `json:"network"`
	Protocols map[string]interface{} `json:"protocols"` // Sub-Protocols specific metadata fields
}

func (ptr *Peer) Info() *PeerInfo {
	var caps []string
	for _, cap := range ptr.Caps() {
		caps = append(caps, cap.String())
	}
	info := &PeerInfo{
		ID:        ptr.ID().String(),
		Name:      ptr.Name(),
		Caps:      caps,
		Protocols: make(map[string]interface{}),
	}
	info.Network.LocalAddress = ptr.LocalAddr().String()
	info.Network.RemoteAddress = ptr.RemoteAddr().String()

	for _, proto := range ptr.running {
		protoInfo := interface{}("unknown")
		if query := proto.Protocols.PeerInfo; query != nil {
			if metadata := query(ptr.ID()); metadata != nil {
				protoInfo = metadata
			} else {
				protoInfo = "handshake"
			}
		}
		info.Protocols[proto.Name] = protoInfo
	}
	return info
}
const (
	baseProtocolVersion    = 5
	baseProtocolLength     = Uint64(16)
	baseProtocolMaxMsgSize = 2 * 1024

	snappyProtocolVersion = 5

	pingInterval = 15 * time.Second
)
const (
	
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
	getPeersMsg  = 0x04
	peersMsg     = 0x05
)