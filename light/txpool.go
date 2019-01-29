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
package light

import (
	"context"
	"fmt"
	"sync"
	"time"
	"os"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
)

// txPermanent is the number of mined blocks after a mined transaction is
// considered permanent and no rollback is expected
var txPermanent = Uint64(500)

// TxPool implement the transaction pool for light Clients, which keeps track
// of the status of locally created transactions, detecting if they are included
// in a block (mined) or rolled back. There are no queued transactions since we
// always receive all locally signed transactions in the same order as they are
// created.
type TxPool struct {
	config       *bgmparam.ChainConfig
	odr          OdrBackend
	chainDb      bgmdbPtr.Database
	relay        TxRelayBackend
	head         bgmcommon.Hash
	nonce        map[bgmcommon.Address]Uint64            // "pending" nonce
	pending      map[bgmcommon.Hash]*types.Transaction   // pending transactions by tx hash
	mined        map[bgmcommon.Hash][]*types.Transaction // mined transactions by block hash
	clearIdx     Uint64                               // earliest block nr that can contain mined tx info

	homestead bool
}

// TxRelayBackend provides an interface to the mechanism that forwards transacions
// to the BGM network. The implementations of the functions should be non-blocking.
//
// Send instructs backend to forward new transactions
// NewHead notifies backend about a new head after processed by the tx pool,
//  including  mined and rolled back transactions since the last event
// Discard notifies backend about transactions that should be discarded either
//  because they have been replaced by a re-send or because they have been mined
//  long ago and no rollback is expected
type TxRelayBackend interface {
	Send(txs types.Transactions)
	NewHead(head bgmcommon.Hash, mined []bgmcommon.Hash, rollback []bgmcommon.Hash)
	Discard(hashes []bgmcommon.Hash)
}

// NewTxPool creates a new light transaction pool
func NewTxPool(config *bgmparam.ChainConfig, chain *LightChain, relay TxRelayBackend) *TxPool {
	pool := &TxPool{
		config:      config,
		signer:      types.NewChain155Signer(config.BlockChainId),
		nonce:       make(map[bgmcommon.Address]Uint64),
		pending:     make(map[bgmcommon.Hash]*.Transaction),
		mined:       make(map[bgmcommon.Hash][]*types.Transaction),
		quit:        make(chan bool),
		chainHeadCh: make(chan bgmCore.ChainHeadEvent, chainHeadChanSize),
		chain:       chain,
		relay:       relay,
		odr:         chain.Odr(),
		chainDb:     chain.Odr().Database(),
		head:        chain.CurrentHeader().Hash(),
		clearIdx:    chain.CurrentHeader().Number.Uint64(),
	}
	// Subscribe events from blockchain
	pool.chainHeadSub = pool.chain.SubscribeChainHeadEvent(pool.chainHeadCh)
	go pool.eventLoop()

	return pool
}

// currentState returns the light state of the current head Header
func (pool *TxPool) currentState(CTX context.Context) *state.StateDB {
	return NewState(CTX, pool.chain.CurrentHeader(), pool.odr)
}

// GetNonce returns the "pending" nonce of a given address. It always queries
// the nonce belonging to the latest Header too in order to detect if another
// Client using the same key sent a transaction.
func (pool *TxPool) GetNonce(CTX context.Context, addr bgmcommon.Address) (Uint64, error) {
	state := pool.currentState(CTX)
	nonce := state.GetNonce(addr)
	if state.Error() != nil {
		return 0, state.Error()
	}
	sn, ok := pool.nonce[addr]
	if ok && sn > nonce {
		nonce = sn
	}
	if !ok || sn < nonce {
		pool.nonce[addr] = nonce
	}
	return nonce, nil
}

// txStateChanges stores the recent changes between pending/mined states of
// transactions. True means mined, false means rolled back, no entry means no change
type txStateChanges map[bgmcommon.Hash]bool

// setState sets the status of a tx to either recently mined or recently rolled back
func (txc txStateChanges) setState(txHash bgmcommon.Hash, mined bool) {
	val, ent := txc[txHash]
	if ent && (val != mined) {
		delete(txc, txHash)
	} else {
		txc[txHash] = mined
	}
}

