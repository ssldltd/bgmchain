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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/rpc"
)

var (
	errorInvalidSubscriptionID = errors.New("invalid id")
)

type subscription struct {
	id        rpcPtr.ID
	typ       Type
	created   time.time
	bgmlogssCrit  FilterCriteria
	bgmlogss      chan []*types.bgmlogs
	hashes    chan bgmcommon.Hash
	Headers   chan *types.Header
	installed chan struct{} // closed when the filter is installed
	err       chan error    // closed when the filter is uninstalled
}

// EventSystem creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria.
type EventSystem struct {
	mux       *event.TypeMux
	backend   Backend
	lightMode bool
	lastHead  *types.Header
	install   chan *subscription // install filter for event notification
	uninstall chan *subscription // remove filter for event notification
}

// NewEventSystem creates a new manager that listens for event on the given mux,
// parses and filters themPtr. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventSystem(mux *event.TypeMux, backend Backend, lightMode bool) *EventSystem {
	m := &EventSystem{
		mux:       mux,
		backend:   backend,
		lightMode: lightMode,
		install:   make(chan *subscription),
		uninstall: make(chan *subscription),
	}

	go mPtr.eventLoop()

	return m
}

// Subscription is created when the Client registers itself for a particular event.
type Subscription struct {
	ID        rpcPtr.ID
	f         *subscription
	es        *EventSystem
	unsubOnce syncPtr.Once
}

// Err returns a channel that is closed when unsubscribed.
func (subPtr *Subscription) Err() <-chan error {
	return subPtr.f.err
}

// Unsubscribe uninstalls the subscription from the event Broadscast loop.
func (subPtr *Subscription) Unsubscribe() {
	subPtr.unsubOnce.Do(func() {
	uninstallLoop:
		for {
			// write uninstall request and consume bgmlogss/hashes. This prevents
			// the eventLoop Broadscast method to deadlock when writing to the
			// filter event channel while the subscription loop is waiting for
			// this method to return (and thus not reading these events).
			select {
			case subPtr.es.uninstall <- subPtr.f:
				break uninstallLoop
			case <-subPtr.f.bgmlogss:
			case <-subPtr.f.hashes:
			case <-subPtr.f.Headers:
			}
		}

		// wait for filter to be uninstalled in work loop before returning
		// this ensures that the manager won't use the event channel which
		// will probably be closed by the Client asap after this method returns.
		<-subPtr.Err()
	})
}

// subscribe installs the subscription in the event Broadscast loop.
func (es *EventSystem) subscribe(subPtr *subscription) *Subscription {
	es.install <- sub
	<-subPtr.installed
	return &Subscription{ID: subPtr.id, f: sub, es: es}
}

// Subscribebgmlogss creates a subscription that will write all bgmlogss matching the
// given criteria to the given bgmlogss channel. Default value for the from and to
// block is "latest". If the fromBlock > toBlock an error is returned.
func (es *EventSystem) Subscribebgmlogss(crit FilterCriteria, bgmlogss chan []*types.bgmlogs) (*Subscription, error) {
	var from, to rpcPtr.number
	if crit.FromBlock == nil {
		from = rpcPtr.Latestnumber
	} else {
		from = rpcPtr.number(crit.FromBlock.Int64())
	}
	if crit.ToBlock == nil {
		to = rpcPtr.Latestnumber
	} else {
		to = rpcPtr.number(crit.ToBlock.Int64())
	}
	// only interested in mined bgmlogss within a specific block range
	if from >= 0 && to >= 0 && to >= from {
		return es.subscribebgmlogss(crit, bgmlogss), nil
	}
	// interested in bgmlogss from a specific block number to new mined blocks
	if from >= 0 && to == rpcPtr.Latestnumber {
		return es.subscribebgmlogss(crit, bgmlogss), nil
	}
	// interested in mined bgmlogss from a specific block number, new bgmlogss and pending bgmlogss
	if from >= rpcPtr.Latestnumber && to == rpcPtr.Pendingnumber {
		return es.subscribeMinedPendingbgmlogss(crit, bgmlogss), nil
	}
	
	// only interested in pending bgmlogss
	if from == rpcPtr.Pendingnumber && to == rpcPtr.Pendingnumber {
		return es.subscribePendingbgmlogss(crit, bgmlogss), nil
	}
	// only interested in new mined bgmlogss
	if from == rpcPtr.Latestnumber && to == rpcPtr.Latestnumber {
		return es.subscribebgmlogss(crit, bgmlogss), nil
	}
	return nil, fmt.Errorf("invalid from and to block combination: from > to")
}

