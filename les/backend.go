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
// Package les implement the Light Bgmchain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/dpos"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/filters"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/internal/bgmapi"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/node"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discv5"
	"github.com/ssldltd/bgmchain/bgmparam"
	rpc "github.com/ssldltd/bgmchain/rpc"
)

type LightBgmchain struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *bgmparam.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb bgmdbPtr.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *bgmCore.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     Uint64
	netRPCService *bgmapi.PublicNetAPI

	wg syncPtr.WaitGroup
}

func New(CTX *node.ServiceContext, config *bgmPtr.Config) (*LightBgmchain, error) {
	chainDb, err := bgmPtr.CreateDB(CTX, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := bgmCore.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*bgmparam.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	bgmlogs.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lbgm := &LightBgmchain{
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         CTX.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   CTX.AccountManager,
		engine:           dpos.New(chainConfig.Dpos, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     bgmPtr.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lbgmPtr.relay = NewLesTxRelay(peers, lbgmPtr.reqDist)
	lbgmPtr.serverPool = newServerPool(chainDb, quitSync, &lbgmPtr.wg)
	lbgmPtr.retriever = newRetrieveManager(peers, lbgmPtr.reqDist, lbgmPtr.serverPool)
	lbgmPtr.odr = NewLesOdr(chainDb, lbgmPtr.chtIndexer, lbgmPtr.bloomTrieIndexer, lbgmPtr.bloomIndexer, lbgmPtr.retriever)
	if lbgmPtr.blockchain, err = light.NewLightChain(lbgmPtr.odr, lbgmPtr.chainConfig, lbgmPtr.engine); err != nil {
		return nil, err
	}
	lbgmPtr.bloomIndexer.Start(lbgmPtr.blockchain)
	// Rewind the chain in case of an Notcompatible config upgrade.
	if compat, ok := genesisErr.(*bgmparam.ConfigCompatError); ok {
		bgmlogs.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lbgmPtr.blockchain.SetHead(compat.RewindTo)
		bgmCore.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lbgmPtr.txPool = light.NewTxPool(lbgmPtr.chainConfig, lbgmPtr.blockchain, lbgmPtr.relay)
	if lbgmPtr.protocolManager, err = NewProtocolManager(lbgmPtr.chainConfig, true, ClientProtocolVersions, config.NetworkId, lbgmPtr.eventMux, lbgmPtr.engine, lbgmPtr.peers, lbgmPtr.blockchain, nil, chainDb, lbgmPtr.odr, lbgmPtr.relay, quitSync, &lbgmPtr.wg); err != nil {
		return nil, err
	}
	lbgmPtr.ApiBackend = &LesApiBackend{lbgm, nil}
	gpobgmparam := config.GPO
	if gpobgmparam.Default == nil {
		gpobgmparam.Default = config.GasPrice
	}
	lbgmPtr.ApiBackend.gpo = gasprice.NewOracle(lbgmPtr.ApiBackend, gpobgmparam)
	return lbgm, nil
}

func lesTopic(genesisHash bgmcommon.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + bgmcommon.Bytes2Hex(genesisHashPtr.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Coinbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Coinbase() (bgmcommon.Address, error) {
	return bgmcommon.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the bgmchain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightBgmchain) APIs() []rpcPtr.apiPtr {
	return append(bgmapi.GetAPIs(s.ApiBackend), []rpcPtr.apiPtr{
		{
			Namespace: "bgm",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightBgmchain) ResetWithGenesisBlock(gbPtr *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightBgmchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightBgmchain) TxPool() *light.TxPool              { return s.txPool }
func (s *LightBgmchain) Engine() consensus.Engine           { return s.engine }
func (s *LightBgmchain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightBgmchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightBgmchain) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implement node.Service, returning all the currently configured
// network protocols to start.
func (s *LightBgmchain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implement node.Service, starting all internal goroutines needed by the
// Bgmchain protocol implementation.
func (s *LightBgmchain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	bgmlogs.Warn("Light client mode is an experimental feature")
	s.netRPCService = bgmapi.NewPublicNetAPI(srvr, s.networkId)
	// search the topic belonging to the oldest supported protocol because
	// servers always advertise all supported protocols
	protocolVersion := ClientProtocolVersions[len(ClientProtocolVersions)-1]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start()
	return nil
}

// Stop implement node.Service, terminating all internal goroutines used by the
// Bgmchain protocol.
func (s *LightBgmchain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDbPtr.Close()
	close(s.shutdownChan)

	return nil
}
