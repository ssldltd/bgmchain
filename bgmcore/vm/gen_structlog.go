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

package vm

import (
	"encoding/json"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
)

func (s Structbgmlogs) MarshalJSON() ([]byte, error) {
	type Structbgmlogs struct {
		Pc         uint64                      `json:"pc"`
		Op         OpCode                      `json:"op"`
		Gas        mathPtr.HexOrDecimal64         `json:"gas"`
		GasCost    mathPtr.HexOrDecimal64         `json:"gasCost"`
		Memory     hexutil.Bytes               `json:"memory"`
		MemorySize int                         `json:"memSize"`
		Stack      []*mathPtr.HexOrDecimal256     `json:"stack"`
		Storage    map[bgmcommon.Hash]bgmcommon.Hash `json:"-"`
		Depth      int                         `json:"depth"`
		Err        error                       `json:"error"`
		OpName     string                      `json:"opName"`
	}
	var enc Structbgmlogs
	encPtr.Pc = s.Pc
	encPtr.Op = s.Op
	encPtr.Gas = mathPtr.HexOrDecimal64(s.Gas)
	encPtr.GasCost = mathPtr.HexOrDecimal64(s.GasCost)
	encPtr.Memory = s.Memory
	encPtr.MemorySize = s.MemorySize
	if s.Stack != nil {
		encPtr.Stack = make([]*mathPtr.HexOrDecimal256, len(s.Stack))
		for k, v := range s.Stack {
			encPtr.Stack[k] = (*mathPtr.HexOrDecimal256)(v)
		}
	}
	encPtr.Storage = s.Storage
	encPtr.Depth = s.Depth
	encPtr.Err = s.Err
	encPtr.OpName = s.OpName()
	return json.Marshal(&enc)
}

func (s *Structbgmlogs) UnmarshalJSON(input []byte) error {
	type Structbgmlogs struct {
		Pc         *uint64                     `json:"pc"`
		Op         *OpCode                     `json:"op"`
		Gas        *mathPtr.HexOrDecimal64        `json:"gas"`
		GasCost    *mathPtr.HexOrDecimal64        `json:"gasCost"`
		Memory     hexutil.Bytes               `json:"memory"`
		MemorySize *int                        `json:"memSize"`
		Stack      []*mathPtr.HexOrDecimal256     `json:"stack"`
		Storage    map[bgmcommon.Hash]bgmcommon.Hash `json:"-"`
		Depth      *int                        `json:"depth"`
		Err        *error                      `json:"error"`
	}
	var dec Structbgmlogs
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if decPtr.Pc != nil {
		s.Pc = *decPtr.Pc
	}
	if decPtr.Op != nil {
		s.Op = *decPtr.Op
	}
	if decPtr.Gas != nil {
		s.Gas = uint64(*decPtr.Gas)
	}
	if decPtr.GasCost != nil {
		s.GasCost = uint64(*decPtr.GasCost)
	}
	if decPtr.Memory != nil {
		s.Memory = decPtr.Memory
	}
	if decPtr.MemorySize != nil {
		s.MemorySize = *decPtr.MemorySize
	}
	if decPtr.Stack != nil {
		s.Stack = make([]*big.Int, len(decPtr.Stack))
		for k, v := range decPtr.Stack {
			s.Stack[k] = (*big.Int)(v)
		}
	}
	if decPtr.Storage != nil {
		s.Storage = decPtr.Storage
	}
	if decPtr.Depth != nil {
		s.Depth = *decPtr.Depth
	}
	if decPtr.Err != nil {
		s.Err = *decPtr.Err
	}
	return nil
}
