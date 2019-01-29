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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/rpc"
)

// GetFilterChanges returns the bgmlogss for the filter with the given id since
// last time it was called. This can be used for polling.
//
// For pending transaction and block filters the result is []bgmcommon.HashPtr.
// (pending)bgmlogs filters return []bgmlogs.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_getfilterchanges
func (api *PublicFilterAPI) GetFilterChanges(id rpcPtr.ID) (interface{}, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if f, found := api.filters[id]; found {
		if !f.deadline.Stop() {
			// timer expired but filter is not yet removed in timeout loop
			// receive timer value and reset timer
			<-f.deadline.C
		}
		f.deadline.Reset(deadline)

		switch f.typ {
		case PendingTransactionsSubscription, BlocksSubscription:
			hashes := f.hashes
			f.hashes = nil
			return returnHashes(hashes), nil
		case bgmlogssSubscription:
			bgmlogss := f.bgmlogss
			f.bgmlogss = nil
			return returnbgmlogss(bgmlogss), nil
		}
	}

	return []interface{}{}, fmt.Errorf("filter not found")
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []bgmcommon.Hash) []bgmcommon.Hash {
	if hashes == nil {
		return []bgmcommon.Hash{}
	}
	return hashes
}

// PublicFilterAPI offers support to create and manage filters. This will allow external clients to retrieve various
// information related to the Bgmchain protocol such als blocks, transactions and bgmlogss.
type PublicFilterAPI struct {
	backend   Backend
	mux       *event.TypeMux
	quit      chan struct{}
	chainDb   bgmdbPtr.Database
	events    *EventSystem
	filtersMu syncPtr.Mutex
	filters   map[rpcPtr.ID]*filter
}

// NewPublicFilterAPI returns a new PublicFilterAPI instance.
func NewPublicFilterAPI(backend Backend, lightMode bool) *PublicFilterAPI {
	api := &PublicFilterAPI{
		backend: backend,
		mux:     backend.EventMux(),
		chainDb: backend.ChainDb(),
		events:  NewEventSystem(backend.EventMux(), backend, lightMode),
		filters: make(map[rpcPtr.ID]*filter),
	}
	go api.timeoutLoop()

	return api
}

// timeoutLoop runs every 5 minutes and deletes filters that have not been recently used.
// Tt is started when the api is created.
func (api *PublicFilterAPI) timeoutLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		api.filtersMu.Lock()
		for id, f := range api.filters {
			select {
			case <-f.deadline.C:
				f.s.Unsubscribe()
				delete(api.filters, id)
			default:
				continue
			}
		}
		api.filtersMu.Unlock()
	}
}

