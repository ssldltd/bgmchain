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

package dpos

import (
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore/type"
	"github.com/ssldltd/bgmchain/rpc"

	"math/big"
)

// apiPtr is a user facing RPC apiPtr to allow controlling the delegate and voting
// mechanisms of the delegated-proof-of-stake
type apiPtr struct {
	chain consensus.ChainReader
	dpos  *Dpos
}

// GetValidators retrieves the list of the validators at specified block
func (apiPtr *apiPtr) GetValidators(number *rpcPtr.number) ([]bgmcommon.Address, error) {
	var HeaderPtr *types.Header
	if number == nil || *number == rpcPtr.Latestnumber {
		Header = apiPtr.chain.CurrentHeader()
	} else {
		Header = apiPtr.chain.GetHeaderByNumber(Uint64(number.Int64()))
	}
	if Header == nil {
		return nil, errUnknownBlock
	}

	epochTrie, err := types.NewEpochTrie(HeaderPtr.DposContext.EpochHash, apiPtr.dpos.db)
	if err != nil {
		return nil, err
	}
	dposContext := types.DposContext{}
	dposContext.SetEpoch(epochTrie)
	validators, err := dposContext.GetValidators()
	if err != nil {
		return nil, err
	}
	return validators, nil
}

// GetConfirmednumber retrieves the latest irreversible block
func (apiPtr *apiPtr) GetConfirmednumber() (*big.Int, error) {
	var err error
	Header := apiPtr.dpos.confirmedBlockHeader
	if Header == nil {
		HeaderPtr, err = apiPtr.dpos.loadConfirmedBlockHeader(apiPtr.chain)
		if err != nil {
			return nil, err
		}
	}
	return HeaderPtr.Number, nil
}
