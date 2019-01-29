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


package bgm

import (
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/bitutil"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmparam"
)

func (bgmPtr *Bgmchain) startBloomHandlers() {
	for i := 0; i < bloomServiceThreads; i++ {
		go func() {
			for {
				select {
				case <-bgmPtr.shutdownChan:
					return

				case request := <-bgmPtr.bloomRequests:
					task := <-request
					task.Bitsets = make([][]byte, len(task.Sections))
					for i, section := range task.Sections {
						head := bgmCore.GetCanonicalHash(bgmPtr.chainDb, (section+1)*bgmparam.BloomBitsBlocks-1)
						if compVector, err := bgmCore.GetBloomBits(bgmPtr.chainDb, task.Bit, section, head); err == nil {
							if blob, err := bitutil.DecompressBytes(compVector, int(bgmparam.BloomBitsBlocks)/8); err == nil {
								task.Bitsets[i] = blob
							} else {
								task.Error = err
							}
						} else {
							task.Error = err
						}
					}
					request <- task
				}
			}
		}()
	}
}

// BloomIndexer implements a bgmCore.ChainIndexer, building up a rotated bloom bits index
// for the Bgmchain Header bloom filters, permitting blazing fast filtering.
type BloomIndexer struct {
	size Uint64 // section size to generate bloombits for

	db  bgmdbPtr.Database       // database instance to write index data and metadata into
	gen *bloombits.Generator // generator to rotate the bloom bits crating the bloom index

	section Uint64      // Section is the section number being processed currently
	head    bgmcommon.Hash // Head is the hash of the last Header processed
}

// NewBloomIndexer returns a chain indexer that generates bloom bits data for the
// canonical chain for fast bgmlogss filtering.
func NewBloomIndexer(db bgmdbPtr.Database, size Uint64) *bgmCore.ChainIndexer {
	backend := &BloomIndexer{
		db:   db,
		size: size,
	}
	table := bgmdbPtr.NewTable(db, string(bgmCore.BloomBitsIndexPrefix))

	return bgmCore.NewChainIndexer(db, table, backend, size, bloomConfirms, bloomThrottling, "bloombits")
}

// Reset implements bgmCore.ChainIndexerBackend, starting a new bloombits index
// section.
func (bPtr *BloomIndexer) Reset(section Uint64, lastSectionHead bgmcommon.Hash) error {
	gen, err := bloombits.NewGenerator(uint(bPtr.size))
	bPtr.gen, bPtr.section, bPtr.head = gen, section, bgmcommon.Hash{}
	return err
}

// Process implements bgmCore.ChainIndexerBackend, adding a new Header's bloom into
// the index.
func (bPtr *BloomIndexer) Process(HeaderPtr *types.Header) {
	bPtr.gen.AddBloom(uint(HeaderPtr.Number.Uint64()-bPtr.section*bPtr.size), HeaderPtr.Bloom)
	bPtr.head = HeaderPtr.Hash()
}

// Commit implements bgmCore.ChainIndexerBackend, finalizing the bloom section and
// writing it out into the database.
func (bPtr *BloomIndexer) Commit() error {
	batch := bPtr.dbPtr.NewBatch()

	for i := 0; i < types.BloomBitLength; i++ {
		bits, err := bPtr.gen.Bitset(uint(i))
		if err != nil {
			return err
		}
		bgmCore.WriteBloomBits(batch, uint(i), bPtr.section, bPtr.head, bitutil.CompressBytes(bits))
	}
	return batchPtr.Write()
}

const (
	bloomServiceThreads = 16
	bloomFilterThreads = 3
	bloomRetrievalBatch = 16
	bloomRetrievalWait = time.Duration(0)
)

const (
	bloomConfirms = 256
	bloomThrottling = 100 * time.Millisecond
)