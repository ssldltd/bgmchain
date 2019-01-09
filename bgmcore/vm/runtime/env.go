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

package runtime

import (
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgmcommon"
	
)

func NewEnv(cfg *Config) *vmPtr.EVM {
	context := vmPtr.Context{
		CanTransfer: bgmcore.CanTransfer,
		Transfer:    bgmcore.Transfer,
		GetHash:     func(uint64) bgmcommon.Hash { return bgmcommon.Hash{} },
		BlockNumber: cfg.BlockNumber,
		Time:        cfg.Time,
		Difficulty:  cfg.Difficulty,
		GasLimit:    new(big.Int).SetUint64(cfg.GasLimit),
		GasPrice:    cfg.GasPrice,
		Origin:      cfg.Origin,
		Coinbase:    cfg.Coinbase,
		
	}

	return vmPtr.NewEVM(context, cfg.State, cfg.ChainConfig, cfg.EVMConfig)
}
