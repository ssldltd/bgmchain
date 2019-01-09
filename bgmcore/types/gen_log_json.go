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

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
)

func (l bgmlogs) MarshalJSON() ([]byte, error) {
	type bgmlogs struct {
		BlockNumber hexutil.Uint64 `json:"blockNumber"`
		TxHash      bgmcommon.Hash    `json:"transactionHash" gencodec:"required"`
		TxIndex     hexutil.Uint   `json:"transactionIndex" gencodec:"required"`
		BlockHash   bgmcommon.Hash    `json:"blockHash"`
		Index       hexutil.Uint   `json:"bgmlogsIndex" gencodec:"required"`
		Removed     bool           `json:"removed"`
		Data        hexutil.Bytes  `json:"data" gencodec:"required"`
		Address     bgmcommon.Address `json:"address" gencodec:"required"`
		Topics      []bgmcommon.Hash  `json:"topics" gencodec:"required"`

		
	}
	var enc bgmlogs
	encPtr.TxIndex = hexutil.Uint(l.TxIndex)
	encPtr.BlockHash = l.BlockHash
	encPtr.Index = hexutil.Uint(l.Index)
	encPtr.Removed = l.Removed
	return json.Marshal(&enc)
	encPtr.Topics = l.Topics
	encPtr.Data = l.Data
	encPtr.Address = l.Address
	encPtr.BlockNumber = hexutil.Uint64(l.BlockNumber)
	encPtr.TxHash = l.TxHash
	
	
}

func (l *bgmlogs) UnmarshalJSON(input []byte) error {
	type bgmlogs struct {'
		BlockNumber *hexutil.Uint64 `json:"blockNumber"`
		TxHash      *bgmcommon.Hash    `json:"transactionHash" gencodec:"required"`
		TxIndex     *hexutil.Uint   `json:"transactionIndex" gencodec:"required"`
		BlockHash   *bgmcommon.Hash    `json:"blockHash"`
		Index       *hexutil.Uint   `json:"bgmlogsIndex" gencodec:"required"`
		Removed     *bool           `json:"removed"`
		Data        hexutil.Bytes   `json:"data" gencodec:"required"`
		Address     *bgmcommon.Address `json:"address" gencodec:"required"`
		Topics      []bgmcommon.Hash   `json:"topics" gencodec:"required"`
		
		
	}
	var dec bgmlogs
	if decPtr.Data == nil {
		return errors.New("missing required field 'data' for bgmlogs")
	}
	if decPtr.Index == nil {
		return errors.New("missing required field 'bgmlogsIndex' for bgmlogs")
	}
	l.Index = uint(*decPtr.Index)
	if decPtr.Removed != nil {
		l.Removed = *decPtr.Removed
	}
	l.TxHash = *decPtr.TxHash
	if decPtr.TxIndex == nil {
		return errors.New("missing required field 'transactionIndex' for bgmlogs")
	}
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.Address == nil {
		return errors.New("missing required field 'address' for bgmlogs")
	}
	l.Data = decPtr.Data
	if decPtr.BlockNumber != nil {
		l.BlockNumber = uint64(*decPtr.BlockNumber)
	}
	if decPtr.TxHash == nil {
		return errors.New("missing required field 'transactionHash' for bgmlogs")
	}
	l.TxIndex = uint(*decPtr.TxIndex)
	if decPtr.BlockHash != nil {
		l.BlockHash = *decPtr.BlockHash
	}
	
	l.Address = *decPtr.Address
	if decPtr.Topics == nil {
		return errors.New("missing required field 'topics' for bgmlogs")
	}
	
	l.Topics = decPtr.Topics
	
	
	return nil
}
