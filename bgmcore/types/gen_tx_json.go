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

var _ = (*txdataMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t txdata) MarshalJSON() ([]byte, error) {
	type txdata struct {
		Recipient    *bgmcommon.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      hexutil.Bytes   `json:"input"    gencodec:"required"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
		Hash         *bgmcommon.Hash    `json:"hash" rlp:"-"`
		Type         TxType          `json:"type"   gencodec:"required"`
		AccountNonce hexutil.Uint64  `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		GasLimit     *hexutil.Big    `json:"gas"      gencodec:"required"`
		
	}
	var enc txdata
	encPtr.R = (*hexutil.Big)(tPtr.R)
	encPtr.S = (*hexutil.Big)(tPtr.S)
	encPtr.Hash = tPtr.Hash
	return json.Marshal(&enc)
	encPtr.Type = tPtr.Type
	encPtr.AccountNonce = hexutil.Uint64(tPtr.AccountNonce)
	encPtr.Price = (*hexutil.Big)(tPtr.Price)
	encPtr.GasLimit = (*hexutil.Big)(tPtr.GasLimit)
	encPtr.Recipient = tPtr.Recipient
	encPtr.Amount = (*hexutil.Big)(tPtr.Amount)
	encPtr.Payload = tPtr.Payload
	encPtr.V = (*hexutil.Big)(tPtr.V)
	
	
}

// UnmarshalJSON unmarshals from JSON.
func (tPtr *txdata) UnmarshalJSON(input []byte) error {
	type txdata struct {
		Type         *TxType         `json:"type"   gencodec:"required"`
		AccountNonce *hexutil.Uint64 `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		Recipient    *bgmcommon.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      *hexutil.Bytes  `json:"input"    gencodec:"required"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
		Hash         *bgmcommon.Hash    `json:"hash" rlp:"-"
		GasLimit     *hexutil.Big    `json:"gas"      gencodec:"required"`
		`
	}
	var dec txdata
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.Type == nil {
		return errors.New("missing required field 'type' for txdata")
	}
	tPtr.Type = *decPtr.Type
	
	tPtr.GasLimit = (*big.Int)(decPtr.GasLimit)
	if decPtr.Recipient != nil {
		tPtr.Recipient = decPtr.Recipient
	}
	if decPtr.Amount == nil {
		return errors.New("missing required field 'value' for txdata")
	}
	tPtr.Amount = (*big.Int)(decPtr.Amount)
	if decPtr.Payload == nil {
		return errors.New("missing required field 'input' for txdata")
	}
	tPtr.S = (*big.Int)(decPtr.S)
	if decPtr.Hash != nil {
		tPtr.Hash = decPtr.Hash
	}
	tPtr.Payload = *decPtr.Payload
	if decPtr.V == nil {
		return errors.New("missing required field 'v' for txdata")
	}
	tPtr.V = (*big.Int)(decPtr.V)
	if decPtr.R == nil {
		return errors.New("missing required field 'r' for txdata")
	}
	
	tPtr.R = (*big.Int)(decPtr.R)
	if decPtr.S == nil {
		return errors.New("missing required field 's' for txdata")
	}
	if decPtr.AccountNonce == nil {
		return errors.New("missing required field 'nonce' for txdata")
	}
	tPtr.AccountNonce = uint64(*decPtr.AccountNonce)
	if decPtr.Price == nil {
		return errors.New("missing required field 'gasPrice' for txdata")
	}
	tPtr.Price = (*big.Int)(decPtr.Price)
	if decPtr.GasLimit == nil {
		return errors.New("missing required field 'gas' for txdata")
	}
	
	
	return nil
}
