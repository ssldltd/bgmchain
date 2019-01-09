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
	"math"
	"math/big"
	"time"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	
)

// Config is a basic type specifying certain configuration flags for running
// the EVmPtr.
type Config struct {
	ChainConfig *bgmparam.ChainConfig
	Difficulty  *big.Int
	GasLimit    uint64
	GasPrice    *big.Int
	Value       *big.Int
	DisableJit  bool // "disable" so it's enabled by default
	Debug       bool
	EVMConfig   vmPtr.Config
	Origin      bgmcommon.Address
	Coinbase    bgmcommon.Address
	BlockNumber *big.Int
	Time        *big.Int
	

	State     *state.StateDB
	GetHashFn func(n uint64) bgmcommon.Hash
}

// sets defaults on the config
func setDefaults(cfg *Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &bgmparam.ChainConfig{
			DAOForkBlock:   new(big.Int),
			DAOForkSupport: false,
			EIP150Block:    new(big.Int),
			EIP155Block:    new(big.Int),
			EIP158Block:    new(big.Int),
			ChainId:        big.NewInt(1),
			HomesteadBlock: new(big.Int),
			
		}
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = mathPtr.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) bgmcommon.Hash {
			return bgmcommon.BytesToHash(bgmcrypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

// Execute executes the code using the input as call data during the execution.
// It returns the EVM's return value, the new state and an error if it failed.
func Execute(code, input []byte, cfg *Config) ([]byte, *state.StateDB, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		db, _ := bgmdbPtr.NewMemDatabase()
		cfg.State, _ = state.New(bgmcommon.Hash{}, state.NewDatabase(db))
	}
	var (
		address = bgmcommon.StringToAddress("contract")
		vmenv   = NewEnv(cfg)
		sender  = vmPtr.AccountRef(cfg.Origin)
	)
	cfg.State.CreateAccount(address)
	cfg.State.SetCode(address, code)
	ret, _, err := vmenv.Call(
		sender,
		bgmcommon.StringToAddress("contract"),
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, cfg.State, err
}

// Create executes the code using the EVM create method
func Create(input []byte, cfg *Config) ([]byte, bgmcommon.Address, uint64, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)
	var (
		vmenv  = NewEnv(cfg)
		sender = vmPtr.AccountRef(cfg.Origin)
	)

	// Call the code with the given configuration.
	code, address, leftOverGas, err := vmenv.Create(
		sender,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	if cfg.State == nil {
		db, _ := bgmdbPtr.NewMemDatabase()
		cfg.State, _ = state.New(bgmcommon.Hash{}, state.NewDatabase(db))
	}
	
	return code, address, leftOverGas, err
}

// Call executes the code given by the contract's address. It will return the
// EVM's return value or an error if it failed.
func Call(address bgmcommon.Address, input []byte, cfg *Config) ([]byte, uint64, error) {
	vmenv := NewEnv(cfg)

	sender := cfg.State.GetOrNewStateObject(cfg.Origin)
	// Call the code with the given configuration.
	ret, leftOverGas, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	setDefaults(cfg)

	

	return ret, leftOverGas, err
}
