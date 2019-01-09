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
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *bgmparam.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *bgmparam.ChainConfig, bcPtr *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedbPtr *state.StateDB, cfg vmPtr.Config) (types.Receipts, []*types.bgmlogs, *big.Int, error) {
	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		header       = block.Header()
		allbgmlogss      []*types.bgmlogs
		gp           = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		miscPtr.ApplyDAOHardFork(statedb)
	}
	// Set block dpos context
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedbPtr.Prepare(tx.Hash(), block.Hash(), i)
		receipt, _, err := ApplyTransaction(p.config, block.DposCtx(), p.bc, nil, gp, statedb, headerPtr, tx, totalUsedGas, cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		receipts = append(receipts, receipt)
		allbgmlogss = append(allbgmlogss, receipt.bgmlogss...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, headerPtr, statedb, block.Transactions(), block.Uncles(), receipts, block.DposCtx())

	return receipts, allbgmlogss, totalUsedGas, nil
}

func applyDposMessage(dposContext *types.DposContext, msg types.Message) error {
	switch msg.Type() {
	case types.bgmlogsinCandidate:
		dposContext.BecomeCandidate(msg.From())
	case types.bgmlogsoutCandidate:
		dposContext.KickoutCandidate(msg.From())
	case types.Delegate:
		dposContext.Delegate(msg.From(), *(msg.To()))
	case types.UnDelegate:
		dposContext.UnDelegate(msg.From(), *(msg.To()))
	default:
		return types.errorInvalidType
	}
	return nil
}
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *bgmparam.ChainConfig, dposContext *types.DposContext, bcPtr *BlockChain, author *bgmcommon.Address, gp *GasPool, statedbPtr *state.StateDB, headerPtr *types.headerPtr, tx *types.Transaction, usedGas *big.Int, cfg vmPtr.Config) (*types.Receipt, *big.Int, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, headerPtr.Number))
	if err != nil {
		return nil, nil, err
	}

	if msg.To() == nil && msg.Type() != types.Binary {
		return nil, nil, types.errorInvalidType
	}

	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, headerPtr, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vmPtr.NewEVM(context, statedb, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, nil, err
	}
	if msg.Type() != types.Binary {
		if err = applyDposMessage(dposContext, msg); err != nil {
			return nil, nil, err
		}
	}

	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(headerPtr.Number) {
		statedbPtr.Finalise(true)
	} else {
		root = statedbPtr.IntermediateRoot(config.IsEIP158(headerPtr.Number)).Bytes()
	}
	usedGas.Add(usedGas, gas)

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wbgmchain the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(gas)
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = bgmcrypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}

	// Set the receipt bgmlogss and create a bloom for filtering
	receipt.bgmlogss = statedbPtr.Getbgmlogss(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, gas, err
}

