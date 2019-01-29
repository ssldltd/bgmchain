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
	"math/big"

	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
)

// Validator is an interface which defines the standard for block validation. It
// done by the specific consensus engines.
//
type Validator interface {
	// ValidateBody validates the given block's content.
	ValidateBody(block *types.Block) error

	ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error
	// ValidateDposState validates the given dpos state
	ValidateDposState(block *types.Block) error
}

// Processor is an interface for processing blocks using a given initial state.
//
// Process takes the block to be processed and the statedb upon which the
type Processor interface {
	Process(block *types.Block, statedbPtr *state.StateDB, cfg vmPtr.Config) (types.Receipts, []*types.bgmlogs, *big.Int, error)
}
