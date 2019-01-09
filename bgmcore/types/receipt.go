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
	"bytes"
	"fmt"
	

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/rlp"
)

//go:generate gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
)

const (
	// ReceiptStatusFailed is the status code of a transaction if execution failed.
	ReceiptStatusFailed = uint(0)

	// ReceiptStatusSuccessful is the status code of a transaction if execution succeeded.
	ReceiptStatusSuccessful = uint(1)
)

// Receipt represents the results of a transaction.
type Receipt struct {

	// Implementation fields (don't reorder!)
	TxHash          bgmcommon.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress bgmcommon.Address `json:"contractAddress"`
	GasUsed         *big.Int       `json:"gasUsed" gencodec:"required"`
	// Consensus fields
	PostState         []byte   `json:"root"`
	Status            uint     `json:"status"`
	CumulativeGasUsed *big.Int `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             Bloom    `json:"bgmlogssBloom"         gencodec:"required"`
	bgmlogss              []*bgmlogs   `json:"bgmlogss"              gencodec:"required"`

	
}

type receiptMarshaling struct {
	PostState         hexutil.Bytes
	Status            hexutil.Uint
	CumulativeGasUsed *hexutil.Big
	GasUsed           *hexutil.Big
}


type receiptStorageRLP struct {
	TxHash            bgmcommon.Hash
	ContractAddress   bgmcommon.Address
	bgmlogss              []*bgmlogsForStorage
	GasUsed           *big.Int
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	
}

// receiptRLP is the consensus encoding of a receipt.
type receiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	bgmlogss              []*bgmlogs
}


// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed *big.Int) *Receipt {
	r := &Receipt{PostState: bgmcommon.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumulativeGasUsed)}
	if failed {
		r.Status = ReceiptStatusFailed
	} else {
		r.Status = ReceiptStatusSuccessful
	}
	return r
}
// Receipts is a wrapper around a Receipt array to implement DerivableList.
type Receipts []*Receipt



// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP streamPtr. If no post state is present, byzantium fork is assumed.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.bgmlogss})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP streamPtr.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	var dec receiptRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	if err := r.setStatus(decPtr.PostStateOrStatus); err != nil {
		return err
	}
	r.CumulativeGasUsed, r.Bloom, r.bgmlogss = decPtr.CumulativeGasUsed, decPtr.Bloom, decPtr.bgmlogss
	return nil
}

// Len returns the number of receipts in this list.
func (r Receipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r Receipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}

func (r *Receipt) setStatus(postStateOrStatus []byte) error {
	switch {
	case bytes.Equal(postStateOrStatus, receiptStatusSuccessfulRLP):
		r.Status = ReceiptStatusSuccessful
	case bytes.Equal(postStateOrStatus, receiptStatusFailedRLP):
		r.Status = ReceiptStatusFailed
	case len(postStateOrStatus) == len(bgmcommon.Hash{}):
		r.PostState = postStateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", postStateOrStatus)
	}
	return nil
}



// String implements the Stringer interface.
func (r *Receipt) String() string {
	if len(r.PostState) == 0 {
		return fmt.Sprintf("receipt{status=%-d cgas=%v bloom=%x bgmlogss=%v}", r.Status, r.CumulativeGasUsed, r.Bloom, r.bgmlogss)
	}
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x bgmlogss=%v}", r.PostState, r.CumulativeGasUsed, r.Bloom, r.bgmlogss)
}

func (r *Receipt) statusEncoding() []byte {
	if len(r.PostState) == 0 {
		if r.Status == ReceiptStatusFailed {
			return receiptStatusFailedRLP
		}
		return receiptStatusSuccessfulRLP
	}
	return r.PostState
}

// entire content of a receipt, as opposed to only the consensus fields originally.
type ReceiptForStorage Receipt


// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var dec receiptStorageRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	if err := (*Receipt)(r).setStatus(decPtr.PostStateOrStatus); err != nil {
		return err
	}
	// Assign the consensus fields
	r.CumulativeGasUsed, r.Bloom = decPtr.CumulativeGasUsed, decPtr.Bloom
	r.bgmlogss = make([]*bgmlogs, len(decPtr.bgmlogss))
	for i, bgmlogs := range decPtr.bgmlogss {
		r.bgmlogss[i] = (*bgmlogs)(bgmlogs)
	}
	// Assign the implementation fields
	r.TxHash, r.ContractAddress, r.GasUsed = decPtr.TxHash, decPtr.ContractAddress, decPtr.GasUsed
	return nil
}

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP streamPtr.
func (r *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	enc := &receiptStorageRLP{
		PostStateOrStatus: (*Receipt)(r).statusEncoding(),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             r.Bloom,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		bgmlogss:              make([]*bgmlogsForStorage, len(r.bgmlogss)),
		GasUsed:           r.GasUsed,
	}
	for i, bgmlogs := range r.bgmlogss {
		encPtr.bgmlogss[i] = (*bgmlogsForStorage)(bgmlogs)
	}
	return rlp.Encode(w, enc)
}
