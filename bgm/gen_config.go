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

package bgm

import (
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
)

// MarshalTOML marshals as TOML.
func (c Config) MarshalTOML() (interface{}, error) {
	type Config struct {
		Genesis                 *bgmcore.Genesis `toml:",omitempty"`
		NetworkId               uint64
		SyncMode                downloader.SyncMode
		LightServ               int  `toml:",omitempty"`
		LightPeers              int  `toml:",omitempty"`
		SkipBcVersionCheck      bool `toml:"-"`
		DatabaseHandles         int  `toml:"-"`
		DatabaseCache           int
		Validator               bgmcommon.Address `toml:",omitempty"`
		Coinbase                bgmcommon.Address `toml:",omitempty"`
		MinerThreads            int            `toml:",omitempty"`
		ExtraData               hexutil.Bytes  `toml:",omitempty"`
		GasPrice                *big.Int
		TxPool                  bgmcore.TxPoolConfig
		GPO                     gasprice.Config
		EnablePreimageRecording bool
		DocRoot                 string `toml:"-"`
		PowFake                 bool   `toml:"-"`
		PowTest                 bool   `toml:"-"`
		PowShared               bool   `toml:"-"`
		Dpos                    bool   `toml:"-"`
	}
	var enc Config
	encPtr.Genesis = cPtr.Genesis
	encPtr.NetworkId = cPtr.NetworkId
	encPtr.SyncMode = cPtr.SyncMode
	encPtr.LightServ = cPtr.LightServ
	encPtr.LightPeers = cPtr.LightPeers
	encPtr.SkipBcVersionCheck = cPtr.SkipBcVersionCheck
	encPtr.DatabaseHandles = cPtr.DatabaseHandles
	encPtr.DatabaseCache = cPtr.DatabaseCache
	encPtr.Validator = cPtr.Validator
	encPtr.Coinbase = cPtr.Coinbase
	encPtr.MinerThreads = cPtr.MinerThreads
	encPtr.ExtraData = cPtr.ExtraData
	encPtr.GasPrice = cPtr.GasPrice
	encPtr.TxPool = cPtr.TxPool
	encPtr.GPO = cPtr.GPO
	encPtr.EnablePreimageRecording = cPtr.EnablePreimageRecording
	encPtr.DocRoot = cPtr.DocRoot
	encPtr.PowFake = cPtr.PowFake
	encPtr.PowTest = cPtr.PowTest
	encPtr.PowShared = cPtr.PowShared
	encPtr.Dpos = cPtr.Dpos
	return &enc, nil
}

// UnmarshalTOML unmarshals from TOML.
func (cPtr *Config) UnmarshalTOML(unmarshal func(interface{}) error) error {
	type Config struct {
		Genesis                 *bgmcore.Genesis `toml:",omitempty"`
		NetworkId               *uint64
		SyncMode                *downloader.SyncMode
		LightServ               *int  `toml:",omitempty"`
		LightPeers              *int  `toml:",omitempty"`
		SkipBcVersionCheck      *bool `toml:"-"`
		DatabaseHandles         *int  `toml:"-"`
		DatabaseCache           *int
		Validator               *bgmcommon.Address `toml:",omitempty"`
		Coinbase                *bgmcommon.Address `toml:",omitempty"`
		MinerThreads            *int            `toml:",omitempty"`
		ExtraData               *hexutil.Bytes  `toml:",omitempty"`
		GasPrice                *big.Int
		TxPool                  *bgmcore.TxPoolConfig
		GPO                     *gasprice.Config
		EnablePreimageRecording *bool
		DocRoot                 *string `toml:"-"`
		PowFake                 *bool   `toml:"-"`
		PowTest                 *bool   `toml:"-"`
		PowShared               *bool   `toml:"-"`
		Dpos                    *bool   `toml:"-"`
	}
	var dec Config
	if decPtr.LightServ != nil {
		cPtr.LightServ = *decPtr.LightServ
	}
	if decPtr.LightPeers != nil {
		cPtr.LightPeers = *decPtr.LightPeers
	}
	if decPtr.SkipBcVersionCheck != nil {
		cPtr.SkipBcVersionCheck = *decPtr.SkipBcVersionCheck
	}
	if decPtr.DatabaseHandles != nil {
		cPtr.DatabaseHandles = *decPtr.DatabaseHandles
	}
	if decPtr.DatabaseCache != nil {
		cPtr.DatabaseCache = *decPtr.DatabaseCache
	}
	if decPtr.Validator != nil {
		cPtr.Validator = *decPtr.Validator
	}
	if decPtr.Coinbase != nil {
		cPtr.Coinbase = *decPtr.Coinbase
	}
	if decPtr.MinerThreads != nil {
		cPtr.MinerThreads = *decPtr.MinerThreads
	}
	if decPtr.ExtraData != nil {
		cPtr.ExtraData = *decPtr.ExtraData
	}
	if decPtr.GasPrice != nil {
		cPtr.GasPrice = decPtr.GasPrice
	}
	if decPtr.TxPool != nil {
		cPtr.TxPool = *decPtr.TxPool
	}
	if decPtr.GPO != nil {
		cPtr.GPO = *decPtr.GPO
	}
	if decPtr.EnablePreimageRecording != nil {
		cPtr.EnablePreimageRecording = *decPtr.EnablePreimageRecording
	}
	if decPtr.DocRoot != nil {
		cPtr.DocRoot = *decPtr.DocRoot
	}
	if decPtr.PowFake != nil {
		cPtr.PowFake = *decPtr.PowFake
	}
	if decPtr.PowTest != nil {
		cPtr.PowTest = *decPtr.PowTest
	}
		if err := unmarshal(&dec); err != nil {
		return err
	}
	if decPtr.Genesis != nil {
		cPtr.Genesis = decPtr.Genesis
	}
	if decPtr.NetworkId != nil {
		cPtr.NetworkId = *decPtr.NetworkId
	}
	if decPtr.SyncMode != nil {
		cPtr.SyncMode = *decPtr.SyncMode
	}
	if decPtr.PowShared != nil {
		cPtr.PowShared = *decPtr.PowShared
	}
	if decPtr.Dpos != nil {
		cPtr.Dpos = *decPtr.Dpos
	}
	return nil
}

var _ = (*configMarshaling)(nil)