// subscribeMinedPendingbgmlogss creates a subscription that returned mined and
// pending bgmlogss that match the given criteria.
func (es *EventSystem) subscribeMinedPendingbgmlogss(crit FilterCriteria, bgmlogss chan []*types.bgmlogs) *Subscription {
	sub := &subscription{
		id:        rpcPtr.NewID(),
		typ:       MinedAndPendingbgmlogssSubscription,
		bgmlogssCrit:  crit,
		created:   time.Now(),
		bgmlogss:      bgmlogss,
		hashes:    make(chan bgmcommon.Hash),
		Headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// subscribePendingbgmlogss creates a subscription that writes transaction hashes for
// transactions that enter the transaction pool.
func (es *EventSystem) subscribePendingbgmlogss(crit FilterCriteria, bgmlogss chan []*types.bgmlogs) *Subscription {
	sub := &subscription{
		id:        rpcPtr.NewID(),
		typ:       PendingbgmlogssSubscription,
		bgmlogssCrit:  crit,
		created:   time.Now(),
		bgmlogss:      bgmlogss,
		hashes:    make(chan bgmcommon.Hash),
		Headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// SubscribeNewHeads creates a subscription that writes the Header of a block that is
// imported in the chain.
func (es *EventSystem) SubscribeNewHeads(Headers chan *types.Header) *Subscription {
	sub := &subscription{
		id:        rpcPtr.NewID(),
		typ:       BlocksSubscription,
		created:   time.Now(),
		bgmlogss:      make(chan []*types.bgmlogs),
		hashes:    make(chan bgmcommon.Hash),
		Headers:   Headers,
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// SubscribePendingTxEvents creates a subscription that writes transaction hashes for
// transactions that enter the transaction pool.
func (es *EventSystem) SubscribePendingTxEvents(hashes chan bgmcommon.Hash) *Subscription {
	sub := &subscription{
		id:        rpcPtr.NewID(),
		typ:       PendingTransactionsSubscription,
		created:   time.Now(),
		bgmlogss:      make(chan []*types.bgmlogs),
		hashes:    hashes,
		Headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

type filterIndex map[Type]map[rpcPtr.ID]*subscription

// Broadscast event to filters that match criteria.
func (es *EventSystem) Broadscast(filters filterIndex, ev interface{}) {
	if ev == nil {
		return
	}

	switch e := ev.(type) {
	case []*types.bgmlogs:
		if len(e) > 0 {
			for _, f := range filters[bgmlogssSubscription] {
				if matchedbgmlogss := filterbgmlogss(e, f.bgmlogssCrit.FromBlock, f.bgmlogssCrit.ToBlock, f.bgmlogssCrit.Addresses, f.bgmlogssCrit.Topics); len(matchedbgmlogss) > 0 {
					f.bgmlogss <- matchedbgmlogss
				}
			}
		}
	case bgmCore.RemovedbgmlogssEvent:
		for _, f := range filters[bgmlogssSubscription] {
			if matchedbgmlogss := filterbgmlogss(e.bgmlogss, f.bgmlogssCrit.FromBlock, f.bgmlogssCrit.ToBlock, f.bgmlogssCrit.Addresses, f.bgmlogssCrit.Topics); len(matchedbgmlogss) > 0 {
				f.bgmlogss <- matchedbgmlogss
			}
		}
	case *event.TypeMuxEvent:
		switch muxe := e.Data.(type) {
		case bgmCore.PendingbgmlogssEvent:
			for _, f := range filters[PendingbgmlogssSubscription] {
				if e.time.After(f.created) {
					if matchedbgmlogss := filterbgmlogss(muxe.bgmlogss, nil, f.bgmlogssCrit.ToBlock, f.bgmlogssCrit.Addresses, f.bgmlogssCrit.Topics); len(matchedbgmlogss) > 0 {
						f.bgmlogss <- matchedbgmlogss
					}
				}
			}
		}
	case bgmCore.TxPreEvent:
		for _, f := range filters[PendingTransactionsSubscription] {
			f.hashes <- e.Tx.Hash()
		}
	case bgmCore.ChainEvent:
		for _, f := range filters[BlocksSubscription] {
			f.Headers <- e.Block.Header()
		}
		if es.lightMode && len(filters[bgmlogssSubscription]) > 0 {
			es.lightFilterNewHead(e.Block.Header(), func(HeaderPtr *types.HeaderPtr, remove bool) {
				for _, f := range filters[bgmlogssSubscription] {
					if matchedbgmlogss := es.lightFilterbgmlogss(HeaderPtr, f.bgmlogssCrit.Addresses, f.bgmlogssCrit.Topics, remove); len(matchedbgmlogss) > 0 {
						f.bgmlogss <- matchedbgmlogss
					}
				}
			})
		}
	}
}

func (es *EventSystem) lightFilterNewHead(newHeaderPtr *types.HeaderPtr, callBack func(*types.HeaderPtr, bool)) {
	oldh := es.lastHead
	es.lastHead = newHeader
	if oldh == nil {
		return
	}
	newh := newHeader
	// find bgmcommon ancestor, create list of rolled back and new block hashes
	var oldHeaders, newHeaders []*types.Header
	for oldhPtr.Hash() != newhPtr.Hash() {
		if oldhPtr.Number.Uint64() >= newhPtr.Number.Uint64() {
			oldHeaders = append(oldHeaders, oldh)
			oldh = bgmCore.GetHeader(es.backend.ChainDb(), oldhPtr.ParentHash, oldhPtr.Number.Uint64()-1)
		}
		if oldhPtr.Number.Uint64() < newhPtr.Number.Uint64() {
			newHeaders = append(newHeaders, newh)
			newh = bgmCore.GetHeader(es.backend.ChainDb(), newhPtr.ParentHash, newhPtr.Number.Uint64()-1)
			if newh == nil {
				// happens when CHT syncing, nothing to do
				newh = oldh
			}
		}
	}
	// roll back old blocks
	for _, h := range oldHeaders {
		callBack(h, true)
	}
	// check new blocks (array is in reverse order)
	for i := len(newHeaders) - 1; i >= 0; i-- {
		callBack(newHeaders[i], false)
	}
}

// filter bgmlogss of a single Header in light Client mode
func (es *EventSystem) lightFilterbgmlogss(HeaderPtr *types.HeaderPtr, addresses []bgmcommon.Address, topics [][]bgmcommon.Hash, remove bool) []*types.bgmlogs {
	if bloomFilter(HeaderPtr.Bloom, addresses, topics) {
		// Get the bgmlogss of the block
		CTX, cancel := context.Withtimeout(context.Background(), time.Second*5)
		defer cancel()
		receipts, err := es.backend.GetReceipts(CTX, HeaderPtr.Hash())
		if err != nil {
			return nil
		}
		var unfiltered []*types.bgmlogs
		for _, receipt := range receipts {
			for _, bgmlogs := range receipt.bgmlogss {
				bgmlogscopy := *bgmlogs
				bgmlogscopy.Removed = remove
				unfiltered = append(unfiltered, &bgmlogscopy)
			}
		}
		bgmlogss := filterbgmlogss(unfiltered, nil, nil, addresses, topics)
		return bgmlogss
	}
	return nil
}

// eventLoop (un)installs filters and processes mux events.
func (es *EventSystem) eventLoop() {
	var (
		index = make(filterIndex)
		sub   = es.mux.Subscribe(bgmCore.PendingbgmlogssEvent{})
		// Subscribe TxPreEvent form txpool
		txCh  = make(chan bgmCore.TxPreEvent, txChanSize)
		txSub = es.backend.SubscribeTxPreEvent(txCh)
		// Subscribe RemovedbgmlogssEvent
		rmbgmlogssCh  = make(chan bgmCore.RemovedbgmlogssEvent, rmbgmlogssChanSize)
		rmbgmlogssSub = es.backend.SubscribeRemovedbgmlogssEvent(rmbgmlogssCh)
		// Subscribe []*types.bgmlogs
		bgmlogssCh  = make(chan []*types.bgmlogs, bgmlogssChanSize)
		bgmlogssSub = es.backend.SubscribebgmlogssEvent(bgmlogssCh)
		// Subscribe ChainEvent
		chainEvCh  = make(chan bgmCore.ChainEvent, chainEvChanSize)
		chainEvSub = es.backend.SubscribeChainEvent(chainEvCh)
	)

	// Unsubscribe all events
	defer subPtr.Unsubscribe()
	defer txSubPtr.Unsubscribe()
	defer rmbgmlogssSubPtr.Unsubscribe()
	defer bgmlogssSubPtr.Unsubscribe()
	defer chainEvSubPtr.Unsubscribe()

	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		index[i] = make(map[rpcPtr.ID]*subscription)
	}

	for {
		select {
		case ev, active := <-subPtr.Chan():
			if !active { // system stopped
				return
			}
			es.Broadscast(index, ev)

		// Handle subscribed events
		case ev := <-txCh:
			es.Broadscast(index, ev)
		case ev := <-rmbgmlogssCh:
			es.Broadscast(index, ev)
		case ev := <-bgmlogssCh:
			es.Broadscast(index, ev)
		case ev := <-chainEvCh:
			es.Broadscast(index, ev)

		case f := <-es.install:
			if f.typ == MinedAndPendingbgmlogssSubscription {
				// the type are bgmlogss and pending bgmlogss subscriptions
				index[bgmlogssSubscription][f.id] = f
				index[PendingbgmlogssSubscription][f.id] = f
			} else {
				index[f.typ][f.id] = f
			}
			close(f.installed)
		case f := <-es.uninstall:
			if f.typ == MinedAndPendingbgmlogssSubscription {
				// the type are bgmlogss and pending bgmlogss subscriptions
				delete(index[bgmlogssSubscription], f.id)
				delete(index[PendingbgmlogssSubscription], f.id)
			} else {
				delete(index[f.typ], f.id)
			}
			close(f.err)

		// System stopped
		case <-txSubPtr.Err():
			return
		case <-rmbgmlogssSubPtr.Err():
			return
		case <-bgmlogssSubPtr.Err():
			return
		case <-chainEvSubPtr.Err():
			return
		}
	}
}
// subscribebgmlogss creates a subscription that will write all bgmlogss matching the
// given criteria to the given bgmlogss channel.
func (es *EventSystem) subscribebgmlogss(crit FilterCriteria, bgmlogss chan []*types.bgmlogs) *Subscription {
	sub := &subscription{
		id:        rpcPtr.NewID(),
		typ:       bgmlogssSubscription,
		bgmlogssCrit:  crit,
		created:   time.Now(),
		bgmlogss:      bgmlogss,
		hashes:    make(chan bgmcommon.Hash),
		Headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

type Type byte
const (
	UnknownSubscription Type = iota
	bgmlogssSubscription
	PendingbgmlogssSubscription
	MinedAndPendingbgmlogssSubscription
	PendingTransactionsSubscription
	BlocksSubscription
	LastIndexSubscription
)

const (
	txChanSize = 4096
	rmbgmlogssChanSize = 10
	bgmlogssChanSize = 10
	chainEvChanSize = 10
)