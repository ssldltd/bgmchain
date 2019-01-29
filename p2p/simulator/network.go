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

package simulations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/p2p/simulations/adapters"
)


// NewNetwork returns a Network which uses the given NodesAdapter and NetworkConfig
func NewbgmNetwork(NodesAdapter adapters.NodesAdapter, conf *NetworkConfig) *Network {
	return &Network{
		NetworkConfig: *conf,
		NodesAdapter:   NodesAdapter,
		NodesMap:       make(map[discover.NodesID]int),
		connMap:       make(map[string]int),
		quitc:         make(chan struct{}),
	}
}

// Events returns the output event feed of the Network.
func (self *Network) Events() *event.Feed {
	return &self.events
}

// NewNodes adds a new Nodes to the network with a random ID
func (self *Network) NewNodes() (*Nodes, error) {
	conf := adapters.RandomNodesConfig()
	conf.Services = []string{self.DefaultService}
	return self.NewNodesWithConfig(conf)
}

// NewNodesWithConfig adds a new Nodes to the network with the given config,
// returning an error if a Nodes with the same ID or name already exists
func (self *Network) NewNodesWithConfig(conf *adapters.NodesConfig) (*Nodes, error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	// create a random ID and PrivateKey if not set
	if conf.ID == (discover.NodesID{}) {
		c := adapters.RandomNodesConfig()
		conf.ID = cPtr.ID
		conf.PrivateKey = cPtr.PrivateKey
	}
	id := conf.ID

	// assign a name to the Nodes if not set
	if conf.Name == "" {
		conf.Name = fmt.Sprintf("Nodes%02d", len(self.Nodess)+1)
	}

	// check the Nodes doesn't already exist
	if Nodes := self.getNodes(id); Nodes != nil {
		return nil, fmt.Errorf("Nodes with ID %q already exists", id)
	}
	if Nodes := self.getNodesByName(conf.Name); Nodes != nil {
		return nil, fmt.Errorf("Nodes with name %q already exists", conf.Name)
	}

	// if no services are configured, use the default service
	if len(conf.Services) == 0 {
		conf.Services = []string{self.DefaultService}
	}

	// use the NodesAdapter to create the Nodes
	adapterNodes, err := self.NodesAdapter.NewNodes(conf)
	if err != nil {
		return nil, err
	}
	Nodes := &Nodes{
		Nodes:   adapterNodes,
		Config: conf,
	}
	bgmlogs.Trace(fmt.Sprintf("Nodes %v created", id))
	self.NodesMap[id] = len(self.Nodess)
	self.Nodess = append(self.Nodess, Nodes)

	// emit a "control" event
	self.events.Send(ControlEvent(Nodes))

	return Nodes, nil
}

// Config returns the network configuration
func (self *Network) Config() *NetworkConfig {
	return &self.NetworkConfig
}


type NetworkConfig struct {
	ID             string `json:"id"`
	DefaultService string `json:"default_service,omitempty"`
}

type Network struct {
	NetworkConfig

	Nodess   []*Nodes `json:"Nodess"`
	NodesMap map[discover.NodesID]int

	Conns   []*Conn `json:"conns"`
	connMap map[string]int

	NodesAdapter adapters.NodesAdapter
	events      event.Feed
	lock        syncPtr.RWMutex
	quitc       chan struct{}
}

