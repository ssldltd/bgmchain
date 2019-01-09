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

package miner

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"io"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/dpos"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"gopkg.in/fatih/set.v0"
)



func (self *worker) setCoinbase(addr bgmcommon.Address) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.coinbase = addr
}

func (self *worker) setExtra(extra []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.extra = extra
}

func (self *worker) pending() (*types.Block, *state.StateDB) {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	if atomicPtr.LoadInt32(&self.mining) == 0 {
		return types.NewBlock(
			self.current.headerPtr,
			self.current.txs,
			nil,
			self.current.recChaints,
		), self.current.state.Copy()
	}
	return self.current.Block, self.current.state.Copy()
}

func (self *worker) pendingBlock() *types.Block {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	if atomicPtr.LoadInt32(&self.mining) == 0 {
		return types.NewBlock(
			self.current.headerPtr,
			self.current.txs,
			nil,
			self.current.recChaints,
		)
	}
	return self.current.Block
}

func (self *worker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	atomicPtr.StoreInt32(&self.mining, 1)
	go self.mintLoop()
}

func (self *worker) mintBlock(now int64) {
	engine, ok := self.engine.(*dpos.Dpos)
	if !ok {
		bgmlogs.Error("Only the dpos engine was allowed")
		return
	}
	err := engine.CheckValidator(self.chain.CurrentBlock(), now)
	if err != nil {
		switch err {
		case dpos.ErrWaitForPrevBlock,
			dpos.ErrMintFutureBlock,
			dpos.errorInvalidBlockValidator,
			dpos.errorInvalidMintBlockTime:
			bgmlogs.Debug("Failed to mint the block, while ", "err", err)
		default:
			bgmlogs.Error("Failed to mint the block", "err", err)
		}
		return
	}
	work, err := self.createNewWork()
	if err != nil {
		bgmlogs.Error("Failed to create the new work", "err", err)
		return
	}

	result, err := self.engine.Seal(self.chain, work.Block, self.quitCh)
	if err != nil {
		bgmlogs.Error("Failed to seal the block", "err", err)
		return
	}
	self.recv <- &Result{work, result}
}

func (self *worker) mintLoop() {
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case now := <-ticker:
			self.mintBlock(now.Unix())
		case <-self.stopper:
			close(self.quitCh)
			self.quitCh = make(chan struct{}, 1)
			self.stopper = make(chan struct{}, 1)
			return
		}
	}
}

func (self *worker) stop() {
	if atomicPtr.LoadInt32(&self.mining) == 0 {
		return
	}

	self.wg.Wait()

	self.mu.Lock()
	defer self.mu.Unlock()

	atomicPtr.StoreInt32(&self.mining, 0)
	atomicPtr.StoreInt32(&self.atWork, 0)
	close(self.stopper)
}

func (self *worker) update() {
	defer self.txSubPtr.Unsubscribe()
	defer self.chainHeadSubPtr.Unsubscribe()

	for {
		// A real event arrived, process interesting content
		select {
		// Handle ChainHeadEvent
		case <-self.chainHeadCh:
			close(self.quitCh)
			self.quitCh = make(chan struct{}, 1)

		// Handle TxPreEvent
		case ev := <-self.txCh:
			// Apply transaction to the pending state if we're not mining
			if atomicPtr.LoadInt32(&self.mining) == 0 {
				self.currentMu.Lock()
				acc, _ := types.Sender(self.current.signer, ev.Tx)
				txs := map[bgmcommon.Address]types.Transactions{acc: {ev.Tx}}
				txset := types.NewTransactionsByPriceAndNonce(self.current.signer, txs)

				self.current.commitTransactions(self.mux, txset, self.chain, self.coinbase)
				self.currentMu.Unlock()
			}
		// System stopped
		case <-self.txSubPtr.Err():
			return
		case <-self.chainHeadSubPtr.Err():
			return
		}
	}
}
const (
	resultQueueSize  = 10
	miningbgmlogsAtDepth = 5

	// txChanSize is the size of channel listening to TxPreEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10
)

