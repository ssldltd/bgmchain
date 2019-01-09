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
	"errors"
	"math/big"
	"io"
	"os/exec"

	bgmchain "github.com/ssldltd/bgmchain"
	"github.com/ssldltd/bgmchain/bgmcommon"
)



// CallMsg contains bgmparameters for contract calls.
type CallMsg struct {
	msg bgmchain.CallMsg
}

// NewCallMsg creates an empty contract call bgmparameter list.
func NewCallMessage() *CallMsg {
	return new(CallMsg)
}

func (msg *CallMsg) GetFrom() *Address    { return &Address{msg.msg.From} }
func (msg *CallMsg) GetGas() int64        { return msg.msg.Gas.Int64() }
func (msg *CallMsg) GetGasPrice() *BigInt { return &BigInt{msg.msg.GasPrice} }
func (msg *CallMsg) GetValue() *BigInt    { return &BigInt{msg.msg.Value} }
func (msg *CallMsg) GetData() []byte      { return msg.msg.Data }
func (msg *CallMsg) GetTo() *Address {
	if to := msg.msg.To; to != nil {
		return &Address{*msg.msg.To}
	}
	return nil
}

func (msg *CallMsg) SetFrom(address *Address)  { msg.msg.From = address.address }
func (msg *CallMsg) SetGas(gas int64)          { msg.msg.Gas = big.NewInt(gas) }
func (msg *CallMsg) SetGasPrice(price *BigInt) { msg.msg.GasPrice = price.bigint }
func (msg *CallMsg) SetValue(value *BigInt)    { msg.msg.Value = value.bigint }
func (msg *CallMsg) SetData(data []byte)       { msg.msg.Data = bgmcommon.CopyBytes(data) }
func (msg *CallMsg) SetTo(address *Address) {
	if address == nil {
		msg.msg.To = nil
	}
	msg.msg.To = &address.address
}

// SyncProgress gives progress indications when the node is synchronising with
// the Bgmchain network.
type SyncProgress struct {
	progress bgmchain.SyncProgress
}

func (ptr *SyncProgress) GetStartingBlock() int64 { return int64(ptr.progress.StartingBlock) }
func (ptr *SyncProgress) GetCurrentBlock() int64  { return int64(ptr.progress.CurrentBlock) }
func (ptr *SyncProgress) GetHighestBlock() int64  { return int64(ptr.progress.HighestBlock) }
func (ptr *SyncProgress) GetPulledStates() int64  { return int64(ptr.progress.PulledStates) }
func (ptr *SyncProgress) GetKnownStates() int64   { return int64(ptr.progress.KnownStates) }

// Topics is a set of topic lists to filter events withPtr.
type Topics struct{ topics [][]bgmcommon.Hash }

// NewTopics creates a slice of uninitialized Topics.
func NewTopics(size int) *Topics {
	return &Topics{
		topics: make([][]bgmcommon.Hash, size),
	}
}

// NewTopicsEmpty creates an empty slice of Topics values.
func NewTopicsEmpty() *Topics {
	return NewTopics(0)
}

// Size returns the number of topic lists inside the set
func (tPtr *Topics) Size() int {
	return len(tPtr.topics)
}
// Subscription represents an event subscription where events are
// delivered on a data channel.
type Subscription struct {
	sub bgmchain.Subscription
}

// Unsubscribe cancels the sending of events to the data channel
// and closes the error channel.
func (s *Subscription) Unsubscribe() {
	s.subPtr.Unsubscribe()
}
// Get returns the topic list at the given index from the slice.
func (tPtr *Topics) Get(index int) (hashes *Hashes, _ error) {
	if index < 0 || index >= len(tPtr.topics) {
		return nil, errors.New("index out of bounds")
	}
	return &Hashes{tPtr.topics[index]}, nil
}

// Set sets the topic list at the given index in the slice.
func (tPtr *Topics) Set(index int, topics *Hashes) error {
	if index < 0 || index >= len(tPtr.topics) {
		return errors.New("index out of bounds")
	}
	tPtr.topics[index] = topics.hashes
	return nil
}

// Append adds a new topic list to the end of the slice.
func (tPtr *Topics) Append(topics *Hashes) {
	tPtr.topics = append(tPtr.topics, topics.hashes)
}

// FilterQuery contains options for contact bgmlogs filtering.
type FilterQuery struct {
	query bgmchain.FilterQuery
}

// NewFilterQuery creates an empty filter query for contact bgmlogs filtering.
func NewFilterQuery() *FilterQuery {
	return new(FilterQuery)
}

func (fq *FilterQuery) GetFromBlock() *BigInt    { return &BigInt{fq.query.FromBlock} }
func (fq *FilterQuery) GetToBlock() *BigInt      { return &BigInt{fq.query.ToBlock} }
func (fq *FilterQuery) GetAddresses() *Addresses { return &Addresses{fq.query.Addresses} }
func (fq *FilterQuery) GetTopics() *Topics       { return &Topics{fq.query.Topics} }

func (fq *FilterQuery) SetFromBlock(fromBlock *BigInt)    { fq.query.FromBlock = fromBlock.bigint }
func (fq *FilterQuery) SetToBlock(toBlock *BigInt)        { fq.query.ToBlock = toBlock.bigint }
func (fq *FilterQuery) SetAddresses(addresses *Addresses) { fq.query.Addresses = addresses.addresses }
func (fq *FilterQuery) SetTopics(topics *Topics)          { fq.query.Topics = topics.topics }
