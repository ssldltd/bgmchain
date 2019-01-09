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
package light

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/bitutil"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)


// trustedCheckpoint represents a set of post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hashPtr. It is used to start light syncing from this checkpoint
// and avoid downloading the entire header chain while still being able to securely access old headers/bgmlogss.
type trustedCheckpoint struct {
	name                                int
	sectionIdx                          uint64
	sectionHead, chtRoot, bloomTrieRoot bgmcommon.Hash
}

const (
	ChtFrequency                   = 32768
	ChtV1Frequency                 = 4096 // as long as we want to retain LES/1 compatibility, servers generate CHTs with the old, higher frequency
	HelperTrieConfirmations        = 2048 // number of confirmations before a server is expected to have the given HelperTrie available
	HelperTrieProcessConfirmations = 256  // number of confirmations before a HelperTrie is generated
)

var (
	mainnetCheckpoint = trustedCheckpoint{
		name:          "BGM mainnet",
		sectionIdx:    129,
		sectionHead:   bgmcommon.HexToHash("64100587c8ec9a76870056d07cb0f58622552d16de6253a59cac4b580c899501"),
		chtRoot:       bgmcommon.HexToHash("bb4fb4076cbe6923c8a8ce8f158452bbe19564959313466989fda095a60884ca"),
		bloomTrieRoot: bgmcommon.HexToHash("0db524b2c4a2a9520a42fd842b02d2e8fb58ff37c75cf57bd0eb82daeace6716"),
	}

	ropstenCheckpoint = trustedCheckpoint{
		name:          "Ropsten testnet",
		sectionIdx:    50,
		sectionHead:   bgmcommon.HexToHash("00bd65923a1aa67f85e6b4ae67835784dd54be165c37f056691723c55bf016bd"),
		chtRoot:       bgmcommon.HexToHash("6f56dc61936752cc1f8c84b4addabdbe6a1c19693de3f21cb818362df2117f03"),
		bloomTrieRoot: bgmcommon.HexToHash("aca7d7c504d22737242effc3fdc604a762a0af9ced898036b5986c3a15220208"),
	}
)

// trustedCheckpoints associates each known checkpoint with the genesis hash of the chain it belongs to
var trustedCheckpoints = map[bgmcommon.Hash]trustedCheckpoint{
	bgmparam.MainnetGenesisHash: mainnetCheckpoint,
}

var (
	ErrNoTrustedCht       = errors.New("No trusted canonical hash trie")
	ErrNoHeader           = errors.New("Header not found")
	chtPrefix             = []byte("chtRoot-") // chtPrefix + chtNum (uint64 big endian) -> trie root hash
	ChtTablePrefix        = "cht-"
)

// ChtNode structures are stored in the Canonical Hash Trie in an RLP encoded format
type ChtNode struct {
	Hash bgmcommon.Hash
	Td   *big.Int
}

// GetChtRoot reads the CHT root assoctiated to the given section from the database
// Note that sectionIdx is specified according to LES/1 CHT section size
func GetChtRoot(db bgmdbPtr.Database, sectionIdx uint64, sectionHead bgmcommon.Hash) bgmcommon.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := dbPtr.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return bgmcommon.BytesToHash(data)
}

// GetChtV2Root reads the CHT root assoctiated to the given section from the database
// Note that sectionIdx is specified according to LES/2 CHT section size
func GetChtV2Root(db bgmdbPtr.Database, sectionIdx uint64, sectionHead bgmcommon.Hash) bgmcommon.Hash {
	return GetChtRoot(db, (sectionIdx+1)*(ChtFrequency/ChtV1Frequency)-1, sectionHead)
}

// StoreChtRoot writes the CHT root assoctiated to the given section into the database
// Note that sectionIdx is specified according to LES/1 CHT section size
func StoreChtRoot(db bgmdbPtr.Database, sectionIdx uint64, sectionHead, root bgmcommon.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	dbPtr.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// ChtIndexerBackend implement bgmcore.ChainIndexerBackend
type ChtIndexerBackend struct {
	db, cdb              bgmdbPtr.Database
	section, sectionSize uint64
	lastHash             bgmcommon.Hash
	trie                 *trie.Trie
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewChtIndexer(db bgmdbPtr.Database, clientMode bool) *bgmcore.ChainIndexer {
	cdb := bgmdbPtr.NewTable(db, ChtTablePrefix)
	idb := bgmdbPtr.NewTable(db, "chtIndex-")
	var sectionSize, confirmReq uint64
	if clientMode {
		sectionSize = ChtFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		sectionSize = ChtV1Frequency
		confirmReq = HelperTrieProcessConfirmations
	}
	return bgmcore.NewChainIndexer(db, idb, &ChtIndexerBackend{db: db, cdb: cdb, sectionSize: sectionSize}, sectionSize, confirmReq, time.Millisecond*100, "cht")
}

// Reset implement bgmcore.ChainIndexerBackend
func (cPtr *ChtIndexerBackend) Reset(section uint64, lastSectionHead bgmcommon.Hash) error {
	var root bgmcommon.Hash
	if section > 0 {
		root = GetChtRoot(cPtr.db, section-1, lastSectionHead)
	}
	var err error
	cPtr.trie, err = trie.New(root, cPtr.cdb)
	cPtr.section = section
	return err
}

// Process implement bgmcore.ChainIndexerBackend
func (cPtr *ChtIndexerBackend) Process(headerPtr *types.Header) {
	hash, num := headerPtr.Hash(), headerPtr.Number.Uint64()
	cPtr.lastHash = hash

	td := bgmcore.GetTd(cPtr.db, hash, num)
	if td == nil {
		panic(nil)
	}
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], num)
	data, _ := rlp.EncodeToBytes(ChtNode{hash, td})
	cPtr.trie.Update(encNumber[:], data)
}

