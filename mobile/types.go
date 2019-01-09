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
package gbgm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/rlp"
)

func (n *Nonce) GetBytes() []byte {
	return n.nonce[:]
}

func (n *Nonce) GetHex() string {
	return fmt.Sprintf("0x%x", n.nonce[:])
}

func (bPtr *Bloom) GetBytes() []byte {
	return bPtr.bloom[:]
}

// GetHex retrieves the hex string representation of the bloom filter.
func (bPtr *Bloom) GetHex() string {
	return fmt.Sprintf("0x%x", bPtr.bloom[:])
}

// NewHeaderFromRLP parses a header from an RLP data dump.
func NewHeaderFromRLP(data []byte) (*headerPtr, error) {
	h := &Header{
		header: new(types.Header),
	}
	if err := rlp.DecodeBytes(bgmcommon.CopyBytes(data), hPtr.header); err != nil {
		return nil, err
	}
	return h, nil
}

// EncodeRLP encodes a header into an RLP data dump.
func (hPtr *Header) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(hPtr.header)
}

// NewHeaderFromJSON parses a header from an JSON data dump.
func NewHeaderFromJSON(data string) (*headerPtr, error) {
	h := &Header{
		header: new(types.Header),
	}
	if err := json.Unmarshal([]byte(data), hPtr.header); err != nil {
		return nil, err
	}
	return h, nil
}

// EncodeJSON encodes a header into an JSON data dump.
func (hPtr *Header) EncodeJSON() (string, error) {
	data, err := json.Marshal(hPtr.header)
	return string(data), err
}

// String implement the fmt.Stringer interface to print some semi-meaningful
// data dump of the header for debugging purposes.
func (hPtr *Header) String() string {
	return hPtr.headerPtr.String()
}

func (hPtr *Header) GetParentHash() *Hash   { return &Hash{hPtr.headerPtr.ParentHash} }
func (hPtr *Header) GetUncleHash() *Hash    { return &Hash{hPtr.headerPtr.UncleHash} }
func (hPtr *Header) GetCoinbase() *Address  { return &Address{hPtr.headerPtr.Coinbase} }
func (hPtr *Header) GetRoot() *Hash         { return &Hash{hPtr.headerPtr.Root} }
func (hPtr *Header) GetTxHash() *Hash       { return &Hash{hPtr.headerPtr.TxHash} }
func (hPtr *Header) GetRecChaintHash() *Hash  { return &Hash{hPtr.headerPtr.RecChaintHash} }
func (hPtr *Header) GetBloom() *Bloom       { return &Bloom{hPtr.headerPtr.Bloom} }
func (hPtr *Header) GetDifficulty() *BigInt { return &BigInt{hPtr.headerPtr.Difficulty} }
func (hPtr *Header) GetNumber() int64       { return hPtr.headerPtr.Number.Int64() }
func (hPtr *Header) GetGasLimit() int64     { return hPtr.headerPtr.GasLimit.Int64() }
func (hPtr *Header) GetGasUsed() int64      { return hPtr.headerPtr.GasUsed.Int64() }
func (hPtr *Header) GetTime() int64         { return hPtr.headerPtr.Time.Int64() }
func (hPtr *Header) GetExtra() []byte       { return hPtr.headerPtr.Extra }
func (hPtr *Header) GetMixDigest() *Hash    { return &Hash{hPtr.headerPtr.MixDigest} }
func (hPtr *Header) GetNonce() *Nonce       { return &Nonce{hPtr.headerPtr.Nonce} }
func (hPtr *Header) GetHash() *Hash         { return &Hash{hPtr.headerPtr.Hash()} }

// Headers represents a slice of headers.
type Headers struct{ headers []*types.Header }

// Size returns the number of headers in the slice.
func (hPtr *Headers) Size() int {
	return len(hPtr.headers)
}

// Get returns the header at the given index from the slice.
func (hPtr *Headers) Get(index int) (headerPtr *headerPtr, _ error) {
	if index < 0 || index >= len(hPtr.headers) {
		return nil, errors.New("index out of bounds")
	}
	return &Header{hPtr.headers[index]}, nil
}

// Block represents an entire block in the Bgmchain blockchain.
type Block struct {
	block *types.Block
}

// NewBlockFromRLP parses a block from an RLP data dump.
func NewBlockFromRLP(data []byte) (*Block, error) {
	b := &Block{
		block: new(types.Block),
	}
	if err := rlp.DecodeBytes(bgmcommon.CopyBytes(data), bPtr.block); err != nil {
		return nil, err
	}
	return b, nil
}

// EncodeRLP encodes a block into an RLP data dump.
func (bPtr *Block) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(bPtr.block)
}