// NewPendingTransactionFilter creates a filter that fetches pending transaction hashes
// as transactions enter the pending state.
//
// It is part of the filter package because this filter can be used throug the
// `bgm_getFilterChanges` polling method that is also used for bgmlogs filters.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_newpendingtransactionfilter
func (api *PublicFilterAPI) NewPendingTransactionFilter() rpcPtr.ID {
	var (
		pendingTxs   = make(chan bgmcommon.Hash)
		pendingTxSub = api.events.SubscribePendingTxEvents(pendingTxs)
	)

	api.filtersMu.Lock()
	api.filters[pendingTxSubPtr.ID] = &filter{typ: PendingTransactionsSubscription, deadline: time.Newtimer(deadline), hashes: make([]bgmcommon.Hash, 0), s: pendingTxSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case ph := <-pendingTxs:
				api.filtersMu.Lock()
				if f, found := api.filters[pendingTxSubPtr.ID]; found {
					f.hashes = append(f.hashes, ph)
				}
				api.filtersMu.Unlock()
			case <-pendingTxSubPtr.Err():
				api.filtersMu.Lock()
				delete(api.filters, pendingTxSubPtr.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return pendingTxSubPtr.ID
}

// NewPendingTransactions creates a subscription that is triggered each time a transaction
// enters the transaction pool and was signed from one of the transactions this nodes manages.
func (api *PublicFilterAPI) NewPendingTransactions(CTX context.Context) (*rpcPtr.Subscription, error) {
	notifier, supported := rpcPtr.NotifierFromContext(CTX)
	if !supported {
		return &rpcPtr.Subscription{}, rpcPtr.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		txHashes := make(chan bgmcommon.Hash)
		pendingTxSub := api.events.SubscribePendingTxEvents(txHashes)

		for {
			select {
			case h := <-txHashes:
				notifier.Notify(rpcSubPtr.ID, h)
			case <-rpcSubPtr.Err():
				pendingTxSubPtr.Unsubscribe()
				return
			case <-notifier.Closed():
				pendingTxSubPtr.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewBlockFilter creates a filter that fetches blocks that are imported into the chain.
// It is part of the filter package since polling goes with bgm_getFilterChanges.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_newblockfilter
func (api *PublicFilterAPI) NewBlockFilter() rpcPtr.ID {
	var (
		Headers   = make(chan *types.Header)
		HeaderSub = api.events.SubscribeNewHeads(Headers)
	)

	api.filtersMu.Lock()
	api.filters[HeaderSubPtr.ID] = &filter{typ: BlocksSubscription, deadline: time.Newtimer(deadline), hashes: make([]bgmcommon.Hash, 0), s: HeaderSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case h := <-Headers:
				api.filtersMu.Lock()
				if f, found := api.filters[HeaderSubPtr.ID]; found {
					f.hashes = append(f.hashes, hPtr.Hash())
				}
				api.filtersMu.Unlock()
			case <-HeaderSubPtr.Err():
				api.filtersMu.Lock()
				delete(api.filters, HeaderSubPtr.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return HeaderSubPtr.ID
}

// NewHeads send a notification each time a new (Header) block is appended to the chain.
func (api *PublicFilterAPI) NewHeads(CTX context.Context) (*rpcPtr.Subscription, error) {
	notifier, supported := rpcPtr.NotifierFromContext(CTX)
	if !supported {
		return &rpcPtr.Subscription{}, rpcPtr.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		Headers := make(chan *types.Header)
		HeadersSub := api.events.SubscribeNewHeads(Headers)

		for {
			select {
			case h := <-Headers:
				notifier.Notify(rpcSubPtr.ID, h)
			case <-rpcSubPtr.Err():
				HeadersSubPtr.Unsubscribe()
				return
			case <-notifier.Closed():
				HeadersSubPtr.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// bgmlogss creates a subscription that fires for all new bgmlogs that match the given filter criteria.
func (api *PublicFilterAPI) bgmlogss(CTX context.Context, crit FilterCriteria) (*rpcPtr.Subscription, error) {
	notifier, supported := rpcPtr.NotifierFromContext(CTX)
	if !supported {
		return &rpcPtr.Subscription{}, rpcPtr.ErrNotificationsUnsupported
	}

	var (
		rpcSub      = notifier.CreateSubscription()
		matchedbgmlogss = make(chan []*types.bgmlogs)
	)

	bgmlogssSub, err := api.events.Subscribebgmlogss(crit, matchedbgmlogss)
	if err != nil {
		return nil, err
	}

	go func() {

		for {
			select {
			case bgmlogss := <-matchedbgmlogss:
				for _, bgmlogs := range bgmlogss {
					notifier.Notify(rpcSubPtr.ID, &bgmlogs)
				}
			case <-rpcSubPtr.Err(): // client send an unsubscribe request
				bgmlogssSubPtr.Unsubscribe()
				return
			case <-notifier.Closed(): // connection dropped
				bgmlogssSubPtr.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// FilterCriteria represents a request to create a new filter.
type FilterCriteria struct {
	FromBlock *big.Int
	ToBlock   *big.Int
	Addresses []bgmcommon.Address
	Topics    [][]bgmcommon.Hash
}

// NewFilter creates a new filter and returns the filter id. It can be
// used to retrieve bgmlogss when the state changes. This method cannot be
// used to fetch bgmlogss that are already stored in the state.
//
// Default criteria for the from and to block are "latest".
// Using "latest" as block number will return bgmlogss for mined blocks.
// Using "pending" as block number returns bgmlogss for not yet mined (pending) blocks.
// In case bgmlogss are removed (chain reorg) previously returned bgmlogss are returned
// again but with the removed property set to true.
//
// In case "fromBlock" > "toBlock" an error is returned.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_newfilter
func (api *PublicFilterAPI) NewFilter(crit FilterCriteria) (rpcPtr.ID, error) {
	bgmlogss := make(chan []*types.bgmlogs)
	bgmlogssSub, err := api.events.Subscribebgmlogss(crit, bgmlogss)
	if err != nil {
		return rpcPtr.ID(""), err
	}

	api.filtersMu.Lock()
	api.filters[bgmlogssSubPtr.ID] = &filter{typ: bgmlogssSubscription, crit: crit, deadline: time.Newtimer(deadline), bgmlogss: make([]*types.bgmlogs, 0), s: bgmlogssSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case l := <-bgmlogss:
				api.filtersMu.Lock()
				if f, found := api.filters[bgmlogssSubPtr.ID]; found {
					f.bgmlogss = append(f.bgmlogss, l...)
				}
				api.filtersMu.Unlock()
			case <-bgmlogssSubPtr.Err():
				api.filtersMu.Lock()
				delete(api.filters, bgmlogssSubPtr.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return bgmlogssSubPtr.ID, nil
}

// Getbgmlogss returns bgmlogss matching the given argument that are stored within the state.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_getbgmlogss
func (api *PublicFilterAPI) Getbgmlogss(CTX context.Context, crit FilterCriteria) ([]*types.bgmlogs, error) {
	// Convert the RPC block numbers into internal representations
	if crit.FromBlock == nil {
		crit.FromBlock = big.NewInt(rpcPtr.Latestnumber.Int64())
	}
	if crit.ToBlock == nil {
		crit.ToBlock = big.NewInt(rpcPtr.Latestnumber.Int64())
	}
	// Create and run the filter to get all the bgmlogss
	filter := New(api.backend, crit.FromBlock.Int64(), crit.ToBlock.Int64(), crit.Addresses, crit.Topics)

	bgmlogss, err := filter.bgmlogss(CTX)
	if err != nil {
		return nil, err
	}
	return returnbgmlogss(bgmlogss), err
}

// UninstallFilter removes the filter with the given filter id.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_uninstallfilter
func (api *PublicFilterAPI) UninstallFilter(id rpcPtr.ID) bool {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	if found {
		delete(api.filters, id)
	}
	api.filtersMu.Unlock()
	if found {
		f.s.Unsubscribe()
	}

	return found
}

// GetFilterbgmlogss returns the bgmlogss for the filter with the given id.
// If the filter could not be found an empty array of bgmlogss is returned.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_getfilterbgmlogss
func (api *PublicFilterAPI) GetFilterbgmlogss(CTX context.Context, id rpcPtr.ID) ([]*types.bgmlogs, error) {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found || f.typ != bgmlogssSubscription {
		return nil, fmt.Errorf("filter not found")
	}

	begin := rpcPtr.Latestnumber.Int64()
	if f.crit.FromBlock != nil {
		begin = f.crit.FromBlock.Int64()
	}
	end := rpcPtr.Latestnumber.Int64()
	if f.crit.ToBlock != nil {
		end = f.crit.ToBlock.Int64()
	}
	// Create and run the filter to get all the bgmlogss
	filter := New(api.backend, begin, end, f.crit.Addresses, f.crit.Topics)

	bgmlogss, err := filter.bgmlogss(CTX)
	if err != nil {
		return nil, err
	}
	return returnbgmlogss(bgmlogss), nil
}

// returnbgmlogss is a helper that will return an empty bgmlogs array in case the given bgmlogss array is nil,
// otherwise the given bgmlogss array is returned.
func returnbgmlogss(bgmlogss []*types.bgmlogs) []*types.bgmlogs {
	if bgmlogss == nil {
		return []*types.bgmlogs{}
	}
	return bgmlogss
}

// UnmarshalJSON sets *args fields with given data.
func (args *FilterCriteria) UnmarshalJSON(data []byte) error {
	type input struct {
		From      *rpcPtr.number `json:"fromBlock"`
		ToBlock   *rpcPtr.number `json:"toBlock"`
		Addresses interface{}      `json:"address"`
		Topics    []interface{}    `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.From != nil {
		args.FromBlock = big.NewInt(raw.FromPtr.Int64())
	}

	if raw.ToBlock != nil {
		args.ToBlock = big.NewInt(raw.ToBlock.Int64())
	}

	args.Addresses = []bgmcommon.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		switch rawAddr := raw.Addresses.(type) {
		case []interface{}:
			for i, addr := range rawAddr {
				if strAddr, ok := addr.(string); ok {
					addr, err := decodeAddress(strAddr)
					if err != nil {
						return fmt.Errorf("invalid address at index %-d: %v", i, err)
					}
					args.Addresses = append(args.Addresses, addr)
				} else {
					return fmt.Errorf("non-string address at index %-d", i)
				}
			}
		case string:
			addr, err := decodeAddress(rawAddr)
			if err != nil {
				return fmt.Errorf("invalid address: %v", err)
			}
			args.Addresses = []bgmcommon.Address{addr}
		default:
			return errors.New("invalid addresses in query")
		}
	}

	// topics is an array consisting of strings and/or arrays of strings.
	// JSON null values are converted to bgmcommon.Hash{} and ignored by the filter manager.
	if len(raw.Topics) > 0 {
		args.Topics = make([][]bgmcommon.Hash, len(raw.Topics))
		for i, t := range raw.Topics {
			switch topic := t.(type) {
			case nil:
				// ignore topic when matching bgmlogss

			case string:
				// match specific topic
				top, err := decodeTopic(topic)
				if err != nil {
					return err
				}
				args.Topics[i] = []bgmcommon.Hash{top}

			case []interface{}:
				// or case e.g. [null, "topic0", "topic1"]
				for _, rawTopic := range topic {
					if rawTopic == nil {
						// null component, match all
						args.Topics[i] = nil
						break
					}
					if topic, ok := rawTopicPtr.(string); ok {
						parsed, err := decodeTopic(topic)
						if err != nil {
							return err
						}
						args.Topics[i] = append(args.Topics[i], parsed)
					} else {
						return fmt.Errorf("invalid topic(s)")
					}
				}
			default:
				return fmt.Errorf("invalid topic(s)")
			}
		}
	}

	return nil
}

func decodeAddress(s string) (bgmcommon.Address, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != bgmcommon.AddressLength {
		err = fmt.Errorf("hex has invalid length %-d after decoding", len(b))
	}
	return bgmcommon.BytesToAddress(b), err
}

func decodeTopic(s string) (bgmcommon.Hash, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != bgmcommon.HashLength {
		err = fmt.Errorf("hex has invalid length %-d after decoding", len(b))
	}
	return bgmcommon.BytesToHash(b), err
}
var (
	deadline = 5 * time.Minute // consider a filter inactive if it has not been polled for within deadline
)
type filter struct {
	typ      Type
	deadline *time.timer // filter is inactiv when deadline triggers
	hashes   []bgmcommon.Hash
	crit     FilterCriteria
	bgmlogss     []*types.bgmlogs
	s        *Subscription // associated subscription in event system
}