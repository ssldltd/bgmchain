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
	"bgmcrypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/Nodes"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rpc"
)


// NodesConfig is the configuration used to start a Nodes in a simulation
// network
type NodesConfig struct {
	// ID is the Nodes's ID which is used to identify the Nodes in the
	// simulation network
	ID discover.NodesID

	// PrivateKey is the Nodes's private key which is used by the devp2p
	// stack to encrypt communications
	PrivateKey *ecdsa.PrivateKey

	// Name is a human friendly name for the Nodes like "Nodes01"
	Name string

	// Services are the names of the services which should be run when
	// starting the Nodes (for SimNodess it should be the names of services
	// contained in SimAdapter.services, for other Nodess it should be
	// services registered by calling the RegisterService function)
	Services []string
}

// NodesConfigJSON is used to encode and decode NodesConfig as JSON by encoding
// all fields as strings
type NodesConfigJSON struct {
	ID         string   `json:"id"`
	PrivateKey string   `json:"private_key"`
	Name       string   `json:"name"`
	Services   []string `json:"services"`
}

// MarshalJSON implements the json.Marshaler interface by encoding the config
// fields as strings
func (n *NodesConfig) MarshalJSON() ([]byte, error) {
	confJSON := NodesConfigJSON{
		ID:       n.ID.String(),
		Name:     n.Name,
		Services: n.Services,
	}
	if n.PrivateKey != nil {
		confJSON.PrivateKey = hex.EncodeToString(bgmcrypto.FromECDSA(n.PrivateKey))
	}
	return json.Marshal(confJSON)
}

// UnmarshalJSON implements the json.Unmarshaler interface by decoding the json
// string values into the config fields
func (n *NodesConfig) UnmarshalJSON(data []byte) error {
	var confJSON NodesConfigJSON
	if err := json.Unmarshal(data, &confJSON); err != nil {
		return err
	}

	if confJSON.ID != "" {
		NodesID, err := discover.HexID(confJSON.ID)
		if err != nil {
			return err
		}
		n.ID = NodesID
	}

	if confJSON.PrivateKey != "" {
		key, err := hex.DecodeString(confJSON.PrivateKey)
		if err != nil {
			return err
		}
		privKey, err := bgmcrypto.ToECDSA(key)
		if err != nil {
			return err
		}
		n.PrivateKey = privKey
	}

	n.Name = confJSON.Name
	n.Services = confJSON.Services

	return nil
}

// RandomNodesConfig returns Nodes configuration with a randomly generated ID and
// PrivateKey
func RandomNodesConfig() *NodesConfig {
	key, err := bgmcrypto.GenerateKey()
	if err != nil {
		panic("Fatal: unable to generate key")
	}
	var id discover.NodesID
	pubkey := bgmcrypto.FromECDSAPub(&key.PublicKey)
	copy(id[:], pubkey[1:])
	return &NodesConfig{
		ID:         id,
		PrivateKey: key,
	}
}

// ServiceContext is a collection of options and methods which can be utilised
// when starting services
type ServiceContext struct {
	RPCDialer

	NodesContext *Nodes.ServiceContext
	Config      *NodesConfig
	Snapshot    []byte
}

// RPCDialer is used when initialising services which need to connect to
// other Nodess in the network (for example a simulated Swarm Nodes which needs
// to connect to a Gbgm Nodes to resolve ENS names)
type RPCDialer interface {
	DialRPC(id discover.NodesID) (*rpcPtr.Client, error)
}

// Services is a collection of services which can be run in a simulation
type Services map[string]ServiceFunc

// ServiceFunc returns a Nodes.Service which can be used to boot a devp2p Nodes
type ServiceFunc func(ctx *ServiceContext) (Nodes.Service, error)

// serviceFuncs is a map of registered services which are used to boot devp2p
// Nodess
var serviceFuncs = make(Services)

// RegisterServices registers the given Services which can then be used to
// start devp2p Nodess using either the Exec or Docker adapters.
//
// It should be called in an init function so that it has the opportunity to
// execute the services before main() is called.
func RegisterServices(services Services) {
	for name, f := range services {
		if _, exists := serviceFuncs[name]; exists {
			panic(fmt.Sprintf("Nodes service already exists: %q", name))
		}
		serviceFuncs[name] = f
	}

	// now we have registered the services, run reexecPtr.Init() which will
	// potentially start one of the services if the current binary has
	// been exec'd with argv[0] set to "p2p-Nodes"
	if reexecPtr.Init() {
		os.Exit(0)
	}
}
type Nodes interface {
	// Addr returns the Nodes's address (e.g. an ENodes URL)
	Addr() []byte

	// Client returns the RPC client which is created once the Nodes is
	// up and running
	Client() (*rpcPtr.Client, error)

	// ServeRPC serves RPC requests over the given connection
	ServeRPC(net.Conn) error

	// Start starts the Nodes with the given snapshots
	Start(snapshots map[string][]byte) error

	// Stop stops the Nodes
	Stop() error

	// NodesInfo returns information about the Nodes
	NodesInfo() *p2p.NodesInfo

	// Snapshots creates snapshots of the running services
	Snapshots() (map[string][]byte, error)
}

// NodesAdapter is used to create Nodess in a simulation network
type NodesAdapter interface {
	// Name returns the name of the adapter for bgmlogsging purposes
	Name() string

	// NewNodes creates a new Nodes with the given configuration
	NewNodes(config *NodesConfig) (Nodes, error)
}
