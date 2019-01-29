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

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
)

// ChainContext supports retrieving Headers and consensus bgmparameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hashPtr.
	GetHeader(bgmcommon.Hash, Uint64) *types.Header
}

// NewEVMContext creates a new context for use in the EVmPtr.


// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vmPtr.StateDB, sender, recipient bgmcommon.Address, amount *big.Int) {
	dbPtr.SubBalance(sender, amount)
	dbPtr.AddBalance(recipient, amount)
}
func NewEVMContext(msg Message, HeaderPtr *types.HeaderPtr, chain ChainContext, author *bgmcommon.Address) vmPtr.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the Header
	var beneficiary bgmcommon.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(Header) // Ignore error, we're past Header validation
	} else {
		beneficiary = *author
	}
	return vmPtr.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(HeaderPtr, chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		number: new(big.Int).Set(HeaderPtr.Number),
		time:        new(big.Int).Set(HeaderPtr.time),
		Difficulty:  new(big.Int).Set(HeaderPtr.Difficulty),
		GasLimit:    new(big.Int).Set(HeaderPtr.GasLimit),
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves Header hashes by number
func GetHashFn(ref *types.HeaderPtr, chain ChainContext) func(n Uint64) bgmcommon.Hash {
	return func(n Uint64) bgmcommon.Hash {
		for Header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); Header != nil; Header = chain.GetHeader(HeaderPtr.ParentHash, HeaderPtr.Number.Uint64()-1) {
			if HeaderPtr.Number.Uint64() == n {
				return HeaderPtr.Hash()
			}
		}

		return bgmcommon.Hash{}
	}
}

// CanTransfer checks wbgmchain there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vmPtr.StateDB, addr bgmcommon.Address, amount *big.Int) bool {
	return dbPtr.GetBalance(addr).Cmp(amount) >= 0
}