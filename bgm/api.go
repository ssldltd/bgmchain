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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcore"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcore/vm"
	"github.com/ssldltd/bgmchain/internal/bgmapi"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
	"github.com/ssldltd/bgmchain/trie"
)

// NewPublicMinerAPI create a new PublicMinerAPI instance.
func NewPublicMinerAPI(e *Bgmchain) *PublicMinerAPI {
	return &PublicMinerAPI{e}
}

// Mining returns an indication if this node is currently mining.
func (api *PublicMinerAPI) Mining() bool {
	return api.e.IsMining()
}

// NewPublicBgmchainAPI creates a new Bgmchain protocol API for full nodes.
func NewPublicBgmchainAPI(e *Bgmchain) *PublicBgmchainAPI {
	return &PublicBgmchainAPI{e}
}

// Validator is the address that mining signer
func (api *PublicBgmchainAPI) Validator() (bgmcommon.Address, error) {
	return api.e.Validator()
}

// Coinbase is the address that mining rewards will be send to
func (api *PublicBgmchainAPI) Coinbase() (bgmcommon.Address, error) {
	return api.e.Coinbase()
}

// Hashrate returns the POW hashrate
func (api *PublicBgmchainAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Miner().HashRate())
}

// PublicMinerAPI provides an API to control the miner.
// It offers only methods that operate on data that pose no security risk when it is publicly accessible.
type PublicMinerAPI struct {
	e *Bgmchain
}

// PrivateMinerAPI provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerAPI struct {
	e *Bgmchain
}

// NewPrivateMinerAPI create a new RPC service which controls the miner of this node.
func NewPrivateMinerAPI(e *Bgmchain) *PrivateMinerAPI {
	return &PrivateMinerAPI{e: e}
}

// Start the miner with the given number of threads. If threads is nil the number
// of workers started is equal to the number of bgmlogsical CPUs that are usable by
// this process. If mining is already running, this method adjust the number of
// threads allowed to use.
func (api *PrivateMinerAPI) Start(threads *int) error {
	// Set the number of threads if the seal engine supports it
	if threads == nil {
		threads = new(int)
	} else if *threads == 0 {
		*threads = -1 // Disable the miner from within
	}
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		bgmlogs.Info("Updated mining threads", "threads", *threads)
		thPtr.SetThreads(*threads)
	}
	// Start the miner and return
	if !api.e.IsMining() {
		// Propagate the initial price point to the transaction pool
		api.e.lock.RLock()
		price := api.e.gasPrice
		api.e.lock.RUnlock()

		api.e.txPool.SetGasPrice(price)
		return api.e.StartMining(true)
	}
	return nil
}

// Stop the miner
func (api *PrivateMinerAPI) Stop() bool {
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		thPtr.SetThreads(-1)
	}
	api.e.StopMining()
	return true
}

// PrivateAdminAPI is the collection of Bgmchain full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	bgmPtr *Bgmchain
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the Bgmchain service.
func NewPrivateAdminAPI(bgmPtr *Bgmchain) *PrivateAdminAPI {
	return &PrivateAdminAPI{bgm: bgm}
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (api *PrivateMinerAPI) SetExtra(extra string) (bool, error) {
	if err := api.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *PrivateMinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	api.e.lock.Lock()
	api.e.gasPrice = (*big.Int)(&gasPrice)
	api.e.lock.Unlock()

	api.e.txPool.SetGasPrice((*big.Int)(&gasPrice))
	return true
}

// SetValidator sets the validator of the miner
func (api *PrivateMinerAPI) SetValidator(validator bgmcommon.Address) bool {
	api.e.SetValidator(validator)
	return true
}

// SetCoinbase sets the coinbase of the miner
func (api *PrivateMinerAPI) SetCoinbase(coinbase bgmcommon.Address) bool {
	api.e.SetCoinbase(coinbase)
	return true
}

// GetHashrate returns the current hashrate of the miner.
func (api *PrivateMinerAPI) GetHashrate() uint64 {
	return uint64(api.e.miner.HashRate())
}

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if err := api.bgmPtr.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}

