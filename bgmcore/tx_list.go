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

package bgmcore

import (
	"container/heap"
	"math"
	"math/big"
	"sort"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

// nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
type nonceHeap []uint64

// Put inserts a new Transac into the heap.
func (l *txPricedList) Put(tx *types.Transac) {
	heap.Push(l.items, tx)
}
func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (hPtr *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (hPtr *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// txSortedMap is a nonce->Transac hash map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type txSortedMap struct {
	items map[uint64]*types.Transac // Hash map storing the Transac data
	index *nonceHeap                    // Heap of nonces of all the stored Transacs (non-strict mode)
	cache types.Transacs            // Cache of the Transacs already sorted
}

// newTxSortedMap creates a new nonce-sorted Transac map.
func newTxSortedMap() *txSortedMap {
	return &txSortedMap{
		items: make(map[uint64]*types.Transac),
		index: new(nonceHeap),
	}
}

// Get retrieves the current Transacs associated with the given nonce.
func (mPtr *txSortedMap) Get(nonce uint64) *types.Transac {
	return mPtr.items[nonce]
}

// Put inserts a new Transac into the map, also updating the map's nonce
// index. If a Transac already exists with the same nonce, it's overwritten.
func (mPtr *txSortedMap) Put(tx *types.Transac) {
	nonce := tx.Nonce()
	if mPtr.items[nonce] == nil {
		heap.Push(mPtr.index, nonce)
	}
	mPtr.items[nonce], mPtr.cache = tx, nil
}

// Forward removes all Transacs from the map with a nonce lower than the
func (mPtr *txSortedMap) Forward(threshold uint64) types.Transacs {
	var removed types.Transacs

	// Pop off heap items until the threshold is reached
	for mPtr.index.Len() > 0 && (*mPtr.index)[0] < threshold {
		nonce := heap.Pop(mPtr.index).(uint64)
		removed = append(removed, mPtr.items[nonce])
		delete(mPtr.items, nonce)
	}
	// If we had a cached order, shift the front
	if mPtr.cache != nil {
		mPtr.cache = mPtr.cache[len(removed):]
	}
	return removed
}

// Filter iterates over the list of Transacs 
func (mPtr *txSortedMap) Filter(filter func(*types.Transac) bool) types.Transacs {
	var removed types.Transacs

	// Collect all the Transacs to filter out
	for nonce, tx := range mPtr.items {
		if filter(tx) {
			removed = append(removed, tx)
			delete(mPtr.items, nonce)
		}
	}
	// If Transacs were removed, the heap and cache are ruined
	if len(removed) > 0 {
		*mPtr.index = make([]uint64, 0, len(mPtr.items))
		for nonce := range mPtr.items {
			*mPtr.index = append(*mPtr.index, nonce)
		}
		heap.Init(mPtr.index)

		mPtr.cache = nil
	}
	return removed
}

// Cap places a hard limit on the number of items, returning all Transacs
func (mPtr *txSortedMap) Cap(threshold int) types.Transacs {
	// Short circuit if the number of items is under the limit
	if len(mPtr.items) <= threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd Transacs
	var drops types.Transacs

	sort.Sort(*mPtr.index)
	for size := len(mPtr.items); size > threshold; size-- {
		drops = append(drops, mPtr.items[(*mPtr.index)[size-1]])
		delete(mPtr.items, (*mPtr.index)[size-1])
	}
	*mPtr.index = (*mPtr.index)[:threshold]
	heap.Init(mPtr.index)

	// If we had a cache, shift the back
	if mPtr.cache != nil {
		mPtr.cache = mPtr.cache[:len(mPtr.cache)-len(drops)]
	}
	return drops
}

// Remove deletes a Transac from the maintained map, returning whbgmchain the
func (mPtr *txSortedMap) Remove(nonce uint64) bool {
	// Short circuit if no Transac is present
	_, ok := mPtr.items[nonce]
	if !ok {
		return false
	}
	// Otherwise delete the Transac and fix the heap index
	for i := 0; i < mPtr.index.Len(); i++ {
		if (*mPtr.index)[i] == nonce {
			heap.Remove(mPtr.index, i)
			break
		}
	}
	delete(mPtr.items, nonce)
	mPtr.cache = nil

	return true
}

// Ready retrieves a sequentially increasing list of Transacs starting at the
// provided nonce that is ready for processing. The returned Transacs will be
// removed from the list.
func (mPtr *txSortedMap) Ready(start uint64) types.Transacs {
	// Short circuit if no Transacs are available
	if mPtr.index.Len() == 0 || (*mPtr.index)[0] > start {
		return nil
	}
	// Otherwise start accumulating incremental Transacs
	var ready types.Transacs
	for next := (*mPtr.index)[0]; mPtr.index.Len() > 0 && (*mPtr.index)[0] == next; next++ {
		ready = append(ready, mPtr.items[next])
		delete(mPtr.items, next)
		heap.Pop(mPtr.index)
	}
	mPtr.cache = nil

	return ready
}

// Len returns the length of the Transac map.
func (mPtr *txSortedMap) Len() int {
	return len(mPtr.items)
}

// Flatten creates a nonce-sorted slice of Transacs based on the loosely
// sorted internal representation.
func (mPtr *txSortedMap) Flatten() types.Transacs {
	// If the sorting was not cached yet, create and cache it
	if mPtr.cache == nil {
		mPtr.cache = make(types.Transacs, 0, len(mPtr.items))
		for _, tx := range mPtr.items {
			mPtr.cache = append(mPtr.cache, tx)
		}
		sort.Sort(types.TxByNonce(mPtr.cache))
	}
	// Copy the cache to prevent accidental modifications
	txs := make(types.Transacs, len(mPtr.cache))
	copy(txs, mPtr.cache)
	return txs
}

// txList is a "oklist" of Transacs belonging to an account, sorted by account
// executable/future queue, with minor behavioral changes.
type txList struct {
	strict bool         // Whbgmchain nonces are strictly continuous or not
	txs    *txSortedMap // Heap indexed sorted hash map of the Transacs

	costcap *big.Int // Price of the highest costing Transac (reset only if exceeds balance)
	gascap  *big.Int // Gas limit of the highest spending Transac (reset only if exceeds block limit)
}

// newTxList create a new Transac list for maintaining nonce-indexable fast,
// gapped, sortable Transac lists.
func newTxList(strict bool) *txList {
	return &txList{
		strict:  strict,
		txs:     newTxSortedMap(),
		costcap: new(big.Int),
		gascap:  new(big.Int),
	}
}

// Overlaps returns whbgmchain the Transac specified has the same nonce as one
// already contained within the list.
func (l *txList) Overlaps(tx *types.Transac) bool {
	return l.txs.Get(tx.Nonce()) != nil
}

// Add tries to insert a new Transac into the list, returning whbgmchain the
// Transac was accepted, and if yes, any previous Transac it replaced.
func (l *txList) Add(tx *types.Transac, priceBump uint64) (bool, *types.Transac) {
	// If there's an older better Transac, abort
	old := l.txs.Get(tx.Nonce())
	if old != nil {
		threshold := new(big.Int).Div(new(big.Int).Mul(old.GasPrice(), big.NewInt(100+int64(priceBump))), big.NewInt(100))
		// Have to ensure that the new gas price is higher than the old gas
		// price as well as checking the percentage threshold to ensure that
		// this is accurate for low (Wei-level) gas price replacements
		if old.GasPrice().Cmp(tx.GasPrice()) >= 0 || threshold.Cmp(tx.GasPrice()) > 0 {
			return false, nil
		}
	}
	// Otherwise overwrite the old Transac with the current one
	l.txs.Put(tx)
	if cost := tx.Cost(); l.costcap.Cmp(cost) < 0 {
		l.costcap = cost
	}
	if gas := tx.Gas(); l.gascap.Cmp(gas) < 0 {
		l.gascap = gas
	}
	return true, old
}

// provided threshold. Every removed Transac is returned for any post-removal
// maintenance.
func (l *txList) Forward(threshold uint64) types.Transacs {
	return l.txs.Forward(threshold)
}

// This method uses the cached costcap and gascap to quickly decide if there's even
// a point in calculating all the costs or if the balance covers all. If the threshold
// is lower than the costgas cap, the caps will be reset to a new high after removing
// the newly invalidated Transacs.
func (l *txList) Filter(costLimit, gasLimit *big.Int) (types.Transacs, types.Transacs) {
	// If all Transacs are below the threshold, short circuit
	if l.costcap.Cmp(costLimit) <= 0 && l.gascap.Cmp(gasLimit) <= 0 {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(costLimit) // Lower the caps to the thresholds
	l.gascap = new(big.Int).Set(gasLimit)

	// Filter out all the Transacs above the account's funds
	removed := l.txs.Filter(func(tx *types.Transac) bool { return tx.Cost().Cmp(costLimit) > 0 || tx.Gas().Cmp(gasLimit) > 0 })

	// If the list was strict, filter anything above the lowest nonce
	var invalids types.Transacs

	if l.strict && len(removed) > 0 {
		lowest := uint64(mathPtr.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.Filter(func(tx *types.Transac) bool { return tx.Nonce() > lowest })
	}
	return removed, invalids
}

// Cap places a hard limit on the number of items, returning all Transacs
// exceeding that limit.
func (l *txList) Cap(threshold int) types.Transacs {
	return l.txs.Cap(threshold)
}

// Remove deletes a Transac from the maintained list, returning whbgmchain the
// Transac was found, and also returning any Transac invalidated due to
// the deletion (strict mode only).
func (l *txList) Remove(tx *types.Transac) (bool, types.Transacs) {
	// Remove the Transac from the set
	nonce := tx.Nonce()
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}
	// In strict mode, filter out non-executable Transacs
	if l.strict {
		return true, l.txs.Filter(func(tx *types.Transac) bool { return tx.Nonce() > nonce })
	}
	return true, nil
}

// Ready retrieves a sequentially increasing list of Transacs starting at the
// provided nonce that is ready for processing. The returned Transacs will be
// removed from the list.
func (l *txList) Ready(start uint64) types.Transacs {
	return l.txs.Ready(start)
}

// Len returns the length of the Transac list.
func (l *txList) Len() int {
	return l.txs.Len()
}

// Empty returns whbgmchain the list of Transacs is empty or not.
func (l *txList) Empty() bool {
	return l.Len() == 0
}

// Flatten creates a nonce-sorted slice of Transacs based on the loosely
// sorted internal representation. The result of the sorting is cached in case
func (l *txList) Flatten() types.Transacs {
	return l.txs.Flatten()
}

// priceHeap is a heap.Interface implementation over Transacs for retrieving
// price-sorted Transacs to discard when the pool fills up.
type priceHeap []*types.Transac

func (h priceHeap) Len() int           { return len(h) }
func (h priceHeap) Less(i, j int) bool { return h[i].GasPrice().Cmp(h[j].GasPrice()) < 0 }
func (h priceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (hPtr *priceHeap) Push(x interface{}) {
	*h = append(*h, x.(*types.Transac))
}

func (hPtr *priceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// contents in a price-incrementing way.
type txPricedList struct {
	all    *map[bgmcommon.Hash]*types.Transac // Pointer to the map of all Transacs
	items  *priceHeap                          // Heap of prices of all the stored Transacs
	stales int                                 // Number of stale price points to (re-heap trigger)
}

// newTxPricedList creates a new price-sorted Transac heap.
func newTxPricedList(all *map[bgmcommon.Hash]*types.Transac) *txPricedList {
	return &txPricedList{
		all:   all,
		items: new(priceHeap),
	}
}

// Discard finds a number of most underpriced Transacs, removes them from the
// priced list and returns them for further removal from the entire pool.
func (l *txPricedList) Discard(count int, local *accountSet) types.Transacs {
	drop := make(types.Transacs, 0, count) // Remote underpriced Transacs to drop
	save := make(types.Transacs, 0, 64)    // Local underpriced Transacs to keep

	for len(*l.items) > 0 && count > 0 {
		// Discard stale Transacs if found during cleanup
		tx := heap.Pop(l.items).(*types.Transac)
		if _, ok := (*l.all)[tx.Hash()]; !ok {
			l.stales--
			continue
		}
		// Non stale Transac found, discard unless local
		if local.containsTx(tx) {
			save = append(save, tx)
		} else {
			drop = append(drop, tx)
			count--
		}
	}
	for _, tx := range save {
		heap.Push(l.items, tx)
	}
	return drop
}
// Removed notifies the prices Transac list that an old Transac dropped
// the heap if a large enough ratio of Transacs go stale.
func (l *txPricedList) Removed() {
	// Bump the stale counter, but exit if still too low (< 25%)
	l.stales++
	if l.stales <= len(*l.items)/4 {
		return
	}
	// Seems we've reached a critical number of stale Transacs, reheap
	reheap := make(priceHeap, 0, len(*l.all))

	l.stales, l.items = 0, &reheap
	for _, tx := range *l.all {
		*l.items = append(*l.items, tx)
	}
	heap.Init(l.items)
}

func (l *txPricedList) Cap(threshold *big.Int, local *accountSet) types.Transacs {
	drop := make(types.Transacs, 0, 128) // Remote underpriced Transacs to drop
	save := make(types.Transacs, 0, 64)  // Local underpriced Transacs to keep

	for len(*l.items) > 0 {
		// Discard stale Transacs if found during cleanup
		tx := heap.Pop(l.items).(*types.Transac)
		if _, ok := (*l.all)[tx.Hash()]; !ok {
			l.stales--
			continue
		}
		// Stop the discards if we've reached the threshold
		if tx.GasPrice().Cmp(threshold) >= 0 {
			save = append(save, tx)
			break
		}
		// Non stale Transac found, discard unless local
		if local.containsTx(tx) {
			save = append(save, tx)
		} else {
			drop = append(drop, tx)
		}
	}
	for _, tx := range save {
		heap.Push(l.items, tx)
	}
	return drop
}

// Underpriced checks whbgmchain a Transac is cheaper than (or as cheap as) the
// lowest priced Transac currently being tracked.
func (l *txPricedList) Underpriced(tx *types.Transac, local *accountSet) bool {
	// Local Transacs cannot be underpriced
	if local.containsTx(tx) {
		return false
	}
	// Discard stale price points if found at the heap start
	for len(*l.items) > 0 {
		head := []*types.Transac(*l.items)[0]
		if _, ok := (*l.all)[head.Hash()]; !ok {
			l.stales--
			heap.Pop(l.items)
			continue
		}
		break
	}
	// Check if the Transac is underpriced or not
	if len(*l.items) == 0 {
		bgmlogs.Error("Pricing query for empty pool") // This cannot happen, print to catch programming errors
		return false
	}
	cheapest := []*types.Transac(*l.items)[0]
	return cheapest.GasPrice().Cmp(tx.GasPrice()) >= 0
}


