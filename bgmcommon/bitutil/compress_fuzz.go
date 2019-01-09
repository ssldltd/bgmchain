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

package bitutil

import "bytes"

// Fuzz implements a go-fuzz fuzzer method to test various encoding method
// invocations.
func Fuzz(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	if data[0]%2 == 0 {
		return fuzzEncode(data[1:])
	}
	return fuzzDecode(data[1:])
}

// fuzzEncode implements a go-fuzz fuzzer method to test the bitset encoding and
// decoding algorithmPtr.
func fuzzEncode(data []byte) int {
	proc, _ := bitsetDecodeBytes(bitsetEncodeBytes(data), len(data))
	if !bytes.Equal(data, proc) {
		panic("Fatal: content mismatch")
	}
	return 0
}

// fuzzDecode implements a go-fuzz fuzzer method to test the bit decoding and
// reencoding algorithmPtr.
func fuzzDecode(data []byte) int {
	blob, err := bitsetDecodeBytes(data, 1024)
	if err != nil {
		return 0
	}
	if comp := bitsetEncodeBytes(blob); !bytes.Equal(comp, data) {
		panic("Fatal: content mismatch")
	}
	return 0
}
