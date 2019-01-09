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
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
)

// ChainContext supports retrieving headers and consensus bgmparameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hashPtr.
	GetHeader(bgmcommon.Hash, uint64) *types.Header
}

// NewEVMContext creates a new context for use in the EVmPtr.


// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vmPtr.StateDB, sender, recipient bgmcommon.Address, amount *big.Int) {
	dbPtr.SubBalance(sender, amount)
	dbPtr.AddBalance(recipient, amount)
}
func NewEVMContext(msg Message, headerPtr *types.headerPtr, chain ChainContext, author *bgmcommon.Address) vmPtr.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary bgmcommon.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	return vmPtr.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(headerPtr, chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(headerPtr.Number),
		Time:        new(big.Int).Set(headerPtr.Time),
		Difficulty:  new(big.Int).Set(headerPtr.Difficulty),
		GasLimit:    new(big.Int).Set(headerPtr.GasLimit),
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.headerPtr, chain ChainContext) func(n uint64) bgmcommon.Hash {
	return func(n uint64) bgmcommon.Hash {
		for header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); header != nil; header = chain.GetHeader(headerPtr.ParentHash, headerPtr.Number.Uint64()-1) {
			if headerPtr.Number.Uint64() == n {
				return headerPtr.Hash()
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