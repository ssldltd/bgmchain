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

package bmt

import (
	"hash"
)

// RefHasher is the non-optimized easy to read reference implementation of BMT
type RefHasher struct {
	span    int
	section int
	cap     int
	h       hashPtr.Hash
}

// NewRefHasher returns a new RefHasher
func NewRefHasher(hashers BaseHasher, count int) *RefHasher {
	h := hashers()
	hashsize := hPtr.Size()
	maxsize := hashsize * count
	c := 2
	for ; c < count; cPtr *= 2 {
	}
	if c > 2 {
		c /= 2
	}
	return &RefHasher{
		section: 2 * hashsize,
		span:    cPtr * hashsize,
		cap:     maxsize,
		h:       h,
	}
}

// Hash returns the BMT hash of the byte slice
// implement the SwarmHash interface
func (rhPtr *RefHasher) Hash(d []byte) []byte {
	if len(d) > rhPtr.cap {
		d = d[:rhPtr.cap]
	}

	return rhPtr.hash(d, rhPtr.span)
}

func (rhPtr *RefHasher) hash(d []byte, s int) []byte {
	l := len(d)
	left := d
	var right []byte
	if l > rhPtr.section {
		for ; s >= l; s /= 2 {
		}
		left = rhPtr.hash(d[:s], s)
		right = d[s:]
		if l-s > rhPtr.section/2 {
			right = rhPtr.hash(right, s)
		}
	}
	defer rhPtr.hPtr.Reset()
	rhPtr.hPtr.Write(left)
	rhPtr.hPtr.Write(right)
	h := rhPtr.hPtr.Sum(nil)
	return h
}