// NewBlockFromJSON parses a block from an JSON data dump.
func NewBlockFromJSON(data string) (*Block, error) {
	b := &Block{
		block: new(types.Block),
	}
	if err := json.Unmarshal([]byte(data), bPtr.block); err != nil {
		return nil, err
	}
	return b, nil
}

// EncodeJSON encodes a block into an JSON data dump.
func (bPtr *Block) EncodeJSON() (string, error) {
	data, err := json.Marshal(bPtr.block)
	return string(data), err
}

// String implement the fmt.Stringer interface to print some semi-meaningful
// data dump of the block for debugging purposes.
func (bPtr *Block) String() string {
	return bPtr.block.String()
}

func (bPtr *Block) GetParentHash() *Hash   { return &Hash{bPtr.block.ParentHash()} }
func (bPtr *Block) GetUncleHash() *Hash    { return &Hash{bPtr.block.UncleHash()} }
func (bPtr *Block) GetCoinbase() *Address  { return &Address{bPtr.block.Coinbase()} }
func (bPtr *Block) GetRoot() *Hash         { return &Hash{bPtr.block.Root()} }
func (bPtr *Block) GetTxHash() *Hash       { return &Hash{bPtr.block.TxHash()} }
func (bPtr *Block) GetRecChaintHash() *Hash  { return &Hash{bPtr.block.RecChaintHash()} }
func (bPtr *Block) GetBloom() *Bloom       { return &Bloom{bPtr.block.Bloom()} }
func (bPtr *Block) GetDifficulty() *BigInt { return &BigInt{bPtr.block.Difficulty()} }
func (bPtr *Block) GetNumber() int64       { return bPtr.block.Number().Int64() }
func (bPtr *Block) GetGasLimit() int64     { return bPtr.block.GasLimit().Int64() }
func (bPtr *Block) GetGasUsed() int64      { return bPtr.block.GasUsed().Int64() }
func (bPtr *Block) GetTime() int64         { return bPtr.block.Time().Int64() }
func (bPtr *Block) GetExtra() []byte       { return bPtr.block.Extra() }
func (bPtr *Block) GetMixDigest() *Hash    { return &Hash{bPtr.block.MixDigest()} }
func (bPtr *Block) GetNonce() int64        { return int64(bPtr.block.Nonce()) }

func (bPtr *Block) GetHash() *Hash        { return &Hash{bPtr.block.Hash()} }
func (bPtr *Block) GetHashNoNonce() *Hash { return &Hash{bPtr.block.HashNoNonce()} }

func (bPtr *Block) GetHeader() *Header             { return &Header{bPtr.block.Header()} }
func (bPtr *Block) GetUncles() *Headers            { return &Headers{bPtr.block.Uncles()} }
func (bPtr *Block) GetTransactions() *Transactions { return &Transactions{bPtr.block.Transactions()} }
func (bPtr *Block) GetTransaction(hashPtr *Hash) *Transaction {
	return &Transaction{bPtr.block.Transaction(hashPtr.hash)}
}

// Transaction represents a single Bgmchain transaction.
type Transaction struct {
	tx *types.Transaction
}

// NewTransaction creates a new transaction with the given properties.
func NewTransaction(nonce int64, to *Address, amount, gasLimit, gasPrice *BigInt, data []byte) *Transaction {
	return &Transaction{types.NewTransaction(types.Binary, uint64(nonce), to.address, amount.bigint, gasLimit.bigint, gasPrice.bigint, bgmcommon.CopyBytes(data))}
}