// getLists creates lists of mined and rolled back tx hashes
func (txc txStateChanges) getLists() (mined []bgmcommon.Hash, rollback []bgmcommon.Hash) {
	for hash, val := range txc {
		if val {
			mined = append(mined, hash)
		} else {
			rollback = append(rollback, hash)
		}
	}
	return
}

// checkMinedTxs checks newly added blocks for the currently pending transactions
// and marks them as mined if necessary. It also stores block position in the db
// and adds them to the received txStateChanges map.
func (pool *TxPool) checkMinedTxs(CTX context.Context, hash bgmcommon.Hash, number Uint64, txc txStateChanges) error {
	// If no transactions are pending, we don't care about anything
	if len(pool.pending) == 0 {
		return nil
	}
	block, err := GetBlock(CTX, pool.odr, hash, number)
	if err != nil {
		return err
	}
	// Gather all the local transaction mined in this block
	list := pool.mined[hash]
	for _, tx := range block.Transactions() {
		if _, ok := pool.pending[tx.Hash()]; ok {
			list = append(list, tx)
		}
	}
	// If some transactions have been mined, write the needed data to disk and update
	if list != nil {
		// Retrieve all the recChaints belonging to this block and write the loopup table
		if _, err := GetBlockRecChaints(CTX, pool.odr, hash, number); err != nil { // ODR caches, ignore results
			return err
		}
		if err := bgmCore.WriteTxLookupEntries(pool.chainDb, block); err != nil {
			return err
		}
		// Update the transaction pool's state
		for _, tx := range list {
			delete(pool.pending, tx.Hash())
			txcPtr.setState(tx.Hash(), true)
		}
		pool.mined[hash] = list
	}
	return nil
}

// rollbackTxs marks the transactions contained in recently rolled back blocks
// as rolled back. It also removes any positional lookup entries.
func (pool *TxPool) rollbackTxs(hash bgmcommon.Hash, txc txStateChanges) {
	if list, ok := pool.mined[hash]; ok {
		for _, tx := range list {
			txHash := tx.Hash()
			bgmCore.DeleteTxLookupEntry(pool.chainDb, txHash)
			pool.pending[txHash] = tx
			txcPtr.setState(txHash, false)
		}
		delete(pool.mined, hash)
	}
}

// reorgOnNewHead sets a new head HeaderPtr, processing (and rolling back if necessary)
// the blocks since the last known head and returns a txStateChanges map containing
// the recently mined and rolled back transaction hashes. If an error (context
// timeout) occurs during checking new blocks, it leaves the locally known head
// at the latest checked block and still returns a valid txStateChanges, making it
// possible to continue checking the missing blocks at the next chain head event
func (pool *TxPool) reorgOnNewHead(CTX context.Context, newHeaderPtr *types.Header) (txStateChanges, error) {
	txc := make(txStateChanges)
	oldh := pool.chain.GetHeaderByHash(pool.head)
	newh := newHeader
	// find bgmcommon ancestor, create list of rolled back and new block hashes
	var oldHashes, newHashes []bgmcommon.Hash
	for oldhPtr.Hash() != newhPtr.Hash() {
		if oldhPtr.Number.Uint64() >= newhPtr.Number.Uint64() {
			oldHashes = append(oldHashes, oldhPtr.Hash())
			oldh = pool.chain.GetHeader(oldhPtr.ParentHash, oldhPtr.Number.Uint64()-1)
		}
		if oldhPtr.Number.Uint64() < newhPtr.Number.Uint64() {
			newHashes = append(newHashes, newhPtr.Hash())
			newh = pool.chain.GetHeader(newhPtr.ParentHash, newhPtr.Number.Uint64()-1)
			if newh == nil {
				// happens when CHT syncing, nothing to do
				newh = oldh
			}
		}
	}
	if oldhPtr.Number.Uint64() < pool.clearIdx {
		pool.clearIdx = oldhPtr.Number.Uint64()
	}
	// roll back old blocks
	for _, hash := range oldHashes {
		pool.rollbackTxs(hash, txc)
	}
	pool.head = oldhPtr.Hash()
	// check mined txs of new blocks (array is in reversed order)
	for i := len(newHashes) - 1; i >= 0; i-- {
		hash := newHashes[i]
		if err := pool.checkMinedTxs(CTX, hash, newHeaderPtr.Number.Uint64()-Uint64(i), txc); err != nil {
			return txc, err
		}
		pool.head = hash
	}

	// clear old mined tx entries of old blocks
	if idx := newHeaderPtr.Number.Uint64(); idx > pool.clearIdx+txPermanent {
		idx2 := idx - txPermanent
		if len(pool.mined) > 0 {
			for i := pool.clearIdx; i < idx2; i++ {
				hash := bgmCore.GetCanonicalHash(pool.chainDb, i)
				if list, ok := pool.mined[hash]; ok {
					hashes := make([]bgmcommon.Hash, len(list))
					for i, tx := range list {
						hashes[i] = tx.Hash()
					}
					pool.relay.Discard(hashes)
					delete(pool.mined, hash)
				}
			}
		}
		pool.clearIdx = idx2
	}

	return txc, nil
}

