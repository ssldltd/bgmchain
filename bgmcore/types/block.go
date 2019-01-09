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

package types

import (
	"sort"
	"sync/atomic"
	"time"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	
	
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/bgcomon"
	"github.com/ssldltd/bgmchain/bgcomon/hexutil"
	
)

var (
	EmptyRootHash  = DeriveSha(Transactions{})
	EmptyUncleHash = CalcUncleHash(nil)
)

// A BlockNonce is a 64-bit hash which proves (combined with the
type BlockNonce [8]byte

func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}



//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

// Header represents a block header in the Bgmchain blockchain.
type Header struct {
	ParentHash  bgcomon.Hash       `json:"parentHash"       gencodec:"required"`
	UncleHash   bgcomon.Hash       `json:"sha3Uncles"       gencodec:"required"`
	Validator   bgcomon.Address    `json:"validator"        gencodec:"required"`
	Coinbase    bgcomon.Address    `json:"coinbase"         gencodec:"required"`
	ReceiptHash bgcomon.Hash       `json:"receiptsRoot"     gencodec:"required"`
	DposContext *DposContextProto `json:"dposContext"      gencodec:"required"`
	Bloom       Bloom             `json:"bgmlogssBloom"        gencodec:"required"`
	Difficulty  *big.Int          `json:"difficulty"       gencodec:"required"`
	Number      *big.Int          `json:"number"           gencodec:"required"`
	GasLimit    *big.Int          `json:"gasLimit"         gencodec:"required"`
	GasUsed     *big.Int          `json:"gasUsed"          gencodec:"required"`
	Time        *big.Int          `json:"timestamp"        gencodec:"required"`
	Extra       []byte            `json:"extraData"        gencodec:"required"`
	MixDigest   bgcomon.Hash       `json:"mixHash"          gencodec:"required"`
	Nonce       BlockNonce        `json:"nonce"            gencodec:"required"`
	Root        bgcomon.Hash       `json:"stateRoot"        gencodec:"required"`
	TxHash      bgcomon.Hash       `json:"transactionsRoot" gencodec:"required"`
	
}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty *hexutil.Big
	GasUsed    *hexutil.Big
	Time       *hexutil.Big
	Extra      hexutil.Bytes
	Number     *hexutil.Big
	GasLimit   *hexutil.Big
	Hash       bgcomon.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the headerPtr, which is simply the keccak256 hash of its
func (hPtr *Header) Hash() bgcomon.Hash {
	return rlpHash(h)
}

// HashNoNonce returns the hash which is used as input for the proof-of-work searchPtr.
func (hPtr *Header) HashNoNonce() bgcomon.Hash {
	return rlpHash([]interface{}{
		hPtr.ParentHash,
		hPtr.UncleHash,
		hPtr.Validator,
		hPtr.Bloom,
		hPtr.Difficulty,
		hPtr.Number,
		hPtr.GasLimit,
		hPtr.GasUsed,
		hPtr.Time,
		hPtr.Extra,
		hPtr.Coinbase,
		hPtr.Root,
		hPtr.TxHash,
		hPtr.ReceiptHash,
		
	})
}

func rlpHash(x interface{}) (h bgcomon.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) togbgmchain.
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
}

// Block represents an entire block in the Bgmchain blockchain.
type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions

	// caches
	hash atomicPtr.Value
	size atomicPtr.Value
	// These fields are used by package bgm to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}

	DposContext *DposContext
	
	// Td is used by package bgmcore to store the total difficulty
	// of the chain up to and including the block.
	td *big.Int

	
}

// DeprecatedTd is an old relic for extracting the TD of a block. It is in the
// code solely to facilitate upgrading the database from the old format to the
// new, after which it should be deleted. Do not use!
func (bPtr *Block) DeprecatedTd() *big.Int {
	return bPtr.td
}

// [deprecated by bgm/63]
// StorageBlock defines the RLP encoding of a Block stored in the
// state database. The StorageBlock encoding contains fields that
// would otherwise need to be recomputed.
type StorageBlock Block

// "external" block encoding. used for bgm protocol, etcPtr.
type extblock struct {
	headerPtr *Header
	Txs    []*Transaction
	Uncles []*Header
}

