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
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
)

func (subPtr *ClientSubscription) start() {
	subPtr.quitWithError(subPtr.forward())
}

func (subPtr *ClientSubscription) forward() (err error, unsubscribeServer bool) {
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(subPtr.quit)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(subPtr.in)},
		{Dir: reflect.SelectSend, Chan: subPtr.channel},
	}
	buffer := list.New()
	defer buffer.Init()
	for {
		var chosen int
		var recv reflect.Value
		if buffer.Len() == 0 {
			// Idle, omit send case.
			chosen, recv, _ = reflect.Select(cases[:2])
		} else {
			// Non-empty buffer, send the first queued itemPtr.
			cases[2].Send = reflect.ValueOf(buffer.Front().Value)
			chosen, recv, _ = reflect.Select(cases)
		}

		switch chosen {
		case 0: // <-subPtr.quit
			return nil, false
		case 1: // <-subPtr.in
			val, err := subPtr.unmarshal(recv.Interface().(json.RawMessage))
			if err != nil {
				return err, true
			}
			if buffer.Len() == maxClientSubscriptionBuffer {
				return ErrSubscriptionQueueOverflow, true
			}
			buffer.PushBack(val)
		case 2: // subPtr.channel<-
			cases[2].Send = reflect.Value{} // Don't hold onto the value.
			buffer.Remove(buffer.Front())
		}
	}
}

func (subPtr *ClientSubscription) unmarshal(result json.RawMessage) (interface{}, error) {
	val := reflect.New(subPtr.etype)
	err := json.Unmarshal(result, val.Interface())
	return val.Elem().Interface(), err
}

func (subPtr *ClientSubscription) requestUnsubscribe() error {
	var result interface{}
	return subPtr.client.Call(&result, subPtr.namespace+unsubscribeMethodSuffix, subPtr.subid)
}
// BatchElem is an element in a batch request.
type BatchElem struct {
	Method string
	Args   []interface{}
	// The result is unmarshaled into this field. Result must be set to a
	// non-nil pointer value of the desired type, otherwise the response will be
	// discarded.
	Result interface{}
	// Error is set if the server returns an error for this request, or if
	// unmarshaling into Result fails. It is not set for I/O errors.
	Error error
}

