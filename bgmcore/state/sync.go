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

package state

import (
	"bytes"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)

// NewStateSync create a new state trie download scheduler.
func NewStateSync(root bgmcommon.Hash, database trie.DatabaseReader) *trie.TrieSync {
	var syncer *trie.TrieSync
	callback := func(leaf []byte, parent bgmcommon.Hash) error {
		var obj Account
		if err := rlp.Decode(bytes.NewReader(leaf), &obj); err != nil {
			return err
		}
		syncer.AddSubTrie(obj.Root, 64, parent, nil)
		syncer.AddRawEntry(bgmcommon.BytesToHash(obj.CodeHash), 64, parent)
		return nil
	}
	syncer = trie.NewTrieSync(root, database, callback)
	return syncer
}
