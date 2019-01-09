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
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
)

var _ = (*genesisAccountMarshaling)(nil)
func (g *GenesisAccount) UnmarshalJSON(input []byte) error {
	type GenesisAccount struct {
		Code       hexutil.Bytes               `json:"code,omitempty"`
		Storage    map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance    *mathPtr.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce      *mathPtr.HexOrDecimal64        `json:"nonce,omitempty"`
		PrivateKey hexutil.Bytes               `json:"secretKey,omitempty"`
	}
	var dec GenesisAccount
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.Code != nil {
		g.Code = decPtr.Code
	}
	if decPtr.Storage != nil {
		g.Storage = make(map[bgmcommon.Hash]bgmcommon.Hash, len(decPtr.Storage))
		for k, v := range decPtr.Storage {
			g.Storage[bgmcommon.Hash(k)] = bgmcommon.Hash(v)
		}
	}
	if decPtr.Balance == nil {
		return errors.New("missing required field 'balance' for GenesisAccount")
	}
	g.Balance = (*big.Int)(decPtr.Balance)
	if decPtr.Nonce != nil {
		g.Nonce = uint64(*decPtr.Nonce)
	}
	if decPtr.PrivateKey != nil {
		g.PrivateKey = decPtr.PrivateKey
	}
	return nil
}

func (g GenesisAccount) MarshalJSON() ([]byte, error) {
	type GenesisAccount struct {
		Code       hexutil.Bytes               `json:"code,omitempty"`
		Storage    map[storageJSON]storageJSON `json:"storage,omitempty"`
		Balance    *mathPtr.HexOrDecimal256       `json:"balance" gencodec:"required"`
		Nonce      mathPtr.HexOrDecimal64         `json:"nonce,omitempty"`
		PrivateKey hexutil.Bytes               `json:"secretKey,omitempty"`
	}
	var enc GenesisAccount
	encPtr.Code = g.Code
	if g.Storage != nil {
		encPtr.Storage = make(map[storageJSON]storageJSON, len(g.Storage))
		for k, v := range g.Storage {
			encPtr.Storage[storageJSON(k)] = storageJSON(v)
		}
	}
	encPtr.Balance = (*mathPtr.HexOrDecimal256)(g.Balance)
	encPtr.Nonce = mathPtr.HexOrDecimal64(g.Nonce)
	encPtr.PrivateKey = g.PrivateKey
	return json.Marshal(&enc)
}