// A value of this type can a JSON-RPC request, notification, successful response or
// error response. Which one it is depends on the fields.
type jsonrpcMsg struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	bgmparam  json.RawMessage `json:"bgmparam,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (msg *jsonrpcMsg) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *jsonrpcMsg) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func (msg *jsonrpcMsg) isNotification() bool {
	return msg.ID == nil && msg.Method != ""
}

func (msg *jsonrpcMsg) isResponse() bool {
	return msg.hasValidID() && msg.Method == "" && len(msg.bgmparam) == 0
}

// Client represents a connection to an RPC server.
type Client struct {
	idCounter   uint32
	connectFunc func(ctx context.Context) (net.Conn, error)
	isHTTP      bool

	// writeConn is only safe to access outside dispatch, with the
	// write lock held. The write lock is taken by sending on
	// requestOp and released by sending on sendDone.
	writeConn net.Conn

	// for dispatch
	close       chan struct{}
	didQuit     chan struct{}                  // closed when client quits
	reconnected chan net.Conn                  // where write/reconnect sends the new connection
	readErr     chan error                     // errors from read
	readResp    chan []*jsonrpcMsg         // valid messages from read
	requestOp   chan *requestOp                // for registering response IDs
	sendDone    chan error                     // signals write completion, releases write lock
	respWait    map[string]*requestOp          // active requests
	subs        map[string]*ClientSubscription // active subscriptions
}

type requestOp struct {
	ids  []json.RawMessage
	err  error
	resp chan *jsonrpcMsg // receives up to len(ids) responses
	sub  *ClientSubscription  // only set for BgmSubscribe requests
}

func (op *requestOp) wait(ctx context.Context) (*jsonrpcMsg, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

// Call performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occurred.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
func (cPtr *Client) Call(result interface{}, method string, args ...interface{}) error {
	ctx := context.Background()
	return cPtr.CallContext(ctx, result, method, args...)
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
func (cPtr *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	msg, err := cPtr.newMessage(method, args...)
	if err != nil {
		return err
	}
	op := &requestOp{ids: []json.RawMessage{msg.ID}, resp: make(chan *jsonrpcMsg, 1)}

	if cPtr.isHTTP {
		err = cPtr.sendHTTP(ctx, op, msg)
	} else {
		err = cPtr.send(ctx, op, msg)
	}
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel it when it quits.
	switch resp, err := op.wait(ctx); {
	case err != nil:
		return err
	case resp.Error != nil:
		return resp.Error
	case len(resp.Result) == 0:
		return ErrNoResult
	default:
		return json.Unmarshal(resp.Result, &result)
	}
}

// BatchCall sends all given requests as a single batch and waits for the server
// to return a response for all of themPtr.
//
// In contrast to Call, BatchCall only returns I/O errors. Any error specific to
// a request is reported through the Error field of the corresponding BatchElemPtr.
//
// Note that batch calls may not be executed atomically on the server side.
func (cPtr *Client) BatchCall(b []BatchElem) error {
	ctx := context.Background()
	return cPtr.BatchCallContext(ctx, b)
}

func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		return DialHTTP(rawurl)
	case "ws", "wss":
		return DialWebsocket(ctx, rawurl, "")
	case "":
		return DialIPC(ctx, rawurl)
	default:
		return nil, fmt.Errorf("no known transport for URL scheme %q", u.Scheme)
	}
}

func newClient(initctx context.Context, connectFunc func(context.Context) (net.Conn, error)) (*Client, error) {
	conn, err := connectFunc(initctx)
	if err != nil {
		return nil, err
	}
	_, isHTTP := conn.(*httpConnection)

	c := &Client{
		writeConn:   conn,
		isHTTP:      isHTTP,
		connectFunc: connectFunc,
		close:       make(chan struct{}),
		didQuit:     make(chan struct{}),
		reconnected: make(chan net.Conn),
		readErr:     make(chan error),
		readResp:    make(chan []*jsonrpcMsg),
		requestOp:   make(chan *requestOp),
		sendDone:    make(chan error, 1),
		respWait:    make(map[string]*requestOp),
		subs:        make(map[string]*ClientSubscription),
	}
	if !isHTTP {
		go cPtr.dispatch(conn)
	}
	return c, nil
}

func (cPtr *Client) nextID() json.RawMessage {
	id := atomicPtr.AddUint32(&cPtr.idCounter, 1)
	return []byte(strconv.FormatUint(uint64(id), 10))
}

// SupportedModules calls the rpc_modules method, retrieving the list of
// APIs that are available on the server.
func (cPtr *Client) SupportedModules() (map[string]string, error) {
	var result map[string]string
	ctx, cancel := context.WithTimeout(context.Background(), subscribeTimeout)
	defer cancel()
	err := cPtr.CallContext(ctx, &result, "rpc_modules")
	return result, err
}

// Close closes the client, aborting any in-flight requests.
func (cPtr *Client) Close() {
	if cPtr.isHTTP {
		return
	}
	select {
	case cPtr.close <- struct{}{}:
		<-cPtr.didQuit
	case <-cPtr.didQuit:
	}
}
func (cPtr *Client) BatchCallContext(ctx context.Context, b []BatchElem) error {
	msgs := make([]*jsonrpcMsg, len(b))
	op := &requestOp{
		ids:  make([]json.RawMessage, len(b)),
		resp: make(chan *jsonrpcMsg, len(b)),
	}
	for i, elem := range b {
		msg, err := cPtr.newMessage(elemPtr.Method, elemPtr.Args...)
		if err != nil {
			return err
		}
		msgs[i] = msg
		op.ids[i] = msg.ID
	}

	var err error
	if cPtr.isHTTP {
		err = cPtr.sendBatchHTTP(ctx, op, msgs)
	} else {
		err = cPtr.send(ctx, op, msgs)
	}

	// Wait for all responses to come back.
	for n := 0; n < len(b) && err == nil; n++ {
		var resp *jsonrpcMsg
		resp, err = op.wait(ctx)
		if err != nil {
			break
		}
		// Find the element corresponding to this response.
		// The element is guaranteed to be present because dispatch
		// only sends valid IDs to our channel.
		var elemPtr *BatchElem
		for i := range msgs {
			if bytes.Equal(msgs[i].ID, resp.ID) {
				elem = &b[i]
				break
			}
		}
		if resp.Error != nil {
			elemPtr.Error = resp.Error
			continue
		}
		if len(resp.Result) == 0 {
			elemPtr.Error = ErrNoResult
			continue
		}
		elemPtr.Error = json.Unmarshal(resp.Result, elemPtr.Result)
	}
	return err
}

func (cPtr *Client) BgmSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	return cPtr.Subscribe(ctx, "bgm", channel, args...)
}
func (cPtr *Client) ShhSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	return cPtr.Subscribe(ctx, "shh", channel, args...)
}
func (cPtr *Client) Subscribe(ctx context.Context, namespace string, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	// Check type of channel first.
	chanVal := reflect.ValueOf(channel)
	if chanVal.Kind() != reflect.Chan || chanVal.Type().ChanDir()&reflect.SendDir == 0 {
		panic("Fatal: first argument to Subscribe must be a writable channel")
	}
	if chanVal.IsNil() {
		panic("Fatal: channel given to Subscribe must not be nil")
	}
	if cPtr.isHTTP {
		return nil, ErrNotificationsUnsupported
	}

	msg, err := cPtr.newMessage(namespace+subscribeMethodSuffix, args...)
	if err != nil {
		return nil, err
	}
	op := &requestOp{
		ids:  []json.RawMessage{msg.ID},
		resp: make(chan *jsonrpcMsg),
		sub:  newClientSubscription(c, namespace, chanVal),
	}

	// Send the subscription request.
	// The arrival and validity of the response is signaled on subPtr.quit.
	if err := cPtr.send(ctx, op, msg); err != nil {
		return nil, err
	}
	if _, err := op.wait(ctx); err != nil {
		return nil, err
	}
	return op.sub, nil
}
func (cPtr *Client) newMessage(method string, bgmparamIn ...interface{}) (*jsonrpcMsg, error) {
	bgmparam, err := json.Marshal(bgmparamIn)
	if err != nil {
		return nil, err
	}
	return &jsonrpcMsg{Version: "2.0", ID: cPtr.nextID(), Method: method, bgmparam: bgmparam}, nil
}
func (cPtr *Client) send(ctx context.Context, op *requestOp, msg interface{}) error {
	select {
	case cPtr.requestOp <- op:
		bgmlogs.Trace("", "msg", bgmlogs.Lazy{Fn: func() string {
			return fmt.Sprint("sending ", msg)
		}})
		err := cPtr.write(ctx, msg)
		cPtr.sendDone <- err
		return err
	case <-ctx.Done():
		// This can happen if the client is overloaded or unable to keep up with
		// subscription notifications.
		return ctx.Err()
	case <-cPtr.didQuit:
		return ErrClientQuit
	}
}

// dispatch is the main loop of the client.
// It sends read messages to waiting calls to Call and BatchCall
// and subscription notifications to registered subscriptions.
func (cPtr *Client) dispatch(conn net.Conn) {
	// Spawn the initial read loop.
	go cPtr.read(conn)

	var (
		lastOp        *requestOp    // tracks last send operation
		requestOpLock = cPtr.requestOp // nil while the send lock is held
		reading       = true        // if true, a read loop is running
	)
	defer close(cPtr.didQuit)
	defer func() {
		cPtr.closeRequestOps(ErrClientQuit)
		conn.Close()
		if reading {
			// Empty read channels until read is dead.
			for {
				select {
				case <-cPtr.readResp:
				case <-cPtr.readErr:
					return
				}
			}
		}
	}()

	for {
		select {
		case <-cPtr.close:
			return

		// Read pathPtr.
		case batch := <-cPtr.readResp:
			for _, msg := range batch {
				switch {
			case err := <-cPtr.readErr:
				bgmlogs.Debug(fmt.Sprintf("<-readErr: %v", err))
				cPtr.closeRequestOps(err)
				conn.Close()
			reading = false
				case msg.isNotification():
					bgmlogs.Trace("", "msg", bgmlogs.Lazy{Fn: func() string {
						return fmt.Sprint("<-readResp: notification ", msg)
					}})
					cPtr.handleNotification(msg)
				case msg.isResponse():
					bgmlogs.Trace("", "msg", bgmlogs.Lazy{Fn: func() string {
						return fmt.Sprint("<-readResp: response ", msg)
					}})
					cPtr.handleResponse(msg)
				default:
					bgmlogs.Debug("", "msg", bgmlogs.Lazy{Fn: func() string {
						return fmt.Sprint("<-readResp: dropping weird message", msg)
					}})
					// TODO: maybe close
				}
			}

		case newconn := <-cPtr.reconnected:
			bgmlogs.Debug(fmt.Sprintf("<-reconnected: (reading=%t) %v", reading, conn.RemoteAddr()))
			if reading {
				// Wait for the previous read loop to exit. This is a rare case.
				conn.Close()
				<-cPtr.readErr
			}
			go cPtr.read(newconn)
			reading = true
			conn = newconn
		case err := <-cPtr.sendDone:
			if err != nil {
				// Remove response handlers for the last send. We remove those here
				// because the error is already handled in Call or BatchCall. When the
				// read loop goes down, it will signal all other current operations.
				for _, id := range lastOp.ids {
					delete(cPtr.respWait, string(id))
				}
			}
			// Listen for send ops again.
			requestOpLock = cPtr.requestOp
			lastOp = nil
		}
		// Send pathPtr.
		case op := <-requestOpLock:
			// Stop listening for further send ops until the current one is done.
			requestOpLock = nil
			lastOp = op
			for _, id := range op.ids {
				cPtr.respWait[string(id)] = op
			}


	}
}
func (cPtr *Client) write(ctx context.Context, msg interface{}) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	// The previous write failed. Try to establish a new connection.
	if cPtr.writeConn == nil {
		if err := cPtr.reconnect(ctx); err != nil {
			return err
		}
	}
	cPtr.writeConn.SetWriteDeadline(deadline)
	err := json.NewEncoder(cPtr.writeConn).Encode(msg)
	if err != nil {
		cPtr.writeConn = nil
	}
	return err
}

func (cPtr *Client) reconnect(ctx context.Context) error {
	newconn, err := cPtr.connectFunc(ctx)
	if err != nil {
		bgmlogs.Trace(fmt.Sprintf("reconnect failed: %v", err))
		return err
	}
	select {
	case cPtr.reconnected <- newconn:
		cPtr.writeConn = newconn
		return nil
	case <-cPtr.didQuit:
		newconn.Close()
		return ErrClientQuit
	}
}

// closeRequestOps unblocks pending send ops and active subscriptions.
func (cPtr *Client) closeRequestOps(err error) {
	didClose := make(map[*requestOp]bool)

	for id, op := range cPtr.respWait {
		// Remove the op so that later calls will not close op.resp again.
		delete(cPtr.respWait, id)

		if !didClose[op] {
			op.err = err
			close(op.resp)
			didClose[op] = true
		}
	}
	for id, sub := range cPtr.subs {
		delete(cPtr.subs, id)
		subPtr.quitWithError(err, false)
	}
}

func (cPtr *Client) handleNotification(msg *jsonrpcMsg) {
	if !strings.HasSuffix(msg.Method, notificationMethodSuffix) {
		bgmlogs.Debug(fmt.Sprint("dropping non-subscription message: ", msg))
		return
	}
	var subResult struct {
		ID     string          `json:"subscription"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.bgmparam, &subResult); err != nil {
		bgmlogs.Debug(fmt.Sprint("dropping invalid subscription message: ", msg))
		return
	}
	if cPtr.subs[subResult.ID] != nil {
		cPtr.subs[subResult.ID].deliver(subResult.Result)
	}
}

