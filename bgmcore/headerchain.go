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

package bgmCore

import (
	crand "bgmcrypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/hashicorp/golang-lru"
)



// HeaderChain implements the basic block Header chain bgmlogsic that is shared by
// bgmCore.BlockChain and light.LightChain. It is not usable in itself, only as
type HeaderChain struct {
	config *bgmparam.ChainConfig

	chainDb       bgmdbPtr.Database
	genesisHeaderPtr *types.Header

	currentHeader     *types.Header // Current head of the Header chain (may be above the block chain!)
	currentHeaderHash bgmcommon.Hash   // Hash of the current head of the Header chain (prevent recomputing all the time)

	HeaderCache *lru.Cache // Cache for the most recent block Headers
	tdCache     *lru.Cache // Cache for the most recent block total difficulties
	numberCache *lru.Cache // Cache for the most recent block numbers

	procInterrupt func() bool

	rand   *mrand.Rand
	engine consensus.Engine
}

const (
	HeaderCacheLimit = 512
	tdCacheLimit     = 1024
	numberCacheLimit = 4096
)

// NewHeaderChain creates a new HeaderChain structure.
//  getValidator should return the parent's validator
func NewHeaderChain(chainDb bgmdbPtr.Database, config *bgmparam.ChainConfig, engine consensus.Engine, procInterrupt func() bool) (*HeaderChain, error) {
	HeaderCache, _ := lru.New(HeaderCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)

	// Seed a fast but bgmcrypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(mathPtr.MaxInt64))
	if err != nil {
		return nil, err
	}

	hc := &HeaderChain{
		config:        config,
		chainDb:       chainDb,
		HeaderCache:   HeaderCache,
		tdCache:       tdCache,
		numberCache:   numberCache,
		procInterrupt: procInterrupt,
		rand:          mrand.New(mrand.NewSource(seed.Int64())),
		engine:        engine,
	}

	hcPtr.genesisHeader = hcPtr.GetHeaderByNumber(0)
	if hcPtr.genesisHeader == nil {
		return nil, ErrNoGenesis
	}

	hcPtr.currentHeader = hcPtr.genesisHeader
	if head := GetHeadhash(chainDb); head != (bgmcommon.Hash{}) {
		if chead := hcPtr.GetHeaderByHash(head); chead != nil {
			hcPtr.currentHeader = chead
		}
	}
	hcPtr.currentHeaderHash = hcPtr.currentHeaderPtr.Hash()

	return hc, nil
}

// Getnumber retrieves the block number belonging to the given hash
func (hcPtr *HeaderChain) Getnumber(hash bgmcommon.Hash) Uint64 {
	if cached, ok := hcPtr.numberCache.Get(hash); ok {
		return cached.(Uint64)
	}
	number := Getnumber(hcPtr.chainDb, hash)
	if number != missingNumber {
		hcPtr.numberCache.Add(hash, number)
	}
	return number
}

