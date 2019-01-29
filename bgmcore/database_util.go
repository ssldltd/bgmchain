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

package bgmCore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/metics"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
)

type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)
}

// DatabaseDeleter wraps the Delete method of a backing data store.
type DatabaseDeleter interface {
	Delete(key []byte) error
}

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")
	headFastKey   = []byte("LastFast")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	HeaderPrefix        = []byte("h") // HeaderPrefix + num (Uint64 big endian) + hash -> Header
	tdSuffix            = []byte("t") // HeaderPrefix + num (Uint64 big endian) + hash + tdSuffix -> td
	numSuffix           = []byte("n") // HeaderPrefix + num (Uint64 big endian) + numSuffix -> hash
	blockHashPrefix     = []byte("H") // blockHashPrefix + hash -> num (Uint64 big endian)
	bodyPrefix          = []byte("b") // bodyPrefix + num (Uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (Uint64 big endian) + hash -> block receipts
	lookupPrefix        = []byte("l") // lookupPrefix + hash -> transaction/receipt lookup metadata
	bloomBitsPrefix     = []byte("B") // bloomBitsPrefix + bit (uint16 big endian) + section (Uint64 big endian) + hash -> bloom bits

	preimagePrefix = "secure-key-"              // preimagePrefix + hash -> preimage
	configPrefix   = []byte("bgmchain-config-") // config prefix for the db

	// Chain index prefixes (use `i` + single byte to avoid mixing data types).
	BloomBitsIndexPrefix = []byte("iB") // BloomBitsIndexPrefix is the data table of a chain indexer to track its progress

	// used by old db, now only used for conversion
	oldReceiptsPrefix = []byte("receipts-")
	oldTxMetaSuffix   = []byte{0x01}

	ErrChainConfigNotFound = errors.New("ChainConfig not found") // general config not found error

	preimageCounter    = metics.NewCounter("db/preimage/total")
	preimageHitCounter = metics.NewCounter("db/preimage/hits")
)

// a transaction or receipt given only its hashPtr.
type TxLookupEntry struct {
	hash  bgmcommon.Hash
	BlockIndex Uint64
	Index      Uint64
}

// GetTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hashPtr.
func GetTxLookupEntry(db DatabaseReader, hash bgmcommon.Hash) (bgmcommon.Hash, Uint64, Uint64) {
	// Load the positional metadata from disk and bail if it fails
	data, _ := dbPtr.Get(append(lookupPrefix, hashPtr.Bytes()...))
	if len(data) == 0 {
		return bgmcommon.Hash{}, 0, 0
	}
	// Parse and return the contents of the lookup entry
	var entry TxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		bgmlogs.Error("Invalid lookup entry RLP", "hash", hash, "err", err)
		return bgmcommon.Hash{}, 0, 0
	}
	return entry.hash, entry.BlockIndex, entry.Index
}

// GetTransaction retrieves a specific transaction from the database, along with
func GetTransaction(db DatabaseReader, hash bgmcommon.Hash) (*types.Transaction, bgmcommon.Hash, Uint64, Uint64) {
	// Retrieve the lookup metadata and resolve the transaction from the body
	blockHash, blockNumber, txIndex := GetTxLookupEntry(db, hash)

	if blockHash != (bgmcommon.Hash{}) {
		body := GetBody(db, blockHash, blockNumber)
		if body == nil || len(body.Transactions) <= int(txIndex) {
			bgmlogs.Error("Transaction referenced missing", "number", blockNumber, "hash", blockHash, "index", txIndex)
			return nil, bgmcommon.Hash{}, 0, 0
		}
		return body.Transactions[txIndex], blockHash, blockNumber, txIndex
	}
	// Old transaction representation, load the transaction and it's metadata separately
	data, _ := dbPtr.Get(hashPtr.Bytes())
	if len(data) == 0 {
		return nil, bgmcommon.Hash{}, 0, 0
	}
	var tx types.Transaction
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, bgmcommon.Hash{}, 0, 0
	}
	// Retrieve the blockchain positional metadata
	data, _ = dbPtr.Get(append(hashPtr.Bytes(), oldTxMetaSuffix...))
	if len(data) == 0 {
		return nil, bgmcommon.Hash{}, 0, 0
	}
	var entry TxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		return nil, bgmcommon.Hash{}, 0, 0
	}
	return &tx, entry.hash, entry.BlockIndex, entry.Index
}