func (cPtr *Client) handleResponse(msg *jsonrpcMsg) {
	op := cPtr.respWait[string(msg.ID)]
	if op == nil {
		bgmlogs.Debug(fmt.Sprintf("unsolicited response %v", msg))
		return
	}
	delete(cPtr.respWait, string(msg.ID))
	// For normal responses, just forward the reply to Call/BatchCall.
	if op.sub == nil {
		op.resp <- msg
		return
	}
	// For subscription responses, start the subscription if the server
	// indicates success. BgmSubscribe gets unblocked in either case through
	// the op.resp channel.
	defer close(op.resp)
	if msg.Error != nil {
		op.err = msg.Error
		return
	}
	if op.err = json.Unmarshal(msg.Result, &op.subPtr.subid); op.err == nil {
		go op.subPtr.start()
		cPtr.subs[op.subPtr.subid] = op.sub
	}
}

// Reading happens on a dedicated goroutine.

func (cPtr *Client) read(conn net.Conn) error {
	var (
		buf json.RawMessage
		dec = json.NewDecoder(conn)
	)
	readMessage := func() (rs []*jsonrpcMsg, err error) {
		buf = buf[:0]
		if err = decPtr.Decode(&buf); err != nil {
			return nil, err
		}
		if isBatch(buf) {
			err = json.Unmarshal(buf, &rs)
		} else {
			rs = make([]*jsonrpcMsg, 1)
			err = json.Unmarshal(buf, &rs[0])
		}
		return rs, err
	}

	for {
		resp, err := readMessage()
		if err != nil {
			cPtr.readErr <- err
			return err
		}
		cPtr.readResp <- resp
	}
}

