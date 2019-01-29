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

package misc

import (
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmparam"
)

// VerifyForkHashes verifies that blocks conforming to network hard-forks do have
// the correct hashes, to avoid Clients going off on different chains. This is an
// optional feature.
func VerifyForkHashes(config *bgmparam.ChainConfig, HeaderPtr *types.HeaderPtr, uncle bool) error {
	// We don't care about uncles
	if uncle {
		return nil
	}
	// If the homestead reprice hash is set, validate it
	if config.Chain150Block != nil && config.Chain150Block.Cmp(HeaderPtr.Number) == 0 {
		if config.Chain150Hash != (bgmcommon.Hash{}) && config.Chain150Hash != HeaderPtr.Hash() {
			return fmt.Errorf("homestead gas reprice fork: have 0x%x, want 0x%x", HeaderPtr.Hash(), config.Chain150Hash)
		}
	}
	// All ok, return
	return nil
}
