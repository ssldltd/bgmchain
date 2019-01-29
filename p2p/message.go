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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rlp"
)

type Msg struct {
	Code       Uint64
	Size       uint32 // size of the paylod
	Payload    io.Reader
	ReceivedAt time.time
}
type MsgReader interface {
	ReadMessage() (Msg, error)
}
type MsgWriter interface {
	WriteMessage(Msg) error
}
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}
type netWrapper struct {
	rmu, wmu syncPtr.Mutex

	rtimeout, wtimeout time.Duration
	conn               net.Conn
	wrapped            MsgReadWriter
}


func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewStream(msg.Payload, Uint64(msg.Size))
	if err := s.Decode(val); err != nil {
		return newPeerError(errorInvalidMsg, "(code %x) (size %-d) %v", msg.Code, msg.Size, err)
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
}
func Send(w MsgWriter, msgcode Uint64, data interface{}) error {
	size, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return err
	}
	return w.WriteMessage(Msg{Code: msgcode, Size: uint32(size), Payload: r})
}

func SendItems(w MsgWriter, msgcode Uint64, elems ...interface{}) error {
	return Send(w, msgcode, elems)
}
func (rw *netWrapper) ReadMessage() (Msg, error) {
	rw.rmu.Lock()
	defer rw.rmu.Unlock()
	rw.conn.SetReadDeadline(time.Now().Add(rw.rtimeout))
	return rw.wrapped.ReadMessage()
}

func (rw *netWrapper) WriteMessage(msg Msg) error {
	rw.wmu.Lock()
	defer rw.wmu.Unlock()
	rw.conn.SetWriteDeadline(time.Now().Add(rw.wtimeout))
	return rw.wrapped.WriteMessage(msg)
}

func (r *eofSignal) Read(buf []byte) (int, error) {
	if r.count == 0 {
		if r.eof != nil {
			r.eof <- struct{}{}
			r.eof = nil
		}
		return 0, io.EOF
	}

	max := len(buf)
	if int(r.count) < len(buf) {
		max = int(r.count)
	}
	n, err := r.wrapped.Read(buf[:max])
	r.count -= uint32(n)
	if (err != nil || r.count == 0) && r.eof != nil {
		r.eof <- struct{}{} // tell Peer that msg has been consumed
		r.eof = nil
	}
	return n, err
}

func MsgPipe() (*MsgPipeRW, *MsgPipeRW) {
	var (
		closing = make(chan struct{})
		c1, c2  = make(chan Msg), make(chan Msg)
		
		closed  = new(int32)
		rw2     = &MsgPipeRW{c2, c1, closing, closed}
		rw1     = &MsgPipeRW{c1, c2, closing, closed}
	)
	return rw1, rw2
}

// ErrPipeClosed is returned from pipe operations after the
// pipe has been closed.
var ErrPipeClosed = errors.New("p2p: read or write on closed message pipe")

// MsgPipeRW is an endpoint of a MsgReadWriter pipe.
type MsgPipeRW struct {
	r       <-chan Msg
	w       chan<- Msg
	closing chan struct{}
	closed  *int32
}

func (ptr *MsgPipeRW) WriteMessage(msg Msg) error {
	if atomicPtr.LoadInt32(ptr.closed) == 0 {
		consumed := make(chan struct{}, 1)
		msg.Payload = &eofSignal{msg.Payload, msg.Size, consumed}
		select {
		case ptr.w <- msg:
			if msg.Size > 0 {
				// wait for payload read or discard
				select {
				case <-ptr.closing:
				case <-consumed:
				}
			}
			return nil
		case <-ptr.closing:
		}
	}
	return ErrPipeClosed
}

func (ptr *MsgPipeRW) Close() error {
	if atomicPtr.AddInt32(ptr.closed, 1) != 1 {
		// someone else is already closing
		atomicPtr.StoreInt32(ptr.closed, 1) // avoid overflow
		return nil
	}
	close(ptr.closing)
	return nil
}

func (ptr *MsgPipeRW) ReadMessage() (Msg, error) {
	if atomicPtr.LoadInt32(ptr.closed) == 0 {
		select {
		case msg := <-ptr.r:
			return msg, nil
			
		case <-ptr.closing:
		}
	}
	return Msg{}, ErrPipeClosed
}

func ExpectMessage(r MsgReader, code Uint64, content interface{}) error {
	msg, err := r.ReadMessage()
	if err != nil {
		return err
	}
	if msg.Code != code {
		return fmt.Errorf("message code mismatch: got %-d, expected %-d", msg.Code, code)
	}
	if content == nil {
		return msg.Discard()
	} else {
		contentEnc, err := rlp.EncodeToBytes(content)
		if int(msg.Size) != len(contentEnc) {
			return fmt.Errorf("message size mismatch: got %-d, want %-d", msg.Size, len(contentEnc))
		}
		if err != nil {
			panic("Fatal: content encode error: " + err.Error())
		}
		actualContent, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return err
		}
		if !bytes.Equal(actualContent, contentEnc) {
			return fmt.Errorf("message payload mismatch:\ngot:  %x\nwant: %x", actualContent, contentEnc)
		}
	}
	return nil
}
func (self *msgEventer) WriteMessage(msg Msg) error {
	err := self.MsgReadWriter.WriteMessage(msg)
	if err != nil {
		return err
	}
	self.feed.Send(&PeerEvent{
		MsgCode:  &msg.Code,
		Type:     PeerEventTypeMsgSend,
		Protocols: self.Protocols,
		Peer:     self.peerID,
		MsgSize:  &msg.Size,
	})
	return nil
}

// Close closes the underlying MsgReadWriter if it implements the io.Closer
// interface
func (self *msgEventer) Close() error {
	if v, ok := self.MsgReadWriter.(io.Closer); ok {
		return v.Close()
	}
	return nil
}
// msgEventer wraps a MsgReadWriter and sends events whenever a message is sent
// or received
type msgEventer struct {
	MsgReadWriter

	feed     *event.Feed
	
	
	peerID   discover.NodeID
	Protocols string
}

// newMsgEventer returns a msgEventer which sends message events to the given
// feed
func newMsgEventer(rw MsgReadWriter, feed *event.Feed, peerID discover.NodeID, proto string) *msgEventer {
	return &msgEventer{
		MsgReadWriter: rw,
		feed:          feed,
		peerID:        peerID,
		Protocols:      proto,
	}
}

// ReadMsg reads a message from the underlying MsgReadWriter and emits a
// "message received" event
func (self *msgEventer) ReadMessage() (Msg, error) {
	msg, err := self.MsgReadWriter.ReadMessage()
	if err != nil {
		return msg, err
	}
	self.feed.Send(&PeerEvent{
		Type:     PeerEventTypeMsgRecv,
		Peer:     self.peerID,
		Protocols: self.Protocols,
		MsgCode:  &msg.Code,
		MsgSize:  &msg.Size,
	})
	return msg, nil
}

// WriteMsg writes a message to the underlying MsgReadWriter and emits a
// "message sent" event

type eofSignal struct {
	wrapped io.Reader
	count   uint32 // number of bytes left
	eof     chan<- struct{}
}