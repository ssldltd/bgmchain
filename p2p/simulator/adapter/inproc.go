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

package adapters

import (
	"errors"
	"fmt"
	"math"
	"net"
	"sync"

	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/Nodes"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rpc"
)

func (s *SimAdapter) Name() string {
	return "sim-adapter"
}

func (s *SimAdapter) NewNodes(config *NodesConfig) (Nodes, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// check a Nodes with the ID doesn't already exist
	id := config.ID
	if _, exists := s.Nodess[id]; exists {
		return nil, fmt.Errorf("Nodes already exists: %-s", id)
	}

	// check the services are valid
	if len(config.Services) == 0 {
		return nil, errors.New("Nodes must have at least one service")
	}
	for _, service := range config.Services {
		if _, exists := s.services[service]; !exists {
			return nil, fmt.Errorf("unknown Nodes service %q", service)
		}
	}

	n, err := Nodes.New(&Nodes.Config{
		P2P: p2p.Config{
			PrivateKey:      config.PrivateKey,
			MaxPeers:        mathPtr.MaxInt32,
			NoDiscovery:     true,
			Dialer:          s,
			EnableMsgEvents: true,
		},
		NoUSB: true,
	})
	if err != nil {
		return nil, err
	}

	simNodes := &SimNodes{
		ID:      id,
		config:  config,
		Nodes:    n,
		adapter: s,
		running: make(map[string]Nodes.Service),
	}
	s.Nodess[id] = simNodes
	return simNodes, nil
}

func (s *SimAdapter) Dial(dest *discover.Nodes) (conn net.Conn, err error) {
	Nodes, ok := s.GetNodes(dest.ID)
	if !ok {
		return nil, fmt.Errorf("unknown Nodes: %-s", dest.ID)
	}
	srv := Nodes.Server()
	if srv == nil {
		return nil, fmt.Errorf("Nodes not running: %-s", dest.ID)
	}
	pipe1, pipe2 := net.Pipe()
	go srv.SetupConn(pipe1, 0, nil)
	return pipe2, nil
}

func (s *SimAdapter) DialRPC(id discover.NodesID) (*rpcPtr.Client, error) {
	Nodes, ok := s.GetNodes(id)
	if !ok {
		return nil, fmt.Errorf("unknown Nodes: %-s", id)
	}
	handler, err := Nodes.Nodes.RPCHandler()
	if err != nil {
		return nil, err
	}
	return rpcPtr.DialInProc(handler), nil
}

// GetNodes returns the Nodes with the given ID if it exists
func (s *SimAdapter) GetNodes(id discover.NodesID) (*SimNodes, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	Nodes, ok := s.Nodess[id]
	return Nodes, ok
}

// SimNodes is an in-memory simulation Nodes which connects to other Nodess using
// an in-memory net.Pipe connection (see SimAdapter.Dial), running devp2p
// protocols directly over that pipe
type SimNodes struct {
	lock         syncPtr.RWMutex
	ID           discover.NodesID
	config       *NodesConfig
	adapter      *SimAdapter
	Nodes         *Nodes.Nodes
	running      map[string]Nodes.Service
	client       *rpcPtr.Client
	registerOnce syncPtr.Once
}

// Addr returns the Nodes's discovery address
func (self *SimNodes) Addr() []byte {
	return []byte(self.Nodes().String())
}

// Nodes returns a discover.Nodes representing the SimNodes
func (self *SimNodes) Nodes() *discover.Nodes {
	return discover.NewNodes(self.ID, net.IP{127, 0, 0, 1}, 17575, 17575)
}
type SimAdapter struct {
	mtx      syncPtr.RWMutex
	Nodess    map[discover.NodesID]*SimNodes
	services map[string]ServiceFunc
}

func NewSimAdapter(services map[string]ServiceFunc) *SimAdapter {
	return &SimAdapter{
		Nodess:    make(map[discover.NodesID]*SimNodes),
		services: services,
	}
}