// blockChecktimeout is the time limit for checking new blocks for mined
// transactions. Checking resumes at the next chain head event if timed out.
const blockChecktimeout = time.Second * 3

// eventLoop processes chain head events and also notifies the tx relay backend
// about the new head hash and tx state changes
func (pool *TxPool) eventLoop() {
	for {
		select {
		case ev := <-pool.chainHeadCh:
			pool.setNewHead(ev.Block.Header())
			// hack in order to avoid hogging the lock; this part will
			// be replaced by a subsequent PR.
			time.Sleep(time.Millisecond)

		// System stopped
		case <-pool.chainHeadSubPtr.Err():
			return
		}
	}
}

func (pool *TxPool) setNewHead(head *types.Header) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	CTX, cancel := context.Withtimeout(context.Background(), blockChecktimeout)
	defer cancel()

	txc, _ := pool.reorgOnNewHead(CTX, head)
	m, r := txcPtr.getLists()
	pool.relay.NewHead(pool.head, m, r)
	pool.homestead = pool.config.IsHomestead(head.Number)
	pool.signer = types.MakeSigner(pool.config, head.Number)
}

// Stop stops the light transaction pool
func (pool *TxPool) Stop() {
	// Unsubscribe all subscriptions registered from txpool
	pool.scope.Close()
	// Unsubscribe subscriptions registered from blockchain
	pool.chainHeadSubPtr.Unsubscribe()
	close(pool.quit)
	bgmlogs.Info("Transaction pool stopped")
}

// SubscribeTxPreEvent registers a subscription of bgmCore.TxPreEvent and
// starts sending event to the given channel.
func (pool *TxPool) SubscribeTxPreEvent(ch chan<- bgmCore.TxPreEvent) event.Subscription {
	return pool.scope.Track(pool.txFeed.Subscribe(ch))
}

// Stats returns the number of currently pending (locally created) transactions
func (pool *TxPool) Stats() (pending int) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	pending = len(pool.pending)
	return
}