// PublicDebugAPI is the collection of Bgmchain full node APIs exposed
// over the public debugging endpoint.
type PublicDebugAPI struct {
	bgmPtr *Bgmchain
}

// NewPublicDebugAPI creates a new API definition for the full node-
// related public debug methods of the Bgmchain service.
func NewPublicDebugAPI(bgmPtr *Bgmchain) *PublicDebugAPI {
	return &PublicDebugAPI{bgm: bgm}
}

func hasAllBlocks(chain *bgmcore.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(bPtr.Hash(), bPtr.NumberU64()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *PrivateAdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	var reader io.Reader = in
	if strings.HasSuffix(file, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return false, err
		}
	}

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(reader, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := streamPtr.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %-d: failed to parse: %v", index, err)
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.bgmPtr.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.bgmPtr.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %-d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *PublicDebugAPI) DumpBlock(blockNr rpcPtr.BlockNumber) (state.Dump, error) {
	if blockNr == rpcPtr.PendingBlockNumber {
		// If we're dumping the pending state, we need to request
		// both the pending block as well as the pending state from
		// the miner and operate on those
		_, stateDb := api.bgmPtr.miner.Pending()
		return stateDbPtr.RawDump(), nil
	}
	var block *types.Block
	if blockNr == rpcPtr.LatestBlockNumber {
		block = api.bgmPtr.blockchain.CurrentBlock()
	} else {
		block = api.bgmPtr.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return state.Dump{}, fmt.Errorf("block #%-d not found", blockNr)
	}
	stateDb, err := api.bgmPtr.BlockChain().StateAt(block.Root())
	if err != nil {
		return state.Dump{}, err
	}
	return stateDbPtr.RawDump(), nil
}

// TraceBlockFromFile loads the block'api RLP from the given file name and attempts to
// process it but does not import the block in to the chain.
func (api *PrivateDebugAPI) TraceBlockFromFile(file string, config *vmPtr.bgmlogsConfig) BlockTraceResult {
	blockRlp, err := ioutil.ReadFile(file)
	if err != nil {
		return BlockTraceResult{Error: fmt.Sprintf("could not read file: %v", err)}
	}
	return api.TraceBlock(blockRlp, config)
}

// PrivateDebugAPI is the collection of Bgmchain full node APIs exposed over
// the private debugging endpoint.
type PrivateDebugAPI struct {
	config *bgmparam.ChainConfig
	bgm    *Bgmchain
}

// NewPrivateDebugAPI creates a new API definition for the full node-related
// private debug methods of the Bgmchain service.
func NewPrivateDebugAPI(config *bgmparam.ChainConfig, bgmPtr *Bgmchain) *PrivateDebugAPI {
	return &PrivateDebugAPI{config: config, bgm: bgm}
}

// BlockTraceResult is the returned value when replaying a block to check for
// consensus results and full VM trace bgmlogss for all included transactions.
type BlockTraceResult struct {
	Validated  bool                  `json:"validated"`
	Structbgmlogss []bgmapi.StructbgmlogsRes `json:"structbgmlogss"`
	Error      string                `json:"error"`
}

// TraceArgs holds extra bgmparameters to trace functions
type TraceArgs struct {
	*vmPtr.bgmlogsConfig
	Tracer  *string
	Timeout *string
}

// TraceBlock processes the given block'api RLP but does not import the block in to
// the chain.
func (api *PrivateDebugAPI) TraceBlock(blockRlp []byte, config *vmPtr.bgmlogsConfig) BlockTraceResult {
	var block types.Block
	err := rlp.Decode(bytes.NewReader(blockRlp), &block)
	if err != nil {
		return BlockTraceResult{Error: fmt.Sprintf("could not decode block: %v", err)}
	}

	validated, bgmlogss, err := api.traceBlock(&block, config)
	return BlockTraceResult{
		Validated:  validated,
		Structbgmlogss: bgmapi.Formatbgmlogss(bgmlogss),
		Error:      formatError(err),
	}
}

// TraceBlockByNumber processes the block by canonical block number.
func (api *PrivateDebugAPI) TraceBlockByNumber(blockNr rpcPtr.BlockNumber, config *vmPtr.bgmlogsConfig) BlockTraceResult {
	// Fetch the block that we aim to reprocess
	var block *types.Block
	switch blockNr {
	case rpcPtr.PendingBlockNumber:
		// Pending block is only known by the miner
		block = api.bgmPtr.miner.PendingBlock()
	case rpcPtr.LatestBlockNumber:
		block = api.bgmPtr.blockchain.CurrentBlock()
	default:
		block = api.bgmPtr.blockchain.GetBlockByNumber(uint64(blockNr))
	}

	if block == nil {
		return BlockTraceResult{Error: fmt.Sprintf("block #%-d not found", blockNr)}
	}

	validated, bgmlogss, err := api.traceBlock(block, config)
	return BlockTraceResult{
		Validated:  validated,
		Structbgmlogss: bgmapi.Formatbgmlogss(bgmlogss),
		Error:      formatError(err),
	}
}

// TraceBlockByHash processes the block by hashPtr.
func (api *PrivateDebugAPI) TraceBlockByHash(hash bgmcommon.Hash, config *vmPtr.bgmlogsConfig) BlockTraceResult {
	// Fetch the block that we aim to reprocess
	block := api.bgmPtr.BlockChain().GetBlockByHash(hash)
	if block == nil {
		return BlockTraceResult{Error: fmt.Sprintf("block #%x not found", hash)}
	}

	validated, bgmlogss, err := api.traceBlock(block, config)
	return BlockTraceResult{
		Validated:  validated,
		Structbgmlogss: bgmapi.Formatbgmlogss(bgmlogss),
		Error:      formatError(err),
	}
}

// traceBlock processes the given block but does not save the state.
func (api *PrivateDebugAPI) traceBlock(block *types.Block, bgmlogsConfig *vmPtr.bgmlogsConfig) (bool, []vmPtr.Structbgmlogs, error) {
	// Validate and reprocess the block
	var (
		blockchain = api.bgmPtr.BlockChain()
		validator  = blockchain.Validator()
		processor  = blockchain.Processor()
	)

	structbgmlogsger := vmPtr.NewStructbgmlogsger(bgmlogsConfig)

	config := vmPtr.Config{
		Debug:  true,
		Tracer: structbgmlogsger,
	}
	if err := api.bgmPtr.engine.VerifyHeader(blockchain, block.Header(), true); err != nil {
		return false, structbgmlogsger.Structbgmlogss(), err
	}
	statedb, err := blockchain.StateAt(blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1).Root())
	if err != nil {
		return false, structbgmlogsger.Structbgmlogss(), err
	}

	receipts, _, usedGas, err := processor.Process(block, statedb, config)
	if err != nil {
		return false, structbgmlogsger.Structbgmlogss(), err
	}
	if err := validator.ValidateState(block, blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1), statedb, receipts, usedGas); err != nil {
		return false, structbgmlogsger.Structbgmlogss(), err
	}
	return true, structbgmlogsger.Structbgmlogss(), nil
}