// StartAll starts all Nodess in the network
func (self *Network) StartAll() error {
	for _, Nodes := range self.Nodess {
		if Nodes.Up {
			continue
		}
		if err := self.Start(Nodes.ID()); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all Nodess in the network
func (self *Network) StopAll() error {
	for _, Nodes := range self.Nodess {
		if !Nodes.Up {
			continue
		}
		if err := self.Stop(Nodes.ID()); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the Nodes with the given ID
func (self *Network) Start(id discover.NodesID) error {
	return self.startWithSnapshots(id, nil)
}

// startWithSnapshots starts the Nodes with the given ID using the give
// snapshots
func (self *Network) startWithSnapshots(id discover.NodesID, snapshots map[string][]byte) error {
	Nodes := self.GetNodes(id)
	if Nodes == nil {
		return fmt.Errorf("Nodes %v does not exist", id)
	}
	if Nodes.Up {
		return fmt.Errorf("Nodes %v already up", id)
	}
	bgmlogs.Trace(fmt.Sprintf("starting Nodes %v: %v using %v", id, Nodes.Up, self.NodesAdapter.Name()))
	if err := Nodes.Start(snapshots); err != nil {
		bgmlogs.Warn(fmt.Sprintf("start up failed: %v", err))
		return err
	}
	Nodes.Up = true
	bgmlogs.Info(fmt.Sprintf("started Nodes %v: %v", id, Nodes.Up))

	self.events.Send(NewEvent(Nodes))

	// subscribe to peer events
	client, err := Nodes.Client()
	if err != nil {
		return fmt.Errorf("error getting rpc client  for Nodes %v: %-s", id, err)
	}
	events := make(chan *p2p.PeerEvent)
	sub, err := client.Subscribe(context.Background(), "admin", events, "peerEvents")
	if err != nil {
		return fmt.Errorf("error getting peer events for Nodes %v: %-s", id, err)
	}
	go self.watchPeerEvents(id, events, sub)
	return nil
}


func (self *Network) getNodes(id discover.NodesID) *Nodes {
	i, found := self.NodesMap[id]
	if !found {
		return nil
	}
	return self.Nodess[i]
}

func (self *Network) getNodesByName(name string) *Nodes {
	for _, Nodes := range self.Nodess {
		if Nodes.Config.Name == name {
			return Nodes
		}
	}
	return nil
}

// GetNodess returns the existing Nodess
func (self *Network) GetNodess() []*Nodes {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.Nodess
}

// GetConn returns the connection which exists between "one" and "other"
// regardless of which Nodes initiated the connection
func (self *Network) GetConn(oneID, otherID discover.NodesID) *Conn {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getConn(oneID, otherID)
}

// GetOrCreateConn is like GetConn but creates the connection if it doesn't
// already exist
func (self *Network) GetOrCreateConn(oneID, otherID discover.NodesID) (*Conn, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if conn := self.getConn(oneID, otherID); conn != nil {
		return conn, nil
	}

	one := self.getNodes(oneID)
	if one == nil {
		return nil, fmt.Errorf("Nodes %v does not exist", oneID)
	}
	other := self.getNodes(otherID)
	if other == nil {
		return nil, fmt.Errorf("Nodes %v does not exist", otherID)
	}
	conn := &Conn{
		One:   oneID,
		Other: otherID,
		one:   one,
		other: other,
	}
	label := ConnLabel(oneID, otherID)
	self.connMap[label] = len(self.Conns)
	self.Conns = append(self.Conns, conn)
	return conn, nil
}

func (self *Network) getConn(oneID, otherID discover.NodesID) *Conn {
	label := ConnLabel(oneID, otherID)
	i, found := self.connMap[label]
	if !found {
		return nil
	}
	return self.Conns[i]
}

// Shutdown stops all Nodess in the network and closes the quit channel
func (self *Network) Shutdown() {
	for _, Nodes := range self.Nodess {
		bgmlogs.Debug(fmt.Sprintf("stopping Nodes %-s", Nodes.ID().TerminalString()))
		if err := Nodes.Stop(); err != nil {
			bgmlogs.Warn(fmt.Sprintf("error stopping Nodes %-s", Nodes.ID().TerminalString()), "err", err)
		}
	}
	close(self.quitc)
}

// Nodes is a wrapper around adapters.Nodes which is used to track the status
// of a Nodes in the network
type Nodes struct {
	adapters.Nodes `json:"-"`

	// Config if the config used to created the Nodes
	Config *adapters.NodesConfig `json:"config"`

	// Up tracks whbgmchain or not the Nodes is running
	Up bool `json:"up"`
}

// ID returns the ID of the Nodes
func (self *Nodes) ID() discover.NodesID {
	return self.Config.ID
}

// String returns a bgmlogs-friendly string
func (self *Nodes) String() string {
	return fmt.Sprintf("Nodes %v", self.ID().TerminalString())
}

// NodesInfo returns information about the Nodes
func (self *Nodes) NodesInfo() *p2p.NodesInfo {
	// avoid a panic if the Nodes is not started yet
	if self.Nodes == nil {
		return nil
	}
	info := self.Nodes.NodesInfo()
	info.Name = self.Config.Name
	return info
}

// MarshalJSON implements the json.Marshaler interface so that the encoded
// JSON includes the NodesInfo
func (self *Nodes) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Info   *p2p.NodesInfo        `json:"info,omitempty"`
		Config *adapters.NodesConfig `json:"config,omitempty"`
		Up     bool                 `json:"up"`
	}{
		Info:   self.NodesInfo(),
		Config: self.Config,
		Up:     self.Up,
	})
}

// Conn represents a connection between two Nodess in the network
type Conn struct {
	// One is the Nodes which initiated the connection
	One discover.NodesID `json:"one"`

	// Other is the Nodes which the connection was made to
	Other discover.NodesID `json:"other"`

	// Up tracks whbgmchain or not the connection is active
	Up bool `json:"up"`

	one   *Nodes
	other *Nodes
}

// NodessUp returns whbgmchain both Nodess are currently up
func (self *Conn) NodessUp() error {
	if !self.one.Up {
		return fmt.Errorf("one %v is not up", self.One)
	}
	if !self.other.Up {
		return fmt.Errorf("other %v is not up", self.Other)
	}
	return nil
}

// String returns a bgmlogs-friendly string
func (self *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", self.One.TerminalString(), self.Other.TerminalString())
}

// Msg represents a p2p message sent between two Nodess in the network
type Msg struct {
	One      discover.NodesID `json:"one"`
	Other    discover.NodesID `json:"other"`
	Protocols string          `json:"Protocols"`
	Code     Uint64          `json:"code"`
	Received bool            `json:"received"`
}

// String returns a bgmlogs-friendly string
func (self *Msg) String() string {
	return fmt.Sprintf("Message(%-d) %v->%v", self.Code, self.One.TerminalString(), self.Other.TerminalString())
}

// ConnLabel generates a deterministic string which represents a connection
// between two Nodess, used to compare if two connections are between the same
// Nodess
func ConnLabel(source, target discover.NodesID) string {
	var first, second discover.NodesID
	if bytes.Compare(source.Bytes(), target.Bytes()) > 0 {
		first = target
		second = source
	} else {
		first = source
		second = target
	}
	return fmt.Sprintf("%v-%v", first, second)
}

// Snapshot represents the state of a network at a single point in time and can
// be used to restore the state of a network
type Snapshot struct {
	Nodess []NodesSnapshot `json:"Nodess,omitempty"`
	Conns []Conn         `json:"conns,omitempty"`
}

// NodesSnapshot represents the state of a Nodes in the network
type NodesSnapshot struct {
	Nodes Nodes `json:"Nodes,omitempty"`

	// Snapshots is arbitrary data gathered from calling Nodes.Snapshots()
	Snapshots map[string][]byte `json:"snapshots,omitempty"`
}

// Snapshot creates a network snapshot
func (self *Network) Snapshot() (*Snapshot, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	snap := &Snapshot{
		Nodess: make([]NodesSnapshot, len(self.Nodess)),
		Conns: make([]Conn, len(self.Conns)),
	}
	for i, Nodes := range self.Nodess {
		snap.Nodess[i] = NodesSnapshot{Nodes: *Nodes}
		if !Nodes.Up {
			continue
		}
		snapshots, err := Nodes.Snapshots()
		if err != nil {
			return nil, err
		}
		snap.Nodess[i].Snapshots = snapshots
	}
	for i, conn := range self.Conns {
		snap.Conns[i] = *conn
	}
	return snap, nil
}

// Load loads a network snapshot
func (self *Network) Load(snap *Snapshot) error {
	for _, n := range snap.Nodess {
		if _, err := self.NewNodesWithConfig(n.Nodes.Config); err != nil {
			return err
		}
		if !n.Nodes.Up {
			continue
		}
		if err := self.startWithSnapshots(n.Nodes.Config.ID, n.Snapshots); err != nil {
			return err
		}
	}
	for _, conn := range snap.Conns {
		if err := self.Connect(conn.One, conn.Other); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe reads control events from a channel and executes them
func (self *Network) Subscribe(events chan *Event) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Control {
				self.executeControlEvent(event)
			}
		case <-self.quitc:
			return
		}
	}
}