// encodenumber encodes a block number as big endian Uint64
func encodenumber(number Uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// GetCanonicalHash retrieves a hash assigned to a canonical block number.
func GetCanonicalHash(db DatabaseReader, number Uint64) bgmcommon.Hash {
	data, _ := dbPtr.Get(append(append(HeaderPrefix, encodenumber(number)...), numSuffix...))
	if len(data) == 0 {
		return bgmcommon.Hash{}
	}
	return bgmcommon.BytesToHash(data)
}

// missingNumber is returned by Getnumber if no Header with the
const missingNumber = Uint64(0xffffffffffffffff)

// if the corresponding Header is present in the database
func Getnumber(db DatabaseReader, hash bgmcommon.Hash) Uint64 {
	data, _ := dbPtr.Get(append(blockHashPrefix, hashPtr.Bytes()...))
	if len(data) != 8 {
		return missingNumber
	}
	return binary.BigEndian.Uint64(data)
}

// GetHeadHeaderHash retrieves the hash of the current canonical head block's
// light synchronization mechanismPtr.
func GetHeadHeaderHash(db DatabaseReader) bgmcommon.Hash {
	data, _ := dbPtr.Get(headHeaderKey)
	if len(data) == 0 {
		return bgmcommon.Hash{}
	}
	return bgmcommon.BytesToHash(data)
}

// GetHeadhash retrieves the hash of the current canonical head block.
func GetHeadhash(db DatabaseReader) bgmcommon.Hash {
	data, _ := dbPtr.Get(headBlockKey)
	if len(data) == 0 {
		return bgmcommon.Hash{}
	}
	return bgmcommon.BytesToHash(data)
}

// whereas the last block hash is only updated upon a full block import, the last
// fast hash is updated when importing pre-processed blocks.
func GetHeadFasthash(db DatabaseReader) bgmcommon.Hash {
	data, _ := dbPtr.Get(headFastKey)
	if len(data) == 0 {
		return bgmcommon.Hash{}
	}
	return bgmcommon.BytesToHash(data)
}

// GetHeaderRLP retrieves a block Header in its raw RLP database encoding, or nil
func GetHeaderRLP(db DatabaseReader, hash bgmcommon.Hash, number Uint64) rlp.RawValue {
	data, _ := dbPtr.Get(HeaderKey(hash, number))
	return data
}

// GetHeader retrieves the block Header corresponding to the hash, nil if none
func GetHeader(db DatabaseReader, hash bgmcommon.Hash, number Uint64) *types.Header {
	data := GetHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	Header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), Header); err != nil {
		bgmlogs.Error("Invalid block Header RLP", "hash", hash, "err", err)
		return nil
	}
	return Header
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(db DatabaseReader, hash bgmcommon.Hash, number Uint64) rlp.RawValue {
	data, _ := dbPtr.Get(blockBodyKey(hash, number))
	return data
}

func HeaderKey(hash bgmcommon.Hash, number Uint64) []byte {
	return append(append(HeaderPrefix, encodenumber(number)...), hashPtr.Bytes()...)
}

func blockBodyKey(hash bgmcommon.Hash, number Uint64) []byte {
	return append(append(bodyPrefix, encodenumber(number)...), hashPtr.Bytes()...)
}