// WriteHeader writes a Header into the local chain, given that its parent is
// already known. If the total difficulty of the newly inserted Header becomes
// greater than the current known TD, the canonical chain is re-routed.
func (hcPtr *HeaderChain) WriteHeader(HeaderPtr *types.Header) (status WriteStatus, err error) {
	// Cache some values to prevent constant recalculation
	var (
		hash   = HeaderPtr.Hash()
		number = HeaderPtr.Number.Uint64()
	)
	// Calculate the total difficulty of the Header
	ptd := hcPtr.GetTd(HeaderPtr.ParentHash, number-1)
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	localTd := hcPtr.GetTd(hcPtr.currentHeaderHash, hcPtr.currentHeaderPtr.Number.Uint64())
	externTd := new(big.Int).Add(HeaderPtr.Difficulty, ptd)

	// Irrelevant of the canonical status, write the td and Header to the database
	if err := hcPtr.WriteTd(hash, number, externTd); err != nil {
		bgmlogs.Crit("Failed to write Header total difficulty", "err", err)
	}
	if err := WriteHeader(hcPtr.chainDb, Header); err != nil {
		bgmlogs.Crit("Failed to write Header content", "err", err)
	}
	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFcPtr.pdf
	if externTd.Cmp(localTd) > 0 || (externTd.Cmp(localTd) == 0 && mrand.Float64() < 0.5) {
		// Delete any canonical number assignments above the new head
		for i := number + 1; ; i++ {
			hash := GetCanonicalHash(hcPtr.chainDb, i)
			if hash == (bgmcommon.Hash{}) {
				break
			}
			DeleteCanonicalHash(hcPtr.chainDb, i)
		}
		// Overwrite any stale canonical number assignments
		var (
			headHash   = HeaderPtr.ParentHash
			headNumber = HeaderPtr.Number.Uint64() - 1
			headHeader = hcPtr.GetHeader(headHash, headNumber)
		)
		for GetCanonicalHash(hcPtr.chainDb, headNumber) != headHash {
			WriteCanonicalHash(hcPtr.chainDb, headHash, headNumber)

			headHash = headHeaderPtr.ParentHash
			headNumber = headHeaderPtr.Number.Uint64() - 1
			headHeader = hcPtr.GetHeader(headHash, headNumber)
		}
		// Extend the canonical chain with the new Header
		if err := WriteCanonicalHash(hcPtr.chainDb, hash, number); err != nil {
			bgmlogs.Crit("Failed to insert Header number", "err", err)
		}
		if err := WriteHeadHeaderHash(hcPtr.chainDb, hash); err != nil {
			bgmlogs.Crit("Failed to insert head Header hash", "err", err)
		}
		hcPtr.currentHeaderHash, hcPtr.currentHeader = hash, types.CopyHeader(Header)

		status = CanonStatTy
	} else {
		status = SideStatTy
	}

	hcPtr.HeaderCache.Add(hash, Header)
	hcPtr.numberCache.Add(hash, number)

	return
}
// necessary since chain events are sent after inserting blocks. Second, the
// Header writes should be protected by the parent chain mutex individually.
type WhCallback func(*types.Header) error

func (hcPtr *HeaderChain) ValidateHeaderChain(chain []*types.HeaderPtr, checkFreq int) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].Number.Uint64() != chain[i-1].Number.Uint64()+1 || chain[i].ParentHash != chain[i-1].Hash() {
			// Chain broke ancestry, bgmlogs a messge (programming error) and skip insertion
			bgmlogs.Error("Non contiguous Header insert", "number", chain[i].Number, "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash, "prevnumber", chain[i-1].Number, "prevhash", chain[i-1].Hash())

			return 0, fmt.Errorf("non contiguous insert: item %-d is #%-d [%x…], item %-d is #%-d [%x…] (parent [%x…])", i-1, chain[i-1].Number,
				chain[i-1].Hash().Bytes()[:4], i, chain[i].Number, chain[i].Hash().Bytes()[:4], chain[i].ParentHash[:4])
		}
	}

	// Generate the list of seal verification requests, and start the parallel verifier
	seals := make([]bool, len(chain))
	for i := 0; i < len(seals)/checkFreq; i++ {
		index := i*checkFreq + hcPtr.rand.Intn(checkFreq)
		if index >= len(seals) {
			index = len(seals) - 1
		}
		seals[index] = true
	}
	seals[len(seals)-1] = true // Last should always be verified to avoid junk

	abort, results := hcPtr.engine.VerifyHeaders(hc, chain, seals)
	defer close(abort)

	// Iterate over the Headers and ensure they all check out
	for i, Header := range chain {
		if hcPtr.procInterrupt() {
			bgmlogs.Debug("Premature abort during Headers verification")
			return 0, errors.New("aborted")
		}
		if BadHashes[HeaderPtr.Hash()] {
			return i, ErrBlacklistedHash
		}
		if err := <-results; err != nil {
			return i, err
		}
	}

	return 0, nil
}

