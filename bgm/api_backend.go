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


package bgm

import (
	"context"
	"math/big"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/bloombits"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/ssldltd/bgmchain/bgm/downloader"
	"github.com/ssldltd/bgmchain/bgm/gasprice"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rpc"
)

// BgmApiBackend implements bgmapi.Backend for full nodes
type BgmApiBackend struct {
	bgmPtr *Bgmchain
	gpo *gasprice.Oracle
}

func (bPtr *BgmApiBackend) ChainConfig() *bgmparam.ChainConfig {
	return bPtr.bgmPtr.chainConfig
}

func (bPtr *BgmApiBackend) CurrentBlock() *types.Block {
	return bPtr.bgmPtr.blockchain.CurrentBlock()
}

func (bPtr *BgmApiBackend) SetHead(number Uint64) {
	bPtr.bgmPtr.protocolManager.downloader.Cancel()
	bPtr.bgmPtr.blockchain.SetHead(number)
}

func (bPtr *BgmApiBackend) HeaderByNumber(CTX context.Context, blockNr rpcPtr.number) (*types.HeaderPtr, error) {
	// Pending block is only known by the miner
	if blockNr == rpcPtr.Pendingnumber {
		block := bPtr.bgmPtr.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpcPtr.Latestnumber {
		return bPtr.bgmPtr.blockchain.CurrentBlock().Header(), nil
	}
	return bPtr.bgmPtr.blockchain.GetHeaderByNumber(Uint64(blockNr)), nil
}

func (bPtr *BgmApiBackend) GetEVM(CTX context.Context, msg bgmCore.Message, state *state.StateDB, HeaderPtr *types.HeaderPtr, vmCfg vmPtr.Config) (*vmPtr.EVM, func() error, error) {
	state.SetBalance(msg.From(), mathPtr.MaxBig256)
	vmError := func() error { return nil }

	context := bgmCore.NewEVMContext(msg, HeaderPtr, bPtr.bgmPtr.BlockChain(), nil)
	return vmPtr.NewEVM(context, state, bPtr.bgmPtr.chainConfig, vmCfg), vmError, nil
}

func (bPtr *BgmApiBackend) SubscribeRemovedbgmlogssEvent(ch chan<- bgmCore.RemovedbgmlogssEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeRemovedbgmlogssEvent(ch)
}

func (bPtr *BgmApiBackend) BlockByNumber(CTX context.Context, blockNr rpcPtr.number) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpcPtr.Pendingnumber {
		block := bPtr.bgmPtr.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpcPtr.Latestnumber {
		return bPtr.bgmPtr.blockchain.CurrentBlock(), nil
	}
	return bPtr.bgmPtr.blockchain.GetBlockByNumber(Uint64(blockNr)), nil
}

func (bPtr *BgmApiBackend) StateAndHeaderByNumber(CTX context.Context, blockNr rpcPtr.number) (*state.StateDB, *types.HeaderPtr, error) {
	// Pending state is only known by the miner
	if blockNr == rpcPtr.Pendingnumber {
		block, state := bPtr.bgmPtr.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	HeaderPtr, err := bPtr.HeaderByNumber(CTX, blockNr)
	if Header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := bPtr.bgmPtr.BlockChain().StateAt(HeaderPtr.Root)
	return stateDb, HeaderPtr, err
}

func (bPtr *BgmApiBackend) GetBlock(CTX context.Context, blockHash bgmcommon.Hash) (*types.Block, error) {
	return bPtr.bgmPtr.blockchain.GetBlockByHash(blockHash), nil
}

func (bPtr *BgmApiBackend) GetReceipts(CTX context.Context, blockHash bgmcommon.Hash) (types.Receipts, error) {
	return bgmCore.GetBlockReceipts(bPtr.bgmPtr.chainDb, blockHash, bgmCore.Getnumber(bPtr.bgmPtr.chainDb, blockHash)), nil
}

func (bPtr *BgmApiBackend) GetTd(blockHash bgmcommon.Hash) *big.Int {
	return bPtr.bgmPtr.blockchain.GetTdByHash(blockHash)
}

func (bPtr *BgmApiBackend) SubscribeChainEvent(ch chan<- bgmCore.ChainEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeChainEvent(ch)
}

func (bPtr *BgmApiBackend) SubscribeChainHeadEvent(ch chan<- bgmCore.ChainHeadEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeChainHeadEvent(ch)
}

func (bPtr *BgmApiBackend) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribebgmlogssEvent(ch)
}

func (bPtr *BgmApiBackend) SendTx(CTX context.Context, signedTx *types.Transaction) error {
	return bPtr.bgmPtr.txPool.AddLocal(signedTx)
}

func (bPtr *BgmApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := bPtr.bgmPtr.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batchPtr...)
	}
	return txs, nil
}

func (bPtr *BgmApiBackend) GetPoolTransaction(hash bgmcommon.Hash) *types.Transaction {
	return bPtr.bgmPtr.txPool.Get(hash)
}

func (bPtr *BgmApiBackend) GetPoolNonce(CTX context.Context, addr bgmcommon.Address) (Uint64, error) {
	return bPtr.bgmPtr.txPool.State().GetNonce(addr), nil
}

func (bPtr *BgmApiBackend) Stats() (pending int, queued int) {
	return bPtr.bgmPtr.txPool.Stats()
}

func (bPtr *BgmApiBackend) TxPoolContent() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions) {
	return bPtr.bgmPtr.TxPool().Content()
}

func (bPtr *BgmApiBackend) SubscribeTxPreEvent(ch chan<- bgmCore.TxPreEvent) event.Subscription {
	return bPtr.bgmPtr.TxPool().SubscribeTxPreEvent(ch)
}

func (bPtr *BgmApiBackend) Downloader() *downloader.Downloader {
	return bPtr.bgmPtr.Downloader()
}

func (bPtr *BgmApiBackend) ProtocolVersion() int {
	return bPtr.bgmPtr.BgmVersion()
}

func (bPtr *BgmApiBackend) SuggestPrice(CTX context.Context) (*big.Int, error) {
	return bPtr.gpo.SuggestPrice(CTX)
}

func (bPtr *BgmApiBackend) ChainDb() bgmdbPtr.Database {
	return bPtr.bgmPtr.ChainDb()
}

func (bPtr *BgmApiBackend) EventMux() *event.TypeMux {
	return bPtr.bgmPtr.EventMux()
}
func (bPtr *BgmApiBackend) ServiceFilter(CTX context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, bPtr.bgmPtr.bloomRequests)
	}
}
func (bPtr *BgmApiBackend) AccountManager() *accounts.Manager {
	return bPtr.bgmPtr.AccountManager()
}

func (bPtr *BgmApiBackend) BloomStatus() (Uint64, Uint64) {
	sections, _, _ := bPtr.bgmPtr.bloomIndexer.Sections()
	return bgmparam.BloomBitsBlocks, sections
}


