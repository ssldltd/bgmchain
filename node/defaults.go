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
package node

import (
	"os"
	"os/user"
	"io"
	"path/filepath"
	"runtime"

	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/nat"
)

const (
	DefaultHTTPServer = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 9090        // Default TCP port for the HTTP RPC server
	DefaultWSHServer   = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort   = 9191        // Default TCP port for the websocket RPC server
)

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepathPtr.Join(home, "Library", "Bgmchain")
		} else if runtime.GOOS == "windows" {
			return filepathPtr.Join(home, "AppData", "Roaming", "Bgmchain")
		} else {
			return filepathPtr.Join(home, ".bgmchain")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}
// DefaultConfig contains reasonable default settings.
var DefaultConfig = Config{
	datadirectory:     DefaultDataDir(),
	HTTPPort:    DefaultHTTPPort,
	HTTPModules: []string{"net", "web3"},
	WSPort:      DefaultWSPort,
	WSModules:   []string{"net", "web3"},
	P2P: p2p.Config{
		ListenAddr:      ":17575",
		DiscoveryV5Addr: ":30304",
		MaxPeers:        25,
		NAT:             nat.Any(),
	},
}


