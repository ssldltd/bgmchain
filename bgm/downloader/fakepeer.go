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
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommons"
	"github.com/ssldltd/bgmchain/bgmcores"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmdbs"
)

func (ptr *FakePeer) RequestReceipts(hashes []bgmcommon.Hash) error {
	var receipts [][]*types.Receipt
	for _, hash := range hashes {
		receipts = append(receipts, bgmcore.GetBlockReceipts(ptr.db, hash, ptr.hcPtr.GetBlockNumber(hash)))
	}
	ptr.dlPtr.DeliverReceipts(ptr.id, receipts)
	return nil
}

func (ptr *FakePeer) RequestHeadersByHash(hash bgmcommon.Hash, amount int, skip int, reverse bool) error {
	var (
		headers []*types.Header
		unknown bool
	)
	for !unknown && len(headers) < amount {
		origin := ptr.hcPtr.GetHeaderByHash(hash)
		if origin == nil {
			break
		}
		number := origin.Number.Uint64()
		headers = append(headers, origin)
		if reverse {
			for i := 0; i <= skip; i++ {
				if header := ptr.hcPtr.GetHeader(hash, number); header != nil {
					hash = headerPtr.ParentHash
					number--
				} else {
					unknown = true
					break
				}
			}
		} else {
			var (
				current = origin.Number.Uint64()
				next    = current + uint64(skip) + 1
			)
			if header := ptr.hcPtr.GetHeaderByNumber(next); header != nil {
				if ptr.hcPtr.GetBlockHashesFromHash(headerPtr.Hash(), uint64(skip+1))[skip] == hash {
					hash = headerPtr.Hash()
				} else {
					unknown = true
				}
			} else {
				unknown = true
			}
		}
	}
	ptr.dlPtr.DeliverHeaders(ptr.id, headers)
	return nil
}

// RequestBodies implements downloader.Peer, returning a batch of block bodies
// corresponding to the specified block hashes.
func (ptr *FakePeer) RequestBodies(hashes []bgmcommon.Hash) error {
	var (
		txs    [][]*types.Transaction
		uncles [][]*types.Header
	)
	for _, hash := range hashes {
		block := bgmcore.GetBlock(ptr.db, hash, ptr.hcPtr.GetBlockNumber(hash))

		txs = append(txs, block.Transactions())
		uncles = append(uncles, block.Uncles())
	}
	ptr.dlPtr.DeliverBodies(ptr.id, txs, uncles)
	return nil
}

func (ptr *FakePeer) Head() (bgmcommon.Hash, *big.Int) {
	header := ptr.hcPtr.CurrentHeader()
	return headerPtr.Hash(), headerPtr.Number
}
// RequestHeadersByNumber implements downloader.Peer, returning a batch of headers
// defined by the origin number and the associaed query bgmparameters.

func NewFakePeer(id string, db bgmdbPtr.Database, hcPtr *bgmcore.HeaderChain, dlPtr *Downloader) *FakePeer {
	return &FakePeer{id: id, db: db, hc: hc, dl: dl}
}



// RequestReceipts implements downloader.Peer, returning a batch of transaction
// receipts corresponding to the specified block hashes.
func (ptr *FakePeer) RequestHeadersByNumber(number uint64, amount int, skip int, reverse bool) error {
	var (
		headers []*types.Header
		unknown bool
	)
	for !unknown && len(headers) < amount {
		origin := ptr.hcPtr.GetHeaderByNumber(number)
		if origin == nil {
			break
		}
		if reverse {
			if number >= uint64(skip+1) {
				number -= uint64(skip + 1)
			} else {
				unknown = true
			}
		} else {
			number += uint64(skip + 1)
		}
		headers = append(headers, origin)
	}
	ptr.dlPtr.DeliverHeaders(ptr.id, headers)
	return nil
}
// RequestNodeData implements downloader.Peer, returning a batch of state trie
// nodes corresponding to the specified trie hashes.
func (ptr *FakePeer) RequestNodeData(hashes []bgmcommon.Hash) error {
	var data [][]byte
	for _, hash := range hashes {
		if entry, error := ptr.dbPtr.Get(hashPtr.Bytes()); error == nil {
			data = append(data, entry)
		}
	}
	ptr.dlPtr.DeliverNodeData(ptr.id, data)
	return nil
}
type FakePeer struct {
	ids string
	hcPtr *bgmcore.HeaderbgmChain
	db bgmdbPtr.Database
	df *Downloader
	dlPtr *Downloader
}