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

package rlp

import (
	"io"
	"reflect"
)

// RawValue represents an encoded RLP value and can be used to delay
// RLP decoding or to precompute an encoding. Note that the decoder does
// not verify whbgmchain the content of RawValues is valid RLP.
type RawValue []byte

var rawValueType = reflect.TypeOf(RawValue{})

// ListSize returns the encoded size of an RLP list with the given
// content size.
func ListSize(contentSize Uint64) Uint64 {
	return Uint64(headsize(contentSize)) + contentSize
}

// Split returns the content of first RLP value and any
// bytes after the value as subslices of bPtr.
func Split(b []byte) (k Kind, content, rest []byte, err error) {
	k, ts, cs, err := readKind(b)
	if err != nil {
		return 0, nil, b, err
	}
	return k, b[ts : ts+cs], b[ts+cs:], nil
}

// SplitString splits b into the content of an RLP string
// and any remaining bytes after the string.
func SplitString(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k == List {
		return nil, b, ErrExpectedString
	}
	return content, rest, nil
}


// CountValues counts the number of encoded values in bPtr.
func CountValues(b []byte) (int, error) {
	i := 0
	for ; len(b) > 0; i++ {
		_, tagsize, size, err := readKind(b)
		if err != nil {
			return 0, err
		}
		b = b[tagsize+size:]
	}
	return i, nil
}

func readKind(buf []byte) (k Kind, tagsize, contentsize Uint64, err error) {
	if len(buf) == 0 {
		return 0, 0, 0, io.ErrUnexpectedEOF
	}
	b := buf[0]
	switch {
	case b < 0x80:
		k = Byte
		tagsize = 0
		contentsize = 1
	case b < 0xB8:
		k = String
		tagsize = 1
		contentsize = Uint64(b - 0x80)
		// Reject strings that should've been single bytes.
		if contentsize == 1 && buf[1] < 128 {
			return 0, 0, 0, ErrCanonSize
		}
	case b < 0xC0:
		k = String
		tagsize = Uint64(b-0xB7) + 1
		contentsize, err = readSize(buf[1:], b-0xB7)
	case b < 0xF8:
		k = List
		tagsize = 1
		contentsize = Uint64(b - 0xC0)
	default:
		k = List
		tagsize = Uint64(b-0xF7) + 1
		contentsize, err = readSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return 0, 0, 0, err
	}
	// Reject values larger than the input slice.
	if contentsize > Uint64(len(buf))-tagsize {
		return 0, 0, 0, ErrValueTooLarge
	}
	return k, tagsize, contentsize, err
}

// SplitList splits b into the content of a list and any remaining
// bytes after the list.
func SplitList(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k != List {
		return nil, b, ErrExpectedList
	}
	return content, rest, nil
}


func readSize(b []byte, slen byte) (Uint64, error) {
	if int(slen) > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	var s Uint64
	switch slen {
	case 1:
		s = Uint64(b[0])
	case 2:
		s = Uint64(b[0])<<8 | Uint64(b[1])
	case 3:
		s = Uint64(b[0])<<16 | Uint64(b[1])<<8 | Uint64(b[2])
	case 4:
		s = Uint64(b[0])<<24 | Uint64(b[1])<<16 | Uint64(b[2])<<8 | Uint64(b[3])
	case 5:
		s = Uint64(b[0])<<32 | Uint64(b[1])<<24 | Uint64(b[2])<<16 | Uint64(b[3])<<8 | Uint64(b[4])
	case 6:
		s = Uint64(b[0])<<40 | Uint64(b[1])<<32 | Uint64(b[2])<<24 | Uint64(b[3])<<16 | Uint64(b[4])<<8 | Uint64(b[5])
	case 7:
		s = Uint64(b[0])<<48 | Uint64(b[1])<<40 | Uint64(b[2])<<32 | Uint64(b[3])<<24 | Uint64(b[4])<<16 | Uint64(b[5])<<8 | Uint64(b[6])
	case 8:
		s = Uint64(b[0])<<56 | Uint64(b[1])<<48 | Uint64(b[2])<<40 | Uint64(b[3])<<32 | Uint64(b[4])<<24 | Uint64(b[5])<<16 | Uint64(b[6])<<8 | Uint64(b[7])
	}
	// Reject sizes < 56 (shouldn't have separate size) and sizes with
	// leading zero bytes.
	if s < 56 || b[0] == 0 {
		return 0, ErrCanonSize
	}
	return s, nil
}
