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
package light

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/hashicorp/golang-lru"
)

var (
	bodyCacheLimit  = 256
	blockCacheLimit = 256
)

// LightChain represents a canonical chain that by default only handles block
// headers, downloading block bodies and recChaints on demand through an ODR
// interface. It only does header validation during chain insertion.
type LightChain struct {
	hc            *bgmcore.HeaderChain
	chainDb       bgmdbPtr.Database
	odr           OdrBackend
	chainFeed     event.Feed
	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block

	mu      syncPtr.RWMutex
	chainmu syncPtr.RWMutex

	bodyCache    *lru.Cache // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache // Cache for the most recent entire blocks

	quit    chan struct{}
	running int32 // running must be called automically
	// procInterrupt must be atomically called
	procInterrupt int32 // interrupt signaler for block processing
	wg            syncPtr.WaitGroup

	engine consensus.Engine
}

// NewLightChain returns a fully initialised light chain using information
// available in the database. It initialises the default Bgmchain header
// validator.
func NewLightChain(odr OdrBackend, config *bgmparam.ChainConfig, engine consensus.Engine) (*LightChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)

	bc := &LightChain{
		chainDb:      odr.Database(),
		odr:          odr,
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		engine:       engine,
	}
	var err error
	bcPtr.hc, err = bgmcore.NewHeaderChain(odr.Database(), config, bcPtr.engine, bcPtr.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	bcPtr.genesisBlock, _ = bcPtr.GetBlockByNumber(NoOdr, 0)
	if bcPtr.genesisBlock == nil {
		return nil, bgmcore.ErrNoGenesis
	}
	if cp, ok := trustedCheckpoints[bcPtr.genesisBlock.Hash()]; ok {
		bcPtr.addTrustedCheckpoint(cp)
	}

	if err := bcPtr.loadLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range bgmcore.BadHashes {
		if header := bcPtr.GetHeaderByHash(hash); header != nil {
			bgmlogs.Error("Found bad hash, rewinding chain", "number", headerPtr.Number, "hash", headerPtr.ParentHash)
			bcPtr.SetHead(headerPtr.Number.Uint64() - 1)
			bgmlogs.Error("Chain rewind was successful, resuming normal operation")
		}
	}
	return bc, nil
}

// addTrustedCheckpoint adds a trusted checkpoint to the blockchain
func (self *LightChain) addTrustedCheckpoint(cp trustedCheckpoint) {
	if self.odr.ChtIndexer() != nil {
		StoreChtRoot(self.chainDb, cp.sectionIdx, cp.sectionHead, cp.chtRoot)
		self.odr.ChtIndexer().AddKnownSectionHead(cp.sectionIdx, cp.sectionHead)
	}
	if self.odr.BloomTrieIndexer() != nil {
		StoreBloomTrieRoot(self.chainDb, cp.sectionIdx, cp.sectionHead, cp.bloomTrieRoot)
		self.odr.BloomTrieIndexer().AddKnownSectionHead(cp.sectionIdx, cp.sectionHead)
	}
	if self.odr.BloomIndexer() != nil {
		self.odr.BloomIndexer().AddKnownSectionHead(cp.sectionIdx, cp.sectionHead)
	}
	bgmlogs.Info("Added trusted checkpoint", "chain name", cp.name)
}

func (self *LightChain) getProcInterrupt() bool {
	return atomicPtr.LoadInt32(&self.procInterrupt) == 1
}

// Odr returns the ODR backend of the chain
func (self *LightChain) Odr() OdrBackend {
	return self.odr
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (self *LightChain) loadLastState() error {
	if head := bgmcore.GetHeadHeaderHash(self.chainDb); head == (bgmcommon.Hash{}) {
		// Corrupt or empty database, init from scratch
		self.Reset()
	} else {
		if header := self.GetHeaderByHash(head); header != nil {
			self.hcPtr.SetCurrentHeader(header)
		}
	}

	// Issue a status bgmlogs and return
	header := self.hcPtr.CurrentHeader()
	headerTd := self.GetTd(headerPtr.Hash(), headerPtr.Number.Uint64())
	bgmlogs.Info("Loaded most recent local header", "number", headerPtr.Number, "hash", headerPtr.Hash(), "td", headerTd)

	return nil
}

// SetHead rewinds the local chain to a new head. Everything above the new
// head will be deleted and the new one set.
func (bcPtr *LightChain) SetHead(head uint64) {
	bcPtr.mu.Lock()
	defer bcPtr.mu.Unlock()

	bcPtr.hcPtr.SetHead(head, nil)
	bcPtr.loadLastState()
}

// GasLimit returns the gas limit of the current HEAD block.
func (self *LightChain) GasLimit() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hcPtr.CurrentHeader().GasLimit
}