// The verify bgmparameter can be used to fine tune whbgmchain nonce verification
// should be done or not. The reason behind the optional check is because some
// of the Header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check eachPtr.
func (hcPtr *HeaderChain) InsertHeaderChain(chain []*types.HeaderPtr, writeHeader WhCallback, start time.time) (int, error) {
	// Collect some import statistics to report on
	stats := struct{ processed, ignored int }{}
	// All Headers passed verification, import them into the database
	for i, Header := range chain {
		// Short circuit insertion if shutting down
		if hcPtr.procInterrupt() {
			bgmlogs.Debug("Premature abort during Headers import")
			return i, errors.New("aborted")
		}
		// If the Header's already known, skip it, otherwise store
		if hcPtr.HasHeader(HeaderPtr.Hash(), HeaderPtr.Number.Uint64()) {
			stats.ignored++
			continue
		}
		if err := writeHeader(Header); err != nil {
			return i, err
		}
		stats.processed++
	}
	// Report some public statistics so the user has a clue what's going on
	last := chain[len(chain)-1]
	bgmlogs.Info("Imported new block Headers", "count", stats.processed, "elapsed", bgmcommon.PrettyDuration(time.Since(start)),
		"number", last.Number, "hash", last.Hash(), "ignored", stats.ignored)

	return 0, nil
}

// GethashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (hcPtr *HeaderChain) GethashesFromHash(hash bgmcommon.Hash, max Uint64) []bgmcommon.Hash {
	// Get the origin Header from which to fetch
	Header := hcPtr.GetHeaderByHash(hash)
	if Header == nil {
		return nil
	}
	// Iterate the Headers until enough is collected or the genesis reached
	chain := make([]bgmcommon.Hash, 0, max)
	for i := Uint64(0); i < max; i++ {
		next := HeaderPtr.ParentHash
		if Header = hcPtr.GetHeader(next, HeaderPtr.Number.Uint64()-1); Header == nil {
			break
		}
		chain = append(chain, next)
		if HeaderPtr.Number.Sign() == 0 {
			break
		}
	}
	return chain
}

// database by hash and number, caching it if found.
func (hcPtr *HeaderChain) GetTd(hash bgmcommon.Hash, number Uint64) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := hcPtr.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := GetTd(hcPtr.chainDb, hash, number)
	if td == nil {
		return nil
	}
	// Cache the found body for next time and return
	hcPtr.tdCache.Add(hash, td)
	return td
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
func (hcPtr *HeaderChain) GetTdByHash(hash bgmcommon.Hash) *big.Int {
	return hcPtr.GetTd(hash, hcPtr.Getnumber(hash))
}

// WriteTd stores a block's total difficulty into the database, also caching it
// along the way.
func (hcPtr *HeaderChain) WriteTd(hash bgmcommon.Hash, number Uint64, td *big.Int) error {
	if err := WriteTd(hcPtr.chainDb, hash, number, td); err != nil {
		return err
	}
	hcPtr.tdCache.Add(hash, new(big.Int).Set(td))
	return nil
}

// GetHeader retrieves a block Header from the database by hash and number,
func (hcPtr *HeaderChain) GetHeader(hash bgmcommon.Hash, number Uint64) *types.Header {
	// Short circuit if the Header's already in the cache, retrieve otherwise
	if HeaderPtr, ok := hcPtr.HeaderCache.Get(hash); ok {
		return HeaderPtr.(*types.Header)
	}
	Header := GetHeader(hcPtr.chainDb, hash, number)
	if Header == nil {
		return nil
	}
	// Cache the found Header for next time and return
	hcPtr.HeaderCache.Add(hash, Header)
	return Header
}