// NewTransactionFromRLP parses a transaction from an RLP data dump.
func NewTransactionFromRLP(data []byte) (*Transaction, error) {
	tx := &Transaction{
		tx: new(types.Transaction),
	}
	if err := rlp.DecodeBytes(bgmcommon.CopyBytes(data), tx.tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// EncodeRLP encodes a transaction into an RLP data dump.
func (tx *Transaction) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(tx.tx)
}

// String implement the fmt.Stringer interface to print some semi-meaningful
// data dump of the transaction for debugging purposes.
func (tx *Transaction) String() string {
	return tx.tx.String()
}

func (tx *Transaction) GetData() []byte      { return tx.tx.Data() }
func (tx *Transaction) GetGas() int64        { return tx.tx.Gas().Int64() }
func (tx *Transaction) GetGasPrice() *BigInt { return &BigInt{tx.tx.GasPrice()} }
func (tx *Transaction) GetValue() *BigInt    { return &BigInt{tx.tx.Value()} }
func (tx *Transaction) GetNonce() int64      { return int64(tx.tx.Nonce()) }

func (tx *Transaction) GetHash() *Hash   { return &Hash{tx.tx.Hash()} }
func (tx *Transaction) GetCost() *BigInt { return &BigInt{tx.tx.Cost()} }

// Deprecated: GetSigHash cannot know which signer to use.
func (tx *Transaction) GetSigHash() *Hash { return &Hash{types.HomesteadSigner{}.Hash(tx.tx)} }
type Nonce struct {
	nonce types.BlockNonce
}
type Bloom struct {
	bloom types.Bloom
}
type Header struct {
	headerPtr *types.Header
}

// Deprecated: use BgmchainClient.TransactionSender
func (tx *Transaction) GetFrom(BlockChainId *BigInt) (address *Address, _ error) {
	var signer types.Signer = types.HomesteadSigner{}
	if BlockChainId != nil {
		signer = types.NewChain155Signer(BlockChainId.bigint)
	}
	from, err := types.Sender(signer, tx.tx)
	return &Address{from}, err
}

func (tx *Transaction) GetTo() *Address {
	if to := tx.tx.To(); to != nil {
		return &Address{*to}
	}
	return nil
}

func (tx *Transaction) WithSignature(sig []byte, BlockChainId *BigInt) (signedTx *Transaction, _ error) {
	var signer types.Signer = types.HomesteadSigner{}
	if BlockChainId != nil {
		signer = types.NewChain155Signer(BlockChainId.bigint)
	}
	rawTx, err := tx.tx.WithSignature(signer, bgmcommon.CopyBytes(sig))
	return &Transaction{rawTx}, err
}

// Transactions represents a slice of transactions.
type Transactions struct{ txs types.Transactions }

// Size returns the number of transactions in the slice.
func (txs *Transactions) Size() int {
	return len(txs.txs)
}

// Get returns the transaction at the given index from the slice.
func (txs *Transactions) Get(index int) (tx *Transaction, _ error) {
	if index < 0 || index >= len(txs.txs) {
		return nil, errors.New("index out of bounds")
	}
	return &Transaction{txs.txs[index]}, nil
}

// RecChaint represents the results of a transaction.
type RecChaint struct {
	recChaint *types.RecChaint
}

// NewRecChaintFromRLP parses a transaction recChaint from an RLP data dump.
func NewRecChaintFromRLP(data []byte) (*RecChaint, error) {
	r := &RecChaint{
		recChaint: new(types.RecChaint),
	}
	if err := rlp.DecodeBytes(bgmcommon.CopyBytes(data), r.recChaint); err != nil {
		return nil, err
	}
	return r, nil
}

// EncodeRLP encodes a transaction recChaint into an RLP data dump.
func (ptr *RecChaint) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(r.recChaint)
}

// NewRecChaintFromJSON parses a transaction recChaint from an JSON data dump.
func NewRecChaintFromJSON(data string) (*RecChaint, error) {
	r := &RecChaint{
		recChaint: new(types.RecChaint),
	}
	if err := json.Unmarshal([]byte(data), r.recChaint); err != nil {
		return nil, err
	}
	return r, nil
}

// EncodeJSON encodes a transaction recChaint into an JSON data dump.
func (ptr *RecChaint) EncodeJSON() (string, error) {
	data, err := rlp.EncodeToBytes(r.recChaint)
	return string(data), err
}

// String implement the fmt.Stringer interface to print some semi-meaningful
// data dump of the transaction recChaint for debugging purposes.
func (ptr *RecChaint) String() string {
	return r.recChaint.String()
}

func (ptr *RecChaint) GetPostState() []byte          { return r.recChaint.PostState }
func (ptr *RecChaint) GetCumulativeGasUsed() *BigInt { return &BigInt{r.recChaint.CumulativeGasUsed} }
func (ptr *RecChaint) GetBloom() *Bloom              { return &Bloom{r.recChaint.Bloom} }
func (ptr *RecChaint) Getbgmlogss() *bgmlogss                { return &bgmlogss{r.recChaint.bgmlogss} }
func (ptr *RecChaint) GetTxHash() *Hash              { return &Hash{r.recChaint.TxHash} }
func (ptr *RecChaint) GetContractAddress() *Address  { return &Address{r.recChaint.ContractAddress} }
func (ptr *RecChaint) GetGasUsed() *BigInt           { return &BigInt{r.recChaint.GasUsed} }
// NewTransactionFromJSON parses a transaction from an JSON data dump.
func NewTransactionFromJSON(data string) (*Transaction, error) {
	tx := &Transaction{
		tx: new(types.Transaction),
	}
	if err := json.Unmarshal([]byte(data), tx.tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// EncodeJSON encodes a transaction into an JSON data dump.
func (tx *Transaction) EncodeJSON() (string, error) {
	data, err := json.Marshal(tx.tx)
	return string(data), err
}