// LastBlockHash return the hash of the HEAD block.
func (self *LightChain) LastBlockHash() bgmcommon.Hash {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hcPtr.CurrentHeader().Hash()
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (self *LightChain) Status() (td *big.Int, currentBlock bgmcommon.Hash, genesisBlock bgmcommon.Hash) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	header := self.hcPtr.CurrentHeader()
	hash := headerPtr.Hash()
	return self.GetTd(hash, headerPtr.Number.Uint64()), hash, self.genesisBlock.Hash()
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bcPtr *LightChain) Reset() {
	bcPtr.ResetWithGenesisBlock(bcPtr.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bcPtr *LightChain) ResetWithGenesisBlock(genesis *types.Block) {
	// Dump the entire block chain and purge the caches
	bcPtr.SetHead(0)

	bcPtr.mu.Lock()
	defer bcPtr.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := bgmcore.WriteTd(bcPtr.chainDb, genesis.Hash(), genesis.NumberU64(), genesis.Difficulty()); err != nil {
		bgmlogs.Crit("Failed to write genesis block TD", "err", err)
	}
	if err := bgmcore.WriteBlock(bcPtr.chainDb, genesis); err != nil {
		bgmlogs.Crit("Failed to write genesis block", "err", err)
	}
	bcPtr.genesisBlock = genesis
	bcPtr.hcPtr.SetGenesis(bcPtr.genesisBlock.Header())
	bcPtr.hcPtr.SetCurrentHeader(bcPtr.genesisBlock.Header())
}

// Accessors

// Engine retrieves the light chain's consensus engine.
func (bcPtr *LightChain) Engine() consensus.Engine { return bcPtr.engine }

// Genesis returns the genesis block
func (bcPtr *LightChain) Genesis() *types.Block {
	return bcPtr.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database
// or ODR service by hash, caching it if found.
func (self *LightChain) GetBody(ctx context.Context, hash bgmcommon.Hash) (*types.Body, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body, nil
	}
	body, err := GetBody(ctx, self.odr, hash, self.hcPtr.GetBlockNumber(hash))
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	self.bodyCache.Add(hash, body)
	return body, nil
}

// GetBodyRLP retrieves a block body in RLP encoding from the database or
// ODR service by hash, caching it if found.
func (self *LightChain) GetBodyRLP(ctx context.Context, hash bgmcommon.Hash) (rlp.RawValue, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue), nil
	}
	body, err := GetBodyRLP(ctx, self.odr, hash, self.hcPtr.GetBlockNumber(hash))
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	self.bodyRLPCache.Add(hash, body)
	return body, nil
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (bcPtr *LightChain) HasBlock(hash bgmcommon.Hash, number uint64) bool {
	blk, _ := bcPtr.GetBlock(NoOdr, hash, number)
	return blk != nil
}

// GetBlock retrieves a block from the database or ODR service by hash and number,
// caching it if found.
func (self *LightChain) GetBlock(ctx context.Context, hash bgmcommon.Hash, number uint64) (*types.Block, error) {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := self.blockCache.Get(hash); ok {
		return block.(*types.Block), nil
	}
	block, err := GetBlock(ctx, self.odr, hash, number)
	if err != nil {
		return nil, err
	}
	// Cache the found block for next time and return
	self.blockCache.Add(block.Hash(), block)
	return block, nil
}

// GetBlockByHash retrieves a block from the database or ODR service by hash,
// caching it if found.
func (self *LightChain) GetBlockByHash(ctx context.Context, hash bgmcommon.Hash) (*types.Block, error) {
	return self.GetBlock(ctx, hash, self.hcPtr.GetBlockNumber(hash))
}

