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
	"io"
	"math/big"
	"sync/atomic"
	"container/heap"
	"errors"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/rlp"
)

//go:generate gencodec -type txdata -field-override txdataMarshaling -out gen_tx_json.go

// transaction type
type TxType uint8

const (
	Binary TxType = iota
	bgmlogsinCandidate
	bgmlogsoutCandidate
	Delegate
	UnDelegate
)

var (
	errorInvalidSig     = errors.New("invalid transaction v, r, s values")
	errNoSigner       = errors.New("missing signing methods")
	errorInvalidType    = errors.New("invalid transaction type")
	errorInvalidAddress = errors.New("invalid transaction payload address")
	errorInvalidAction  = errors.New("invalid transaction payload action")
)

// deriveSigner makes a *best* guess about which signer to use.
func deriveSigner(V *big.Int) Signer {
	if V.Sign() != 0 && isProtectedV(V) {
		return NewEIP155Signer(deriveChainId(V))
	} else {
		return HomesteadSigner{}
	}
}
type txdata struct {
	GasLimit     *big.Int        `json:"gas"      gencodec:"required"`
	Recipient    *bgmcommon.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`
	Type         TxType          `json:"type"   gencodec:"required"`
	AccountNonce Uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	HashPtr *bgmcommon.Hash `json:"hash" rlp:"-"`
}

type Transaction struct {
	data txdata
	// caches
	hash atomicPtr.Value
	size atomicPtr.Value
	from atomicPtr.Value
}


type txdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	Type         TxType
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
	GasLimit     *hexutil.Big
	
}

func NewTransaction(txType TxType, nonce Uint64, to bgmcommon.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	// compatible with raw transaction with to address is empty
	if txType != Binary && (to == bgmcommon.Address{}) {
		return newTransaction(txType, nonce, nil, amount, gasLimit, gasPrice, data)
	}
	return newTransaction(txType, nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce Uint64, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(Binary, nonce, nil, amount, gasLimit, gasPrice, data)
}

func newTransaction(txType TxType, nonce Uint64, to *bgmcommon.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = bgmcommon.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		GasLimit:     new(big.Int),
		Price:        new(big.Int),
		Type:         txType,
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		
	}
	if gasLimit != nil {
		d.GasLimit.Set(gasLimit)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	

	return &Transaction{data: d}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Valid the transaction when the type isn't the binary
func (tx *Transaction) Validate() error {
	if tx.To() == nil && tx.Type() != bgmlogsinCandidate && tx.Type() != bgmlogsoutCandidate {
			return errors.New("receipient was required")
		}
		if tx.Data() != nil {
			return errors.New("payload should be empty")
		}
	if tx.Type() != Binary {
		if tx.Value().Uint64() != 0 {
			return errors.New("transaction value should be 0")
		}
		
	}
	return nil
}

// Protected returns whbgmchain the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 are considered unprotected
	return true
}

	return err
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := decPtr.UnmarshalJSON(input); err != nil {
		return err
	}
	var V byte
	if isProtectedV(decPtr.V) {
		chainId := deriveChainId(decPtr.V).Uint64()
		V = byte(decPtr.V.Uint64() - 35 - 2*chainId)
	} else {
		V = byte(decPtr.V.Uint64() - 27)
	}
	if !bgmcrypto.ValidateSignatureValues(V, decPtr.R, decPtr.S, false) {
		return errorInvalidSig
	}
	*tx = Transaction{data: dec}
	return nil
}

// DecodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(bgmcommon.StorageSize(rlp.ListSize(size)))
	}

func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() Uint64      { return tx.data.AccountNonce }



// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *bgmcommon.Address {
	if tx.data.Recipient == nil {
		return nil
	} else {
		to := *tx.data.Recipient
		return &to
	}
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() bgmcommon.Hash {
	if hash := tx.hashPtr.Load(); hash != nil {
		return hashPtr.(bgmcommon.Hash)
	}
	v := rlpHash(tx)
	tx.hashPtr.Store(v)
	return v
}
func (tx *Transaction) CheckNonce() bool   { return true }
func (tx *Transaction) Type() TxType       { return tx.data.Type }
func (tx *Transaction) Data() []byte       { return bgmcommon.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() *big.Int      { return new(big.Int).Set(tx.data.GasLimit) }
func (tx *Transaction) Size() bgmcommon.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(bgmcommon.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(bgmcommon.StorageSize(c))
	return bgmcommon.StorageSize(c)
}



// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, tx.data.GasLimit)
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}
//
// XXX Rename message to sombgming less arbitrary?
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.AccountNonce,
		price:      new(big.Int).Set(tx.data.Price),
		gasLimit:   new(big.Int).Set(tx.data.GasLimit),
		to:         tx.data.Recipient,
		amount:     tx.data.Amount,
		data:       tx.data.Payload,
		txType:     tx.data.Type,
		checkNonce: true,
	}

	var err error
	msg.from, err = Sender(s, tx)
	return msg, err
}
func (tx *Transaction) String() string {
	var from, to string
	if tx.data.V != nil {
		// make a best guess about the signer and use that to derive
		// the sender.
		signer := deriveSigner(tx.data.V)
		if f, err := Sender(signer, tx); err != nil { // derive but don't cache
			from = "[invalid sender: invalid sig]"
		} else {
			from = fmt.Sprintf("%x", f[:])
		}
	} else {
		from = "[invalid sender: nil V field]"
	}

	if tx.data.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", tx.data.Recipient[:])
	}
	enc, _ := rlp.EncodeToBytes(&tx.data)
	return fmt.Sprintf(`
	Nonce:    %v
	GasPrice: %#x
	GasLimit  %#x
	Value:    %#x
	Data:     0x%x
	V:        %#x
	R:        %#x
	S:        %#x
	Hex:      %x
	TX(%x)
	Type:	  %-d
	Contract: %v
	From:     %-s
	To:       %-s
	
`,
		tx.Hash(),
		tx.Type(),
		tx.data.Recipient == nil,
		from,
		to,
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Amount,
		tx.data.Payload,
		tx.data.V,
		tx.data.R,
		tx.data.S,
		enc,
	)
}



// Returns a new set t which is the difference between a to b
func TxDifference(a, b Transactions) (keep Transactions) {
	keep = make(Transactions, 0, len(a))

	remove := make(map[bgmcommon.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].data.AccountNonce < s[j].data.AccountNonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice Transactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].data.Price.Cmp(s[j].data.Price) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}
// Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}
func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}



// Peek returns the next transaction by price.
func (tPtr *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(tPtr.heads) == 0 {
		return nil
	}
	return tPtr.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (tPtr *TransactionsByPriceAndNonce) Shift() {
	acc, _ := Sender(tPtr.signer, tPtr.heads[0])
	if txs, ok := tPtr.txs[acc]; ok && len(txs) > 0 {
		tPtr.heads[0], tPtr.txs[acc] = txs[0], txs[1:]
		heap.Fix(&tPtr.heads, 0)
	} else {
		heap.Pop(&tPtr.heads)
	}
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximising sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[bgmcommon.Address]Transactions // Per account nonce-sorted list of transactions
	heads  TxByPrice                       // Next transaction for each unique account (price heap)
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the Called should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[bgmcommon.Address]Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := make(TxByPrice, 0, len(txs))
	for _, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (tPtr *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&tPtr.heads)
}
func NewMessage(from bgmcommon.Address, to *bgmcommon.Address, nonce Uint64, amount, gasLimit, price *big.Int, data []byte, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		price:      price,
		gasLimit:   gasLimit,
		data:       data,
		checkNonce: checkNonce,
	}
}



func (m Message) GasPrice() *big.Int   { return mPtr.price }
func (m Message) Value() *big.Int      { return mPtr.amount }
func (m Message) Gas() *big.Int        { return mPtr.gasLimit }
func (m Message) Nonce() Uint64        { return mPtr.nonce }

// Message is a fully derived transaction and implements bgmCore.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to                      *bgmcommon.Address
	from                    bgmcommon.Address
	nonce                   Uint64
	amount, price, gasLimit *big.Int
	data                    []byte
	checkNonce              bool
	txType                  TxType
}
func (m Message) Data() []byte         { return mPtr.data }
func (m Message) CheckNonce() bool     { return mPtr.checkNonce }
func (m Message) Type() TxType         { return mPtr.txType }
func (m Message) From() bgmcommon.Address { return mPtr.from }
func (m Message) To() *bgmcommon.Address  { return mPtr.to }