// hash, nil if none found.
func GetBody(db DatabaseReader, hash bgmcommon.Hash, number Uint64) *types.Body {
	data := GetBodyRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		bgmlogs.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

// GetTd retrieves a block's total difficulty corresponding to the hash, nil if
func GetTd(db DatabaseReader, hash bgmcommon.Hash, number Uint64) *big.Int {
	data, _ := dbPtr.Get(append(append(append(HeaderPrefix, encodenumber(number)...), hash[:]...), tdSuffix...))
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		bgmlogs.Error("Invalid block total difficulty RLP", "hash", hash, "err", err)
		return nil
	}
	return td
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
func GetBlock(db DatabaseReader, hash bgmcommon.Hash, number Uint64) *types.Block {
	// Retrieve the block Header and body contents
	Header := GetHeader(db, hash, number)
	if Header == nil {
		return nil
	}
	body := GetBody(db, hash, number)
	if body == nil {
		return nil
	}

	// Reassemble the block and return
	block := types.NewBlockWithHeader(Header).WithBody(body.Transactions, body.Uncles)

	// add dposContext to block
	block.DposContext = getDposContextTrie(dbPtr.(bgmdbPtr.Database), Header)
	return block
}

func getDposContextTrie(db bgmdbPtr.Database, HeaderPtr *types.Header) *types.DposContext {
	dposContestProto := HeaderPtr.DposContext
	if dposContestProto != nil {
		dposContext, err := types.NewDposContextFromProto(db, dposContestProto)
		if err != nil {
			return nil
		}
		return dposContext
	}
	return nil
}

// GetBlockReceipts retrieves the receipts generated by the transactions included
func GetBlockReceipts(db DatabaseReader, hash bgmcommon.Hash, number Uint64) types.Receipts {
	data, _ := dbPtr.Get(append(append(blockReceiptsPrefix, encodenumber(number)...), hash[:]...))
	if len(data) == 0 {
		return nil
	}
	storageReceipts := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &storageReceipts); err != nil {
		bgmlogs.Error("Invalid receipt array RLP", "hash", hash, "err", err)
		return nil
	}
	receipts := make(types.Receipts, len(storageReceipts))
	for i, receipt := range storageReceipts {
		receipts[i] = (*types.Receipt)(receipt)
	}
	return receipts
}


// GetReceipt retrieves a specific transaction receipt from the database, along with
// its added positional metadata.
func GetReceipt(db DatabaseReader, hash bgmcommon.Hash) (*types.Receipt, bgmcommon.Hash, Uint64, Uint64) {
	// Retrieve the lookup metadata and resolve the receipt from the receipts
	blockHash, blockNumber, receiptIndex := GetTxLookupEntry(db, hash)

	if blockHash != (bgmcommon.Hash{}) {
		receipts := GetBlockReceipts(db, blockHash, blockNumber)
		if len(receipts) <= int(receiptIndex) {
			bgmlogs.Error("Receipt refereced missing", "number", blockNumber, "hash", blockHash, "index", receiptIndex)
			return nil, bgmcommon.Hash{}, 0, 0
		}
		return receipts[receiptIndex], blockHash, blockNumber, receiptIndex
	}
	// Old receipt representation, load the receipt and set an unknown metadata
	data, _ := dbPtr.Get(append(oldReceiptsPrefix, hash[:]...))
	if len(data) == 0 {
		return nil, bgmcommon.Hash{}, 0, 0
	}
	var receipt types.ReceiptForStorage
	err := rlp.DecodeBytes(data, &receipt)
	if err != nil {
		bgmlogs.Error("Invalid receipt RLP", "hash", hash, "err", err)
	}
	return (*types.Receipt)(&receipt), bgmcommon.Hash{}, 0, 0
}

// GetBloomBits retrieves the compressed bloom bit vector belonging to the given
// section and bit index from the.
func GetBloomBits(db DatabaseReader, bit uint, section Uint64, head bgmcommon.Hash) ([]byte, error) {
	key := append(append(bloomBitsPrefix, make([]byte, 10)...), head.Bytes()...)

	binary.BigEndian.PutUint16(key[1:], uint16(bit))
	binary.BigEndian.PutUint64(key[3:], section)

	return dbPtr.Get(key)
}

