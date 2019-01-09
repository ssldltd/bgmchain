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

// Package bgmcore implements the Bgmchain consensus protocol.
package bgmcore

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/dpos"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/metrics"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
	"github.com/hashicorp/golang-lru"
)

var (
	blockInsertTimer = metrics.NewTimer("chain/inserts")

	ErrNoGenesis = errors.New("Genesis not found in chain")
)

const (
	bodyCacheLimit      = 256
	blockCacheLimit     = 256
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	badBlockLimit       = 10

	// BlockChainVersion ensures that an incompatible database forces a resync from scratchPtr.
	BlockChainVersion = 3
)
// The BlockChain also helps in returning blocks fromPtr **any** chain included
// in the database as well as blocks that represents the canonical chain. It's
// important to note that GetBlock can return any block and does not need to be
// included in the canonical one where as GetBlockByNumber always represents the
// canonical chain.
type BlockChain struct {
	config *bgmparam.ChainConfig // chain & network configuration

	hc            *HeaderChain
	chainDb       bgmdbPtr.Database
	rmbgmlogssFeed    event.Feed
	chainFeed     event.Feed
	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	bgmlogssFeed      event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block

	mu      syncPtr.RWMutex // global mutex for locking chain operations
	chainmu syncPtr.RWMutex // blockchain insertion lock
	procmu  syncPtr.RWMutex // block processor lock

	checkpoint       int          // checkpoint counts towards the new checkpoint
	currentBlock     *types.Block // Current head of the block chain
	currentFastBlock *types.Block // Current head of the fast-sync chain (may be above the block chain!)

	stateCache   state.Database // State database to reuse between imports (contains state cache)
	bodyCache    *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache     // Cache for the most recent entire blocks
	futureBlocks *lru.Cache     // future blocks are blocks added for later processing

	quit    chan struct{} // blockchain quit channel
	running int32         // running must be called atomically
	// procInterrupt must be atomically called
	procInterrupt int32          // interrupt signaler for block processing
	wg            syncPtr.WaitGroup // chain processing wait group for shutting down

	engine    consensus.Engine
	processor Processor // block processor interface
	validator Validator // block and state validator interface
	vmConfig  vmPtr.Config

	badBlocks *lru.Cache // Bad block cache
}


