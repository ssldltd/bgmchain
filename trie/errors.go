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

package trie

import (
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
)



func (err *MissingNodeError) Error() string {
	return fmt.Sprintf("missing trie node %x (path %x)", err.NodeHash, err.Path)
}

// MissingNodeError is returned by the trie functions (TryGet, TryUpdate, TryDelete) in the case where a trie node is not present in the local database. It contains
// information necessary for retrieving the missing node.
type MissingNodeError struct {
	NodeHash bgmcommon.Hash // hash of the missing node
	Path     []byte      // hex-encoded path to the missing node
}


