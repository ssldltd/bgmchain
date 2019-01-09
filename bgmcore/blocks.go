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

import "github.com/ssldltd/bgmchain/bgmcommon"

// BadHashes represent a set of manually tracked bad hashes (usually hard forks)
var BadHashes = map[bgmcommon.Hash]bool{
	bgmcommon.HexToHash("05bef30ef572270f654746da22639a7a0c97dd97a7050b9e252391996aaeb689"): true,
	bgmcommon.HexToHash("7d05d08cbc596a2e5e4f13b80a743e53e09221b5323c3a61946b20873e58583f"): true,
}