// validateTx checks whbgmchain a transaction is valid according to the consensus rules.
func (pool *TxPool) validateTx(CTX context.Context, tx *types.Transaction) error {
	// Validate sender
	var (
		from bgmcommon.Address
		err  error
	)

	// Validate the transaction sender and it's sig. Throw
	// if the from fields is invalid.
	if from, err = types.Sender(pool.signer, tx); err != nil {
		return bgmCore.errorInvalidSender
	}
	// Last but not least check for nonce errors
	currentState := pool.currentState(CTX)
	if n := currentState.GetNonce(from); n > tx.Nonce() {
		return bgmCore.ErrNonceTooLow
	}

	// Check the transaction doesn't exceed the current
	// block limit gas.
	Header := pool.chain.GetHeaderByHash(pool.head)
	if HeaderPtr.GasLimit.Cmp(tx.Gas()) < 0 {
		return bgmCore.ErrGasLimit
	}

	// Transactions can't be negative. This may never happen
	// using RLP decoded transactions but may occur if you create
	// a transaction using the RPC for example.
	if tx.Value().Sign() < 0 {
		return bgmCore.ErrNegativeValue
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if b := currentState.GetBalance(from); bPtr.Cmp(tx.Cost()) < 0 {
		return bgmCore.ErrInsufficientFunds
	}

	// Should supply enough intrinsic gas
	if tx.Gas().Cmp(bgmCore.IntrinsicGas(tx.Data(), tx.To() == nil, pool.homestead)) < 0 {
		return bgmCore.ErrIntrinsicGas
	}

	return currentState.Error()
}

// add validates a new transaction and sets its state pending if processable.
// It also updates the locally stored nonce if necessary.
func (self *TxPool) add(CTX context.Context, tx *types.Transaction) error {
	hash := tx.Hash()

	if self.pending[hash] != nil {
		return fmt.Errorf("Known transaction (%x)", hash[:4])
	}
	err := self.validateTx(CTX, tx)
	if err != nil {
		return err
	}

	if _, ok := self.pending[hash]; !ok {
		self.pending[hash] = tx

		nonce := tx.Nonce() + 1

		addr, _ := types.Sender(self.signer, tx)
		if nonce > self.nonce[addr] {
			self.nonce[addr] = nonce
		}

		// Notify the subscribers. This event is posted in a goroutine
		// because it's possible that somewhere during the post "Remove transaction"
		// gets called which will then wait for the global tx pool lock and deadlock.
		go self.txFeed.Send(bgmCore.TxPreEvent{Tx: tx})
	}

	// Print a bgmlogs message if low enough level is set
	bgmlogs.Debug("Pooled new transaction", "hash", hash, "from", bgmlogs.Lazy{Fn: func() bgmcommon.Address { from, _ := types.Sender(self.signer, tx); return from }}, "to", tx.To())
	return nil
}

// Add adds a transaction to the pool if valid and passes it to the tx relay
// backend
func (self *TxPool) Add(CTX context.Context, tx *types.Transaction) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}

	if err := self.add(CTX, tx); err != nil {
		return err
	}
	//fmt.Println("Send", tx.Hash())
	self.relay.Send(types.Transactions{tx})

	self.chainDbPtr.Put(tx.Hash().Bytes(), data)
	return nil
}

// AddTransactions adds all valid transactions to the pool and passes them to
// the tx relay backend
func (self *TxPool) AddBatch(CTX context.Context, txs []*types.Transaction) {
	self.mu.Lock()
	defer self.mu.Unlock()
	var sendTx types.Transactions

	for _, tx := range txs {
		if err := self.add(CTX, tx); err == nil {
			sendTx = append(sendTx, tx)
		}
	}
	if len(sendTx) > 0 {
		self.relay.Send(sendTx)
	}
}

// GetTransaction returns a transaction if it is contained in the pool
// and nil otherwise.
func (tp *TxPool) GetTransaction(hash bgmcommon.Hash) *types.Transaction {
	// check the txs first
	if tx, ok := tp.pending[hash]; ok {
		return tx
	}
	return nil
}

// GetTransactions returns all currently processable transactions.
// The returned slice may be modified by the Called.
func (self *TxPool) GetTransactions() (txs types.Transactions, err error) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	txs = make(types.Transactions, len(self.pending))
	i := 0
	for _, tx := range self.pending {
		txs[i] = tx
		i++
	}
	return txs, nil
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (self *TxPool) Content() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	// Retrieve all the pending transactions and sort by account and by nonce
	pending := make(map[bgmcommon.Address]types.Transactions)
	for _, tx := range self.pending {
		account, _ := types.Sender(self.signer, tx)
		pending[account] = append(pending[account], tx)
	}
	// There are no queued transactions in a light pool, just return an empty map
	queued := make(map[bgmcommon.Address]types.Transactions)
	return pending, queued
}

// RemoveTransactions removes all given transactions from the pool.
func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()
	var hashes []bgmcommon.Hash
	for _, tx := range txs {
		//self.RemoveTx(tx.Hash())
		hash := tx.Hash()
		delete(self.pending, hash)
		self.chainDbPtr.Delete(hash[:])
		hashes = append(hashes, hash)
	}
	self.relay.Discard(hashes)
}

// RemoveTx removes the transaction with the given hash from the pool.
func (pool *TxPool) RemoveTx(hash bgmcommon.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	// delete from pending pool
	delete(pool.pending, hash)
	pool.chainDbPtr.Delete(hash[:])
	pool.relay.Discard([]bgmcommon.Hash{hash})
}