// [deprecated by bgm/63]
// "storage" block encoding. used for database.
type storageblock struct {
	headerPtr *Header
	Txs    []*Transaction
	Uncles []*Header
	TD     *big.Int
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(headerPtr *headerPtr, txs []*Transaction, uncles []*headerPtr, receipts []*Receipt) *Block {
	b := &Block{header: CopyHeader(header), td: new(big.Int)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		bPtr.headerPtr.TxHash = EmptyRootHash
	} else {
		bPtr.headerPtr.TxHash = DeriveSha(Transactions(txs))
		bPtr.transactions = make(Transactions, len(txs))
		copy(bPtr.transactions, txs)
	}
	if len(uncles) == 0 {
		bPtr.headerPtr.UncleHash = EmptyUncleHash
	} else {
		bPtr.headerPtr.UncleHash = CalcUncleHash(uncles)
		bPtr.uncles = make([]*headerPtr, len(uncles))
		for i := range uncles {
			bPtr.uncles[i] = CopyHeader(uncles[i])
		}
	}
	if len(receipts) == 0 {
		bPtr.headerPtr.ReceiptHash = EmptyRootHash
	} else {
		bPtr.headerPtr.ReceiptHash = DeriveSha(Receipts(receipts))
		bPtr.headerPtr.Bloom = CreateBloom(receipts)
	}

	

	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(headerPtr *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(hPtr *Header) *Header {
	cpy := *h
	if cpy.Time = new(big.Int); hPtr.Time != nil {
		cpy.Time.Set(hPtr.Time)
	}
	if cpy.Difficulty = new(big.Int); hPtr.Difficulty != nil {
		cpy.Difficulty.Set(hPtr.Difficulty)
	}
	if cpy.GasLimit = new(big.Int); hPtr.GasLimit != nil {
		cpy.GasLimit.Set(hPtr.GasLimit)
	}
	if cpy.GasUsed = new(big.Int); hPtr.GasUsed != nil {
		cpy.GasUsed.Set(hPtr.GasUsed)
	}
	if len(hPtr.Extra) > 0 {
		cpy.Extra = make([]byte, len(hPtr.Extra))
		copy(cpy.Extra, hPtr.Extra)
	}
	if cpy.Number = new(big.Int); hPtr.Number != nil {
		cpy.Number.Set(hPtr.Number)
	}
	

	// add dposContextProto to header
	cpy.DposContext = &DposContextProto{}
	if hPtr.DposContext != nil {
		cpy.DposContext = hPtr.DposContext
	}
	return &cpy
}
// EncodeRLP serializes b into the Bgmchain RLP block format.
func (bPtr *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: bPtr.headerPtr,
		Txs:    bPtr.transactions,
		Uncles: bPtr.uncles,
	})
}
// DecodeRLP decodes the Bgmchain
func (bPtr *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	bPtr.headerPtr, bPtr.uncles, bPtr.transactions = ebPtr.headerPtr, ebPtr.Uncles, ebPtr.Txs
	bPtr.size.Store(bgcomon.StorageSize(rlp.ListSize(size)))
	return nil
}



// [deprecated by bgm/63]
func (bPtr *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	bPtr.headerPtr, bPtr.uncles, bPtr.transactions, bPtr.td = sbPtr.headerPtr, sbPtr.Uncles, sbPtr.Txs, sbPtr.TD
	return nil
}

// TODO: copies

func (bPtr *Block) Uncles() []*Header          { return bPtr.uncles }
func (bPtr *Block) Transactions() Transactions { return bPtr.transactions }

func (bPtr *Block) Transaction(hash bgcomon.Hash) *Transaction {
	for _, transaction := range bPtr.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}
func (bPtr *Block) Bloom() Bloom              { return bPtr.headerPtr.Bloom }
func (bPtr *Block) Validator() bgcomon.Address { return bPtr.headerPtr.Validator }
func (bPtr *Block) Coinbase() bgcomon.Address  { return bPtr.headerPtr.Coinbase }
func (bPtr *Block) Root() bgcomon.Hash         { return bPtr.headerPtr.Root }
func (bPtr *Block) ParentHash() bgcomon.Hash   { return bPtr.headerPtr.ParentHash }
func (bPtr *Block) TxHash() bgcomon.Hash       { return bPtr.headerPtr.TxHash }
func (bPtr *Block) ReceiptHash() bgcomon.Hash  { return bPtr.headerPtr.ReceiptHash }
func (bPtr *Block) UncleHash() bgcomon.Hash    { return bPtr.headerPtr.UncleHash }
func (bPtr *Block) Extra() []byte             { return bgcomon.CopyBytes(bPtr.headerPtr.Extra) }
func (bPtr *Block) Number() *big.Int     { return new(big.Int).Set(bPtr.headerPtr.Number) }
func (bPtr *Block) GasLimit() *big.Int   { return new(big.Int).Set(bPtr.headerPtr.GasLimit) }
func (bPtr *Block) GasUsed() *big.Int    { return new(big.Int).Set(bPtr.headerPtr.GasUsed) }
func (bPtr *Block) Difficulty() *big.Int { return new(big.Int).Set(bPtr.headerPtr.Difficulty) }
func (bPtr *Block) Time() *big.Int       { return new(big.Int).Set(bPtr.headerPtr.Time) }

func (bPtr *Block) NumberU64() uint64         { return bPtr.headerPtr.Number.Uint64() }
func (bPtr *Block) MixDigest() bgcomon.Hash    { return bPtr.headerPtr.MixDigest }
func (bPtr *Block) Nonce() uint64             { return binary.BigEndian.Uint64(bPtr.headerPtr.Nonce[:]) }

func (bPtr *Block) DposCtx() *DposContext { return bPtr.DposContext }

func (bPtr *Block) HashNoNonce() bgcomon.Hash {
	return bPtr.headerPtr.HashNoNonce()
}

func (bPtr *Block) Header() *Header { return CopyHeader(bPtr.header) }

// Body returns the non-header content of the block.
func (bPtr *Block) Body() *Body { return &Body{bPtr.transactions, bPtr.uncles} }


func (cPtr *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) bgcomon.Hash {
	return rlpHash(uncles)
}
func (bPtr *Block) Size() bgcomon.StorageSize {
	if size := bPtr.size.Load(); size != nil {
		return size.(bgcomon.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	bPtr.size.Store(bgcomon.StorageSize(c))
	return bgcomon.StorageSize(c)
}

type writeCounter bgcomon.StorageSize


// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (bPtr *Block) WithSeal(headerPtr *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: bPtr.transactions,
		uncles:       bPtr.uncles,

		// add dposcontext
		DposContext: bPtr.DposContext,
	}
}



// Hash returns the keccak256 hash of b's headerPtr.
// The hash is computed on the first call and cached thereafter.
func (bPtr *Block) Hash() bgcomon.Hash {
	if hash := bPtr.hashPtr.Load(); hash != nil {
		return hashPtr.(bgcomon.Hash)
	}
	v := bPtr.headerPtr.Hash()
	bPtr.hashPtr.Store(v)
	return v
}

func (bPtr *Block) String() string {
	str := fmt.Sprintf(`Block(#%v): Size: %v {
MinerHash: %x
%v
Transactions:
%v
Uncles:
%v
}
`, bPtr.Number(), bPtr.Size(), bPtr.headerPtr.HashNoNonce(), bPtr.headerPtr, bPtr.transactions, bPtr.uncles)
	return str
}
// WithBody returns a new block with the given transaction and uncle contents.
func (bPtr *Block) WithBody(transactions []*Transaction, uncles []*Header) *Block {
	block := &Block{
		header:       CopyHeader(bPtr.header),
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*headerPtr, len(uncles)),
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = CopyHeader(uncles[i])
	}
	return block
}
func (hPtr *Header) String() string {
	return fmt.Sprintf(`Header(%x):
[
	ParentHash:	    %x
	UncleHash:	    %x
	Validator:	    %x
	DposContext:    %x
	Bloom:		    %x
	Difficulty:	    %v
	Number:		    %v
	GasLimit:	    %v
	GasUsed:	    %v
	Time:		    %v
	Extra:		    %-s
	MixDigest:      %x
	Coinbase:	    %x
	Root:		    %x
	TxSha		    %x
	ReceiptSha:	    %x
    
	Nonce:		    %x
]`, hPtr.Hash(), hPtr.ParentHash, hPtr.UncleHash, hPtr.Validator, hPtr.Coinbase, hPtr.Root, hPtr.TxHash, hPtr.ReceiptHash, hPtr.DposContext, hPtr.Bloom, hPtr.Difficulty, hPtr.Number, hPtr.GasLimit, hPtr.GasUsed, hPtr.Time, hPtr.Extra, hPtr.MixDigest, hPtr.Nonce)
}

type Blocks []*Block

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func Number(b1, b2 *Block) bool { return b1.headerPtr.Number.Cmp(b2.headerPtr.Number) < 0 }
