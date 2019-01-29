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
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus/dpos"
	"github.com/ssldltd/bgmchain/consensus/bgmash"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmparam"
)
// BlockGen creates blocks for testing.
type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	Header  *types.Header
	statedbPtr *state.StateDB

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header

	config *bgmparam.ChainConfig
}
// So we can deterministically seed different blockchains
var (
	canonicalSeed = 1
	forkSeed      = 2
)

// Offsettime modifies the time instance of a block, implicitly changing its
// associated difficulty. It's useful to test scenarios where forking is not
func (bPtr *BlockGen) Offsettime(seconds int64) {
	bPtr.HeaderPtr.time.Add(bPtr.HeaderPtr.time, new(big.Int).SetInt64(seconds))
	if bPtr.HeaderPtr.time.Cmp(bPtr.parent.Header().time) <= 0 {
		panic("Fatal: block time out of range")
	}
	bPtr.HeaderPtr.Difficulty = bgmashPtr.CalcDifficulty(bPtr.config, bPtr.HeaderPtr.time.Uint64(), bPtr.parent.Header())
}
// Blocks created by GenerateChain do not contain valid proof of work
// values. Inserting them into BlockChain requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(config *bgmparam.ChainConfig, parent *types.Block, db bgmdbPtr.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = bgmparam.DposChainConfig
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	genblock := func(i int, hPtr *types.HeaderPtr, statedbPtr *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{parent: parent, i: i, chain: blocks, Header: h, statedb: statedb, config: config}
		// Mutate the state and block according to any hard-fork specs
		if daoBlock := config.DAOForkBlock; daoBlock != nil {
			limit := new(big.Int).Add(daoBlock, bgmparam.DAOForkExtraRange)
			if hPtr.Number.Cmp(daoBlock) >= 0 && hPtr.Number.Cmp(limit) < 0 {
				if config.DAOForkSupport {
					hPtr.Extra = bgmcommon.CopyBytes(bgmparam.DAOForkBlockExtra)
				}
			}
		}
		if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(hPtr.Number) == 0 {
			miscPtr.ApplyDAOHardFork(statedb)
		}
		// Execute any user modifications to the block and finalize it
		if gen != nil {
			gen(i, b)
		}
		dpos.AccumulateRewards(config, statedb, h, bPtr.uncles)
		blockRoot, err := statedb.commitTo(db, config.IsEIP158(hPtr.Number))
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}
		hPtr.Root = blockRoot
		hPtr.DposContext = parent.Header().DposContext
		return types.NewBlock(h, bPtr.txs, bPtr.uncles, bPtr.receipts), bPtr.receipts
	}
	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		Header := makeHeader(config, parent, statedb)
		block, receipt := genblock(i, HeaderPtr, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (bPtr *BlockGen) SetCoinbase(addr bgmcommon.Address) {
	if bPtr.gasPool != nil {
		if len(bPtr.txs) > 0 {
			panic("Fatal: coinbase must be set before adding transactions")
		}
		panic("Fatal: coinbase can only be set once")
	}
	bPtr.HeaderPtr.Coinbase = addr
	bPtr.gasPool = new(GasPool).AddGas(bPtr.HeaderPtr.GasLimit)
}

// SetExtra sets the extra data field of the generated block.
func (bPtr *BlockGen) SetExtra(data []byte) {
	bPtr.HeaderPtr.Extra = data
}

// been set, the block's coinbase is set to the zero address.
//
// AddTx panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etcPtr.), there are some
// will panic during execution.
func (bPtr *BlockGen) AddTx(tx *types.Transaction) {
	if bPtr.gasPool == nil {
		bPtr.SetCoinbase(bgmcommon.Address{})
	}
	bPtr.statedbPtr.Prepare(tx.Hash(), bgmcommon.Hash{}, len(bPtr.txs))
	receipt, _, err := ApplyTransaction(bPtr.config, nil, nil, &bPtr.HeaderPtr.Coinbase, bPtr.gasPool, bPtr.statedb, bPtr.HeaderPtr, tx, bPtr.HeaderPtr.GasUsed, vmPtr.Config{})
	if err != nil {
		panic(err)
	}
	bPtr.txs = append(bPtr.txs, tx)
	bPtr.receipts = append(bPtr.receipts, receipt)
}

// Number returns the block number of the block being generated.
func (bPtr *BlockGen) Number() *big.Int {
	return new(big.Int).Set(bPtr.HeaderPtr.Number)
}

// backing transaction.
//
// AddUncheckedReceipt will cause consensus failures when used during real
// chain processing. This is best used in conjunction with raw block insertion.
func (bPtr *BlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
	bPtr.receipts = append(bPtr.receipts, receipt)
}

// account at addr. It panics if the account does not exist.
func (bPtr *BlockGen) TxNonce(addr bgmcommon.Address) Uint64 {
	if !bPtr.statedbPtr.Exist(addr) {
		panic("Fatal: account does not exist")
	}
	return bPtr.statedbPtr.GetNonce(addr)
}

// AddUncle adds an uncle Header to the generated block.
func (bPtr *BlockGen) AddUncle(hPtr *types.Header) {
	bPtr.uncles = append(bPtr.uncles, h)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
func (bPtr *BlockGen) PrevBlock(index int) *types.Block {
	if index >= bPtr.i {
		panic("Fatal: block index out of range")
	}
	if index == -1 {
		return bPtr.parent
	}
	return bPtr.chain[index]
}



func makeHeader(config *bgmparam.ChainConfig, parent *types.Block, state *state.StateDB) *types.Header {
	var time *big.Int
	if parent.time() == nil {
		time = big.NewInt(10)
	} else {
		time = new(big.Int).Add(parent.time(), big.NewInt(10)) // block time is fixed at 10 seconds
	}

	return &types.Header{
		Root:        state.IntermediateRoot(config.IsEIP158(parent.Number())),
		ParentHash:  parent.Hash(),
		Coinbase:    parent.Coinbase(),
		Difficulty:  parent.Difficulty(),
		DposContext: &types.DposContextProto{},
		GasLimit:    CalcGasLimit(parent),
		GasUsed:     new(big.Int),
		Number:      new(big.Int).Add(parent.Number(), bgmcommon.Big1),
		time:        time,
	}
}

// newCanonical creates a chain database, and injects a deterministic canonical
// chain. Depending on the full flag, if creates either a full block chain or a
// Header only chain.
func newCanonical(n int, full bool) (bgmdbPtr.Database, *BlockChain, error) {
	// Initialize a fresh chain with only a genesis block
	gspec := new(Genesis)
	db, _ := bgmdbPtr.NewMemDatabase()
	genesis := gspecPtr.MustCommit(db)

	blockchain, _ := NewBlockChain(db, bgmparam.AllBgmashProtocolChanges, bgmashPtr.NewFaker(), vmPtr.Config{})
	// Create and inject the requested chain
	if n == 0 {
		return db, blockchain, nil
	}
	if full {
		// Full block-chain requested
		blocks := makeBlockChain(genesis, n, db, canonicalSeed)
		_, err := blockchain.InsertChain(blocks)
		return db, blockchain, err
	}
	// Header-only chain requested
	Headers := makeHeaderChain(genesis.Header(), n, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(Headers, 1)
	return db, blockchain, err
}

// makeHeaderChain creates a deterministic chain of Headers blockRooted at parent.
func makeHeaderChain(parent *types.HeaderPtr, n int, db bgmdbPtr.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, db, seed)
	Headers := make([]*types.HeaderPtr, len(blocks))
	for i, block := range blocks {
		Headers[i] = block.Header()
	}
	return Headers
}

// makeBlockChain creates a deterministic chain of blocks blockRooted at parent.
func makeBlockChain(parent *types.Block, n int, db bgmdbPtr.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(bgmparam.DposChainConfig, parent, db, n, func(i int, bPtr *BlockGen) {
		bPtr.SetCoinbase(bgmcommon.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}