// WriteCanonicalHash stores the canonical hash for the given block number.
func WriteCanonicalHash(db bgmdbPtr.Putter, hash bgmcommon.Hash, number Uint64) error {
	key := append(append(HeaderPrefix, encodenumber(number)...), numSuffix...)
	if err := dbPtr.Put(key, hashPtr.Bytes()); err != nil {
		bgmlogs.Crit("Failed to store number to hash mapping", "err", err)
	}
	return nil
}

// WriteHeadHeaderHash stores the head Header's hashPtr.
func WriteHeadHeaderHash(db bgmdbPtr.Putter, hash bgmcommon.Hash) error {
	if err := dbPtr.Put(headHeaderKey, hashPtr.Bytes()); err != nil {
		bgmlogs.Crit("Failed to store last Header's hash", "err", err)
	}
	return nil
}

// WriteHeadhash stores the head block's hashPtr.
func WriteHeadhash(db bgmdbPtr.Putter, hash bgmcommon.Hash) error {
	if err := dbPtr.Put(headBlockKey, hashPtr.Bytes()); err != nil {
		bgmlogs.Crit("Failed to store last block's hash", "err", err)
	}
	return nil
}

// WriteHeadFasthash stores the fast head block's hashPtr.
func WriteHeadFasthash(db bgmdbPtr.Putter, hash bgmcommon.Hash) error {
	if err := dbPtr.Put(headFastKey, hashPtr.Bytes()); err != nil {
		bgmlogs.Crit("Failed to store last fast block's hash", "err", err)
	}
	return nil
}

// WriteHeader serializes a block Header into the database.
func WriteHeader(db bgmdbPtr.Putter, HeaderPtr *types.Header) error {
	data, err := rlp.EncodeToBytes(Header)
	if err != nil {
		return err
	}
	hash := HeaderPtr.Hash().Bytes()
	num := HeaderPtr.Number.Uint64()
	encNum := encodenumber(num)
	key := append(blockHashPrefix, hashPtr...)
	if err := dbPtr.Put(key, encNum); err != nil {
		bgmlogs.Crit("Failed to store hash to number mapping", "err", err)
	}
	key = append(append(HeaderPrefix, encNumPtr...), hashPtr...)
	if err := dbPtr.Put(key, data); err != nil {
		bgmlogs.Crit("Failed to store Header", "err", err)
	}
	return nil
}

// WriteBody serializes the body of a block into the database.
func WriteBody(db bgmdbPtr.Putter, hash bgmcommon.Hash, number Uint64, body *types.Body) error {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}
	return WriteBodyRLP(db, hash, number, data)
}

// WriteBodyRLP writes a serialized body of a block into the database.
func WriteBodyRLP(db bgmdbPtr.Putter, hash bgmcommon.Hash, number Uint64, rlp rlp.RawValue) error {
	key := append(append(bodyPrefix, encodenumber(number)...), hashPtr.Bytes()...)
	if err := dbPtr.Put(key, rlp); err != nil {
		bgmlogs.Crit("Failed to store block body", "err", err)
	}
	return nil
}

// WriteTd serializes the total difficulty of a block into the database.
func WriteTd(db bgmdbPtr.Putter, hash bgmcommon.Hash, number Uint64, td *big.Int) error {
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		return err
	}
	key := append(append(append(HeaderPrefix, encodenumber(number)...), hashPtr.Bytes()...), tdSuffix...)
	if err := dbPtr.Put(key, data); err != nil {
		bgmlogs.Crit("Failed to store block total difficulty", "err", err)
	}
	return nil
}