// GetHeaderByHash retrieves a block Header from the database by hash, caching it if
// found.
func (hcPtr *HeaderChain) GetHeaderByHash(hash bgmcommon.Hash) *types.Header {
	return hcPtr.GetHeader(hash, hcPtr.Getnumber(hash))
}

// HasHeader checks if a block Header is present in the database or not.
func (hcPtr *HeaderChain) HasHeader(hash bgmcommon.Hash, number Uint64) bool {
	if hcPtr.numberCache.Contains(hash) || hcPtr.HeaderCache.Contains(hash) {
		return true
	}
	ok, _ := hcPtr.chainDbPtr.Has(HeaderKey(hash, number))
	return ok
}

// GetHeaderByNumber retrieves a block Header from the database by number,
// caching it (associated with its hash) if found.
func (hcPtr *HeaderChain) GetHeaderByNumber(number Uint64) *types.Header {
	hash := GetCanonicalHash(hcPtr.chainDb, number)
	if hash == (bgmcommon.Hash{}) {
		return nil
	}
	return hcPtr.GetHeader(hash, number)
}

func (hcPtr *HeaderChain) CurrentHeader() *types.Header {
	return hcPtr.currentHeader
}

// SetCurrentHeader sets the current head Header of the canonical chain.
func (hcPtr *HeaderChain) SetCurrentHeader(head *types.Header) {
	if err := WriteHeadHeaderHash(hcPtr.chainDb, head.Hash()); err != nil {
		bgmlogs.Crit("Failed to insert head Header hash", "err", err)
	}
	hcPtr.currentHeader = head
	hcPtr.currentHeaderHash = head.Hash()
}

type DeleteCallback func(bgmcommon.Hash, Uint64)

func (hcPtr *HeaderChain) SetHead(head Uint64, delFn DeleteCallback) {
	height := Uint64(0)
	if hcPtr.currentHeader != nil {
		height = hcPtr.currentHeaderPtr.Number.Uint64()
	}

	for hcPtr.currentHeader != nil && hcPtr.currentHeaderPtr.Number.Uint64() > head {
		hash := hcPtr.currentHeaderPtr.Hash()
		num := hcPtr.currentHeaderPtr.Number.Uint64()
		if delFn != nil {
			delFn(hash, num)
		}
		DeleteHeader(hcPtr.chainDb, hash, num)
		DeleteTd(hcPtr.chainDb, hash, num)
		hcPtr.currentHeader = hcPtr.GetHeader(hcPtr.currentHeaderPtr.ParentHash, hcPtr.currentHeaderPtr.Number.Uint64()-1)
	}
	// Roll back the canonical chain numbering
	for i := height; i > head; i-- {
		DeleteCanonicalHash(hcPtr.chainDb, i)
	}
	// Clear out any stale content from the caches
	hcPtr.HeaderCache.Purge()
	hcPtr.tdCache.Purge()
	hcPtr.numberCache.Purge()

	if hcPtr.currentHeader == nil {
		hcPtr.currentHeader = hcPtr.genesisHeader
	}
	hcPtr.currentHeaderHash = hcPtr.currentHeaderPtr.Hash()

	if err := WriteHeadHeaderHash(hcPtr.chainDb, hcPtr.currentHeaderHash); err != nil {
		bgmlogs.Crit("Failed to reset head Header hash", "err", err)
	}
}

// SetGenesis sets a new genesis block Header for the chain
func (hcPtr *HeaderChain) SetGenesis(head *types.Header) {
	hcPtr.genesisHeader = head
}

// Config retrieves the Header chain's chain configuration.
func (hcPtr *HeaderChain) Config() *bgmparam.ChainConfig { return hcPtr.config }

// Engine retrieves the Header chain's consensus engine.
func (hcPtr *HeaderChain) Engine() consensus.Engine { return hcPtr.engine }

// a Header chain does not have blocks available for retrieval.
func (hcPtr *HeaderChain) GetBlock(hash bgmcommon.Hash, number Uint64) *types.Block {
	return nil
}
