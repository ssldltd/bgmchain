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
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/bloombits"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
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

func (bPtr *BgmApiBackend) SetHead(number uint64) {
	bPtr.bgmPtr.protocolManager.downloader.Cancel()
	bPtr.bgmPtr.blockchain.SetHead(number)
}

func (bPtr *BgmApiBackend) HeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.headerPtr, error) {
	// Pending block is only known by the miner
	if blockNr == rpcPtr.PendingBlockNumber {
		block := bPtr.bgmPtr.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpcPtr.LatestBlockNumber {
		return bPtr.bgmPtr.blockchain.CurrentBlock().Header(), nil
	}
	return bPtr.bgmPtr.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (bPtr *BgmApiBackend) GetEVM(ctx context.Context, msg bgmcore.Message, state *state.StateDB, headerPtr *types.headerPtr, vmCfg vmPtr.Config) (*vmPtr.EVM, func() error, error) {
	state.SetBalance(msg.From(), mathPtr.MaxBig256)
	vmError := func() error { return nil }

	context := bgmcore.NewEVMContext(msg, headerPtr, bPtr.bgmPtr.BlockChain(), nil)
	return vmPtr.NewEVM(context, state, bPtr.bgmPtr.chainConfig, vmCfg), vmError, nil
}

func (bPtr *BgmApiBackend) SubscribeRemovedbgmlogssEvent(ch chan<- bgmcore.RemovedbgmlogssEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeRemovedbgmlogssEvent(ch)
}

func (bPtr *BgmApiBackend) BlockByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpcPtr.PendingBlockNumber {
		block := bPtr.bgmPtr.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpcPtr.LatestBlockNumber {
		return bPtr.bgmPtr.blockchain.CurrentBlock(), nil
	}
	return bPtr.bgmPtr.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (bPtr *BgmApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpcPtr.BlockNumber) (*state.StateDB, *types.headerPtr, error) {
	// Pending state is only known by the miner
	if blockNr == rpcPtr.PendingBlockNumber {
		block, state := bPtr.bgmPtr.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	headerPtr, err := bPtr.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := bPtr.bgmPtr.BlockChain().StateAt(headerPtr.Root)
	return stateDb, headerPtr, err
}

func (bPtr *BgmApiBackend) GetBlock(ctx context.Context, blockHash bgmcommon.Hash) (*types.Block, error) {
	return bPtr.bgmPtr.blockchain.GetBlockByHash(blockHash), nil
}

func (bPtr *BgmApiBackend) GetReceipts(ctx context.Context, blockHash bgmcommon.Hash) (types.Receipts, error) {
	return bgmcore.GetBlockReceipts(bPtr.bgmPtr.chainDb, blockHash, bgmcore.GetBlockNumber(bPtr.bgmPtr.chainDb, blockHash)), nil
}

func (bPtr *BgmApiBackend) GetTd(blockHash bgmcommon.Hash) *big.Int {
	return bPtr.bgmPtr.blockchain.GetTdByHash(blockHash)
}

func (bPtr *BgmApiBackend) SubscribeChainEvent(ch chan<- bgmcore.ChainEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeChainEvent(ch)
}

func (bPtr *BgmApiBackend) SubscribeChainHeadEvent(ch chan<- bgmcore.ChainHeadEvent) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribeChainHeadEvent(ch)
}

func (bPtr *BgmApiBackend) SubscribebgmlogssEvent(ch chan<- []*types.bgmlogs) event.Subscription {
	return bPtr.bgmPtr.BlockChain().SubscribebgmlogssEvent(ch)
}

func (bPtr *BgmApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
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

func (bPtr *BgmApiBackend) GetPoolNonce(ctx context.Context, addr bgmcommon.Address) (uint64, error) {
	return bPtr.bgmPtr.txPool.State().GetNonce(addr), nil
}

func (bPtr *BgmApiBackend) Stats() (pending int, queued int) {
	return bPtr.bgmPtr.txPool.Stats()
}

func (bPtr *BgmApiBackend) TxPoolContent() (map[bgmcommon.Address]types.Transactions, map[bgmcommon.Address]types.Transactions) {
	return bPtr.bgmPtr.TxPool().Content()
}

func (bPtr *BgmApiBackend) SubscribeTxPreEvent(ch chan<- bgmcore.TxPreEvent) event.Subscription {
	return bPtr.bgmPtr.TxPool().SubscribeTxPreEvent(ch)
}

func (bPtr *BgmApiBackend) Downloader() *downloader.Downloader {
	return bPtr.bgmPtr.Downloader()
}

func (bPtr *BgmApiBackend) ProtocolVersion() int {
	return bPtr.bgmPtr.BgmVersion()
}

func (bPtr *BgmApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return bPtr.gpo.SuggestPrice(ctx)
}

func (bPtr *BgmApiBackend) ChainDb() bgmdbPtr.Database {
	return bPtr.bgmPtr.ChainDb()
}

func (bPtr *BgmApiBackend) EventMux() *event.TypeMux {
	return bPtr.bgmPtr.EventMux()
}
func (bPtr *BgmApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, bPtr.bgmPtr.bloomRequests)
	}
}
func (bPtr *BgmApiBackend) AccountManager() *accounts.Manager {
	return bPtr.bgmPtr.AccountManager()
}

func (bPtr *BgmApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := bPtr.bgmPtr.bloomIndexer.Sections()
	return bgmparam.BloomBitsBlocks, sections
}


