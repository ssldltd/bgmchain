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


// Package bgm implements the Bgmchain protocol.
package bgm

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/dpos"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/filters"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/internal/bgmapi"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/miner"
	"github.com/ssldltd/bgmchain/node"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
)


// set in js bgmconsole via admin interface or wrapper from cli flags
func (self *Bgmchain) SetCoinbase(coinbase bgmcommon.Address) {
	self.lock.Lock()
	self.coinbase = coinbase
	self.lock.Unlock()

	self.miner.SetCoinbase(coinbase)
}

// Bgmchain implements the Bgmchain full node service.
type Bgmchain struct {
	config      *Config
	chainConfig *bgmparam.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the bgmchain
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *bgmCore.TxPool
	blockchain      *bgmCore.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb bgmdbPtr.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *bgmCore.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *BgmApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	validator bgmcommon.Address
	coinbase  bgmcommon.Address

	networkId     Uint64
	netRPCService *bgmapi.PublicNetAPI

	lock syncPtr.RWMutex // Protects the variadic fields (e.g. gas price and coinbase)
}

func (s *Bgmchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Bgmchain object (including the
// initialisation of the bgmcommon Bgmchain object)
func New(CTX *node.ServiceContext, config *Config) (*Bgmchain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run bgmPtr.Bgmchain in light sync mode, use les.LightBgmchain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %-d", config.SyncMode)
	}
	chainDb, err := CreateDB(CTX, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := bgmCore.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*bgmparam.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	bgmlogs.Info("Initialised chain configuration", "config", chainConfig)

	bgm := &Bgmchain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       CTX.EventMux,
		accountManager: CTX.AccountManager,
		engine:         dpos.New(chainConfig.Dpos, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		validator:      config.Validator,
		coinbase:       config.Coinbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, bgmparam.BloomBitsBlocks),
	}

	bgmlogs.Info("Initialising Bgmchain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := bgmCore.GetBlockChainVersion(chainDb)
		if bcVersion != bgmCore.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%-d / %-d). Run gbgm upgradedbPtr.\n", bcVersion, bgmCore.BlockChainVersion)
		}
		bgmCore.WriteBlockChainVersion(chainDb, bgmCore.BlockChainVersion)
	}
	vmConfig := vmPtr.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	bgmPtr.blockchain, err = bgmCore.NewBlockChain(chainDb, bgmPtr.chainConfig, bgmPtr.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*bgmparam.ConfigCompatError); ok {
		bgmlogs.Warn("Rewinding chain to upgrade configuration", "err", compat)
		bgmPtr.blockchain.SetHead(compat.RewindTo)
		bgmCore.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	bgmPtr.bloomIndexer.Start(bgmPtr.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = CTX.ResolvePath(config.TxPool.Journal)
	}
	bgmPtr.txPool = bgmCore.NewTxPool(config.TxPool, bgmPtr.chainConfig, bgmPtr.blockchain)

	if bgmPtr.protocolManager, err = NewProtocolManager(bgmPtr.chainConfig, config.SyncMode, config.NetworkId, bgmPtr.eventMux, bgmPtr.txPool, bgmPtr.engine, bgmPtr.blockchain, chainDb); err != nil {
		return nil, err
	}
	bgmPtr.miner = miner.New(bgm, bgmPtr.chainConfig, bgmPtr.EventMux(), bgmPtr.engine)
	bgmPtr.miner.SetExtra(makeExtraData(config.ExtraData))

	bgmPtr.ApiBackend = &BgmApiBackend{bgm, nil}
	gpobgmparam := config.GPO
	if gpobgmparam.Default == nil {
		gpobgmparam.Default = config.GasPrice
	}
	bgmPtr.ApiBackend.gpo = gasprice.NewOracle(bgmPtr.ApiBackend, gpobgmparam)

	return bgm, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(bgmparam.VersionMajor<<16 | bgmparam.VersionMinor<<8 | bgmparam.VersionPatch),
			"gbgm",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if Uint64(len(extra)) > bgmparam.MaximumExtraDataSize {
		bgmlogs.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", bgmparam.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(CTX *node.ServiceContext, config *Config, name string) (bgmdbPtr.Database, error) {
	db, err := CTX.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := dbPtr.(*bgmdbPtr.LDBDatabase); ok {
		dbPtr.Meter("bgm/db/chaindata/")
	}
	return db, nil
}

// APIs returns the collection of RPC services the bgmchain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Bgmchain) APIs() []rpcPtr.API {
	bgmlogs.Info("BGM get Apis ***********************************************")
	apis := bgmapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpcPtr.API{
		{
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicBgmchainAPI(s),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Bgmchain) ResetWithGenesisBlock(gbPtr *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Bgmchain) Validator() (validator bgmcommon.Address, err error) {
	s.lock.RLock()
	validator = s.validator
	s.lock.RUnlock()

	if validator != (bgmcommon.Address{}) {
		return validator, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return bgmcommon.Address{}, fmt.Errorf("validator address must be explicitly specified")
}

// set in js bgmconsole via admin interface or wrapper from cli flags
func (self *Bgmchain) SetValidator(validator bgmcommon.Address) {
	self.lock.Lock()
	self.validator = validator
	self.lock.Unlock()
}

func (s *Bgmchain) Coinbase() (eb bgmcommon.Address, err error) {
	s.lock.RLock()
	coinbase := s.coinbase
	s.lock.RUnlock()

	if coinbase != (bgmcommon.Address{}) {
		return coinbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return bgmcommon.Address{}, fmt.Errorf("coinbase address must be explicitly specified")
}

func (s *Bgmchain) StartMining(local bool) error {
	validator, err := s.Validator()
	if err != nil {
		bgmlogs.Error("Cannot start mining without validator", "err", err)
		return fmt.Errorf("validator missing: %v", err)
	}
	cb, err := s.Coinbase()
	if err != nil {
		bgmlogs.Error("Cannot start mining without coinbase", "err", err)
		return fmt.Errorf("coinbase missing: %v", err)
	}

	if dpos, ok := s.engine.(*dpos.Dpos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: validator})
		if wallet == nil || err != nil {
			bgmlogs.Error("Coinbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		dpos.Authorize(validator, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomicPtr.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(cb)
	return nil
}

func (s *Bgmchain) StopMining()         { s.miner.Stop() }
func (s *Bgmchain) IsMining() bool      { return s.miner.Mining() }
func (s *Bgmchain) Miner() *miner.Miner { return s.miner }

func (s *Bgmchain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Bgmchain) BlockChain() *bgmCore.BlockChain       { return s.blockchain }
func (s *Bgmchain) TxPool() *bgmCore.TxPool               { return s.txPool }
func (s *Bgmchain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Bgmchain) Engine() consensus.Engine           { return s.engine }
func (s *Bgmchain) ChainDb() bgmdbPtr.Database            { return s.chainDb }
func (s *Bgmchain) IsListening() bool                  { return true } // Always listening
func (s *Bgmchain) BgmVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Bgmchain) NetVersion() Uint64                 { return s.networkId }
func (s *Bgmchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Bgmchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Bgmchain protocol implementation.
func (s *Bgmchain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = bgmapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Bgmchain protocol.
func (s *Bgmchain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDbPtr.Close()
	close(s.shutdownChan)

	return nil
}

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *bgmCore.ChainIndexer)
}