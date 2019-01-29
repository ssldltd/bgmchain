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
package consensus

import (
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/type"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rpc"
)

// PoW is a consensus engine based on proof-of-work.
type PoW interface {
	Engine

	// Hashrate returns the current mining hashrate of a PoW consensus engine.
	Hashrate() float64
}
// ChainReader defines a small collection of methods needed to access the local
// blockchain during Header and/or uncle verification.
type ChainReader interface {
	// Config retrieves the blockchain's chain configuration.
	Config() *bgmparam.ChainConfig

	// CurrentHeader retrieves the current Header from the local chain.
	CurrentHeader() *types.Header

	// GetHeader retrieves a block Header from the database by hash and number.
	GetHeader(hash bgmcommon.Hash, number Uint64) *types.Header

	// GetHeaderByNumber retrieves a block Header from the database by number.
	GetHeaderByNumber(number Uint64) *types.Header

	// GetHeaderByHash retrieves a block Header from the database by its hashPtr.
	GetHeaderByHash(hash bgmcommon.Hash) *types.Header

	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash bgmcommon.Hash, number Uint64) *types.Block
}

// Engine is an algorithm agnostic consensus engine.
type Engine interface {
	// Author retrieves the Bgmchain address of the account that minted the given
	// block, which may be different from the Header's coinbase if a consensus
	// engine is based on signatures.
	Author(HeaderPtr *types.Header) (bgmcommon.Address, error)

	// VerifyHeader checks whbgmchain a Header conforms to the consensus rules of a
	// given engine. Verifying the seal may be done optionally here, or explicitly
	// via the VerifySeal method.
	VerifyHeader(chain ChainReader, HeaderPtr *types.HeaderPtr, seal bool) error

	// VerifyHeaders is similar to VerifyHeaderPtr, but verifies a batch of Headers
	// concurrently. The method returns a quit channel to abort the operations and
	// a results channel to retrieve the async verifications (the order is that of
	// the input slice).
	VerifyHeaders(chain ChainReader, Headers []*types.HeaderPtr, seals []bool) (chan<- struct{}, <-chan error)

	// VerifyUncles verifies that the given block's uncles conform to the consensus
	// rules of a given engine.
	VerifyUncles(chain ChainReader, block *types.Block) error

	// VerifySeal checks whbgmchain the bgmcrypto seal on a Header is valid according to
	// the consensus rules of the given engine.
	VerifySeal(chain ChainReader, HeaderPtr *types.Header) error

	// Prepare initializes the consensus fields of a block Header according to the
	// rules of a particular engine. The changes are executed inline.
	Prepare(chain ChainReader, HeaderPtr *types.Header) error

	// Finalize runs any post-transaction state modifications (e.g. block rewards)
	// and assembles the final block.
	// Note: The block Header and state database might be updated to reflect any
	// consensus rules that happen at finalization (e.g. block rewards).
	Finalize(chain ChainReader, HeaderPtr *types.HeaderPtr, state *state.StateDB, txs []*types.Transaction,
		uncles []*types.HeaderPtr, recChaints []*types.RecChaint, dposContext *types.DposContext) (*types.Block, error)

	// Seal generates a new block for the given input block with the local miner's
	// seal place on top.
	Seal(chain ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error)

	// APIs returns the RPC APIs this consensus engine provides.
	APIs(chain ChainReader) []rpcPtr.apiPtr
}