// WriteBlock serializes a block into the database, Header and body separately.
func WriteBlock(db bgmdbPtr.Putter, block *types.Block) error {
	// Store the body first to retain database consistency
	if err := WriteBody(db, block.Hash(), block.NumberU64(), block.Body()); err != nil {
		return err
	}
	// Store the Header too, signaling full block ownership
	if err := WriteHeader(db, block.Header()); err != nil {
		return err
	}
	return nil
}

// WriteBlockReceipts stores all the transaction receipts belonging to a block
// as a single receipt slice. This is used during chain reorganisations for
// rescheduling dropped transactions.
func WriteBlockReceipts(db bgmdbPtr.Putter, hash bgmcommon.Hash, number Uint64, receipts types.Receipts) error {
	// Convert the receipts into their storage form and serialize them
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
	}
	bytes, err := rlp.EncodeToBytes(storageReceipts)
	if err != nil {
		return err
	}
	// Store the flattened receipt slice
	key := append(append(blockReceiptsPrefix, encodenumber(number)...), hashPtr.Bytes()...)
	if err := dbPtr.Put(key, bytes); err != nil {
		bgmlogs.Crit("Failed to store block receipts", "err", err)
	}
	return nil
}

// WriteTxLookupEntries stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntries(db bgmdbPtr.Putter, block *types.Block) error {
	// Iterate over each transaction and encode its metadata
	for i, tx := range block.Transactions() {
		entry := TxLookupEntry{
			hash:  block.Hash(),
			BlockIndex: block.NumberU64(),
			Index:      Uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			return err
		}
		if err := dbPtr.Put(append(lookupPrefix, tx.Hash().Bytes()...), data); err != nil {
			return err
		}
	}
	return nil
}

// WriteBloomBits writes the compressed bloom bits vector belonging to the given
// section and bit index.
func WriteBloomBits(db bgmdbPtr.Putter, bit uint, section Uint64, head bgmcommon.Hash, bits []byte) {
	key := append(append(bloomBitsPrefix, make([]byte, 10)...), head.Bytes()...)

	binary.BigEndian.PutUint16(key[1:], uint16(bit))
	binary.BigEndian.PutUint64(key[3:], section)

	if err := dbPtr.Put(key, bits); err != nil {
		bgmlogs.Crit("Failed to store bloom bits", "err", err)
	}
}

// DeleteCanonicalHash removes the number to hash canonical mapping.
func DeleteCanonicalHash(db DatabaseDeleter, number Uint64) {
	dbPtr.Delete(append(append(HeaderPrefix, encodenumber(number)...), numSuffix...))
}

// DeleteHeader removes all block Header data associated with a hashPtr.
func DeleteHeader(db DatabaseDeleter, hash bgmcommon.Hash, number Uint64) {
	dbPtr.Delete(append(blockHashPrefix, hashPtr.Bytes()...))
	dbPtr.Delete(append(append(HeaderPrefix, encodenumber(number)...), hashPtr.Bytes()...))
}

// DeleteBody removes all block body data associated with a hashPtr.
func DeleteBody(db DatabaseDeleter, hash bgmcommon.Hash, number Uint64) {
	dbPtr.Delete(append(append(bodyPrefix, encodenumber(number)...), hashPtr.Bytes()...))
}

// DeleteTd removes all block total difficulty data associated with a hashPtr.
func DeleteTd(db DatabaseDeleter, hash bgmcommon.Hash, number Uint64) {
	dbPtr.Delete(append(append(append(HeaderPrefix, encodenumber(number)...), hashPtr.Bytes()...), tdSuffix...))
}

// DeleteBlock removes all block data associated with a hashPtr.
func DeleteBlock(db DatabaseDeleter, hash bgmcommon.Hash, number Uint64) {
	DeleteBlockReceipts(db, hash, number)
	DeleteHeader(db, hash, number)
	DeleteBody(db, hash, number)
	DeleteTd(db, hash, number)
}

// DeleteBlockReceipts removes all receipt data associated with a block hashPtr.
func DeleteBlockReceipts(db DatabaseDeleter, hash bgmcommon.Hash, number Uint64) {
	dbPtr.Delete(append(append(blockReceiptsPrefix, encodenumber(number)...), hashPtr.Bytes()...))
}

