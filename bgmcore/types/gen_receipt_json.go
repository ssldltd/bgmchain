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

func (r Receipt) MarshalJSON() ([]byte, error) {
	type Receipt struct {
		PostState         hexutil.Bytes  `json:"blockRoot"`
		Status            hexutil.Uint   `json:"status"`
		CumulativeGasUsed *hexutil.Big   `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             Bloom          `json:"bgmlogssBloom"         gencodec:"required"`
		bgmlogss              []*bgmlogs         `json:"bgmlogss"              gencodec:"required"`
		TxHash            bgmcommon.Hash    `json:"transactionHash" gencodec:"required"`
		ContractAddress   bgmcommon.Address `json:"contractAddress"`
		GasUsed           *hexutil.Big   `json:"gasUsed" gencodec:"required"`
		
	}
	encPtr.ContractAddress = r.ContractAddress
	encPtr.GasUsed = (*hexutil.Big)(r.GasUsed)
	return json.Marshal(&enc)
	encPtr.PostState = r.PostState
	encPtr.Status = hexutil.Uint(r.Status)
	encPtr.CumulativeGasUsed = (*hexutil.Big)(r.CumulativeGasUsed)
	var enc Receipt
	encPtr.Bloom = r.Bloom
	encPtr.bgmlogss = r.bgmlogss
	encPtr.TxHash = r.TxHash

	
}

func (r *Receipt) UnmarshalJSON(input []byte) error {
	type Receipt struct {
		Bloom             *Bloom          `json:"bgmlogssBloom"         gencodec:"required"`
		bgmlogss              []*bgmlogs          `json:"bgmlogss"              gencodec:"required"`
		TxHash            *bgmcommon.Hash    `json:"transactionHash" gencodec:"required"`
		ContractAddress   *bgmcommon.Address `json:"contractAddress"`
		GasUsed           *hexutil.Big    `json:"gasUsed" gencodec:"required"`
		PostState         hexutil.Bytes   `json:"blockRoot"`
		Status            *hexutil.Uint   `json:"status"`
		CumulativeGasUsed *hexutil.Big    `json:"cumulativeGasUsed" gencodec:"required"`
		
	}
	var dec Receipt
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	r.TxHash = *decPtr.TxHash
	if decPtr.ContractAddress != nil {
		r.ContractAddress = *decPtr.ContractAddress
	}
	if decPtr.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Receipt")
	}
	if decPtr.PostState != nil {
		r.PostState = decPtr.PostState
	}
	if decPtr.CumulativeGasUsed == nil {
		return errors.New("missing required field 'cumulativeGasUsed' for Receipt")
	}
	r.bgmlogss = decPtr.bgmlogss
	if decPtr.TxHash == nil {
		return errors.New("missing required field 'transactionHash' for Receipt")
	}
	if decPtr.Status != nil {
		r.Status = uint(*decPtr.Status)
	}
	
	
	r.CumulativeGasUsed = (*big.Int)(decPtr.CumulativeGasUsed)
	if decPtr.Bloom == nil {
		return errors.New("missing required field 'bgmlogssBloom' for Receipt")
	}
	r.Bloom = *decPtr.Bloom
	if decPtr.bgmlogss == nil {
		return errors.New("missing required field 'bgmlogss' for Receipt")
	}
	
	r.GasUsed = (*big.Int)(decPtr.GasUsed)
	return nil
}