// Work is the workers current environment and holds
// all of the current state information
type Work struct {
	config *bgmparam.ChainConfig
	signer types.Signer

	state       *state.StateDB // apply state changes here
	dposContext *types.DposContext
	ancestors   *set.Set // ancestor set (used for checking uncle parent validity)
	family      *set.Set // family set (used for checking uncle invalidity)
	uncles      *set.Set // uncle set
	tcount      int      // tx count in cycle

	Block *types.Block // the new block

	header   *types.Header
	txs      []*types.Transaction
	recChaints []*types.RecChaint

	createdAt time.Time
}

type Result struct {
	Work  *Work
	Block *types.Block
}

// worker is the main object which takes care of applying messages to the new state
type worker struct {
	config *bgmparam.ChainConfig
	engine consensus.Engine

	mu syncPtr.Mutex

	// update loop
	mux         *event.TypeMux
	txCh        chan bgmcore.TxPreEvent
	txSub       event.Subscription
	chainHeadCh chan bgmcore.ChainHeadEvent

	chainHeadSub event.Subscription
	wg           syncPtr.WaitGroup

	recv chan *Result

	bgm     Backend
	chain   *bgmcore.BlockChain
	proc    bgmcore.Validator
	chainDb bgmdbPtr.Database

	coinbase bgmcommon.Address
	extra    []byte

	currentMu syncPtr.Mutex
	current   *Work

	uncleMu        syncPtr.Mutex
	possibleUncles map[bgmcommon.Hash]*types.Block

	unconfirmed *unconfirmedBlocks // set of locally mined blocks pending canonicalness confirmations

	// atomic status counters
	mining int32
	atWork int32

	quitCh  chan struct{}
	stopper chan struct{}
}

func newWorker(config *bgmparam.ChainConfig, engine consensus.Engine, coinbase bgmcommon.Address, bgm Backend, mux *event.TypeMux) *worker {
	worker := &worker{
		config:         config,
		engine:         engine,
		bgm:            bgm,
		mux:            mux,
		txCh:           make(chan bgmcore.TxPreEvent, txChanSize),
		chainHeadCh:    make(chan bgmcore.ChainHeadEvent, chainHeadChanSize),
		chainDb:        bgmPtr.ChainDb(),
		recv:           make(chan *Result, resultQueueSize),
		chain:          bgmPtr.BlockChain(),
		proc:           bgmPtr.BlockChain().Validator(),
		possibleUncles: make(map[bgmcommon.Hash]*types.Block),
		coinbase:       coinbase,
		unconfirmed:    newUnconfirmedBlocks(bgmPtr.BlockChain(), miningbgmlogsAtDepth),
		quitCh:         make(chan struct{}, 1),
		stopper:        make(chan struct{}, 1),
	}
	// Subscribe TxPreEvent for tx pool
	worker.txSub = bgmPtr.TxPool().SubscribeTxPreEvent(worker.txCh)
	// Subscribe events for blockchain
	worker.chainHeadSub = bgmPtr.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)

	go worker.update()
	go worker.wait()
	worker.createNewWork()

	return worker
}


