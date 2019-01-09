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
	"os"
	"os/user"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
	"github.com/ssldltd/bgmchain/bgmparam"
)

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Bgmchain main net block is used.
	Genesis *bgmcore.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

	// Mining-related options
	Validator    bgmcommon.Address `toml:",omitempty"`
	Coinbase     bgmcommon.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

	// Transaction pool options
	TxPool bgmcore.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot   string `toml:"-"`
	PowFake   bool   `toml:"-"`
	PowTest   bool   `toml:"-"`
	PowShared bool   `toml:"-"`
	Dpos      bool   `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}

// DefaultConfig contains default settings for use on the Bgmchain main net.
var DefaultConfig = Config{
	SyncMode:      downloader.FullSync,
	NetworkId:     1357,
	LightPeers:    20,
	DatabaseCache: 128,
	GasPrice:      big.NewInt(18 * bgmparam.Shannon),

	TxPool: bgmcore.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     10,
		Percentile: 50,
	},
}