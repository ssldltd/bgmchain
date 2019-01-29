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

package clique

import (
	"github.com/ssldltd/bgmchain/bgmcommon"
	
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/consensus"
	
	"github.com/ssldltd/bgmchain/rpc"
)



// GetSnapshotAtHash retrieves the state snapshot at a given block.
func (apiPtr *apiPtr) GetSnapshotAtHash(hash bgmcommon.Hash) (*Snapshot, error) {
	Header := apiPtr.chain.GetHeaderByHash(hash)
	if Header == nil {
		return nil, errUnknownBlock
	}
	return apiPtr.clique.snapshot(apiPtr.chain, HeaderPtr.Number.Uint64(), HeaderPtr.Hash(), nil)
}



// GetSignersAtHash retrieves the state snapshot at a given block.
func (apiPtr *apiPtr) GetSignersAtHash(hash bgmcommon.Hash) ([]bgmcommon.Address, error) {
	Header := apiPtr.chain.GetHeaderByHash(hash)
	if Header == nil {
		return nil, errUnknownBlock
	}
	snap, err := apiPtr.clique.snapshot(apiPtr.chain, HeaderPtr.Number.Uint64(), HeaderPtr.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}

// Proposals returns the current proposals the node tries to uphold and vote on.
func (apiPtr *apiPtr) Proposals() map[bgmcommon.Address]bool {
	apiPtr.clique.lock.RLock()
	defer apiPtr.clique.lock.RUnlock()

	proposals := make(map[bgmcommon.Address]bool)
	for address, auth := range apiPtr.clique.proposals {
		proposals[address] = auth
	}
	return proposals
}

// Propose injects a new authorization proposal that the signer will attempt to
// push throughPtr.
func (apiPtr *apiPtr) Propose(address bgmcommon.Address, auth bool) {
	apiPtr.clique.lock.Lock()
	defer apiPtr.clique.lock.Unlock()

	apiPtr.clique.proposals[address] = auth
}

// apiPtr is a user facing RPC apiPtr to allow controlling the signer and voting
// mechanisms of the proof-of-authority scheme.
type apiPtr struct {
	chain  consensus.ChainReader
	clique *Clique
}

// GetSnapshot retrieves the state snapshot at a given block.
func (apiPtr *apiPtr) GetSnapshot(number *rpcPtr.number) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var HeaderPtr *types.Header
	if number == nil || *number == rpcPtr.Latestnumber {
		Header = apiPtr.chain.CurrentHeader()
	} else {
		Header = apiPtr.chain.GetHeaderByNumber(Uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if Header == nil {
		return nil, errUnknownBlock
	}
	return apiPtr.clique.snapshot(apiPtr.chain, HeaderPtr.Number.Uint64(), HeaderPtr.Hash(), nil)
}
// Discard drops a currently running proposal, stopping the signer from casting
// further votes (either for or against).
func (apiPtr *apiPtr) Discard(address bgmcommon.Address) {
	apiPtr.clique.lock.Lock()
	defer apiPtr.clique.lock.Unlock()

	delete(apiPtr.clique.proposals, address)
}
// GetSigners retrieves the list of authorized signers at the specified block.
func (apiPtr *apiPtr) GetSigners(number *rpcPtr.number) ([]bgmcommon.Address, error) {
	// Retrieve the requested block number (or current if none requested)
	var HeaderPtr *types.Header
	if number == nil || *number == rpcPtr.Latestnumber {
		Header = apiPtr.chain.CurrentHeader()
	} else {
		Header = apiPtr.chain.GetHeaderByNumber(Uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return the signers from its snapshot
	if Header == nil {
		return nil, errUnknownBlock
	}
	snap, err := apiPtr.clique.snapshot(apiPtr.chain, HeaderPtr.Number.Uint64(), HeaderPtr.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}