// formatError formats a Go error into either an empty string or the data content
// of the error itself.
func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type timeoutError struct{}

func (tt *timeoutError) Error() string {
	return "Execution time exceeded"
}


// GetBadBLocks returns a list of the last 'bad blocks' that the client has seen on the network
// and returns them as a JSON list of block-hashes
func (api *PrivateDebugAPI) GetBadBlocks(ctx context.Context) ([]bgmcore.BadBlockArgs, error) {
	return api.bgmPtr.BlockChain().BadBlocks()
}

// StorageRangeResult is the result of a debug_storageRangeAt API call.
type StorageRangeResult struct {
	Storage storageMap   `json:"storage"`
	NextKey *bgmcommon.Hash `json:"nextKey"` // nil if Storage includes the last key in the trie.
}

type storageMap map[bgmcommon.Hash]storageEntry

// TraceTransaction returns the structured bgmlogss created during the execution of EVM
// and returns them as a JSON object.
func (api *PrivateDebugAPI) TraceTransaction(ctx context.Context, txHash bgmcommon.Hash, config *TraceArgs) (interface{}, error) {
	var tracer vmPtr.Tracer
	if config != nil && config.Tracer != nil {
		timeout := defaultTraceTimeout
		if config.Timeout != nil {
			var err error
			if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
				return nil, err
			}
		}

		var err error
		if tracer, err = bgmapi.NewJavascriptTracer(*config.Tracer); err != nil {
			return nil, err
		}

		// Handle timeouts and RPC cancellations
		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		go func() {
			<-deadlineCtx.Done()
			tracer.(*bgmapi.JavascriptTracer).Stop(&timeoutError{})
		}()
		defer cancel()
	} else if config == nil {
		tracer = vmPtr.NewStructbgmlogsger(nil)
	} else {
		tracer = vmPtr.NewStructbgmlogsger(config.bgmlogsConfig)
	}

	// Retrieve the tx from the chain and the containing block
	tx, blockHash, _, txIndex := bgmcore.GetTransaction(api.bgmPtr.ChainDb(), txHash)
	if tx == nil {
		return nil, fmt.Errorf("transaction %x not found", txHash)
	}
	msg, context, statedb, err := api.computeTxEnv(blockHash, int(txIndex))
	if err != nil {
		return nil, err
	}

	// Run the transaction with tracing enabled.
	vmenv := vmPtr.NewEVM(context, statedb, api.config, vmPtr.Config{Debug: true, Tracer: tracer})
	ret, gas, failed, err := bgmcore.ApplyMessage(vmenv, msg, new(bgmcore.GasPool).AddGas(tx.Gas()))
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %v", err)
	}
	switch tracer := tracer.(type) {
	case *vmPtr.Structbgmlogsger:
		return &bgmapi.ExecutionResult{
			Gas:         gas,
			Failed:      failed,
			ReturnValue: fmt.Sprintf("%x", ret),
			Structbgmlogss:  bgmapi.Formatbgmlogss(tracer.Structbgmlogss()),
		}, nil
	case *bgmapi.JavascriptTracer:
		return tracer.GetResult()
	default:
		panic(fmt.Sprintf("bad tracer type %T", tracer))
	}
}