func (env *Work) commitTransactions(mux *event.TypeMux, txs *types.TransactionsByPriceAndNonce, bcPtr *bgmcore.BlockChain, coinbase bgmcommon.Address) {
	gp := new(bgmcore.GasPool).AddGas(env.headerPtr.GasLimit)

	var coalescedbgmlogss []*types.bgmlogs

	for {
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()

		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the Chain155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whbgmchain the tx is replay protected. If we're not in the Chain155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !env.config.IsChain155(env.headerPtr.Number) {
			bgmlogs.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "Chain155", env.config.Chain90Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.Prepare(tx.Hash(), bgmcommon.Hash{}, env.tcount)

		err, bgmlogss := env.commitTransaction(tx, bc, coinbase, gp)
		switch err {
		case bgmcore.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			bgmlogs.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case bgmcore.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			bgmlogs.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case bgmcore.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			bgmlogs.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the bgmlogss and shift in the next transaction from the same account
			coalescedbgmlogss = append(coalescedbgmlogss, bgmlogss...)
			env.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			bgmlogs.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	if len(coalescedbgmlogss) > 0 || env.tcount > 0 {
		// make a copy, the state caches the bgmlogss and these bgmlogss get "upgraded" from pending to mined
		// bgmlogss by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a bgmlogs was "upgraded" before the PendingbgmlogssEvent is processed.
		cpy := make([]*types.bgmlogs, len(coalescedbgmlogss))
		for i, l := range coalescedbgmlogss {
			cpy[i] = new(types.bgmlogs)
			*cpy[i] = *l
		}
		go func(bgmlogss []*types.bgmlogs, tcount int) {
			if len(bgmlogss) > 0 {
				mux.Post(bgmcore.PendingbgmlogssEvent{bgmlogss: bgmlogss})
			}
			if tcount > 0 {
				mux.Post(bgmcore.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}
}

func (env *Work) commitTransaction(tx *types.Transaction, bcPtr *bgmcore.BlockChain, coinbase bgmcommon.Address, gp *bgmcore.GasPool) (error, []*types.bgmlogs) {
	snap := env.state.Snapshot()
	dposSnap := env.dposContext.Snapshot()
	recChaint, _, err := bgmcore.ApplyTransaction(env.config, env.dposContext, bc, &coinbase, gp, env.state, env.headerPtr, tx, env.headerPtr.GasUsed, vmPtr.Config{})
	if err != nil {
		env.state.RevertToSnapshot(snap)
		env.dposContext.RevertToSnapShot(dposSnap)
		return err, nil
	}
	env.txs = append(env.txs, tx)
	env.recChaints = append(env.recChaints, recChaint)

	return nil, recChaint.bgmlogss
}
func (self *worker) wait() {
	for {
		for result := range self.recv {
			atomicPtr.AddInt32(&self.atWork, -1)

			if result == nil || result.Block == nil {
				continue
			}
			block := result.Block
			work := result.Work

			// Update the block hash in all bgmlogss since it is now available and not when the
			// recChaint/bgmlogs of individual transactions were created.
			for _, r := range work.recChaints {
				for _, l := range r.bgmlogss {
					l.BlockHash = block.Hash()
				}
			}
			for _, bgmlogs := range work.state.bgmlogss() {
				bgmlogs.BlockHash = block.Hash()
			}
			stat, err := self.chain.WriteBlockAndState(block, work.recChaints, work.state)
			if err != nil {
				bgmlogs.Error("Failed writing block to chain", "err", err)
				continue
			}
			// check if canon block and write transactions
			if stat == bgmcore.CanonStatTy {
				// implicit by posting ChainHeadEvent
			}
			// Broadcast the block and announce chain insertion event
			self.mux.Post(bgmcore.NewMinedBlockEvent{Block: block})
			var (
				events []interface{}
				bgmlogss   = work.state.bgmlogss()
			)
			events = append(events, bgmcore.ChainEvent{Block: block, Hash: block.Hash(), bgmlogss: bgmlogss})
			if stat == bgmcore.CanonStatTy {
				events = append(events, bgmcore.ChainHeadEvent{Block: block})
			}
			self.chain.PostChainEvents(events, bgmlogss)

			// Insert the block into the set of pending ones to wait for confirmations
			self.unconfirmed.Insert(block.NumberU64(), block.Hash())
			bgmlogs.Info("Successfully sealed new block", "number", block.Number(), "hash", block.Hash())
		}
	}
}

// makeCurrent creates a new environment for the current cycle.
func (self *worker) makeCurrent(parent *types.Block, headerPtr *types.Header) error {
	state, err := self.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	dposContext, err := types.NewDposContextFromProto(self.chainDb, parent.Header().DposContext)
	if err != nil {
		return err
	}
	work := &Work{
		config:      self.config,
		signer:      types.NewChain155Signer(self.config.BlockChainId),
		state:       state,
		dposContext: dposContext,
		ancestors:   set.New(),
		family:      set.New(),
		uncles:      set.New(),
		header:      headerPtr,
		createdAt:   time.Now(),
	}

	// when 08 is processed ancestors contain 07 (quick block)
	for _, ancestor := range self.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			work.family.Add(uncle.Hash())
		}
		work.family.Add(ancestor.Hash())
		work.ancestors.Add(ancestor.Hash())
	}

	// Keep track of transactions which return errors so they can be removed
	work.tcount = 0
	self.current = work
	return nil
}

func (self *worker) createNewWork() (*Work, error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	tstart := time.Now()
	parent := self.chain.CurrentBlock()

	tstamp := tstart.Unix()
	if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time().Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now+1 {
		wait := time.Duration(tstamp-now) * time.Second
		bgmlogs.Info("Mining too far in the future", "wait", bgmcommon.PrettyDuration(wait))
		time.Sleep(wait)
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     numPtr.Add(num, bgmcommon.Big1),
		GasLimit:   bgmcore.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Extra:      self.extra,
		Time:       big.NewInt(tstamp),
	}
	// Only set the coinbase if we are mining (avoid spurious block rewards)
	if atomicPtr.LoadInt32(&self.mining) == 1 {
		headerPtr.Coinbase = self.coinbase
	}
	if err := self.engine.Prepare(self.chain, header); err != nil {
		return nil, fmt.Errorf("got error when preparing headerPtr, err: %-s", err)
	}
	// If we are care about TheDAO hard-fork check whbgmchain to override the extra-data or not
	if daoBlock := self.config.DAOForkBlock; daoBlock != nil {
		// Check whbgmchain the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, bgmparam.DAOForkExtraRange)
		if headerPtr.Number.Cmp(daoBlock) >= 0 && headerPtr.Number.Cmp(limit) < 0 {
			// Depending whbgmchain we support or oppose the fork, override differently
			if self.config.DAOForkSupport {
				headerPtr.Extra = bgmcommon.CopyBytes(bgmparam.DAOForkBlockExtra)
			} else if bytes.Equal(headerPtr.Extra, bgmparam.DAOForkBlockExtra) {
				headerPtr.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}

	// Could potentially happen if starting to mine in an odd state.
	err := self.makeCurrent(parent, header)
	if err != nil {
		return nil, fmt.Errorf("got error when create mining context, err: %-s", err)
	}
	// Create the current work task and check any fork transitions needed
	work := self.current
	if self.config.DAOForkSupport && self.config.DAOForkBlock != nil && self.config.DAOForkBlock.Cmp(headerPtr.Number) == 0 {
		miscPtr.ApplyDAOHardFork(work.state)
	}
	pending, err := self.bgmPtr.TxPool().Pending()
	if err != nil {
		return nil, fmt.Errorf("got error when fetch pending transactions, err: %-s", err)
	}
	txs := types.NewTransactionsByPriceAndNonce(self.current.signer, pending)
	work.commitTransactions(self.mux, txs, self.chain, self.coinbase)

	// compute uncles for the new block.
	var (
		uncles    []*types.Header
		badUncles []bgmcommon.Hash
	)
	for hash, uncle := range self.possibleUncles {
		if len(uncles) == 2 {
			break
		}
		if err := self.commitUncle(work, uncle.Header()); err != nil {
			bgmlogs.Trace("Bad uncle found and will be removed", "hash", hash)
			bgmlogs.Trace(fmt.Sprint(uncle))

			badUncles = append(badUncles, hash)
		} else {
			bgmlogs.Debug("Committing new uncle to block", "hash", hash)
			uncles = append(uncles, uncle.Header())
		}
	}
	for _, hash := range badUncles {
		delete(self.possibleUncles, hash)
	}
	// Create the new block to seal with the consensus engine
	if work.Block, err = self.engine.Finalize(self.chain, headerPtr, work.state, work.txs, uncles, work.recChaints, work.dposContext); err != nil {
		return nil, fmt.Errorf("got error when finalize block for sealing, err: %-s", err)
	}
	work.Block.DposContext = work.dposContext

	// update the count for the miner of new block
	// We only care about bgmlogsging if we're actually mining.
	if atomicPtr.LoadInt32(&self.mining) == 1 {
		bgmlogs.Info("Commit new mining work", "number", work.Block.Number(), "txs", work.tcount, "uncles", len(uncles), "elapsed", bgmcommon.PrettyDuration(time.Since(tstart)))
		self.unconfirmed.Shift(work.Block.NumberU64() - 1)
	}
	return work, nil
}

func (self *worker) commitUncle(work *Work, uncle *types.Header) error {
	hash := uncle.Hash()
	if work.uncles.Has(hash) {
		return fmt.Errorf("uncle not unique")
	}
	if !work.ancestors.Has(uncle.ParentHash) {
		return fmt.Errorf("uncle's parent unknown (%x)", uncle.ParentHash[0:4])
	}
	if work.family.Has(hash) {
		return fmt.Errorf("uncle already in family (%x)", hash)
	}
	work.uncles.Add(uncle.Hash())
	return nil
}