func (batcr *BlockChain) getProcInterrupt() bool {
	return atomicPtr.LoadInt32(&batcr.procInterrupt) == 1
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (batcr *BlockChain) loadLastState() error {
	// Restore the last known head block
	head := GetHeadBlockHash(batcr.chainDb)
	if head == (bgmcommon.Hash{}) {
		// Corrupt or empty database, init from scratch
		bgmlogs.Warn("Empty database, resetting chain")
		return batcr.Reset()
	}
	// Make sure the entire head block is available
	currentBlock := batcr.GetBlockByHash(head)
	if currentBlock == nil {
		// Corrupt or empty database, init from scratch
		bgmlogs.Warn("Head block missing, resetting chain", "hash", head)
		return batcr.Reset()
	}
	// Make sure the state associated with the block is available
	if _, err := state.New(currentBlock.Root(), batcr.stateCache); err != nil {
		// Dangling block without a state associated, init from scratch
		bgmlogs.Warn("Head state missing, resetting chain", "number", currentBlock.Number(), "hash", currentBlock.Hash())
		return batcr.Reset()
	}
	// Everything seems to be fine, set as the head block
	batcr.currentBlock = currentBlock

	// Restore the last known head header
	currentHeader := batcr.currentBlock.Header()
	if head := GetHeadHeaderHash(batcr.chainDb); head != (bgmcommon.Hash{}) {
		if header := batcr.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	batcr.hcPtr.SetCurrentHeader(currentHeader)

	// Restore the last known head fast block
	batcr.currentFastBlock = batcr.currentBlock
	if head := GetHeadFastBlockHash(batcr.chainDb); head != (bgmcommon.Hash{}) {
		if block := batcr.GetBlockByHash(head); block != nil {
			batcr.currentFastBlock = block
		}
	}

	// Issue a status bgmlogs for the user
	headerTd := batcr.GetTd(currentheaderPtr.Hash(), currentheaderPtr.Number.Uint64())
	blockTd := batcr.GetTd(batcr.currentBlock.Hash(), batcr.currentBlock.NumberU64())
	fastTd := batcr.GetTd(batcr.currentFastBlock.Hash(), batcr.currentFastBlock.NumberU64())

	bgmlogs.Info("Loaded most recent local header", "number", currentheaderPtr.Number, "hash", currentheaderPtr.Hash(), "td", headerTd)
	bgmlogs.Info("Loaded most recent local full block", "number", batcr.currentBlock.Number(), "hash", batcr.currentBlock.Hash(), "td", blockTd)
	bgmlogs.Info("Loaded most recent local fast block", "number", batcr.currentFastBlock.Number(), "hash", batcr.currentFastBlock.Hash(), "td", fastTd)

	return nil
}

// SetHead rewinds the local chain to a new head. In the case of headers, everything
// above the new head will be deleted and the new one set. In the case of blocks
// though, the head may be further rewound if block bodies are missing (non-archive
// nodes after a fast sync).
func (batcr *BlockChain) SetHead(head uint64) error {
	bgmlogs.Warn("Rewinding blockchain", "target", head)

	batcr.mu.Lock()
	defer batcr.mu.Unlock()

	// Rewind the header chain, deleting all block bodies until then
	delFn := func(hash bgmcommon.Hash, num uint64) {
		DeleteBody(batcr.chainDb, hash, num)
	}
	batcr.hcPtr.SetHead(head, delFn)
	currentHeader := batcr.hcPtr.CurrentHeader()

	// Clear out any stale content from the caches
	batcr.bodyCache.Purge()
	batcr.bodyRLPCache.Purge()
	batcr.blockCache.Purge()
	batcr.futureBlocks.Purge()

	// Rewind the block chain, ensuring we don't end up with a stateless head block
	if batcr.currentBlock != nil && currentheaderPtr.Number.Uint64() < batcr.currentBlock.NumberU64() {
		batcr.currentBlock = batcr.GetBlock(currentheaderPtr.Hash(), currentheaderPtr.Number.Uint64())
	}
	if batcr.currentBlock != nil {
		if _, err := state.New(batcr.currentBlock.Root(), batcr.stateCache); err != nil {
			// Rewound state missing, rolled back to before pivot, reset to genesis
			batcr.currentBlock = nil
		}
	}
	// Rewind the fast block in a simpleton way to the target head
	if batcr.currentFastBlock != nil && currentheaderPtr.Number.Uint64() < batcr.currentFastBlock.NumberU64() {
		batcr.currentFastBlock = batcr.GetBlock(currentheaderPtr.Hash(), currentheaderPtr.Number.Uint64())
	}
	// If either blocks reached nil, reset to the genesis state
	if batcr.currentBlock == nil {
		batcr.currentBlock = batcr.genesisBlock
	}
	if batcr.currentFastBlock == nil {
		batcr.currentFastBlock = batcr.genesisBlock
	}
	if err := WriteHeadBlockHash(batcr.chainDb, batcr.currentBlock.Hash()); err != nil {
		bgmlogs.Crit("Failed to reset head full block", "err", err)
	}
	if err := WriteHeadFastBlockHash(batcr.chainDb, batcr.currentFastBlock.Hash()); err != nil {
		bgmlogs.Crit("Failed to reset head fast block", "err", err)
	}
	return batcr.loadLastState()
}

// NewBlockChain returns a fully initialised block chain using information
// Processor.
func NewBlockChain(chainDb bgmdbPtr.Database, config *bgmparam.ChainConfig, engine consensus.Engine, vmConfig vmPtr.Config) (*BlockChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)
	badBlocks, _ := lru.New(badBlockLimit)

	bc := &BlockChain{
		config:       config,
		chainDb:      chainDb,
		stateCache:   state.NewDatabase(chainDb),
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		engine:       engine,
		vmConfig:     vmConfig,
		badBlocks:    badBlocks,
	}
	batcr.SetValidator(NewBlockValidator(config, bc, engine))
	batcr.SetProcessor(NewStateProcessor(config, bc, engine))

	var err error
	batcr.hc, err = NewHeaderChain(chainDb, config, engine, batcr.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	batcr.genesisBlock = batcr.GetBlockByNumber(0)
	if batcr.genesisBlock == nil {
		return nil, ErrNoGenesis
	}
	if err := batcr.loadLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range BadHashes {
		if header := batcr.GetHeaderByHash(hash); header != nil {
			// get the canonical block corresponding to the offending header's number
			headerByNumber := batcr.GetHeaderByNumber(headerPtr.Number.Uint64())
			// make sure the headerByNumber (if present) is in our current canonical chain
			if headerByNumber != nil && headerByNumber.Hash() == headerPtr.Hash() {
				bgmlogs.Error("Found bad hash, rewinding chain", "number", headerPtr.Number, "hash", headerPtr.ParentHash)
				batcr.SetHead(headerPtr.Number.Uint64() - 1)
				bgmlogs.Error("Chain rewind was successful, resuming normal operation")
			}
		}
	}
	// Take ownership of this particular state
	go batcr.update()
	return bc, nil
}


// FastSyncCommitHead sets the current head block to the one defined by the hash
func (batcr *BlockChain) FastSyncCommitHead(hash bgmcommon.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := batcr.GetBlockByHash(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x…]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), batcr.chainDb, 0); err != nil {
		return err
	}
	// If all checks out, manually set the head block
	batcr.mu.Lock()
	batcr.currentBlock = block
	batcr.mu.Unlock()

	bgmlogs.Info("Committed new head block", "number", block.Number(), "hash", hash)
	return nil
}

// GasLimit returns the gas limit of the current HEAD block.
func (batcr *BlockChain) GasLimit() *big.Int {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.currentBlock.GasLimit()
}