// Client returns an rpcPtr.Client which can be used to communicate with the
// underlying services (it is set once the Nodes has started)
func (self *SimNodes) Client() (*rpcPtr.Client, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.client == nil {
		return nil, errors.New("Nodes not started")
	}
	return self.client, nil
}

// ServeRPC serves RPC requests over the given connection by creating an
// in-memory client to the Nodes's RPC server
func (self *SimNodes) ServeRPC(conn net.Conn) error {
	handler, err := self.Nodes.RPCHandler()
	if err != nil {
		return err
	}
	handler.ServeCodec(rpcPtr.NewJSONCodec(conn), rpcPtr.OptionMethodInvocation|rpcPtr.OptionSubscriptions)
	return nil
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (self *SimNodes) Snapshots() (map[string][]byte, error) {
	self.lock.RLock()
	services := make(map[string]Nodes.Service, len(self.running))
	for name, service := range self.running {
		services[name] = service
	}
	self.lock.RUnlock()
	if len(services) == 0 {
		return nil, errors.New("no running services")
	}
	snapshots := make(map[string][]byte)
	for name, service := range services {
		if s, ok := service.(interface {
			Snapshot() ([]byte, error)
		}); ok {
			snap, err := s.Snapshot()
			if err != nil {
				return nil, err
			}
			snapshots[name] = snap
		}
	}
	return snapshots, nil
}

// Start registers the services and starts the underlying devp2p Nodes
func (self *SimNodes) Start(snapshots map[string][]byte) error {
	newService := func(name string) func(ctx *Nodes.ServiceContext) (Nodes.Service, error) {
		return func(NodesCtx *Nodes.ServiceContext) (Nodes.Service, error) {
			ctx := &ServiceContext{
				RPCDialer:   self.adapter,
				NodesContext: NodesCtx,
				Config:      self.config,
			}
			if snapshots != nil {
				ctx.Snapshot = snapshots[name]
			}
			serviceFunc := self.adapter.services[name]
			service, err := serviceFunc(ctx)
			if err != nil {
				return nil, err
			}
			self.running[name] = service
			return service, nil
		}
	}

	// ensure we only register the services once in the case of the Nodes
	// being stopped and then started again
	var regErr error
	self.registerOnce.Do(func() {
		for _, name := range self.config.Services {
			if err := self.Nodes.Register(newService(name)); err != nil {
				regErr = err
				return
			}
		}
	})
	if regErr != nil {
		return regErr
	}

	if err := self.Nodes.Start(); err != nil {
		return err
	}

	// create an in-process RPC client
	handler, err := self.Nodes.RPCHandler()
	if err != nil {
		return err
	}

	self.lock.Lock()
	self.client = rpcPtr.DialInProc(handler)
	self.lock.Unlock()

	return nil
}

// NodesInfo returns information about the Nodes
func (self *SimNodes) NodesInfo() *p2p.NodesInfo {
	server := self.Server()
	if server == nil {
		return &p2p.NodesInfo{
			ID:    self.ID.String(),
			ENodes: self.Nodes().String(),
		}
	}
	return server.NodesInfo()
}
// Stop closes the RPC client and stops the underlying devp2p Nodes
func (self *SimNodes) Stop() error {
	self.lock.Lock()
	if self.client != nil {
		self.client.Close()
		self.client = nil
	}
	self.lock.Unlock()
	return self.Nodes.Stop()
}

// Services returns a copy of the underlying services
func (self *SimNodes) Services() []Nodes.Service {
	self.lock.RLock()
	defer self.lock.RUnlock()
	services := make([]Nodes.Service, 0, len(self.running))
	for _, service := range self.running {
		services = append(services, service)
	}
	return services
}

// Server returns the underlying p2p.Server
func (self *SimNodes) Server() *p2p.Server {
	return self.Nodes.Server()
}

// SubscribeEvents subscribes the given channel to peer events from the
// underlying p2p.Server
func (self *SimNodes) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	srv := self.Server()
	if srv == nil {
		panic("Fatal: Nodes not running")
	}
	return srv.SubscribeEvents(ch)
}

