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

package bgmcore

import (
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// BlockValidator is responsible for validating block headers, uncles and
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *bgmparam.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *bgmparam.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the the block
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whbgmchain the block's known, and if not, that it's linkable
	if v.bcPtr.HasBlockAndState(block.Hash()) {
		return ErrKnownBlock
	}
	if !v.bcPtr.HasBlockAndState(block.ParentHash()) {
		return consensus.ErrUnknownAncestor
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != headerPtr.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, headerPtr.UncleHash)
	}
	if hash := types.DeriveSha(block.Transactions()); hash != headerPtr.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, headerPtr.TxHash)
	}
	return nil
}

// ValidateState validates the various changes that happen after a state
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block, parent *types.Block, statedbPtr *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	header := block.Header()
	if block.GasUsed().Cmp(usedGas) != 0 {
		return fmt.Errorf("invalid gas used (remote: %v local: %v)", block.GasUsed(), usedGas)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != headerPtr.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", headerPtr.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != headerPtr.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", headerPtr.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't matchPtr.
	if root := statedbPtr.IntermediateRoot(v.config.IsEIP158(headerPtr.Number)); headerPtr.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", headerPtr.Root, root)
	}
	return nil
}
// This is miner strategy, not consensus protocol.
func CalcGasLimit(parent *types.Block) *big.Int {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(3))
	contrib = contribPtr.Div(contrib, big.NewInt(2))
	contrib = contribPtr.Div(contrib, bgmparam.GasLimitBoundDivisor)

	// decay = parentGasLimit / 1024 -1
	decay := new(big.Int).Div(parent.GasLimit(), bgmparam.GasLimitBoundDivisor)
	decay.Sub(decay, big.NewInt(1))
	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl.Set(mathPtr.BigMax(gl, bgmparam.MinGasLimit))

	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if gl.Cmp(bgmparam.TargetGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(mathPtr.BigMin(gl, bgmparam.TargetGasLimit))
	}
	return gl
}
func (v *BlockValidator) ValidateDposState(block *types.Block) error {
	header := block.Header()
	localRoot := block.DposCtx().Root()
	remoteRoot := headerPtr.DposContext.Root()
	if remoteRoot != localRoot {
		return fmt.Errorf("invalid dpos root (remote: %x local: %x)", remoteRoot, localRoot)
	}
	return nil
}

