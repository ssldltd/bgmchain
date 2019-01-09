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

package types

import (
	"fmt"
	"io"

	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/bgmcommon"
	
)




type bgmlogsMarshaling struct {
	Data        hexutil.Bytes
	BlockNumber hexutil.Uint64
	TxIndex     hexutil.Uint
	Index       hexutil.Uint
}

type rlpbgmlogs struct {
	Address bgmcommon.Address
	Topics  []bgmcommon.Hash
	Data    []byte
}

type rlpStoragebgmlogs struct {
	Address     bgmcommon.Address
	Topics      []bgmcommon.Hash
	Data        []byte
	BlockNumber uint64
	TxHash      bgmcommon.Hash
	TxIndex     uint
	BlockHash   bgmcommon.Hash
	Index       uint
}
// bgmlogs represents a contract bgmlogs event. These events are generated by the bgmlogs opcode and
// stored/indexed by the node.
type bgmlogs struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address bgmcommon.Address `json:"address" gencodec:"required"`
	// list of topics provided by the contract.
	Topics []bgmcommon.Hash `json:"topics" gencodec:"required"`
	// supplied by the contract, usually ABI-encoded
	Data []byte `json:"data" gencodec:"required"`

	// Derived fields. These fields are filled in by the node
	BlockNumber uint64 `json:"blockNumber"`
	// hash of the transaction
	TxHash bgmcommon.Hash `json:"transactionHash" gencodec:"required"`
	// index of the transaction in the block
	TxIndex uint `json:"transactionIndex" gencodec:"required"`
	// hash of the block in which the transaction was included
	BlockHash bgmcommon.Hash `json:"blockHash"`
	// index of the bgmlogs in the receipt
	Index uint `json:"bgmlogsIndex" gencodec:"required"`

	// The Removed field is true if this bgmlogs was reverted due to a chain reorganisation.
	Removed bool `json:"removed"`
}
// EncodeRLP implements rlp.Encoder.
func (l *bgmlogs) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rlpbgmlogs{Address: l.Address, Topics: l.Topics, Data: l.Data})
}



// DecodeRLP implements rlp.Decoder.
func (l *bgmlogs) DecodeRLP(s *rlp.Stream) error {
	var dec rlpbgmlogs
	err := s.Decode(&dec)
	if err == nil {
		l.Address, l.Topics, l.Data = decPtr.Address, decPtr.Topics, decPtr.Data
	}
	return err
}

func (l *bgmlogs) String() string {
	return fmt.Sprintf(`bgmlogs: %x %x %x %x %-d %x %-d`, l.Address, l.Topics, l.Data, l.TxHash, l.TxIndex, l.BlockHash, l.Index)
}

// bgmlogsForStorage is a wrapper around a bgmlogs that flattens and parses the entire content of
type bgmlogsForStorage bgmlogs

// EncodeRLP implements rlp.Encoder.
func (l *bgmlogsForStorage) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rlpStoragebgmlogs{
		Address:     l.Address,
		Topics:      l.Topics,
		Data:        l.Data,
		BlockNumber: l.BlockNumber,
		TxHash:      l.TxHash,
		TxIndex:     l.TxIndex,
		BlockHash:   l.BlockHash,
		Index:       l.Index,
	})
}
// DecodeRLP implements rlp.Decoder.
func (l *bgmlogsForStorage) DecodeRLP(s *rlp.Stream) error {
	var dec rlpStoragebgmlogs
	err := s.Decode(&dec)
	if err == nil {
		*l = bgmlogsForStorage{
			Address:     decPtr.Address,
			Topics:      decPtr.Topics,
			Data:        decPtr.Data,
			BlockNumber: decPtr.BlockNumber,
			TxHash:      decPtr.TxHash,
			TxIndex:     decPtr.TxIndex,
			BlockHash:   decPtr.BlockHash,
			Index:       decPtr.Index,
		}
	}
	return err
}