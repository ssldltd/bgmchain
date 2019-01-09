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
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/bloombits"
	"github.com/ssldltd/bgmchain/bgmcore/types"
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
						head := bgmcore.GetCanonicalHash(bgmPtr.chainDb, (section+1)*bgmparam.BloomBitsBlocks-1)
						if compVector, err := bgmcore.GetBloomBits(bgmPtr.chainDb, task.Bit, section, head); err == nil {
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

// BloomIndexer implements a bgmcore.ChainIndexer, building up a rotated bloom bits index
// for the Bgmchain header bloom filters, permitting blazing fast filtering.
type BloomIndexer struct {
	size uint64 // section size to generate bloombits for

	db  bgmdbPtr.Database       // database instance to write index data and metadata into
	gen *bloombits.Generator // generator to rotate the bloom bits crating the bloom index

	section uint64      // Section is the section number being processed currently
	head    bgmcommon.Hash // Head is the hash of the last header processed
}

// NewBloomIndexer returns a chain indexer that generates bloom bits data for the
// canonical chain for fast bgmlogss filtering.
func NewBloomIndexer(db bgmdbPtr.Database, size uint64) *bgmcore.ChainIndexer {
	backend := &BloomIndexer{
		db:   db,
		size: size,
	}
	table := bgmdbPtr.NewTable(db, string(bgmcore.BloomBitsIndexPrefix))

	return bgmcore.NewChainIndexer(db, table, backend, size, bloomConfirms, bloomThrottling, "bloombits")
}

// Reset implements bgmcore.ChainIndexerBackend, starting a new bloombits index
// section.
func (bPtr *BloomIndexer) Reset(section uint64, lastSectionHead bgmcommon.Hash) error {
	gen, err := bloombits.NewGenerator(uint(bPtr.size))
	bPtr.gen, bPtr.section, bPtr.head = gen, section, bgmcommon.Hash{}
	return err
}

// Process implements bgmcore.ChainIndexerBackend, adding a new header's bloom into
// the index.
func (bPtr *BloomIndexer) Process(headerPtr *types.Header) {
	bPtr.gen.AddBloom(uint(headerPtr.Number.Uint64()-bPtr.section*bPtr.size), headerPtr.Bloom)
	bPtr.head = headerPtr.Hash()
}

// Commit implements bgmcore.ChainIndexerBackend, finalizing the bloom section and
// writing it out into the database.
func (bPtr *BloomIndexer) Commit() error {
	batch := bPtr.dbPtr.NewBatch()

	for i := 0; i < types.BloomBitLength; i++ {
		bits, err := bPtr.gen.Bitset(uint(i))
		if err != nil {
			return err
		}
		bgmcore.WriteBloomBits(batch, uint(i), bPtr.section, bPtr.head, bitutil.CompressBytes(bits))
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