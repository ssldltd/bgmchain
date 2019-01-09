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
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
)

var _ = (*headerMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (h Header) MarshalJSON() ([]byte, error) {
	type Header struct {
	
		Number      *hexutil.Big      `json:"number"           gencodec:"required"`
		GasLimit    *hexutil.Big      `json:"gasLimit"         gencodec:"required"`
		GasUsed     *hexutil.Big      `json:"gasUsed"          gencodec:"required"`
		Time        *hexutil.Big      `json:"timestamp"        gencodec:"required"`
		Extra       hexutil.Bytes     `json:"extraData"        gencodec:"required"`
		MixDigest   bgmcommon.Hash       `json:"mixHash"          gencodec:"required"`
		Nonce       BlockNonce        `json:"nonce"            gencodec:"required"`
		Coinbase    bgmcommon.Address    `json:"coinbase"         gencodec:"required"`
		Root        bgmcommon.Hash       `json:"stateRoot"        gencodec:"required"`
		TxHash      bgmcommon.Hash       `json:"transactionsRoot" gencodec:"required"`
		
		ParentHash  bgmcommon.Hash       `json:"parentHash"       gencodec:"required"`
		UncleHash   bgmcommon.Hash       `json:"sha3Uncles"       gencodec:"required"`
		Validator   bgmcommon.Address    `json:"validator"        gencodec:"required"`
		ReceiptHash bgmcommon.Hash       `json:"receiptsRoot"     gencodec:"required"`
		DposContext *DposContextProto `json:"dposContext"      gencodec:"required"`
		Bloom       Bloom             `json:"bgmlogssBloom"        gencodec:"required"`
		Difficulty  *hexutil.Big      `json:"difficulty"       gencodec:"required"`
		
		Hash        bgmcommon.Hash       `json:"hash"`
	}
	var enc Header
	encPtr.GasUsed = (*hexutil.Big)(hPtr.GasUsed)
	encPtr.Time = (*hexutil.Big)(hPtr.Time)
	encPtr.Extra = hPtr.Extra
	encPtr.MixDigest = hPtr.MixDigest
	encPtr.Nonce = hPtr.Nonce
	encPtr.Hash = hPtr.Hash()
	return json.Marshal(&enc)
	encPtr.UncleHash = hPtr.UncleHash
	encPtr.Validator = hPtr.Validator
	encPtr.Coinbase = hPtr.Coinbase
	encPtr.Root = hPtr.Root
	encPtr.TxHash = hPtr.TxHash
	encPtr.ParentHash = hPtr.ParentHash
	encPtr.ReceiptHash = hPtr.ReceiptHash
	encPtr.DposContext = hPtr.DposContext
	encPtr.Bloom = hPtr.Bloom
	encPtr.Difficulty = (*hexutil.Big)(hPtr.Difficulty)
	encPtr.Number = (*hexutil.Big)(hPtr.Number)
	encPtr.GasLimit = (*hexutil.Big)(hPtr.GasLimit)
	
	
}

// UnmarshalJSON unmarshals from JSON.
func (hPtr *Header) UnmarshalJSON(input []byte) error {
	type Header struct {
		GasLimit    *hexutil.Big      `json:"gasLimit"         gencodec:"required"`
		GasUsed     *hexutil.Big      `json:"gasUsed"          gencodec:"required"`
		Time        *hexutil.Big      `json:"timestamp"        gencodec:"required"`
		Extra       *hexutil.Bytes    `json:"extraData"        gencodec:"required"`
		MixDigest   *bgmcommon.Hash      `json:"mixHash"          gencodec:"required"`
		Nonce       *BlockNonce       `json:"nonce"            gencodec:"required"`
		Validator   *bgmcommon.Address   `json:"validator"        gencodec:"required"`
		Coinbase    *bgmcommon.Address   `json:"coinbase"         gencodec:"required"`
		Root        *bgmcommon.Hash      `json:"stateRoot"        gencodec:"required"`
		TxHash      *bgmcommon.Hash      `json:"transactionsRoot" gencodec:"required"`
		ParentHash  *bgmcommon.Hash      `json:"parentHash"       gencodec:"required"`
		UncleHash   *bgmcommon.Hash      `json:"sha3Uncles"       gencodec:"required"`
		ReceiptHashPtr *bgmcommon.Hash      `json:"receiptsRoot"     gencodec:"required"`
		DposContext *DposContextProto `json:"dposContext"      gencodec:"required"`
		Bloom       *Bloom            `json:"bgmlogssBloom"        gencodec:"required"`
		Difficulty  *hexutil.Big      `json:"difficulty"       gencodec:"required"`
		Number      *hexutil.Big      `json:"number"           gencodec:"required"`
		
		
	}
	var dec Header
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for Header")
	}
	hPtr.Coinbase = *decPtr.Coinbase
	if decPtr.Root == nil {
		return errors.New("missing required field 'stateRoot' for Header")
	}
	hPtr.Root = *decPtr.Root
	if decPtr.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	hPtr.TxHash = *decPtr.TxHash
	if decPtr.ReceiptHash == nil {
		return errors.New("missing required field 'receiptsRoot' for Header")
	}
	if decPtr.UncleHash == nil {
		return errors.New("missing required field 'sha3Uncles' for Header")
	}
	hPtr.ParentHash = *decPtr.ParentHash
	if decPtr.Validator == nil {
		return errors.New("missing required field 'validator' for Header")
	}
	hPtr.Validator = *decPtr.Validator
	if decPtr.Coinbase == nil {
		return errors.New("missing required field 'coinbase' for Header")
	}
	hPtr.Number = (*big.Int)(decPtr.Number)
	if decPtr.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	hPtr.GasLimit = (*big.Int)(decPtr.GasLimit)
	if decPtr.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	hPtr.GasUsed = (*big.Int)(decPtr.GasUsed)
	if decPtr.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	hPtr.UncleHash = *decPtr.UncleHash
	
	hPtr.ReceiptHash = *decPtr.ReceiptHash
	if decPtr.DposContext == nil {
		return errors.New("missing required field 'dposContext' for Header")
	}
	
	hPtr.Time = (*big.Int)(decPtr.Time)
	if decPtr.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	hPtr.Extra = *decPtr.Extra
	if decPtr.MixDigest == nil {
		return errors.New("missing required field 'mixHash' for Header")
	}
	hPtr.MixDigest = *decPtr.MixDigest
	if decPtr.Nonce == nil {
		return errors.New("missing required field 'nonce' for Header")
	}
	hPtr.Nonce = *decPtr.Nonce
	hPtr.DposContext = decPtr.DposContext
	
	
	return nil
}