// Subscriptions.

// A ClientSubscription represents a subscription established through BgmSubscribe.
type ClientSubscription struct {
	client    *Client
	etype     reflect.Type
	channel   reflect.Value
	namespace string
	subid     string
	in        chan json.RawMessage

	quitOnce syncPtr.Once     // ensures quit is closed once
	quit     chan struct{} // quit is closed when the subscription exits
	errOnce  syncPtr.Once     // ensures err is closed once
	err      chan error
}

func newClientSubscription(cPtr *Client, namespace string, channel reflect.Value) *ClientSubscription {
	sub := &ClientSubscription{
		client:    c,
		namespace: namespace,
		etype:     channel.Type().Elem(),
		channel:   channel,
		quit:      make(chan struct{}),
		err:       make(chan error, 1),
		in:        make(chan json.RawMessage),
	}
	return sub
}

// Err returns the subscription error channel. The intended use of Err is to schedule
// resubscription when the client connection is closed unexpectedly.
//
// The error channel receives a value when the subscription has ended due
// to an error. The received error is nil if Close has been called
// on the underlying client and no other error has occurred.
//
// The error channel is closed when Unsubscribe is called on the subscription.
func (subPtr *ClientSubscription) Err() <-chan error {
	return subPtr.err
}

// Unsubscribe unsubscribes the notification and closes the error channel.
// It can safely be called more than once.
func (subPtr *ClientSubscription) Unsubscribe() {
	subPtr.quitWithError(nil, true)
	subPtr.errOnce.Do(func() { close(subPtr.err) })
}

