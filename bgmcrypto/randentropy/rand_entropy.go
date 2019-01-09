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

package randentropy

import (
	crand "bgmcrypto/rand"
	"io"
)

var Reader io.Reader = &randEntropy{}

type randEntropy struct {
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("Fatal: reading from bgmcrypto/rand failed: " + err.Error())
	}
	return mainBuff
}

func (*randEntropy) Read(bytes []byte) (n int, err error) {
	readBytes := GetEntropyCSPRNG(len(bytes))
	copy(bytes, readBytes)
	return len(bytes), nil
}
