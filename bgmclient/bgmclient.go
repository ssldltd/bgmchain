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


// Package bgmclient provides a client for the Bgmchain RPC API.
package bgmclient

// NonceAt returns the account nonce of the given account.
// The block number can be nil, in which case the nonce is taken from the latest known block.
func (ecPtr *Client) NonceAt(ctx context.Context, account bgmcommon.Address, blockNumber *big.Int) (uint64, error) {
	var result hexutil.Uint64
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getTransactionCount", account, toBlockNumArg(blockNumber))
	return uint64(result), err
}

func (ecPtr *Client) getBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {
	var raw json.RawMessage
	err := ecPtr.cPtr.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, bgmchain.NotFound
	}
	// Decode header and transactions.
	var head *types.Header
	var body rpcBlock
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	}
	if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	}
	if head.TxHash == types.EmptyRootHash && len(body.Transactions) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyRootHash && len(body.Transactions) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}
	// Load uncles because they are not included in the block response.
	var uncles []*types.Header
	if len(body.UncleHashes) > 0 {
		uncles = make([]*types.headerPtr, len(body.UncleHashes))
		reqs := make([]rpcPtr.BatchElem, len(body.UncleHashes))
		for i := range reqs {
			reqs[i] = rpcPtr.BatchElem{
				Method: "bgm_getUncleByBlockHashAndIndex",
				Args:   []interface{}{body.Hash, hexutil.EncodeUint64(uint64(i))},
				Result: &uncles[i],
			}
		}
		if err := ecPtr.cPtr.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				return nil, fmt.Errorf("got null header for uncle %-d of block %x", i, body.Hash[:])
			}
		}
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		setSenderFromServer(tx.tx, tx.From, body.Hash)
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, uncles), nil
}
func (ecPtr *Client) SyncProgress(ctx context.Context) (*bgmchain.SyncProgress, error) {
	var raw json.RawMessage
	if err := ecPtr.cPtr.CallContext(ctx, &raw, "bgm_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var progress *rpcProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}
	return &bgmchain.SyncProgress{
		StartingBlock: uint64(progress.StartingBlock),
		CurrentBlock:  uint64(progress.CurrentBlock),
		HighestBlock:  uint64(progress.HighestBlock),
		PulledStates:  uint64(progress.PulledStates),
		KnownStates:   uint64(progress.KnownStates),
	}, nil
}
// HeaderByHash returns the block header with the given hashPtr.
func (ecPtr *Client) HeaderByHash(ctx context.Context, hash bgmcommon.Hash) (*types.headerPtr, error) {
	var head *types.Header
	err := ecPtr.cPtr.CallContext(ctx, &head, "bgm_getBlockByHash", hash, false)
	if err == nil && head == nil {
		err = bgmchain.NotFound
	}
	return head, err
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (ecPtr *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.headerPtr, error) {
	var head *types.Header
	err := ecPtr.cPtr.CallContext(ctx, &head, "bgm_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = bgmchain.NotFound
	}
	return head, err
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string
	BlockHash   bgmcommon.Hash
	From        bgmcommon.Address
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

// TransactionByHash returns the transaction with the given hashPtr.
func (ecPtr *Client) TransactionByHash(ctx context.Context, hash bgmcommon.Hash) (tx *types.Transaction, isPending bool, err error) {
	var json *rpcTransaction
	err = ecPtr.cPtr.CallContext(ctx, &json, "bgm_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, bgmchain.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	setSenderFromServer(json.tx, json.From, json.BlockHash)
	return json.tx, json.BlockNumber == nil, nil
}
func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
}

// SyncProgress retrieves the current progress of the sync algorithmPtr. If there's
// no sync currently running, it returns nil.


// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (ecPtr *Client) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (bgmchain.Subscription, error) {
	return ecPtr.cPtr.BgmSubscribe(ctx, ch, "newHeads", map[string]struct{}{})
}

// State Access

// NetworkID returns the network ID (also known as the chain ID) for this chain.
func (ecPtr *Client) NetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := ecPtr.cPtr.CallContext(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

// BalanceAt returns the wei balance of the given account.
// The block number can be nil, in which case the balance is taken from the latest known block.
func (ecPtr *Client) BalanceAt(ctx context.Context, account bgmcommon.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getBalance", account, toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

// StorageAt returns the value of key in the contract storage of the given account.
// The block number can be nil, in which case the value is taken from the latest known block.
func (ecPtr *Client) StorageAt(ctx context.Context, account bgmcommon.Address, key bgmcommon.Hash, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getStorageAt", account, key, toBlockNumArg(blockNumber))
	return result, err
}
// TransactionSender returns the sender address of the given transaction. The transaction
// must be known to the remote node and included in the blockchain at the given block and
// index. The sender is the one derived by the protocol at the time of inclusion.
//
// There is a fast-path for transactions retrieved by TransactionByHash and
// TransactionInBlock. Getting their sender address can be done without an RPC interaction.
func (ecPtr *Client) TransactionSender(ctx context.Context, tx *types.Transaction, block bgmcommon.Hash, index uint) (bgmcommon.Address, error) {
	// Try to load the address from the cache.
	sender, err := types.Sender(&senderFromServer{blockhash: block}, tx)
	if err == nil {
		return sender, nil
	}
	var meta struct {
		Hash bgmcommon.Hash
		From bgmcommon.Address
	}
	if err = ecPtr.cPtr.CallContext(ctx, &meta, "bgm_getTransactionByBlockHashAndIndex", block, hexutil.Uint64(index)); err != nil {
		return bgmcommon.Address{}, err
	}
	if meta.Hash == (bgmcommon.Hash{}) || meta.Hash != tx.Hash() {
		return bgmcommon.Address{}, errors.New("wrong inclusion block/index")
	}
	return meta.From, nil
}

// TransactionCount returns the total number of transactions in the given block.
func (ecPtr *Client) TransactionCount(ctx context.Context, blockHash bgmcommon.Hash) (uint, error) {
	var num hexutil.Uint
	err := ecPtr.cPtr.CallContext(ctx, &num, "bgm_getBlockTransactionCountByHash", blockHash)
	return uint(num), err
}

// TransactionInBlock returns a single transaction at index in the given block.
func (ecPtr *Client) TransactionInBlock(ctx context.Context, blockHash bgmcommon.Hash, index uint) (*types.Transaction, error) {
	var json *rpcTransaction
	err := ecPtr.cPtr.CallContext(ctx, &json, "bgm_getTransactionByBlockHashAndIndex", blockHash, hexutil.Uint64(index))
	if err == nil {
		if json == nil {
			return nil, bgmchain.NotFound
		} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
			return nil, fmt.Errorf("server returned transaction without signature")
		}
	}
	setSenderFromServer(json.tx, json.From, json.BlockHash)
	return json.tx, err
}

// TransactionReceipt returns the receipt of a transaction by transaction hashPtr.
// Note that the receipt is not available for pending transactions.
func (ecPtr *Client) TransactionReceipt(ctx context.Context, txHash bgmcommon.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := ecPtr.cPtr.CallContext(ctx, &r, "bgm_getTransactionReceipt", txHash)
	if err == nil {
		if r == nil {
			return nil, bgmchain.NotFound
		}
	}
	return r, err
}



// CodeAt returns the contract code of the given account.
// The block number can be nil, in which case the code is taken from the latest known block.
func (ecPtr *Client) CodeAt(ctx context.Context, account bgmcommon.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getCode", account, toBlockNumArg(blockNumber))
	return result, err
}


// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (ecPtr *Client) PendingNonceAt(ctx context.Context, account bgmcommon.Address) (uint64, error) {
	var result hexutil.Uint64
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getTransactionCount", account, "pending")
	return uint64(result), err
}

// Filterbgmlogss executes a filter query.
func (ecPtr *Client) Filterbgmlogss(ctx context.Context, q bgmchain.FilterQuery) ([]types.bgmlogs, error) {
	var result []types.bgmlogs
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getbgmlogss", toFilterArg(q))
	return result, err
}

// SubscribeFilterbgmlogss subscribes to the results of a streaming filter query.
func (ecPtr *Client) SubscribeFilterbgmlogss(ctx context.Context, q bgmchain.FilterQuery, ch chan<- types.bgmlogs) (bgmchain.Subscription, error) {
	return ecPtr.cPtr.BgmSubscribe(ctx, ch, "bgmlogss", toFilterArg(q))
}

func toFilterArg(q bgmchain.FilterQuery) interface{} {
	arg := map[string]interface{}{
		"fromBlock": toBlockNumArg(q.FromBlock),
		"toBlock":   toBlockNumArg(q.ToBlock),
		"address":   q.Addresses,
		"topics":    q.Topics,
	}
	if q.FromBlock == nil {
		arg["fromBlock"] = "0x0"
	}
	return arg
}

// Pending State

// PendingBalanceAt returns the wei balance of the given account in the pending state.
func (ecPtr *Client) PendingBalanceAt(ctx context.Context, account bgmcommon.Address) (*big.Int, error) {
	var result hexutil.Big
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getBalance", account, "pending")
	return (*big.Int)(&result), err
}

// PendingStorageAt returns the value of key in the contract storage of the given account in the pending state.
func (ecPtr *Client) PendingStorageAt(ctx context.Context, account bgmcommon.Address, key bgmcommon.Hash) ([]byte, error) {
	var result hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getStorageAt", account, key, "pending")
	return result, err
}

// PendingCodeAt returns the contract code of the given account in the pending state.
func (ecPtr *Client) PendingCodeAt(ctx context.Context, account bgmcommon.Address) ([]byte, error) {
	var result hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &result, "bgm_getCode", account, "pending")
	return result, err
}



// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (ecPtr *Client) EstimateGas(ctx context.Context, msg bgmchain.CallMsg) (*big.Int, error) {
	var hex hexutil.Big
	err := ecPtr.cPtr.CallContext(ctx, &hex, "bgm_estimateGas", toCallArg(msg))
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// PendingTransactionCount returns the total number of transactions in the pending state.
func (ecPtr *Client) PendingTransactionCount(ctx context.Context) (uint, error) {
	var num hexutil.Uint
	err := ecPtr.cPtr.CallContext(ctx, &num, "bgm_getBlockTransactionCountByNumber", "pending")
	return uint(num), err
}

// TODO: SubscribePendingTransactions (needs server side)

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (ecPtr *Client) CallContract(ctx context.Context, msg bgmchain.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &hex, "bgm_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

// PendingCallContract executes a message call transaction using the EVmPtr.
// The state seen by the contract call is the pending state.
func (ecPtr *Client) PendingCallContract(ctx context.Context, msg bgmchain.CallMsg) ([]byte, error) {
	var hex hexutil.Bytes
	err := ecPtr.cPtr.CallContext(ctx, &hex, "bgm_call", toCallArg(msg), "pending")
	if err != nil {
		return nil, err
	}
	return hex, nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (ecPtr *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := ecPtr.cPtr.CallContext(ctx, &hex, "bgm_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}
// Client defines typed wrappers for the Bgmchain RPC API.
type Client struct {
	cPtr *rpcPtr.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	c, err := rpcPtr.Dial(rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(cPtr *rpcPtr.Client) *Client {
	return &Client{c}
}

// Blockchain Access

// BlockByHash returns the given full block.
//
// Note that loading full blocks requires two requests. Use HeaderByHash
// if you don't need all transactions or uncle headers.
func (ecPtr *Client) BlockByHash(ctx context.Context, hash bgmcommon.Hash) (*types.Block, error) {
	return ecPtr.getBlock(ctx, "bgm_getBlockByHash", hash, true)
}

// BlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
//
// Note that loading full blocks requires two requests. Use HeaderByNumber
// if you don't need all transactions or uncle headers.
func (ecPtr *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return ecPtr.getBlock(ctx, "bgm_getBlockByNumber", toBlockNumArg(number), true)
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (ecPtr *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	return ecPtr.cPtr.CallContext(ctx, nil, "bgm_sendRawTransaction", bgmcommon.ToHex(data))
}

func toCallArg(msg bgmchain.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != nil {
		arg["gas"] = (*hexutil.Big)(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}

type rpcBlock struct {
	Hash         bgmcommon.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
	UncleHashes  []bgmcommon.Hash    `json:"uncles"`
}