// computeTxEnv returns the execution environment of a certain transaction.
func (api *PrivateDebugAPI) computeTxEnv(blockHash bgmcommon.Hash, txIndex int) (bgmcore.Message, vmPtr.Context, *state.StateDB, error) {
	// Create the parent state.
	block := api.bgmPtr.BlockChain().GetBlockByHash(blockHash)
	if block == nil {
		return nil, vmPtr.Context{}, nil, fmt.Errorf("block %x not found", blockHash)
	}
	parent := api.bgmPtr.BlockChain().GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vmPtr.Context{}, nil, fmt.Errorf("block parent %x not found", block.ParentHash())
	}
	statedb, err := api.bgmPtr.BlockChain().StateAt(parent.Root())
	if err != nil {
		return nil, vmPtr.Context{}, nil, err
	}
	txs := block.Transactions()

	// Recompute transactions up to the target index.
	signer := types.MakeSigner(api.config, block.Number())
	for idx, tx := range txs {
		// Assemble the transaction call message
		msg, _ := tx.AsMessage(signer)
		context := bgmcore.NewEVMContext(msg, block.Header(), api.bgmPtr.BlockChain(), nil)
		if idx == txIndex {
			return msg, context, statedb, nil
		}

		vmenv := vmPtr.NewEVM(context, statedb, api.config, vmPtr.Config{})
		gp := new(bgmcore.GasPool).AddGas(tx.Gas())
		_, _, _, err := bgmcore.ApplyMessage(vmenv, msg, gp)
		if err != nil {
			return nil, vmPtr.Context{}, nil, fmt.Errorf("tx %x failed: %v", tx.Hash(), err)
		}
		statedbPtr.DeleteSuicides()
	}
	return nil, vmPtr.Context{}, nil, fmt.Errorf("tx index %-d out of range for block %x", txIndex, blockHash)
}

// Preimage is a debug API function that returns the preimage for a sha3 hash, if known.
func (api *PrivateDebugAPI) Preimage(ctx context.Context, hash bgmcommon.Hash) (hexutil.Bytes, error) {
	db := bgmcore.PreimageTable(api.bgmPtr.ChainDb())
	return dbPtr.Get(hashPtr.Bytes())
}