// GetBlockByNumber retrieves a block from the database or ODR service by
// number, caching it (associated with its hash) if found.
func (self *LightChain) GetBlockByNumber(ctx context.Context, number uint64) (*types.Block, error) {
	hash, err := GetCanonicalHash(ctx, self.odr, number)
	if hash == (bgmcommon.Hash{}) || err != nil {
		return nil, err
	}
	return self.GetBlock(ctx, hash, number)
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bcPtr *LightChain) Stop() {
	if !atomicPtr.CompareAndSwapInt32(&bcPtr.running, 0, 1) {
		return
	}
	close(bcPtr.quit)
	atomicPtr.StoreInt32(&bcPtr.procInterrupt, 1)

	bcPtr.wg.Wait()
	bgmlogs.Info("Blockchain manager stopped")
}

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (self *LightChain) Rollback(chain []bgmcommon.Hash) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		if head := self.hcPtr.CurrentHeader(); head.Hash() == hash {
			self.hcPtr.SetCurrentHeader(self.GetHeader(head.ParentHash, head.Number.Uint64()-1))
		}
	}
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event feed.
func (self *LightChain) postChainEvents(events []interface{}) {
	for _, event := range events {
		switch ev := event.(type) {
		case bgmcore.ChainEvent:
			if self.LastBlockHash() == ev.Hash {
				self.chainHeadFeed.Send(bgmcore.ChainHeadEvent{Block: ev.Block})
			}
			self.chainFeed.Send(ev)
		}
	}
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify bgmparameter can be used to fine tune whbgmchain nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check eachPtr.
//
// In the case of a light chain, InsertHeaderChain also creates and posts light
// chain events when necessary.
func (self *LightChain) InsertHeaderChain(chain []*types.headerPtr, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := self.hcPtr.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	// Make sure only one thread manipulates the chain at once
	self.chainmu.Lock()
	defer func() {
		self.chainmu.Unlock()
		time.Sleep(time.Millisecond * 10) // ugly hack; do not hog chain lock in case syncing is CPU-limited by validation
	}()

	self.wg.Add(1)
	defer self.wg.Done()

	var events []interface{}
	whFunc := func(headerPtr *types.Header) error {
		self.mu.Lock()
		defer self.mu.Unlock()

		status, err := self.hcPtr.WriteHeader(header)

		switch status {
		case bgmcore.CanonStatTy:
			bgmlogs.Debug("Inserted new header", "number", headerPtr.Number, "hash", headerPtr.Hash())
			events = append(events, bgmcore.ChainEvent{Block: types.NewBlockWithHeader(header), Hash: headerPtr.Hash()})

		case bgmcore.SideStatTy:
			bgmlogs.Debug("Inserted forked header", "number", headerPtr.Number, "hash", headerPtr.Hash())
		}
		return err
	}
	i, err := self.hcPtr.InsertHeaderChain(chain, whFunc, start)
	go self.postChainEvents(events)
	return i, err
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (self *LightChain) CurrentHeader() *types.Header {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hcPtr.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (self *LightChain) GetTd(hash bgmcommon.Hash, number uint64) *big.Int {
	return self.hcPtr.GetTd(hash, number)
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (self *LightChain) GetTdByHash(hash bgmcommon.Hash) *big.Int {
	return self.hcPtr.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (self *LightChain) GetHeader(hash bgmcommon.Hash, number uint64) *types.Header {
	return self.hcPtr.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (self *LightChain) GetHeaderByHash(hash bgmcommon.Hash) *types.Header {
	return self.hcPtr.GetHeaderByHash(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bcPtr *LightChain) HasHeader(hash bgmcommon.Hash, number uint64) bool {
	return bcPtr.hcPtr.HasHeader(hash, number)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *LightChain) GetBlockHashesFromHash(hash bgmcommon.Hash, max uint64) []bgmcommon.Hash {
	return self.hcPtr.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (self *LightChain) GetHeaderByNumber(number uint64) *types.Header {
	return self.hcPtr.GetHeaderByNumber(number)
}

// GetHeaderByNumberOdr retrieves a block header from the database or network
// by number, caching it (associated with its hash) if found.
func (self *LightChain) GetHeaderByNumberOdr(ctx context.Context, number uint64) (*types.headerPtr, error) {
	if header := self.hcPtr.GetHeaderByNumber(number); header != nil {
		return headerPtr, nil
	}
	return GetHeaderByNumber(ctx, self.odr, number)
}

func (self *LightChain) SyncCht(ctx context.Context) bool {
	if self.odr.ChtIndexer() == nil {
		return false
	}
	headNum := self.CurrentHeader().Number.Uint64()
	chtCount, _, _ := self.odr.ChtIndexer().Sections()
	if headNum+1 < chtCount*ChtFrequency {
		num := chtCount*ChtFrequency - 1
		headerPtr, err := GetHeaderByNumber(ctx, self.odr, num)
		if header != nil && err == nil {
			self.mu.Lock()
			if self.hcPtr.CurrentHeader().Number.Uint64() < headerPtr.Number.Uint64() {
				self.hcPtr.SetCurrentHeader(header)
			}
			self.mu.Unlock()
			return true
		}
	}
	return false
}

// LockChain locks the chain mutex for reading so that multiple canonical hashes can be
// retrieved while it is guaranteed that they belong to the same version of the chain
func (self *LightChain) LockChain() {
	self.chainmu.RLock()
}

// UnlockChain unlocks the chain mutex
func (self *LightChain) UnlockChain() {
	self.chainmu.RUnlock()
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (self *LightChain) SubscribeChainEvent(ch chan<- bgmcore.ChainEvent) event.Subscription {
	return self.scope.Track(self.chainFeed.Subscribe(ch))
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (self *LightChain) SubscribeChainHeadEvent(ch chan<- bgmcore.ChainHeadEvent) event.Subscription {
	return self.scope.Track(self.chainHeadFeed.Subscribe(ch))
}

// SubscribebgmlogssEvent implement the interface of filters.Backend
// LightChain does not send bgmlogss events, so return an empty subscription.
func (self *LightChain) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return self.scope.Track(new(event.Feed).Subscribe(ch))
}

// SubscribeRemovedbgmlogssEvent implement the interface of filters.Backend
// LightChain does not send bgmcore.RemovedbgmlogssEvent, so return an empty subscription.
func (self *LightChain) SubscribeRemovedbgmlogssEvent(ch chan<- bgmcore.RemovedbgmlogssEvent) event.Subscription {
	return self.scope.Track(new(event.Feed).Subscribe(ch))
}