// LastBlockHash return the hash of the HEAD block.
func (batcr *BlockChain) LastBlockHash() bgmcommon.Hash {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.currentBlock.Hash()
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (batcr *BlockChain) CurrentBlock() *types.Block {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.currentBlock
}

// CurrentFastBlock retrieves the current fast-sync head block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (batcr *BlockChain) CurrentFastBlock() *types.Block {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.currentFastBlock
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (batcr *BlockChain) Status() (td *big.Int, currentBlock bgmcommon.Hash, genesisBlock bgmcommon.Hash) {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.GetTd(batcr.currentBlock.Hash(), batcr.currentBlock.NumberU64()), batcr.currentBlock.Hash(), batcr.genesisBlock.Hash()
}

// SetProcessor sets the processor required for making state modifications.
func (batcr *BlockChain) SetProcessor(processor Processor) {
	batcr.procmu.Lock()
	defer batcr.procmu.Unlock()
	batcr.processor = processor
}

// SetValidator sets the validator which is used to validate incoming blocks.
func (batcr *BlockChain) SetValidator(validator Validator) {
	batcr.procmu.Lock()
	defer batcr.procmu.Unlock()
	batcr.validator = validator
}

// Validator returns the current validator.
func (batcr *BlockChain) Validator() Validator {
	batcr.procmu.RLock()
	defer batcr.procmu.RUnlock()
	return batcr.validator
}

// Processor returns the current processor.
func (batcr *BlockChain) Processor() Processor {
	batcr.procmu.RLock()
	defer batcr.procmu.RUnlock()
	return batcr.processor
}

// State returns a new mutable state based on the current HEAD block.
func (batcr *BlockChain) State() (*state.StateDB, error) {
	return batcr.StateAt(batcr.CurrentBlock().Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (batcr *BlockChain) StateAt(root bgmcommon.Hash) (*state.StateDB, error) {
	return state.New(root, batcr.stateCache)
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (batcr *BlockChain) Reset() error {
	return batcr.ResetWithGenesisBlock(batcr.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (batcr *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// Dump the entire block chain and purge the caches
	if err := batcr.SetHead(0); err != nil {
		return err
	}
	batcr.mu.Lock()
	defer batcr.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := batcr.hcPtr.WriteTd(genesis.Hash(), genesis.NumberU64(), genesis.Difficulty()); err != nil {
		bgmlogs.Crit("Failed to write genesis block TD", "err", err)
	}
	if err := WriteBlock(batcr.chainDb, genesis); err != nil {
		bgmlogs.Crit("Failed to write genesis block", "err", err)
	}
	batcr.genesisBlock = genesis
	batcr.insert(batcr.genesisBlock)
	batcr.currentBlock = batcr.genesisBlock
	batcr.hcPtr.SetGenesis(batcr.genesisBlock.Header())
	batcr.hcPtr.SetCurrentHeader(batcr.genesisBlock.Header())
	batcr.currentFastBlock = batcr.genesisBlock

	return nil
}

// Export writes the active chain to the given writer.
func (batcr *BlockChain) Export(w io.Writer) error {
	return batcr.ExportN(w, uint64(0), batcr.currentBlock.NumberU64())
}

// ExportN writes a subset of the active chain to the given writer.
func (batcr *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	if first > last {
		return fmt.Errorf("export failed: first (%-d) is greater than last (%-d)", first, last)
	}
	bgmlogs.Info("Exporting batch of blocks", "count", last-first+1)

	for nr := first; nr <= last; nr++ {
		block := batcr.GetBlockByNumber(nr)
		if block == nil {
			return fmt.Errorf("export failed on #%-d: not found", nr)
		}

		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}

	return nil
}

// insert injects a new head block into the current block chain. This method
// assumes that the block is indeed a true head. It will also reset the head
// header and the head fast sync block to this very same block if they are older
// or if they are on a different side chain.
//
// Note, this function assumes that the `mu` mutex is held!
func (batcr *BlockChain) insert(block *types.Block) {
	// If the block is on a side chain or an unknown one, force other heads onto it too
	updateHeads := GetCanonicalHash(batcr.chainDb, block.NumberU64()) != block.Hash()

	// Add the block to the canonical chain number scheme and mark as the head
	if err := WriteCanonicalHash(batcr.chainDb, block.Hash(), block.NumberU64()); err != nil {
		bgmlogs.Crit("Failed to insert block number", "err", err)
	}
	if err := WriteHeadBlockHash(batcr.chainDb, block.Hash()); err != nil {
		bgmlogs.Crit("Failed to insert head block hash", "err", err)
	}
	batcr.currentBlock = block

	// If the block is better than out head or is on a different chain, force update heads
	if updateHeads {
		batcr.hcPtr.SetCurrentHeader(block.Header())

		if err := WriteHeadFastBlockHash(batcr.chainDb, block.Hash()); err != nil {
			bgmlogs.Crit("Failed to insert head fast block hash", "err", err)
		}
		batcr.currentFastBlock = block
	}
}

// Genesis retrieves the chain's genesis block.
func (batcr *BlockChain) Genesis() *types.Block {
	return batcr.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (batcr *BlockChain) GetBody(hash bgmcommon.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := batcr.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	body := GetBody(batcr.chainDb, hash, batcr.hcPtr.GetBlockNumber(hash))
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	batcr.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (batcr *BlockChain) GetBodyRLP(hash bgmcommon.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := batcr.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue)
	}
	body := GetBodyRLP(batcr.chainDb, hash, batcr.hcPtr.GetBlockNumber(hash))
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	batcr.bodyRLPCache.Add(hash, body)
	return body
}

// HasBlock checks if a block is fully present in the database or not.
func (batcr *BlockChain) HasBlock(hash bgmcommon.Hash, number uint64) bool {
	if batcr.blockCache.Contains(hash) {
		return true
	}
	ok, _ := batcr.chainDbPtr.Has(blockBodyKey(hash, number))
	return ok
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (batcr *BlockChain) HasBlockAndState(hash bgmcommon.Hash) bool {
	// Check first that the block itself is known
	block := batcr.GetBlockByHash(hash)
	if block == nil {
		return false
	}
	// Ensure the associated state is also present
	_, err := batcr.stateCache.OpenTrie(block.Root())
	return err == nil
}

// GetBlock retrieves a block from the database by hash and number,
// caching it if found.
func (batcr *BlockChain) GetBlock(hash bgmcommon.Hash, number uint64) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := batcr.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	block := GetBlock(batcr.chainDb, hash, number)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	batcr.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (batcr *BlockChain) GetBlockByHash(hash bgmcommon.Hash) *types.Block {
	return batcr.GetBlock(hash, batcr.hcPtr.GetBlockNumber(hash))
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (batcr *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(batcr.chainDb, number)
	if hash == (bgmcommon.Hash{}) {
		return nil
	}
	return batcr.GetBlock(hash, number)
}

// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
// [deprecated by bgm/62]
func (batcr *BlockChain) GetBlocksFromHash(hash bgmcommon.Hash, n int) (blocks []*types.Block) {
	number := batcr.hcPtr.GetBlockNumber(hash)
	for i := 0; i < n; i++ {
		block := batcr.GetBlock(hash, number)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.ParentHash()
		number--
	}
	return
}

// GetUnclesInChain retrieves all the uncles from a given block backwards until
// a specific distance is reached.
func (batcr *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = batcr.GetBlock(block.ParentHash(), block.NumberU64()-1)
	}
	return uncles
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (batcr *BlockChain) Stop() {
	if !atomicPtr.CompareAndSwapInt32(&batcr.running, 0, 1) {
		return
	}
	// Unsubscribe all subscriptions registered from blockchain
	batcr.scope.Close()
	close(batcr.quit)
	atomicPtr.StoreInt32(&batcr.procInterrupt, 1)

	batcr.wg.Wait()
	bgmlogs.Info("Blockchain manager stopped")
}

func (batcr *BlockChain) procFutureBlocks() {
	blocks := make([]*types.Block, 0, batcr.futureBlocks.Len())
	for _, hash := range batcr.futureBlocks.Keys() {
		if block, exist := batcr.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block.(*types.Block))
		}
	}
	if len(blocks) > 0 {
		types.BlockBy(types.Number).Sort(blocks)

		// Insert one by one as chain insertion needs contiguous ancestry between blocks
		for i := range blocks {
			batcr.InsertChain(blocks[i : i+1])
		}
	}
}

// WriteStatus status of write
type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (batcr *BlockChain) Rollback(chain []bgmcommon.Hash) {
	batcr.mu.Lock()
	defer batcr.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		currentHeader := batcr.hcPtr.CurrentHeader()
		if currentheaderPtr.Hash() == hash {
			batcr.hcPtr.SetCurrentHeader(batcr.GetHeader(currentheaderPtr.ParentHash, currentheaderPtr.Number.Uint64()-1))
		}
		if batcr.currentFastBlock.Hash() == hash {
			batcr.currentFastBlock = batcr.GetBlock(batcr.currentFastBlock.ParentHash(), batcr.currentFastBlock.NumberU64()-1)
			WriteHeadFastBlockHash(batcr.chainDb, batcr.currentFastBlock.Hash())
		}
		if batcr.currentBlock.Hash() == hash {
			batcr.currentBlock = batcr.GetBlock(batcr.currentBlock.ParentHash(), batcr.currentBlock.NumberU64()-1)
			WriteHeadBlockHash(batcr.chainDb, batcr.currentBlock.Hash())
		}
	}
}

// SetReceiptsData computes all the non-consensus fields of the receipts
func SetReceiptsData(config *bgmparam.ChainConfig, block *types.Block, receipts types.Receipts) {
	signer := types.MakeSigner(config, block.Number())

	transactions, bgmlogsIndex := block.Transactions(), uint(0)

	for j := 0; j < len(receipts); j++ {
		// The transaction hash can be retrieved from the transaction itself
		receipts[j].TxHash = transactions[j].Hash()

		// The contract address can be derived from the transaction itself
		if transactions[j].To() == nil {
			// Deriving the signer is expensive, only do if it's actually needed
			from, _ := types.Sender(signer, transactions[j])
			receipts[j].ContractAddress = bgmcrypto.CreateAddress(from, transactions[j].Nonce())
		}
		// The used gas can be calculated based on previous receipts
		if j == 0 {
			receipts[j].GasUsed = new(big.Int).Set(receipts[j].CumulativeGasUsed)
		} else {
			receipts[j].GasUsed = new(big.Int).Sub(receipts[j].CumulativeGasUsed, receipts[j-1].CumulativeGasUsed)
		}
		// The derived bgmlogs fields can simply be set from the block and transaction
		for k := 0; k < len(receipts[j].bgmlogss); k++ {
			receipts[j].bgmlogss[k].BlockNumber = block.NumberU64()
			receipts[j].bgmlogss[k].BlockHash = block.Hash()
			receipts[j].bgmlogss[k].TxHash = receipts[j].TxHash
			receipts[j].bgmlogss[k].TxIndex = uint(j)
			receipts[j].bgmlogss[k].Index = bgmlogsIndex
			bgmlogsIndex++
		}
	}
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (batcr *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
	batcr.wg.Add(1)
	defer batcr.wg.Done()

	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(blockChain); i++ {
		if blockChain[i].NumberU64() != blockChain[i-1].NumberU64()+1 || blockChain[i].ParentHash() != blockChain[i-1].Hash() {
			bgmlogs.Error("Non contiguous receipt insert", "number", blockChain[i].Number(), "hash", blockChain[i].Hash(), "parent", blockChain[i].ParentHash(),
				"prevnumber", blockChain[i-1].Number(), "prevhash", blockChain[i-1].Hash())
			return 0, fmt.Errorf("non contiguous insert: item %-d is #%-d [%x…], item %-d is #%-d [%x…] (parent [%x…])", i-1, blockChain[i-1].NumberU64(),
				blockChain[i-1].Hash().Bytes()[:4], i, blockChain[i].NumberU64(), blockChain[i].Hash().Bytes()[:4], blockChain[i].ParentHash().Bytes()[:4])
		}
	}

	var (
		stats = struct{ processed, ignored int32 }{}
		start = time.Now()
		bytes = 0
		batch = batcr.chainDbPtr.NewBatch()
	)
	for i, block := range blockChain {
		receipts := receiptChain[i]
		// Short circuit insertion if shutting down or processing failed
		if atomicPtr.LoadInt32(&batcr.procInterrupt) == 1 {
			return 0, nil
		}
		// Short circuit if the owner header is unknown
		if !batcr.HasHeader(block.Hash(), block.NumberU64()) {
			return i, fmt.Errorf("containing header #%-d [%x…] unknown", block.Number(), block.Hash().Bytes()[:4])
		}
		// Skip if the entire data is already known
		if batcr.HasBlock(block.Hash(), block.NumberU64()) {
			stats.ignored++
			continue
		}
		// Compute all the non-consensus fields of the receipts
		SetReceiptsData(batcr.config, block, receipts)
		// Write all the data out into the database
		if err := WriteBody(batch, block.Hash(), block.NumberU64(), block.Body()); err != nil {
			return i, fmt.Errorf("failed to write block body: %v", err)
		}
		if err := WriteBlockReceipts(batch, block.Hash(), block.NumberU64(), receipts); err != nil {
			return i, fmt.Errorf("failed to write block receipts: %v", err)
		}
		if err := WriteTxLookupEntries(batch, block); err != nil {
			return i, fmt.Errorf("failed to write lookup metadata: %v", err)
		}
		stats.processed++

		if batchPtr.ValueSize() >= bgmdbPtr.IdealBatchSize {
			if err := batchPtr.Write(); err != nil {
				return 0, err
			}
			bytes += batchPtr.ValueSize()
			batch = batcr.chainDbPtr.NewBatch()
		}
	}
	if batchPtr.ValueSize() > 0 {
		bytes += batchPtr.ValueSize()
		if err := batchPtr.Write(); err != nil {
			return 0, err
		}
	}

	// Update the head fast sync block if better
	batcr.mu.Lock()
	head := blockChain[len(blockChain)-1]
	if td := batcr.GetTd(head.Hash(), head.NumberU64()); td != nil { // Rewind may have occurred, skip in that case
		if batcr.GetTd(batcr.currentFastBlock.Hash(), batcr.currentFastBlock.NumberU64()).Cmp(td) < 0 {
			if err := WriteHeadFastBlockHash(batcr.chainDb, head.Hash()); err != nil {
				bgmlogs.Crit("Failed to update head fast block hash", "err", err)
			}
			batcr.currentFastBlock = head
		}
	}
	batcr.mu.Unlock()

	bgmlogs.Info("Imported new block receipts",
		"count", stats.processed,
		"elapsed", bgmcommon.PrettyDuration(time.Since(start)),
		"bytes", bytes,
		"number", head.Number(),
		"hash", head.Hash(),
		"ignored", stats.ignored)
	return 0, nil
}

// WriteBlock writes the block to the chain.
func (batcr *BlockChain) WriteBlockAndState(block *types.Block, receipts []*types.Receipt, state *state.StateDB) (status WriteStatus, err error) {
	batcr.wg.Add(1)
	defer batcr.wg.Done()

	// Calculate the total difficulty of the block
	ptd := batcr.GetTd(block.ParentHash(), block.NumberU64()-1)
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	// Make sure no inconsistent state is leaked during insertion
	batcr.mu.Lock()
	defer batcr.mu.Unlock()

	localTd := batcr.GetTd(batcr.currentBlock.Hash(), batcr.currentBlock.NumberU64())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// Irrelevant of the canonical status, write the block itself to the database
	if err := batcr.hcPtr.WriteTd(block.Hash(), block.NumberU64(), externTd); err != nil {
		return NonStatTy, err
	}
	// Write other block data using a batchPtr.
	batch := batcr.chainDbPtr.NewBatch()
	if err := WriteBlock(batch, block); err != nil {
		return NonStatTy, err
	}
	if _, err := block.DposContext.CommitTo(batch); err != nil {
		return NonStatTy, err
	}
	if _, err := state.CommitTo(batch, batcr.config.IsEIP158(block.Number())); err != nil {
		return NonStatTy, err
	}
	if err := WriteBlockReceipts(batch, block.Hash(), block.NumberU64(), receipts); err != nil {
		return NonStatTy, err
	}

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFcPtr.pdf
	reorg := externTd.Cmp(localTd) > 0
	if !reorg && externTd.Cmp(localTd) == 0 {
		// Split same-difficulty blocks by number, then at random
		reorg = block.NumberU64() < batcr.currentBlock.NumberU64() || (block.NumberU64() == batcr.currentBlock.NumberU64() && mrand.Float64() < 0.5)
	}
	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != batcr.currentBlock.Hash() {
			if err := batcr.reorg(batcr.currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		// Write the positional metadata for transaction and receipt lookups
		if err := WriteTxLookupEntries(batch, block); err != nil {
			return NonStatTy, err
		}
		// Write hash preimages
		if err := WritePreimages(batcr.chainDb, block.NumberU64(), state.Preimages()); err != nil {
			return NonStatTy, err
		}
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	if err := batchPtr.Write(); err != nil {
		return NonStatTy, err
	}

	// Set new head.
	if status == CanonStatTy {
		batcr.insert(block)
	}
	batcr.futureBlocks.Remove(block.Hash())
	return status, nil
}

// InsertChain attempts to insert the given batch of blocks in to the canonical
// chain or, otherwise, create a fork. If an error is returned it will return
// the index number of the failing block as well an error describing what went
// wrong.
//
// After insertion is done, all accumulated events will be fired.
func (batcr *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	n, events, bgmlogss, err := batcr.insertChain(chain)
	batcr.PostChainEvents(events, bgmlogss)
	return n, err
}

// insertChain will execute the actual chain insertion and event aggregation. The
// only reason this method exists as a separate one is to make locking cleaner
// with deferred statements.
func (batcr *BlockChain) insertChain(chain types.Blocks) (int, []interface{}, []*types.bgmlogs, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, bgmlogs a messge (programming error) and skip insertion
			bgmlogs.Error("Non contiguous block insert", "number", chain[i].Number(), "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash(), "prevnumber", chain[i-1].Number(), "prevhash", chain[i-1].Hash())

			return 0, nil, nil, fmt.Errorf("non contiguous insert: item %-d is #%-d [%x…], item %-d is #%-d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
		}
	}
	// Pre-checks passed, start the full block imports
	batcr.wg.Add(1)
	defer batcr.wg.Done()

	batcr.chainmu.Lock()
	defer batcr.chainmu.Unlock()

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		stats         = insertStats{startTime: mclock.Now()}
		events        = make([]interface{}, 0, len(chain))
		lastCanon     *types.Block
		coalescedbgmlogss []*types.bgmlogs
	)
	// Start the parallel header verifier
	headers := make([]*types.headerPtr, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		seals[i] = true
	}
	abort, results := batcr.engine.VerifyHeaders(bc, headers, seals)
	defer close(abort)

	// Iterate over the blocks and insert when the verifier permits
	for i, block := range chain {
		// If the chain is terminating, stop processing blocks
		if atomicPtr.LoadInt32(&batcr.procInterrupt) == 1 {
			bgmlogs.Debug("Premature abort during blocks processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			batcr.reportBlock(block, nil, ErrBlacklistedHash)
			return i, events, coalescedbgmlogss, ErrBlacklistedHash
		}
		// Wait for the block's verification to complete
		bstart := time.Now()

		err := <-results
		if err == nil {
			err = batcr.Validator().ValidateBody(block)
		}
		if err != nil {
			if err == ErrKnownBlock {
				stats.ignored++
				continue
			}

			if err == consensus.ErrFutureBlock {
				// Allow up to MaxFuture second in the future blocks. If this limit
				// is exceeded the chain is discarded and processed at a later time
				// if given.
				max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
				if block.Time().Cmp(max) > 0 {
					return i, events, coalescedbgmlogss, fmt.Errorf("future block: %v > %v", block.Time(), max)
				}
				batcr.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			if err == consensus.ErrUnknownAncestor && batcr.futureBlocks.Contains(block.ParentHash()) {
				batcr.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			batcr.reportBlock(block, nil, err)
			return i, events, coalescedbgmlogss, err
		}
		// Create a new statedb using the parent block and report an
		// error if it fails.
		var parent *types.Block
		if i == 0 {
			parent = batcr.GetBlock(block.ParentHash(), block.NumberU64()-1)
		} else {
			parent = chain[i-1]
		}
		block.DposContext, err = types.NewDposContextFromProto(batcr.chainDb, parent.Header().DposContext)
		if err != nil {
			return i, events, coalescedbgmlogss, err
		}
		state, err := state.New(parent.Root(), batcr.stateCache)
		if err != nil {
			return i, events, coalescedbgmlogss, err
		}
		// Process block using the parent state as reference point.
		receipts, bgmlogss, usedGas, err := batcr.processor.Process(block, state, batcr.vmConfig)
		if err != nil {
			batcr.reportBlock(block, receipts, err)
			return i, events, coalescedbgmlogss, err
		}

		// Validate the state using the default validator
		err = batcr.Validator().ValidateState(block, parent, state, receipts, usedGas)
		if err != nil {
			batcr.reportBlock(block, receipts, err)
			return i, events, coalescedbgmlogss, err
		}
		// Validate the dpos state using the default validator
		err = batcr.Validator().ValidateDposState(block)
		if err != nil {
			batcr.reportBlock(block, receipts, err)
			return i, events, coalescedbgmlogss, err
		}
		// Validate validator
		dposEngine, isDpos := batcr.engine.(*dpos.Dpos)
		if isDpos {
			err = dposEngine.VerifySeal(bc, block.Header())
			if err != nil {
				batcr.reportBlock(block, receipts, err)
				return i, events, coalescedbgmlogss, err
			}
		}

		// Validate the dpos state using the default validator
		// Write the block to the chain and get the status.
		status, err := batcr.WriteBlockAndState(block, receipts, state)
		if err != nil {
			return i, events, coalescedbgmlogss, err
		}
		switch status {
		case CanonStatTy:
			bgmlogs.Debug("Inserted new block", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", bgmcommon.PrettyDuration(time.Since(bstart)))

			coalescedbgmlogss = append(coalescedbgmlogss, bgmlogss...)
			blockInsertTimer.UpdateSince(bstart)
			events = append(events, ChainEvent{block, block.Hash(), bgmlogss})
			lastCanon = block

		case SideStatTy:
			bgmlogs.Debug("Inserted forked block", "number", block.Number(), "hash", block.Hash(), "diff", block.Difficulty(), "elapsed",
				bgmcommon.PrettyDuration(time.Since(bstart)), "txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()))

			blockInsertTimer.UpdateSince(bstart)
		}
		stats.processed++
		stats.usedGas += usedGas.Uint64()
		stats.report(chain, i)
	}
	// Append a single chain head event if we've progressed the chain
	if lastCanon != nil && batcr.LastBlockHash() == lastCanon.Hash() {
		events = append(events, ChainHeadEvent{lastCanon})
	}
	return 0, events, coalescedbgmlogss, nil
}

// insertStats tracks and reports on block insertion.
type insertStats struct {
	queued, processed, ignored int
	usedGas                    uint64
	lastIndex                  int
	startTime                  mclock.AbsTime
}

// statsReportLimit is the time limit during import after which we always print
// out progress. This avoids the user wondering what's going on.
const statsReportLimit = 8 * time.Second

// report prints statistics if some number of blocks have been processed
// or more than a few seconds have passed since the last message.
func (st *insertStats) report(chain []*types.Block, index int) {
	// Fetch the timings for the batch
	var (
		now     = mclock.Now()
		elapsed = time.Duration(now) - time.Duration(st.startTime)
	)
	// If we're at the last block of the batch or report period reached, bgmlogs
	if index == len(chain)-1 || elapsed >= statsReportLimit {
		var (
			end = chain[index]
			txs = countTransactions(chain[st.lastIndex : index+1])
		)
		context := []interface{}{
			"blocks", st.processed, "txs", txs, "mgas", float64(st.usedGas) / 1000000,
			"elapsed", bgmcommon.PrettyDuration(elapsed), "mgasps", float64(st.usedGas) * 1000 / float64(elapsed),
			"number", end.Number(), "hash", end.Hash(),
		}
		if st.queued > 0 {
			context = append(context, []interface{}{"queued", st.queued}...)
		}
		if st.ignored > 0 {
			context = append(context, []interface{}{"ignored", st.ignored}...)
		}
		bgmlogs.Info("Imported new chain segment", context...)

		*st = insertStats{startTime: now, lastIndex: index + 1}
	}
}

func countTransactions(chain []*types.Block) (c int) {
	for _, b := range chain {
		c += len(bPtr.Transactions())
	}
	return c
}

// reorgs takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain and accumulates potential missing transactions and post an
// event about them
func (batcr *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		bgmcommonBlock *types.Block
		deletedTxs  types.Transactions
		deletedbgmlogss []*types.bgmlogs
		// collectbgmlogss collects the bgmlogss that were generated during the
		// processing of the block that corresponds with the given hashPtr.
		// These bgmlogss are later announced as deleted.
		collectbgmlogss = func(h bgmcommon.Hash) {
			// Coalesce bgmlogss and set 'Removed'.
			receipts := GetBlockReceipts(batcr.chainDb, h, batcr.hcPtr.GetBlockNumber(h))
			for _, receipt := range receipts {
				for _, bgmlogs := range receipt.bgmlogss {
					del := *bgmlogs
					del.Removed = true
					deletedbgmlogss = append(deletedbgmlogss, &del)
				}
			}
		}
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = batcr.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

			collectbgmlogss(oldBlock.Hash())
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = batcr.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

	for {
		if oldBlock.Hash() == newBlock.Hash() {
			bgmcommonBlock = oldBlock
			break
		}

		oldChain = append(oldChain, oldBlock)
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectbgmlogss(oldBlock.Hash())

		oldBlock, newBlock = batcr.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1), batcr.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1)
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}
	// Ensure the user sees large reorgs
	if len(oldChain) > 0 && len(newChain) > 0 {
		bgmlogsFn := bgmlogs.Debug
		if len(oldChain) > 63 {
			bgmlogsFn = bgmlogs.Warn
		}
		bgmlogsFn("Chain split detected", "number", bgmcommonBlock.Number(), "hash", bgmcommonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
	} else {
		bgmlogs.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Number(), "oldhash", oldBlock.Hash(), "newnum", newBlock.Number(), "newhash", newBlock.Hash())
	}
	var addedTxs types.Transactions
	// insert blocks. Order does not matter. Last block will be written in ImportChain itself which creates the new head properly
	for _, block := range newChain {
		// insert the block in the canonical way, re-writing history
		batcr.insert(block)
		// write lookup entries for hash based transaction/receipt searches
		if err := WriteTxLookupEntries(batcr.chainDb, block); err != nil {
			return err
		}
		addedTxs = append(addedTxs, block.Transactions()...)
	}

	// calculate the difference between deleted and added transactions
	diff := types.TxDifference(deletedTxs, addedTxs)
	// When transactions get deleted from the database that means the
	// receipts that were created in the fork must also be deleted
	for _, tx := range diff {
		DeleteTxLookupEntry(batcr.chainDb, tx.Hash())
	}
	if len(deletedbgmlogss) > 0 {
		go batcr.rmbgmlogssFeed.Send(RemovedbgmlogssEvent{deletedbgmlogss})
	}

	return nil
}

// PostChainEvents iterates over the events generated by a chain insertion and
// posts them into the event feed.
// TODO: Should not expose PostChainEvents. The chain events should be posted in WriteBlock.
func (batcr *BlockChain) PostChainEvents(events []interface{}, bgmlogss []*types.bgmlogs) {
	// post event bgmlogss for further processing
	if bgmlogss != nil {
		batcr.bgmlogssFeed.Send(bgmlogss)
	}
	for _, event := range events {
		switch ev := event.(type) {
		case ChainEvent:
			batcr.chainFeed.Send(ev)

		case ChainHeadEvent:
			batcr.chainHeadFeed.Send(ev)
		}
	}
}

func (batcr *BlockChain) update() {
	futureTimer := time.Tick(5 * time.Second)
	for {
		select {
		case <-futureTimer:
			batcr.procFutureBlocks()
		case <-batcr.quit:
			return
		}
	}
}

// BadBlockArgs represents the entries in the list returned when bad blocks are queried.
type BadBlockArgs struct {
	Hash   bgmcommon.Hash   `json:"hash"`
	headerPtr *types.Header `json:"header"`
}

// BadBlocks returns a list of the last 'bad blocks' that the client has seen on the network
func (batcr *BlockChain) BadBlocks() ([]BadBlockArgs, error) {
	headers := make([]BadBlockArgs, 0, batcr.badBlocks.Len())
	for _, hash := range batcr.badBlocks.Keys() {
		if hdr, exist := batcr.badBlocks.Peek(hash); exist {
			header := hdr.(*types.Header)
			headers = append(headers, BadBlockArgs{headerPtr.Hash(), header})
		}
	}
	return headers, nil
}

// addBadBlock adds a bad block to the bad-block LRU cache
func (batcr *BlockChain) addBadBlock(block *types.Block) {
	batcr.badBlocks.Add(block.Header().Hash(), block.Header())
}

// reportBlock bgmlogss a bad block error.
func (batcr *BlockChain) reportBlock(block *types.Block, receipts types.Receipts, err error) {
	batcr.addBadBlock(block)

	var receiptString string
	for _, receipt := range receipts {
		receiptString += fmt.Sprintf("\t%v\n", receipt)
	}
	bgmlogs.Error(fmt.Sprintf(`
########## BAD BLOCK #########
Chain config: %v

Number: %v
Hash: 0x%x
%v

Error: %v
##############################
`, batcr.config, block.Number(), block.Hash(), receiptString, err))
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify bgmparameter can be used to fine tune whbgmchain nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check eachPtr.
func (batcr *BlockChain) InsertHeaderChain(chain []*types.headerPtr, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := batcr.hcPtr.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	// Make sure only one thread manipulates the chain at once
	batcr.chainmu.Lock()
	defer batcr.chainmu.Unlock()

	batcr.wg.Add(1)
	defer batcr.wg.Done()

	whFunc := func(headerPtr *types.Header) error {
		batcr.mu.Lock()
		defer batcr.mu.Unlock()

		_, err := batcr.hcPtr.WriteHeader(header)
		return err
	}

	return batcr.hcPtr.InsertHeaderChain(chain, whFunc, start)
}

// writeHeader writes a header into the local chain, given that its parent is
// already known. If the total difficulty of the newly inserted header becomes
// greater than the current known TD, the canonical chain is re-routed.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (batcr *BlockChain) writeHeader(headerPtr *types.Header) error {
	batcr.wg.Add(1)
	defer batcr.wg.Done()

	batcr.mu.Lock()
	defer batcr.mu.Unlock()

	_, err := batcr.hcPtr.WriteHeader(header)
	return err
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (batcr *BlockChain) CurrentHeader() *types.Header {
	batcr.mu.RLock()
	defer batcr.mu.RUnlock()

	return batcr.hcPtr.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (batcr *BlockChain) GetTd(hash bgmcommon.Hash, number uint64) *big.Int {
	return batcr.hcPtr.GetTd(hash, number)
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (batcr *BlockChain) GetTdByHash(hash bgmcommon.Hash) *big.Int {
	return batcr.hcPtr.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (batcr *BlockChain) GetHeader(hash bgmcommon.Hash, number uint64) *types.Header {
	return batcr.hcPtr.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (batcr *BlockChain) GetHeaderByHash(hash bgmcommon.Hash) *types.Header {
	return batcr.hcPtr.GetHeaderByHash(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (batcr *BlockChain) HasHeader(hash bgmcommon.Hash, number uint64) bool {
	return batcr.hcPtr.HasHeader(hash, number)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (batcr *BlockChain) GetBlockHashesFromHash(hash bgmcommon.Hash, max uint64) []bgmcommon.Hash {
	return batcr.hcPtr.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (batcr *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return batcr.hcPtr.GetHeaderByNumber(number)
}

// Config retrieves the blockchain's chain configuration.
func (batcr *BlockChain) Config() *bgmparam.ChainConfig { return batcr.config }

// Engine retrieves the blockchain's consensus engine.
func (batcr *BlockChain) Engine() consensus.Engine { return batcr.engine }

// SubscribeRemovedbgmlogssEvent registers a subscription of RemovedbgmlogssEvent.
func (batcr *BlockChain) SubscribeRemovedbgmlogssEvent(ch chan<- RemovedbgmlogssEvent) event.Subscription {
	return batcr.scope.Track(batcr.rmbgmlogssFeed.Subscribe(ch))
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (batcr *BlockChain) SubscribeChainEvent(ch chan<- ChainEvent) event.Subscription {
	return batcr.scope.Track(batcr.chainFeed.Subscribe(ch))
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (batcr *BlockChain) SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription {
	return batcr.scope.Track(batcr.chainHeadFeed.Subscribe(ch))
}

// SubscribebgmlogssEvent registers a subscription of []*types.bgmlogs.
func (batcr *BlockChain) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return batcr.scope.Track(batcr.bgmlogssFeed.Subscribe(ch))
}