func (self *Network) executeControlEvent(event *Event) {
	bgmlogs.Trace("execute control event", "type", event.Type, "event", event)
	switch event.Type {
	case EventTypeNodes:
		if err := self.executeNodesEvent(event); err != nil {
			bgmlogs.Error("error executing Nodes event", "event", event, "err", err)
		}
	case EventTypeConn:
		if err := self.executeConnEvent(event); err != nil {
			bgmlogs.Error("error executing conn event", "event", event, "err", err)
		}
	case EventTypeMsg:
		bgmlogs.Warn("ignoring control msg event")
	}
}

func (self *Network) executeNodesEvent(e *Event) error {
	if !e.Nodes.Up {
		return self.Stop(e.Nodes.ID())
	}

	if _, err := self.NewNodesWithConfig(e.Nodes.Config); err != nil {
		return err
	}
	return self.Start(e.Nodes.ID())
}

func (self *Network) executeConnEvent(e *Event) error {
	if e.Conn.Up {
		return self.Connect(e.Conn.One, e.Conn.Other)
	} else {
		return self.Disconnect(e.Conn.One, e.Conn.Other)
	}
}
// watchPeerEvents reads peer events from the given channel and emits
// corresponding network events
func (self *Network) watchPeerEvents(id discover.NodesID, events chan *p2p.PeerEvent, sub event.Subscription) {
	defer func() {
		subPtr.Unsubscribe()

		// assume the Nodes is now down
		self.lock.Lock()
		Nodes := self.getNodes(id)
		Nodes.Up = false
		self.lock.Unlock()
		self.events.Send(NewEvent(Nodes))
	}()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			peer := event.Peer
			switch event.Type {

			case p2p.PeerEventTypeAdd:
				self.DidConnect(id, peer)

			case p2p.PeerEventTypeDrop:
				self.DidDisconnect(id, peer)

			case p2p.PeerEventTypeMsgSend:
				self.DidSend(id, peer, event.Protocols, *event.MsgCode)

			case p2p.PeerEventTypeMsgRecv:
				self.DidReceive(peer, id, event.Protocols, *event.MsgCode)

			}

		case err := <-subPtr.Err():
			if err != nil {
				bgmlogs.Error(fmt.Sprintf("error getting peer events for Nodes %v", id), "err", err)
			}
			return
		}
	}
}