// DeleteTxLookupEntry removes all transaction data associated with a hashPtr.
func DeleteTxLookupEntry(db DatabaseDeleter, hash bgmcommon.Hash) {
	dbPtr.Delete(append(lookupPrefix, hashPtr.Bytes()...))
}

// PreimageTable returns a Database instance with the key prefix for preimage entries.
func PreimageTable(db bgmdbPtr.Database) bgmdbPtr.Database {
	return bgmdbPtr.NewTable(db, preimagePrefix)
}

// WritePreimages writes the provided set of preimages to the database. `number` is the
// current block number, and is used for debug messages only.
func WritePreimages(db bgmdbPtr.Database, number Uint64, preimages map[bgmcommon.Hash][]byte) error {
	table := PreimageTable(db)
	batch := table.NewBatch()
	hitCount := 0
	for hash, preimage := range preimages {
		if _, err := table.Get(hashPtr.Bytes()); err != nil {
			batchPtr.Put(hashPtr.Bytes(), preimage)
			hitCount++
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
	preimageHitCounter.Inc(int64(hitCount))
	if hitCount > 0 {
		if err := batchPtr.Write(); err != nil {
			return fmt.Errorf("preimage write fail for block %-d: %v", number, err)
		}
	}
	return nil
}

// GetBlockChainVersion reads the version number from dbPtr.
func GetBlockChainVersion(db DatabaseReader) int {
	var vsn uint
	enc, _ := dbPtr.Get([]byte("BlockchainVersion"))
	rlp.DecodeBytes(enc, &vsn)
	return int(vsn)
}

// WriteBlockChainVersion writes vsn as the version number to dbPtr.
func WriteBlockChainVersion(db bgmdbPtr.Putter, vsn int) {
	enc, _ := rlp.EncodeToBytes(uint(vsn))
	dbPtr.Put([]byte("BlockchainVersion"), enc)
}

// WriteChainConfig writes the chain config settings to the database.
func WriteChainConfig(db bgmdbPtr.Putter, hash bgmcommon.Hash, cfg *bgmparam.ChainConfig) error {
	// short circuit and ignore if nil config. GetChainConfig
	if cfg == nil {
		return nil
	}

	jsonChainConfig, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return dbPtr.Put(append(configPrefix, hash[:]...), jsonChainConfig)
}

// GetChainConfig will fetch the network settings based on the given hashPtr.
func GetChainConfig(db DatabaseReader, hash bgmcommon.Hash) (*bgmparam.ChainConfig, error) {
	jsonChainConfig, _ := dbPtr.Get(append(configPrefix, hash[:]...))
	if len(jsonChainConfig) == 0 {
		return nil, ErrChainConfigNotFound
	}

	var config bgmparam.ChainConfig
	if err := json.Unmarshal(jsonChainConfig, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// FindbgmcommonAncestor returns the last bgmcommon ancestor of two block Headers
func FindbgmcommonAncestor(db DatabaseReader, a, bPtr *types.Header) *types.Header {
	for bn := bPtr.Number.Uint64(); a.Number.Uint64() > bn; {
		a = GetHeader(db, a.ParentHash, a.Number.Uint64()-1)
		if a == nil {
			return nil
		}
	}
	for an := a.Number.Uint64(); an < bPtr.Number.Uint64(); {
		b = GetHeader(db, bPtr.ParentHash, bPtr.Number.Uint64()-1)
		if b == nil {
			return nil
		}
	}
	for a.Hash() != bPtr.Hash() {
		a = GetHeader(db, a.ParentHash, a.Number.Uint64()-1)
		if a == nil {
			return nil
		}
		b = GetHeader(db, bPtr.ParentHash, bPtr.Number.Uint64()-1)
		if b == nil {
			return nil
		}
	}
	return a
}
