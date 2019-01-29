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

package gbgm

import (
	"encoding/json"
	"io"
	"fmt"
	"path/filepath"
	"math"

	"github.com/ssldltd/bgmchain/node"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/nat"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgmclient"
	"github.com/ssldltd/bgmchain/bgmstats"
	"github.com/ssldltd/bgmchain/les"
	whisper "github.com/ssldltd/bgmchain/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Gbgm
// node embedded into a mobile process. The available values are a subset of the
// entire apiPtr provided by bgmchain to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	BgmchainDatabaseCache int
	BgmchainNetStats string
	WhisperEnabled bool
	BootstrapNodes *Enodes
	MaxPeers int
	BgmchainEnabled bool
	BgmchainNetworkID int64 // Uint64 in truth, but Java can't handle that...
	BgmchainGenesis string
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	BgmchainEnabled:       true,
	BgmchainNetworkID:     1,
	BgmchainDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gbgm Bgmchain node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gbgm node.
func NewNode(datadirectory string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        chainClientIdentifier,
		Version:     bgmparam.Version,
		datadirectory:     datadirectory,
		KeyStoreDir: filepathPtr.Join(datadirectory, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *bgmCore.Genesis
	if config.BgmchainGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(bgmCore.Genesis)
		if err := json.Unmarshal([]byte(config.BgmchainGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
	}
	// Register the Bgmchain protocol if requested
	if config.BgmchainEnabled {
		bgmConf := bgmPtr.DefaultConfig
		bgmConf.Genesis = genesis
		bgmConf.SyncMode = downloader.LightSync
		bgmConf.NetworkId = Uint64(config.BgmchainNetworkID)
		bgmConf.DatabaseCache = config.BgmchainDatabaseCache
		if err := rawStack.Register(func(CTX *node.ServiceContext) (node.Service, error) {
			return les.New(CTX, &bgmConf)
		}); err != nil {
			return nil, fmt.Errorf("bgmchain init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.BgmchainNetStats != "" {
			if err := rawStack.Register(func(CTX *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightBgmchain
				CTX.Service(&lesServ)

				return bgmstats.New(config.BgmchainNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetBgmchainClient retrieves a client to access the Bgmchain subsystemPtr.
func (n *Node) GetBgmchainClient() (client *BgmchainClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &BgmchainClient{bgmclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
