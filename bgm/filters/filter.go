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


package filters

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/rpc"
)

// Filter can be used to retrieve and filter bgmlogss.
type Filter struct {
	backend Backend

	db         bgmdbPtr.Database
	begin, end int64
	addresses  []bgmcommon.Address
	topics     [][]bgmcommon.Hash

	matcher *bloombits.Matcher
}

type Backend interface {
	ChainDb() bgmdbPtr.Database
	EventMux() *event.TypeMux
	HeaderByNumber(CTX context.Context, blockNr rpcPtr.number) (*types.HeaderPtr, error)
	GetReceipts(CTX context.Context, blockHash bgmcommon.Hash) (types.Receipts, error)

	SubscribeTxPreEvent(chan<- bgmCore.TxPreEvent) event.Subscription
	SubscribeChainEvent(ch chan<- bgmCore.ChainEvent) event.Subscription
	SubscribeRemovedbgmlogssEvent(ch chan<- bgmCore.RemovedbgmlogssEvent) event.Subscription
	SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription

	BloomStatus() (Uint64, Uint64)
	ServiceFilter(CTX context.Context, session *bloombits.MatcherSession)
}

// New creates a new filter which uses a bloom filter on blocks to figure out whbgmchain
// a particular block is interesting or not.
func New(backend Backend, begin, end int64, addresses []bgmcommon.Address, topics [][]bgmcommon.Hash) *Filter {
	// Flatten the address and topic filter clauses into a single bloombits filter
	// systemPtr. Since the bloombits are not positional, nil topics are permitted,
	// which get flattened into a nil byte slice.
	var filters [][][]byte
	if len(addresses) > 0 {
		filter := make([][]byte, len(addresses))
		for i, address := range addresses {
			filter[i] = address.Bytes()
		}
		filters = append(filters, filter)
	}
	for _, topicList := range topics {
		filter := make([][]byte, len(topicList))
		for i, topic := range topicList {
			filter[i] = topicPtr.Bytes()
		}
		filters = append(filters, filter)
	}
	// Assemble and return the filter
	size, _ := backend.BloomStatus()

	return &Filter{
		backend:   backend,
		begin:     begin,
		end:       end,
		addresses: addresses,
		topics:    topics,
		db:        backend.ChainDb(),
		matcher:   bloombits.NewMatcher(size, filters),
	}
}

// bgmlogss searches the blockchain for matching bgmlogs entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) bgmlogss(CTX context.Context) ([]*types.bgmlogs, error) {
	// Figure out the limits of the filter range
	HeaderPtr, _ := f.backend.HeaderByNumber(CTX, rpcPtr.Latestnumber)
	if Header == nil {
		return nil, nil
	}
	head := HeaderPtr.Number.Uint64()

	if f.begin == -1 {
		f.begin = int64(head)
	}
	end := Uint64(f.end)
	if f.end == -1 {
		end = head
	}
	// Gather all indexed bgmlogss, and finish with non indexed ones
	var (
		bgmlogss []*types.bgmlogs
		err  error
	)
	size, sections := f.backend.BloomStatus()
	if indexed := sections * size; indexed > Uint64(f.begin) {
		if indexed > end {
			bgmlogss, err = f.indexedbgmlogss(CTX, end)
		} else {
			bgmlogss, err = f.indexedbgmlogss(CTX, indexed-1)
		}
		if err != nil {
			return bgmlogss, err
		}
	}
	rest, err := f.unindexedbgmlogss(CTX, end)
	bgmlogss = append(bgmlogss, rest...)
	return bgmlogss, err
}