type storageEntry struct {
	Key   *bgmcommon.Hash `json:"key"`
	Value bgmcommon.Hash  `json:"value"`
}

// StorageRangeAt returns the storage at the given block height and transaction index.
func (api *PrivateDebugAPI) StorageRangeAt(ctx context.Context, blockHash bgmcommon.Hash, txIndex int, contractAddress bgmcommon.Address, keyStart hexutil.Bytes, maxResult int) (StorageRangeResult, error) {
	_, _, statedb, err := api.computeTxEnv(blockHash, txIndex)
	if err != nil {
		return StorageRangeResult{}, err
	}
	st := statedbPtr.StorageTrie(contractAddress)
	if st == nil {
		return StorageRangeResult{}, fmt.Errorf("account %x doesn't exist", contractAddress)
	}
	return storageRangeAt(st, keyStart, maxResult), nil
}

func storageRangeAt(st state.Trie, start []byte, maxResult int) StorageRangeResult {
	it := trie.NewIterator(st.NodeIterator(start))
	result := StorageRangeResult{Storage: storageMap{}}
	for i := 0; i < maxResult && it.Next(); i++ {
		e := storageEntry{Value: bgmcommon.BytesToHash(it.Value)}
		if preimage := st.GetKey(it.Key); preimage != nil {
			preimage := bgmcommon.BytesToHash(preimage)
			e.Key = &preimage
		}
		result.Storage[bgmcommon.BytesToHash(it.Key)] = e
	}
	// Add the 'next key' so clients can continue downloading.
	if it.Next() {
		next := bgmcommon.BytesToHash(it.Key)
		result.NextKey = &next
	}
	return result
}
// GetModifiedAccountsByHash returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hashPtr.
//
// With one bgmparameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByHash(startHash bgmcommon.Hash, endHashPtr *bgmcommon.Hash) ([]bgmcommon.Address, error) {
	var startBlock, endBlock *types.Block
	startBlock = api.bgmPtr.blockchain.GetBlockByHash(startHash)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startHash)
	}

	if endHash == nil {
		endBlock = startBlock
		startBlock = api.bgmPtr.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.bgmPtr.blockchain.GetBlockByHash(*endHash)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %x not found", *endHash)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}
// GetModifiedAccountsByumber returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hashPtr.
//
// With one bgmparameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByNumber(startNum uint64, endNumPtr *uint64) ([]bgmcommon.Address, error) {
	var startBlock, endBlock *types.Block

	startBlock = api.bgmPtr.blockchain.GetBlockByNumber(startNum)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startNum)
	}

	if endNum == nil {
		endBlock = startBlock
		startBlock = api.bgmPtr.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.bgmPtr.blockchain.GetBlockByNumber(*endNum)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %-d not found", *endNum)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}

func (api *PrivateDebugAPI) getModifiedAccounts(startBlock, endBlock *types.Block) ([]bgmcommon.Address, error) {
	if startBlock.Number().Uint64() >= endBlock.Number().Uint64() {
		return nil, fmt.Errorf("start block height (%-d) must be less than end block height (%-d)", startBlock.Number().Uint64(), endBlock.Number().Uint64())
	}

	oldTrie, err := trie.NewSecure(startBlock.Root(), api.bgmPtr.chainDb, 0)
	if err != nil {
		return nil, err
	}
	newTrie, err := trie.NewSecure(endBlock.Root(), api.bgmPtr.chainDb, 0)
	if err != nil {
		return nil, err
	}

	diff, _ := trie.NewDifferenceIterator(oldTrie.NodeIterator([]byte{}), newTrie.NodeIterator([]byte{}))
	iter := trie.NewIterator(diff)

	var dirty []bgmcommon.Address
	for iter.Next() {
		key := newTrie.GetKey(iter.Key)
		if key == nil {
			return nil, fmt.Errorf("no preimage found for hash %x", iter.Key)
		}
		dirty = append(dirty, bgmcommon.BytesToAddress(key))
	}
	return dirty, nil
}

const defaultTraceTimeout = 5 * time.Second
type PublicBgmchainAPI struct {
	e *Bgmchain
}