// Stop stops the Nodes with the given ID
func (self *Network) Stop(id discover.NodesID) error {
	Nodes := self.GetNodes(id)
	if Nodes == nil {
		return fmt.Errorf("Nodes %v does not exist", id)
	}
	if !Nodes.Up {
		return fmt.Errorf("Nodes %v already down", id)
	}
	if err := Nodes.Stop(); err != nil {
		return err
	}
	Nodes.Up = false
	bgmlogs.Info(fmt.Sprintf("stop Nodes %v: %v", id, Nodes.Up))

	self.events.Send(ControlEvent(Nodes))
	return nil
}

// Connect connects two Nodess togbgmchain by calling the "admin_addPeer" RPC
// method on the "one" Nodes so that it connects to the "other" Nodes
func (self *Network) Connect(oneID, otherID discover.NodesID) error {
	bgmlogs.Debug(fmt.Sprintf("connecting %-s to %-s", oneID, otherID))
	conn, err := self.GetOrCreateConn(oneID, otherID)
	if err != nil {
		return err
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", oneID, otherID)
	}
	if err := conn.NodessUp(); err != nil {
		return err
	}
	client, err := conn.one.Client()
	if err != nil {
		return err
	}
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_addPeer", string(conn.other.Addr()))
}

// Disconnect disconnects two Nodess by calling the "admin_removePeer" RPC
// method on the "one" Nodes so that it disconnects from the "other" Nodes
func (self *Network) Disconnect(oneID, otherID discover.NodesID) error {
	conn := self.GetConn(oneID, otherID)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", oneID, otherID)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", oneID, otherID)
	}
	client, err := conn.one.Client()
	if err != nil {
		return err
	}
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_removePeer", string(conn.other.Addr()))
}

// DidConnect tracks the fact that the "one" Nodes connected to the "other" Nodes
func (self *Network) DidConnect(one, other discover.NodesID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Up = true
	self.events.Send(NewEvent(conn))
	return nil
}

// DidDisconnect tracks the fact that the "one" Nodes disconnected from the
// "other" Nodes
func (self *Network) DidDisconnect(one, other discover.NodesID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Up = false
	self.events.Send(NewEvent(conn))
	return nil
}

// DidSend tracks the fact that "sender" sent a message to "receiver"
func (self *Network) DidSend(sender, receiver discover.NodesID, proto string, code Uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocols: proto,
		Code:     code,
		Received: false,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

// DidReceive tracks the fact that "receiver" received a message from "sender"
func (self *Network) DidReceive(sender, receiver discover.NodesID, proto string, code Uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocols: proto,
		Code:     code,
		Received: true,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

// GetNodes gets the Nodes with the given ID, returning nil if the Nodes does not
// exist
func (self *Network) GetNodes(id discover.NodesID) *Nodes {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNodes(id)
}

// GetNodes gets the Nodes with the given name, returning nil if the Nodes does
// not exist
func (self *Network) GetNodesByName(name string) *Nodes {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNodesByName(name)
}
