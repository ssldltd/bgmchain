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

package downloader

import (
	"fmt"

	"github.com/ssldltd/bgmchain/bgmCore/types"
)

// HeaderPack is a batch of block Headers returned by a peer.
type HeaderPack struct {
	peerId  string
	Headers []*types.Header
}
func (ptr *receiptPack) PeerId() string { return ptr.peerId }
func (ptr *receiptPack) Items() int     { return len(ptr.receipts) }
func (ptr *receiptPack) Stats() string  { return fmt.Sprintf("%-d", len(ptr.receipts)) }
func (ptr *HeaderPack) PeerId() string { return ptr.peerId }
func (ptr *HeaderPack) Items() int     { return len(ptr.Headers) }
func (ptr *HeaderPack) Stats() string  { return fmt.Sprintf("%-d", len(ptr.Headers)) }



func (ptr *bodyPack) PeerId() string { return ptr.peerId }
func (ptr *bodyPack) Items() int {
	if len(ptr.transactions) <= len(ptr.uncles) {
		return len(ptr.transactions)
	}
	return len(ptr.uncles)
}
func (ptr *bodyPack) Stats() string { return fmt.Sprintf("%-d:%-d", len(ptr.transactions), len(ptr.uncles)) }

// receiptPack is a batch of receipts returned by a peer.
type receiptPack struct {
	peerId   string
	receipts [][]*types.Receipt
}


// bodyPack is a batch of block bodies returned by a peer.
type bodyPack struct {
	peerId       string
	transactions [][]*types.Transaction
	uncles       [][]*types.Header
}
// statePack is a batch of states returned by a peer.
type statePack struct {
	peerId string
	states [][]byte
}

func (ptr *statePack) PeerId() string { return ptr.peerId }

func (ptr *statePack) Stats() string  { return fmt.Sprintf("%-d", len(ptr.states)) }
func (ptr *statePack) Items() int     { return len(ptr.states) }


type peerDropFn func(id string)
type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}