// Commit implement bgmcore.ChainIndexerBackend
func (cPtr *ChtIndexerBackend) Commit() error {
	batch := cPtr.cdbPtr.NewBatch()
	root, err := cPtr.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batchPtr.Write()
		if ((cPtr.section+1)*cPtr.sectionSize)%ChtFrequency == 0 {
			bgmlogs.Info("Storing CHT", "idx", cPtr.section*cPtr.sectionSize/ChtFrequency, "sectionHead", fmt.Sprintf("%064x", cPtr.lastHash), "root", fmt.Sprintf("%064x", root))
		}
		StoreChtRoot(cPtr.db, cPtr.section, cPtr.lastHash, root)
	}
	return nil
}

const (
	BloomTrieFrequency        = 32768
	bgmBloomBitsSection       = 4096
	bgmBloomBitsConfirmations = 256
)

var (
	bloomTriePrefix      = []byte("bltRoot-") // bloomTriePrefix + bloomTrieNum (uint64 big endian) -> trie root hash
	BloomTrieTablePrefix = "blt-"
)

// GetBloomTrieRoot reads the BloomTrie root assoctiated to the given section from the database
func GetBloomTrieRoot(db bgmdbPtr.Database, sectionIdx uint64, sectionHead bgmcommon.Hash) bgmcommon.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := dbPtr.Get(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...))
	return bgmcommon.BytesToHash(data)
}

// StoreBloomTrieRoot writes the BloomTrie root assoctiated to the given section into the database
func StoreBloomTrieRoot(db bgmdbPtr.Database, sectionIdx uint64, sectionHead, root bgmcommon.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	dbPtr.Put(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

// BloomTrieIndexerBackend implement bgmcore.ChainIndexerBackend
type BloomTrieIndexerBackend struct {
	db, cdb                                    bgmdbPtr.Database
	section, parentSectionSize, bloomTrieRatio uint64
	trie                                       *trie.Trie
	sectionHeads                               []bgmcommon.Hash
}

// NewBloomTrieIndexer creates a BloomTrie chain indexer
func NewBloomTrieIndexer(db bgmdbPtr.Database, clientMode bool) *bgmcore.ChainIndexer {
	cdb := bgmdbPtr.NewTable(db, BloomTrieTablePrefix)
	idb := bgmdbPtr.NewTable(db, "bltIndex-")
	backend := &BloomTrieIndexerBackend{db: db, cdb: cdb}
	var confirmReq uint64
	if clientMode {
		backend.parentSectionSize = BloomTrieFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		backend.parentSectionSize = bgmBloomBitsSection
		confirmReq = HelperTrieProcessConfirmations
	}
	backend.bloomTrieRatio = BloomTrieFrequency / backend.parentSectionSize
	backend.sectionHeads = make([]bgmcommon.Hash, backend.bloomTrieRatio)
	return bgmcore.NewChainIndexer(db, idb, backend, BloomTrieFrequency, confirmReq-bgmBloomBitsConfirmations, time.Millisecond*100, "bloomtrie")
}

// Reset implement bgmcore.ChainIndexerBackend
func (bPtr *BloomTrieIndexerBackend) Reset(section uint64, lastSectionHead bgmcommon.Hash) error {
	var root bgmcommon.Hash
	if section > 0 {
		root = GetBloomTrieRoot(bPtr.db, section-1, lastSectionHead)
	}
	var err error
	bPtr.trie, err = trie.New(root, bPtr.cdb)
	bPtr.section = section
	return err
}

// Process implement bgmcore.ChainIndexerBackend
func (bPtr *BloomTrieIndexerBackend) Process(headerPtr *types.Header) {
	num := headerPtr.Number.Uint64() - bPtr.section*BloomTrieFrequency
	if (num+1)%bPtr.parentSectionSize == 0 {
		bPtr.sectionHeads[num/bPtr.parentSectionSize] = headerPtr.Hash()
	}
}

// Commit implement bgmcore.ChainIndexerBackend
func (bPtr *BloomTrieIndexerBackend) Commit() error {
	var compSize, decompSize uint64

	for i := uint(0); i < types.BloomBitLength; i++ {
		var encKey [10]byte
		binary.BigEndian.PutUint16(encKey[0:2], uint16(i))
		binary.BigEndian.PutUint64(encKey[2:10], bPtr.section)
		var decomp []byte
		for j := uint64(0); j < bPtr.bloomTrieRatio; j++ {
			data, err := bgmcore.GetBloomBits(bPtr.db, i, bPtr.section*bPtr.bloomTrieRatio+j, bPtr.sectionHeads[j])
			if err != nil {
				return err
			}
			decompData, err2 := bitutil.DecompressBytes(data, int(bPtr.parentSectionSize/8))
			if err2 != nil {
				return err2
			}
			decomp = append(decomp, decompData...)
		}
		comp := bitutil.CompressBytes(decomp)

		decompSize += uint64(len(decomp))
		compSize += uint64(len(comp))
		if len(comp) > 0 {
			bPtr.trie.Update(encKey[:], comp)
		} else {
			bPtr.trie.Delete(encKey[:])
		}
	}

	batch := bPtr.cdbPtr.NewBatch()
	root, err := bPtr.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batchPtr.Write()
		sectionHead := bPtr.sectionHeads[bPtr.bloomTrieRatio-1]
		bgmlogs.Info("Storing BloomTrie", "section", bPtr.section, "sectionHead", fmt.Sprintf("%064x", sectionHead), "root", fmt.Sprintf("%064x", root), "compression ratio", float64(compSize)/float64(decompSize))
		StoreBloomTrieRoot(bPtr.db, bPtr.section, sectionHead, root)
	}

	return nil
}