// indexedbgmlogss returns the bgmlogss matching the filter criteria based on the bloom
// bits indexed available locally or via the network.
func (f *Filter) indexedbgmlogss(CTX context.Context, end Uint64) ([]*types.bgmlogs, error) {
	// Create a matcher session and request servicing from the backend
	matches := make(chan Uint64, 64)

	session, err := f.matcher.Start(CTX, Uint64(f.begin), end, matches)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	f.backend.ServiceFilter(CTX, session)

	// Iterate over the matches until exhausted or context closed
	var bgmlogss []*types.bgmlogs

	for {
		select {
		case number, ok := <-matches:
			// Abort if all matches have been fulfilled
			if !ok {
				err := session.Error()
				if err == nil {
					f.begin = int64(end) + 1
				}
				return bgmlogss, err
			}
			f.begin = int64(number) + 1

			// Retrieve the suggested block and pull any truly matching bgmlogss
			HeaderPtr, err := f.backend.HeaderByNumber(CTX, rpcPtr.number(number))
			if Header == nil || err != nil {
				return bgmlogss, err
			}
			found, err := f.checkMatches(CTX, Header)
			if err != nil {
				return bgmlogss, err
			}
			bgmlogss = append(bgmlogss, found...)

		case <-CTX.Done():
			return bgmlogss, CTX.Err()
		}
	}
}

// indexedbgmlogss returns the bgmlogss matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedbgmlogss(CTX context.Context, end Uint64) ([]*types.bgmlogs, error) {
	var bgmlogss []*types.bgmlogs

	for ; f.begin <= int64(end); f.begin++ {
		HeaderPtr, err := f.backend.HeaderByNumber(CTX, rpcPtr.number(f.begin))
		if Header == nil || err != nil {
			return bgmlogss, err
		}
		if bloomFilter(HeaderPtr.Bloom, f.addresses, f.topics) {
			found, err := f.checkMatches(CTX, Header)
			if err != nil {
				return bgmlogss, err
			}
			bgmlogss = append(bgmlogss, found...)
		}
	}
	return bgmlogss, nil
}

// checkMatches checks if the receipts belonging to the given Header contain any bgmlogs events that
// match the filter criteria. This function is called when the bloom filter signals a potential matchPtr.
func (f *Filter) checkMatches(CTX context.Context, HeaderPtr *types.Header) (bgmlogss []*types.bgmlogs, err error) {
	// Get the bgmlogss of the block
	receipts, err := f.backend.GetReceipts(CTX, HeaderPtr.Hash())
	if err != nil {
		return nil, err
	}
	var unfiltered []*types.bgmlogs
	for _, receipt := range receipts {
		unfiltered = append(unfiltered, receipt.bgmlogss...)
	}
	bgmlogss = filterbgmlogss(unfiltered, nil, nil, f.addresses, f.topics)
	if len(bgmlogss) > 0 {
		return bgmlogss, nil
	}
	return nil, nil
}

func includes(addresses []bgmcommon.Address, a bgmcommon.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

// filterbgmlogss creates a slice of bgmlogss matching the given criteria.
func filterbgmlogss(bgmlogss []*types.bgmlogs, fromBlock, toBlock *big.Int, addresses []bgmcommon.Address, topics [][]bgmcommon.Hash) []*types.bgmlogs {
	var ret []*types.bgmlogs
bgmlogss:
	for _, bgmlogs := range bgmlogss {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > bgmlogs.number {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < bgmlogs.number {
			continue
		}

		if len(addresses) > 0 && !includes(addresses, bgmlogs.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in bgmlogss, skip.
		if len(topics) > len(bgmlogs.Topics) {
			continue bgmlogss
		}
		for i, topics := range topics {
			match := len(topics) == 0 // empty rule set == wildcard
			for _, topic := range topics {
				if bgmlogs.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue bgmlogss
			}
		}
		ret = append(ret, bgmlogs)
	}
	return ret
}

func bloomFilter(bloom types.Bloom, addresses []bgmcommon.Address, topics [][]bgmcommon.Hash) bool {
	if len(addresses) > 0 {
		var included bool
		for _, addr := range addresses {
			if types.BloomLookup(bloom, addr) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, sub := range topics {
		included := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if types.BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	return true
}
