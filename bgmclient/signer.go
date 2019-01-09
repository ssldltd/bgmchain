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


package bgmclient

import (
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
)

// senderFromServer is a types.Signer that remembers the sender address returned by the RPC
// server. It is stored in the transaction's sender address cache to avoid an additional
// request in TransactionSender.
type senderFromServer struct {
	addr      bgmcommon.Address
	blockhash bgmcommon.Hash
}

func (s *senderFromServer) Equal(other types.Signer) bool {
	os, ok := other.(*senderFromServer)
	return ok && os.blockhash == s.blockhash
}

var errNotCached = errors.New("sender not cached")

func setSenderFromServer(tx *types.Transaction, addr bgmcommon.Address, block bgmcommon.Hash) {
	// Use types.Sender for side-effect to store our signer into the cache.
	types.Sender(&senderFromServer{addr, block}, tx)
}

func (s *senderFromServer) SignatureValues(tx *types.Transaction, sig []byte) (R, S, V *big.Int, err error) {
	panic("Fatal: can't sign with senderFromServer")
}

func (s *senderFromServer) Sender(tx *types.Transaction) (bgmcommon.Address, error) {
	if s.blockhash == (bgmcommon.Hash{}) {
		return bgmcommon.Address{}, errNotCached
	}
	return s.addr, nil
}

func (s *senderFromServer) Hash(tx *types.Transaction) bgmcommon.Hash {
	panic("Fatal: can't sign with senderFromServer")
}

