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

package types

import (
	"bytes"
	"github.com/ssldltd/bgmchain/trie"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/rlp"
	
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) bgmcommon.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie.Hash()
}