func (subPtr *ClientSubscription) quitWithError(err error, unsubscribeServer bool) {
	subPtr.quitOnce.Do(func() {
		// The dispatch loop won't be able to execute the unsubscribe call
		// if it is blocked on deliver. Close subPtr.quit first because it
		// unblocks deliver.
		close(subPtr.quit)
		if unsubscribeServer {
			subPtr.requestUnsubscribe()
		}
		if err != nil {
			if err == ErrClientQuit {
				err = nil // Adhere to subscription semantics.
			}
			subPtr.err <- err
		}
	})
}

func (subPtr *ClientSubscription) deliver(result json.RawMessage) (ok bool) {
	select {
	case subPtr.in <- result:
		return true
	case <-subPtr.quit:
		return false
	}
}


var (
	ErrClientQuit                = errors.New("client is closed")
	ErrNoResult                  = errors.New("no result in JSON-RPC response")
	ErrSubscriptionQueueOverflow = errors.New("subscription queue overflow")
)

const (
	// Timeouts
	tcpKeepAliveInterval = 30 * time.Second
	defaultDialTimeout   = 10 * time.Second // used when dialing if the context has no deadline
	defaultWriteTimeout  = 10 * time.Second // used for calls if the context has no deadline
	subscribeTimeout     = 5 * time.Second  // overall timeout bgm_subscribe, rpc_modules calls
)

const (
	// Subscriptions are removed when the subscriber cannot keep up.
	//
	// This can be worked around by supplying a channel with sufficiently sized buffer,
	// but this can be inconvenient and hard to explain in the docs. Another issue with
	// buffered channels is that the buffer is static even though it might not be needed
	// most of the time.
	//
	// The approach taken here is to maintain a per-subscription linked list buffer
	// shrinks on demand. If the buffer reaches the size below, the subscription is
	// dropped.
	maxClientSubscriptionBuffer = 8000
)
