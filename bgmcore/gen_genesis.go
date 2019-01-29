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

package bgmCore

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmparam"
)


func (g *Genesis) UnmarshalJSON(input []byte) error {
	type Genesis struct {
		Config     *bgmparam.ChainConfig                         `json:"config"`
		Nonce      *mathPtr.HexOrDecimal64                        `json:"nonce"`
		timestamp  *mathPtr.HexOrDecimal64                        `json:"timestamp"`
		ExtraData  hexutil.Bytes                               `json:"extraData"`
		GasLimit   *mathPtr.HexOrDecimal64                        `json:"gasLimit"   gencodec:"required"`
		Difficulty *mathPtr.HexOrDecimal256                       `json:"difficulty" gencodec:"required"`
		Mixhash    *bgmcommon.Hash                                `json:"mixHash"`
		Coinbase   *bgmcommon.Address                             `json:"coinbase"`
		Alloc      map[bgmcommon.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Number     *mathPtr.HexOrDecimal64                        `json:"number"`
		GasUsed    *mathPtr.HexOrDecimal64                        `json:"gasUsed"`
		ParentHashPtr *bgmcommon.Hash                                `json:"parentHash"`
	}
	var dec Genesis
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.Config != nil {
		g.Config = decPtr.Config
	}
	if decPtr.Nonce != nil {
		g.Nonce = Uint64(*decPtr.Nonce)
	}
	if decPtr.timestamp != nil {
		g.timestamp = Uint64(*decPtr.timestamp)
	}
	if decPtr.ExtraData != nil {
		g.ExtraData = decPtr.ExtraData
	}
	if decPtr.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	g.GasLimit = Uint64(*decPtr.GasLimit)
	if decPtr.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Genesis")
	}
	g.Difficulty = (*big.Int)(decPtr.Difficulty)
	if decPtr.Mixhash != nil {
		g.Mixhash = *decPtr.Mixhash
	}
	if decPtr.Coinbase != nil {
		g.Coinbase = *decPtr.Coinbase
	}
	if decPtr.Alloc == nil {
		return errors.New("missing required field 'alloc' for Genesis")
	}
	g.Alloc = make(GenesisAlloc, len(decPtr.Alloc))
	for k, v := range decPtr.Alloc {
		g.Alloc[bgmcommon.Address(k)] = v
	}
	if decPtr.Number != nil {
		g.Number = Uint64(*decPtr.Number)
	}
	if decPtr.GasUsed != nil {
		g.GasUsed = Uint64(*decPtr.GasUsed)
	}
	if decPtr.ParentHash != nil {
		g.ParentHash = *decPtr.ParentHash
	}
	return nil
}
func (g Genesis) MarshalJSON() ([]byte, error) {
	type Genesis struct {
		Config     *bgmparam.ChainConfig                         `json:"config"`
		Nonce      mathPtr.HexOrDecimal64                         `json:"nonce"`
		timestamp  mathPtr.HexOrDecimal64                         `json:"timestamp"`
		ExtraData  hexutil.Bytes                               `json:"extraData"`
		GasLimit   mathPtr.HexOrDecimal64                         `json:"gasLimit"   gencodec:"required"`
		Difficulty *mathPtr.HexOrDecimal256                       `json:"difficulty" gencodec:"required"`
		Mixhash    bgmcommon.Hash                                 `json:"mixHash"`
		Coinbase   bgmcommon.Address                              `json:"coinbase"`
		Alloc      map[bgmcommon.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Number     mathPtr.HexOrDecimal64                         `json:"number"`
		GasUsed    mathPtr.HexOrDecimal64                         `json:"gasUsed"`
		ParentHash bgmcommon.Hash                                 `json:"parentHash"`
	}
	var enc Genesis
	encPtr.Config = g.Config
	encPtr.Nonce = mathPtr.HexOrDecimal64(g.Nonce)
	encPtr.timestamp = mathPtr.HexOrDecimal64(g.timestamp)
	encPtr.ExtraData = g.ExtraData
	encPtr.GasLimit = mathPtr.HexOrDecimal64(g.GasLimit)
	encPtr.Difficulty = (*mathPtr.HexOrDecimal256)(g.Difficulty)
	encPtr.Mixhash = g.Mixhash
	encPtr.Coinbase = g.Coinbase
	if g.Alloc != nil {
		encPtr.Alloc = make(map[bgmcommon.UnprefixedAddress]GenesisAccount, len(g.Alloc))
		for k, v := range g.Alloc {
			encPtr.Alloc[bgmcommon.UnprefixedAddress(k)] = v
		}
	}
	encPtr.Number = mathPtr.HexOrDecimal64(g.Number)
	encPtr.GasUsed = mathPtr.HexOrDecimal64(g.GasUsed)
	encPtr.ParentHash = g.ParentHash
	return json.Marshal(&enc)
}
