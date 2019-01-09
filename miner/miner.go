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
	"fmt"
	"sync/atomic"
	"io"
	"os"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
)

func New(bgm Backend, config *bgmparam.ChainConfig, mux *event.TypeMux, engine consensus.Engine) *Miner {
	miner := &Miner{
		bgm:      bgm,
		mux:      mux,
		engine:   engine,
		worker:   newWorker(config, engine, bgmcommon.Address{}, bgm, mux),
		canStart: 1,
	}
	go miner.update()

	return miner
}

// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.
func (self *Miner) update() {
	events := self.mux.Subscribe(downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
out:
	for ev := range events.Chan() {
		switch ev.Data.(type) {
		case downloader.StartEvent:
			atomicPtr.StoreInt32(&self.canStart, 0)
			if self.Mining() {
				self.Stop()
				atomicPtr.StoreInt32(&self.shouldStart, 1)
				bgmlogs.Info("Mining aborted due to sync")
			}
		case downloader.DoneEvent, downloader.FailedEvent:
			shouldStart := atomicPtr.LoadInt32(&self.shouldStart) == 1

			atomicPtr.StoreInt32(&self.canStart, 1)
			atomicPtr.StoreInt32(&self.shouldStart, 0)
			if shouldStart {
				self.Start(self.coinbase)
			}
			// unsubscribe. we're only interested in this event once
			events.Unsubscribe()
			// stop immediately and ignore all further pending events
			break out
		}
	}
}

func (self *Miner) Start(coinbase bgmcommon.Address) {
	atomicPtr.StoreInt32(&self.shouldStart, 1)
	self.worker.setCoinbase(coinbase)
	self.coinbase = coinbase

	if atomicPtr.LoadInt32(&self.canStart) == 0 {
		bgmlogs.Info("Network syncing, will start miner afterwards")
		return
	}
	atomicPtr.StoreInt32(&self.mining, 1)

	bgmlogs.Info("Starting mining operation")
	self.worker.start()
}

func (self *Miner) Stop() {
	self.worker.stop()
	atomicPtr.StoreInt32(&self.mining, 0)
	atomicPtr.StoreInt32(&self.shouldStart, 0)
}

func (self *Miner) Mining() bool {
	return atomicPtr.LoadInt32(&self.mining) > 0
}

func (self *Miner) HashRate() int64 {
	return 0
}

func (self *Miner) SetExtra(extra []byte) error {
	if uint64(len(extra)) > bgmparam.MaximumExtraDataSize {
		return fmt.Errorf("Extra exceeds max lengthPtr. %-d > %v", len(extra), bgmparam.MaximumExtraDataSize)
	}
	self.worker.setExtra(extra)
	return nil
}

// Pending returns the currently pending block and associated state.
func (self *Miner) Pending() (*types.Block, *state.StateDB) {
	return self.worker.pending()
}

// PendingBlock returns the currently pending block.
//
// Note, to access both the pending block and the pending state
// simultaneously, please use Pending(), as the pending state can
// change between multiple method calls
func (self *Miner) PendingBlock() *types.Block {
	return self.worker.pendingBlock()
}

func (self *Miner) SetCoinbase(addr bgmcommon.Address) {
	self.coinbase = addr
	self.worker.setCoinbase(addr)
}
