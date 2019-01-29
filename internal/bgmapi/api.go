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
package bgmapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/account/keystore"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmCore/vm"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	defaultGas      = 90000
	defaultGasPrice = 50 * bgmparam.ShannonUnit
)

// PublicBgmchainAPI provides an apiPtr to access Bgmchain related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBgmchainAPI struct {
	b Backend
}

// NewPublicBgmchainAPI creates a new Bgmchain protocol apiPtr.
func NewPublicBgmchainAPI(b Backend) *PublicBgmchainAPI {
	return &PublicBgmchainAPI{b}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicBgmchainAPI) GasPrice(CTX context.Context) (*big.Int, error) {
	return s.bPtr.SuggestPrice(CTX)
}

// ProtocolVersion returns the current Bgmchain protocol version this node supports
func (s *PublicBgmchainAPI) ProtocolVersion() hexutil.Uint {
	return hexutil.Uint(s.bPtr.ProtocolVersion())
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block Headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block Header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicBgmchainAPI) Syncing() (interface{}, error) {
	progress := s.bPtr.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": hexutil.Uint64(progress.StartingBlock),
		"currentBlock":  hexutil.Uint64(progress.CurrentBlock),
		"highestBlock":  hexutil.Uint64(progress.HighestBlock),
		"pulledStates":  hexutil.Uint64(progress.PulledStates),
		"knownStates":   hexutil.Uint64(progress.KnownStates),
	}, nil
}

// PublicTxPoolAPI offers and apiPtr for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	b Backend
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(b Backend) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := s.bPtr.TxPoolContent()

	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%-d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%-d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := s.bPtr.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := s.bPtr.TxPoolContent()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%-s: %v WeiUnit + %v gas × %v WeiUnit", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v WeiUnit + %v gas × %v WeiUnit", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%-d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%-d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an apiPtr to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	amPtr *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(amPtr *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() []bgmcommon.Address {
	addresses := make([]bgmcommon.Address, 0) // return [] instead of nil if empty
	for _, wallet := range s.amPtr.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}
	return addresses
}

// PrivateAccountAPI provides an apiPtr to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	am        *accounts.Manager
	nonceLock *AddrLocker
	b         Backend
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(b Backend, nonceLock *AddrLocker) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		am:        bPtr.AccountManager(),
		nonceLock: nonceLock,
		b:         b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []bgmcommon.Address {
	addresses := make([]bgmcommon.Address, 0) // return [] instead of nil if empty
	for _, wallet := range s.amPtr.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}
	return addresses
}

// rawWallet is a JSON representation of an accounts.Wallet interface, with its
// data contents extracted into plain fields.
type rawWallet struct {
	URL      string             `json:"url"`
	Status   string             `json:"status"`
	Failure  string             `json:"failure,omitempty"`
	Accounts []accounts.Account `json:"accounts,omitempty"`
}

// ListWallets will return a list of wallets this node manages.
func (s *PrivateAccountAPI) ListWallets() []rawWallet {
	wallets := make([]rawWallet, 0) // return [] instead of nil if empty
	for _, wallet := range s.amPtr.Wallets() {
		status, failure := wallet.Status()

		raw := rawWallet{
			URL:      wallet.URL().String(),
			Status:   status,
			Accounts: wallet.Accounts(),
		}
		if failure != nil {
			raw.Failure = failure.Error()
		}
		wallets = append(wallets, raw)
	}
	return wallets
}

// OpenWallet initiates a hardware wallet opening procedure, establishing a USB
// connection and attempting to authenticate via the provided passphrase. Note,
// the method may return an extra challenge requiring a second open (e.g. the
// Trezor PIN matrix challenge).
func (s *PrivateAccountAPI) OpenWallet(url string, passphrase *string) error {
	wallet, err := s.amPtr.Wallet(url)
	if err != nil {
		return err
	}
	pass := ""
	if passphrase != nil {
		pass = *passphrase
	}
	return wallet.Open(pass)
}

// DeriveAccount requests a HD wallet to derive a new account, optionally pinning
// it for later reuse.
func (s *PrivateAccountAPI) DeriveAccount(url string, path string, pin *bool) (accounts.Account, error) {
	wallet, err := s.amPtr.Wallet(url)
	if err != nil {
		return accounts.Account{}, err
	}
	derivPath, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return accounts.Account{}, err
	}
	if pin == nil {
		pin = new(bool)
	}
	return wallet.Derive(derivPath, *pin)
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (bgmcommon.Address, error) {
	acc, err := fetchKeystore(s.am).NewAccount(password)
	if err == nil {
		return accPtr.Address, nil
	}
	return bgmcommon.Address{}, err
}

// fetchKeystore retrives the encrypted keystore from the account manager.
func fetchKeystore(amPtr *accounts.Manager) *keystore.KeyStore {
	return amPtr.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (bgmcommon.Address, error) {
	key, err := bgmcrypto.HexToECDSA(privkey)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	acc, err := fetchKeystore(s.am).ImportECDSA(key, password)
	return accPtr.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(addr bgmcommon.Address, password string, duration *Uint64) (bool, error) {
	const max = Uint64(time.Duration(mathPtr.MaxInt64) / time.Second)
	var d time.Duration
	if duration == nil {
		d = 300 * time.Second
	} else if *duration > max {
		return false, errors.New("unlock duration too large")
	} else {
		d = time.Duration(*duration) * time.Second
	}
	err := fetchKeystore(s.am).timedUnlock(accounts.Account{Address: addr}, password, d)
	return err == nil, err
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr bgmcommon.Address) bool {
	return fetchKeystore(s.am).Lock(addr) == nil
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(CTX context.Context, args SendTxArgs, passwd string) (bgmcommon.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}

	wallet, err := s.amPtr.Find(account)
	if err != nil {
		return bgmcommon.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(CTX, s.b); err != nil {
		return bgmcommon.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	var BlockChainId *big.Int
	if config := s.bPtr.ChainConfig(); config.IsChain155(s.bPtr.CurrentBlock().Number()) {
		BlockChainId = config.BlockChainId
	}
	signed, err := wallet.SignTxWithPassphrase(account, passwd, tx, BlockChainId)
	if err != nil {
		return bgmcommon.Hash{}, err
	}
	return submitTransaction(CTX, s.b, signed)
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature fromPtr.
//
// The hash is calulcated as
//   keccak256("\x19Bgmchain Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Bgmchain Signed Message:\n%-d%-s", len(data), data)
	return bgmcrypto.Keccak256([]byte(msg))
}

// Sign calculates an Bgmchain ECDSA signature for:
// keccack256("\x19Bgmchain Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ssldltd/bgmchain/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(CTX context.Context, data hexutil.Bytes, addr bgmcommon.Address, passwd string) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.bPtr.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignHashWithPassphrase(account, passwd, signHash(data))
	if err != nil {
		return nil, err
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with bgm_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Bgmchain Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// Note, the signature must conform to the secp256k1 curve R, S and V values, where
// the V value must be be 27 or 28 for legacy reasons.
//
// https://github.com/ssldltd/bgmchain/wiki/Management-APIs#personal_ecRecover
func (s *PrivateAccountAPI) EcRecover(CTX context.Context, data, sig hexutil.Bytes) (bgmcommon.Address, error) {
	if len(sig) != 65 {
		return bgmcommon.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return bgmcommon.Address{}, fmt.Errorf("invalid Bgmchain signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := bgmcrypto.Ecrecover(signHash(data), sig)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	pubKey := bgmcrypto.ToECDSAPub(rpk)
	recoveredAddr := bgmcrypto.PubkeyToAddress(*pubKey)
	return recoveredAddr, nil
}

// SignAndSendTransaction was renamed to SendTransaction. This method is deprecated
// and will be removed in the future. It primary goal is to give Clients time to update.
func (s *PrivateAccountAPI) SignAndSendTransaction(CTX context.Context, args SendTxArgs, passwd string) (bgmcommon.Hash, error) {
	return s.SendTransaction(CTX, args, passwd)
}

// PublicBlockChainAPI provides an apiPtr to access the Bgmchain blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	b Backend
}

// NewPublicBlockChainAPI creates a new Bgmchain blockchain apiPtr.
func NewPublicBlockChainAPI(b Backend) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{b}
}

// number returns the block number of the chain head.
func (s *PublicBlockChainAPI) number() *big.Int {
	HeaderPtr, _ := s.bPtr.HeaderByNumber(context.Background(), rpcPtr.Latestnumber) // latest Header should always be available
	return HeaderPtr.Number
}

// GetBalance returns the amount of WeiUnit for the given address in the state of the
// given block number. The rpcPtr.Latestnumber and rpcPtr.Pendingnumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(CTX context.Context, address bgmcommon.Address, blockNr rpcPtr.number) (*big.Int, error) {
	state, _, err := s.bPtr.StateAndHeaderByNumber(CTX, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	b := state.GetBalance(address)
	return b, state.Error()
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(CTX context.Context, blockNr rpcPtr.number, fullTx bool) (map[string]interface{}, error) {
	block, err := s.bPtr.BlockByNumber(CTX, blockNr)
	if block != nil {
		response, err := s.rpcOutputBlock(block, true, fullTx)
		if err == nil && blockNr == rpcPtr.Pendingnumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(CTX context.Context, blockHash bgmcommon.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := s.bPtr.GetBlock(CTX, blockHash)
	if block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, err
}

// GetUncleBynumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleBynumberAndIndex(CTX context.Context, blockNr rpcPtr.number, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.bPtr.BlockByNumber(CTX, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			bgmlogs.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, err
}

// GetUncleByhashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByhashAndIndex(CTX context.Context, blockHash bgmcommon.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.bPtr.GetBlock(CTX, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			bgmlogs.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, err
}

// GetUncleCountBynumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainAPI) GetUncleCountBynumber(CTX context.Context, blockNr rpcPtr.number) *hexutil.Uint {
	if block, _ := s.bPtr.BlockByNumber(CTX, blockNr); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetUncleCountByhash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByhash(CTX context.Context, blockHash bgmcommon.Hash) *hexutil.Uint {
	if block, _ := s.bPtr.GetBlock(CTX, blockHash); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(CTX context.Context, address bgmcommon.Address, blockNr rpcPtr.number) (hexutil.Bytes, error) {
	state, _, err := s.bPtr.StateAndHeaderByNumber(CTX, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpcPtr.Latestnumber and rpcPtr.Pendingnumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(CTX context.Context, address bgmcommon.Address, key string, blockNr rpcPtr.number) (hexutil.Bytes, error) {
	state, _, err := s.bPtr.StateAndHeaderByNumber(CTX, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	res := state.GetState(address, bgmcommon.HexToHash(key))
	return res[:], state.Error()
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     bgmcommon.Address  `json:"from"`
	To       *bgmcommon.Address `json:"to"`
	Gas      hexutil.Big     `json:"gas"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

func (s *PublicBlockChainAPI) doCall(CTX context.Context, args CallArgs, blockNr rpcPtr.number, vmCfg vmPtr.Config) ([]byte, *big.Int, bool, error) {
	defer func(start time.time) { bgmlogs.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, HeaderPtr, err := s.bPtr.StateAndHeaderByNumber(CTX, blockNr)
	if state == nil || err != nil {
		return nil, bgmcommon.Big0, false, err
	}
	// Set sender address or use a default if none specified
	addr := args.From
	if addr == (bgmcommon.Address{}) {
		if wallets := s.bPtr.AccountManager().Wallets(); len(wallets) > 0 {
			if accounts := wallets[0].Accounts(); len(accounts) > 0 {
				addr = accounts[0].Address
			}
		}
	}
	// Set default gas & gas price if none were set
	gas, gasPrice := args.Gas.ToInt(), args.GasPrice.ToInt()
	if gas.Sign() == 0 {
		gas = big.NewInt(50000000)
	}
	if gasPrice.Sign() == 0 {
		gasPrice = new(big.Int).SetUint64(defaultGasPrice)
	}

	// Create new call message
	msg := types.NewMessage(addr, args.To, 0, args.Value.ToInt(), gas, gasPrice, args.Data, false)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if vmCfg.DisableGasMetering {
		CTX, cancel = context.Withtimeout(CTX, time.Second*5)
	} else {
		CTX, cancel = context.WithCancel(CTX)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer func() { cancel() }()

	// Get a new instance of the EVmPtr.
	evm, vmError, err := s.bPtr.GetEVM(CTX, msg, state, HeaderPtr, vmCfg)
	if err != nil {
		return nil, bgmcommon.Big0, false, err
	}
	// Wait for the context to be done and cancel the evmPtr. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-CTX.Done()
		evmPtr.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(bgmCore.GasPool).AddGas(mathPtr.MaxBig256)
	res, gas, failed, err := bgmCore.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, bgmcommon.Big0, false, err
	}
	return res, gas, failed, err
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(CTX context.Context, args CallArgs, blockNr rpcPtr.number) (hexutil.Bytes, error) {
	result, _, _, err := s.doCall(CTX, args, blockNr, vmPtr.Config{DisableGasMetering: true})
	return (hexutil.Bytes)(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *PublicBlockChainAPI) EstimateGas(CTX context.Context, args CallArgs) (*hexutil.Big, error) {
	// Determine the lowest and highest possible gas limits to binary search in between
	var (
		lo  Uint64 = bgmparam.TxGas - 1
		hi  Uint64
		cap Uint64
	)
	if (*big.Int)(&args.Gas).Uint64() >= bgmparam.TxGas {
		hi = (*big.Int)(&args.Gas).Uint64()
	} else {
		// Retrieve the current pending block to act as the gas ceiling
		block, err := s.bPtr.BlockByNumber(CTX, rpcPtr.Pendingnumber)
		if err != nil {
			return nil, err
		}
		hi = block.GasLimit().Uint64()
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas Uint64) bool {
		(*big.Int)(&args.Gas).SetUint64(gas)
		_, _, failed, err := s.doCall(CTX, args, rpcPtr.Pendingnumber, vmPtr.Config{})
		if err != nil || failed {
			return false
		}
		return true
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		if !executable(hi) {
			return nil, fmt.Errorf("gas required exceeds allowance or always failing transaction")
		}
	}
	return (*hexutil.Big)(new(big.Int).SetUint64(hi)), nil
}

// ExecutionResult groups all structured bgmlogss emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         *big.Int       `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	Structbgmlogss  []StructbgmlogsRes `json:"structbgmlogss"`
}

// StructbgmlogsRes stores a structured bgmlogs emitted by the EVM while replaying a
// transaction in debug mode
type StructbgmlogsRes struct {
	Pc      Uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     Uint64             `json:"gas"`
	GasCost Uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   error              `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

// formatbgmlogss formats EVM returned structured bgmlogss for json output
func Formatbgmlogss(bgmlogss []vmPtr.Structbgmlogs) []StructbgmlogsRes {
	formatted := make([]StructbgmlogsRes, len(bgmlogss))
	for index, trace := range bgmlogss {
		formatted[index] = StructbgmlogsRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.Err,
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = fmt.Sprintf("%x", mathPtr.PaddedBigBytes(stackValue, 32))
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(bPtr *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	head := bPtr.Header() // copies the Header once
	fields := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             bPtr.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"bgmlogssBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"validator":        head.Validator,
		"coinbase":         head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"totalDifficulty":  (*hexutil.Big)(s.bPtr.GetTd(bPtr.Hash())),
		"extraData":        hexutil.Bytes(head.Extra),
		"size":             hexutil.Uint64(Uint64(bPtr.Size().Int64())),
		"gasLimit":         (*hexutil.Big)(head.GasLimit),
		"gasUsed":          (*hexutil.Big)(head.GasUsed),
		"timestamp":        (*hexutil.Big)(head.time),
		"transactionsRoot": head.TxHash,
		"recChaintsRoot":     head.RecChaintHash,
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromhash(b, tx.Hash()), nil
			}
		}

		txs := bPtr.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range bPtr.Transactions() {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	uncles := bPtr.Uncles()
	uncleHashes := make([]bgmcommon.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes

	return fields, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	Type             types.TxType    `json:"type"`
	hash        bgmcommon.Hash     `json:"blockHash"`
	number      *hexutil.Big    `json:"blockNumber"`
	From             bgmcommon.Address  `json:"from"`
	Gas              *hexutil.Big    `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             bgmcommon.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *bgmcommon.Address `json:"to"`
	TransactionIndex hexutil.Uint    `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash bgmcommon.Hash, blockNumber Uint64, index Uint64) *RPCTransaction {
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewChain155Signer(tx.BlockChainId())
	}
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()

	result := &RPCTransaction{
		Type:     tx.Type(),
		From:     from,
		Gas:      (*hexutil.Big)(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (bgmcommon.Hash{}) {
		result.hash = blockHash
		result.number = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = hexutil.Uint(index)
	}
	return result
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	return newRPCTransaction(tx, bgmcommon.Hash{}, 0, 0)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(bPtr *types.Block, index Uint64) *RPCTransaction {
	txs := bPtr.Transactions()
	if index >= Uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], bPtr.Hash(), bPtr.NumberU64(), index)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(bPtr *types.Block, index Uint64) hexutil.Bytes {
	txs := bPtr.Transactions()
	if index >= Uint64(len(txs)) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(txs[index])
	return blob
}

// newRPCTransactionFromhash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromhash(bPtr *types.Block, hash bgmcommon.Hash) *RPCTransaction {
	for idx, tx := range bPtr.Transactions() {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, Uint64(idx))
		}
	}
	return nil
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b         Backend
	nonceLock *AddrLocker
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend, nonceLock *AddrLocker) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{b, nonceLock}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(CTX context.Context, blockNr rpcPtr.number) *hexutil.Uint {
	if block, _ := s.bPtr.BlockByNumber(CTX, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hashPtr.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(CTX context.Context, blockHash bgmcommon.Hash) *hexutil.Uint {
	if block, _ := s.bPtr.GetBlock(CTX, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetTransactionBynumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionBynumberAndIndex(CTX context.Context, blockNr rpcPtr.number, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.bPtr.BlockByNumber(CTX, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, Uint64(index))
	}
	return nil
}

// GetTransactionByhashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByhashAndIndex(CTX context.Context, blockHash bgmcommon.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.bPtr.GetBlock(CTX, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, Uint64(index))
	}
	return nil
}

// GetRawTransactionBynumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionBynumberAndIndex(CTX context.Context, blockNr rpcPtr.number, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.bPtr.BlockByNumber(CTX, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, Uint64(index))
	}
	return nil
}

// GetRawTransactionByhashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByhashAndIndex(CTX context.Context, blockHash bgmcommon.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.bPtr.GetBlock(CTX, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, Uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(CTX context.Context, address bgmcommon.Address, blockNr rpcPtr.number) (*hexutil.Uint64, error) {
	state, _, err := s.bPtr.StateAndHeaderByNumber(CTX, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(CTX context.Context, hash bgmcommon.Hash) *RPCTransaction {
	// Try to return an already finalized transaction
	if tx, blockHash, blockNumber, index := bgmCore.GetTransaction(s.bPtr.ChainDb(), hash); tx != nil {
		return newRPCTransaction(tx, blockHash, blockNumber, index)
	}
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.bPtr.GetPoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx)
	}
	// Transaction unknown, return as such
	return nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hashPtr.
func (s *PublicTransactionPoolAPI) GetRawTransactionByHash(CTX context.Context, hash bgmcommon.Hash) (hexutil.Bytes, error) {
	var tx *types.Transaction

	// Retrieve a finalized transaction, or a pooled otherwise
	if tx, _, _, _ = bgmCore.GetTransaction(s.bPtr.ChainDb(), hash); tx == nil {
		if tx = s.bPtr.GetPoolTransaction(hash); tx == nil {
			// Transaction not found anywhere, abort
			return nil, nil
		}
	}
	// Serialize to RLP and return
	return rlp.EncodeToBytes(tx)
}

// GetTransactionRecChaint returns the transaction recChaint for the given transaction hashPtr.
func (s *PublicTransactionPoolAPI) GetTransactionRecChaint(hash bgmcommon.Hash) (map[string]interface{}, error) {
	tx, blockHash, blockNumber, index := bgmCore.GetTransaction(s.bPtr.ChainDb(), hash)
	if tx == nil {
		return nil, nil
	}
	recChaint, _, _, _ := bgmCore.GetRecChaint(s.bPtr.ChainDb(), hash) // Old recChaints don't have the lookup data available

	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewChain155Signer(tx.BlockChainId())
	}
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"type":              tx.Type(),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           (*hexutil.Big)(recChaint.GasUsed),
		"cumulativeGasUsed": (*hexutil.Big)(recChaint.CumulativeGasUsed),
		"contractAddress":   nil,
		"bgmlogss":              recChaint.bgmlogss,
		"bgmlogssBloom":         recChaint.Bloom,
	}

	// Assign recChaint status or post state.
	if len(recChaint.PostState) > 0 {
		fields["blockRoot"] = hexutil.Bytes(recChaint.PostState)
	} else {
		fields["status"] = hexutil.Uint(recChaint.Status)
	}
	if recChaint.bgmlogss == nil {
		fields["bgmlogss"] = [][]*types.bgmlogs{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if recChaint.ContractAddress != (bgmcommon.Address{}) {
		fields["contractAddress"] = recChaint.ContractAddress
	}
	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(addr bgmcommon.Address, tx *types.Transaction) (*types.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return nil, err
	}
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.bPtr.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	var BlockChainId *big.Int
	if config := s.bPtr.ChainConfig(); config.IsChain155(s.bPtr.CurrentBlock().Number()) {
		BlockChainId = config.BlockChainId
	}
	return wallet.SignTx(account, tx, BlockChainId)
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     bgmcommon.Address  `json:"from"`
	To       *bgmcommon.Address `json:"to"`
	Gas      *hexutil.Big    `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Type     types.TxType    `json:"type"`
}

// prepareSendTxArgs is a helper function that fills in default values for unspecified tx fields.
func (args *SendTxArgs) setDefaults(CTX context.Context, b Backend) error {
	if args.Gas == nil {
		args.Gas = (*hexutil.Big)(big.NewInt(defaultGas))
	}
	if args.GasPrice == nil {
		price, err := bPtr.SuggestPrice(CTX)
		if err != nil {
			return err
		}
		args.GasPrice = (*hexutil.Big)(price)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := bPtr.GetPoolNonce(CTX, args.From)
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	return nil
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	if args.Type == types.Binary && args.To == nil {
		return types.NewContractCreation(Uint64(*args.Nonce), (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), args.Data)
	}
	to := bgmcommon.Address{}
	if args.To != nil {
		to = *args.To
	}
	return types.NewTransaction(args.Type, Uint64(*args.Nonce), to, (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), args.Data)
}

// submitTransaction is a helper function that submits tx to txPool and bgmlogss a message.
func submitTransaction(CTX context.Context, b Backend, tx *types.Transaction) (bgmcommon.Hash, error) {
	if err := tx.Validate(); err != nil {
		return bgmcommon.Hash{}, err
	}
	if err := bPtr.SendTx(CTX, tx); err != nil {
		return bgmcommon.Hash{}, err
	}
	if tx.To() == nil {
		signer := types.MakeSigner(bPtr.ChainConfig(), bPtr.CurrentBlock().Number())
		from, err := types.Sender(signer, tx)
		if err != nil {
			return bgmcommon.Hash{}, err
		}
		addr := bgmcrypto.CreateAddress(from, tx.Nonce())
		bgmlogs.Info("Submitted contract creation", "fullhash", tx.Hash().Hex(), "contract", addr.Hex())
	} else {
		bgmlogs.Info("Submitted transaction", "fullhash", tx.Hash().Hex(), "recipient", tx.To())
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(CTX context.Context, args SendTxArgs) (bgmcommon.Hash, error) {

	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}

	wallet, err := s.bPtr.AccountManager().Find(account)
	if err != nil {
		return bgmcommon.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(CTX, s.b); err != nil {
		return bgmcommon.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	var BlockChainId *big.Int
	if config := s.bPtr.ChainConfig(); config.IsChain155(s.bPtr.CurrentBlock().Number()) {
		BlockChainId = config.BlockChainId
	}
	signed, err := wallet.SignTx(account, tx, BlockChainId)
	if err != nil {
		return bgmcommon.Hash{}, err
	}
	return submitTransaction(CTX, s.b, signed)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(CTX context.Context, encodedTx hexutil.Bytes) (bgmcommon.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return bgmcommon.Hash{}, err
	}
	return submitTransaction(CTX, s.b, tx)
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Bgmchain Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// https://github.com/bgmchain/wiki/wiki/JSON-RPC#bgm_sign
func (s *PublicTransactionPoolAPI) Sign(addr bgmcommon.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.bPtr.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignHash(account, signHash(data))
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(CTX context.Context, args SendTxArgs) (*SignTransactionResult, error) {
	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}
	if err := args.setDefaults(CTX, s.b); err != nil {
		return nil, err
	}
	tx, err := s.sign(args.From, args.toTransaction())
	if err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.bPtr.GetPoolTransactions()
	if err != nil {
		return nil, err
	}

	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if tx.Protected() {
			signer = types.NewChain155Signer(tx.BlockChainId())
		}
		from, _ := types.Sender(signer, tx)
		if _, err := s.bPtr.AccountManager().Find(accounts.Account{Address: from}); err == nil {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}
	return transactions, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(CTX context.Context, sendArgs SendTxArgs, gasPrice, gasLimit *hexutil.Big) (bgmcommon.Hash, error) {
	if sendArgs.Nonce == nil {
		return bgmcommon.Hash{}, fmt.Errorf("missing transaction nonce in transaction spec")
	}
	if err := sendArgs.setDefaults(CTX, s.b); err != nil {
		return bgmcommon.Hash{}, err
	}
	matchTx := sendArgs.toTransaction()
	pending, err := s.bPtr.GetPoolTransactions()
	if err != nil {
		return bgmcommon.Hash{}, err
	}

	for _, p := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if p.Protected() {
			signer = types.NewChain155Signer(p.BlockChainId())
		}
		wantSigHash := signer.Hash(matchTx)

		if pFrom, err := types.Sender(signer, p); err == nil && pFrom == sendArgs.From && signer.Hash(p) == wantSigHash {
			// MatchPtr. Re-sign and send the transaction.
			if gasPrice != nil {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := s.sign(sendArgs.From, sendArgs.toTransaction())
			if err != nil {
				return bgmcommon.Hash{}, err
			}
			if err = s.bPtr.SendTx(CTX, signedTx); err != nil {
				return bgmcommon.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}

	return bgmcommon.Hash{}, fmt.Errorf("Transaction %#x not found", matchTx.Hash())
}

// PublicDebugAPI is the collection of Bgmchain APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	b Backend
}

// NewPublicDebugAPI creates a new apiPtr definition for the public debug methods
// of the Bgmchain service.
func NewPublicDebugAPI(b Backend) *PublicDebugAPI {
	return &PublicDebugAPI{b: b}
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (apiPtr *PublicDebugAPI) GetBlockRlp(CTX context.Context, number Uint64) (string, error) {
	block, _ := apiPtr.bPtr.BlockByNumber(CTX, rpcPtr.number(number))
	if block == nil {
		return "", fmt.Errorf("block #%-d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed formPtr.
func (apiPtr *PublicDebugAPI) PrintBlock(CTX context.Context, number Uint64) (string, error) {
	block, _ := apiPtr.bPtr.BlockByNumber(CTX, rpcPtr.number(number))
	if block == nil {
		return "", fmt.Errorf("block #%-d not found", number)
	}
	return block.String(), nil
}

// PrivateDebugAPI is the collection of Bgmchain APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	b Backend
}

// NewPrivateDebugAPI creates a new apiPtr definition for the private debug methods
// of the Bgmchain service.
func NewPrivateDebugAPI(b Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{b: b}
}

// ChaindbProperty returns leveldb properties of the chain database.
func (apiPtr *PrivateDebugAPI) ChaindbProperty(property string) (string, error) {
	ldb, ok := apiPtr.bPtr.ChainDb().(interface {
		LDB() *leveldbPtr.DB
	})
	if !ok {
		return "", fmt.Errorf("chaindbProperty does not work for memory databases")
	}
	if property == "" {
		property = "leveldbPtr.stats"
	} else if !strings.HasPrefix(property, "leveldbPtr.") {
		property = "leveldbPtr." + property
	}
	return ldbPtr.LDB().GetProperty(property)
}

func (apiPtr *PrivateDebugAPI) ChaindbCompact() error {
	ldb, ok := apiPtr.bPtr.ChainDb().(interface {
		LDB() *leveldbPtr.DB
	})
	if !ok {
		return fmt.Errorf("chaindbCompact does not work for memory databases")
	}
	for b := byte(0); b < 255; b++ {
		bgmlogs.Info("Compacting chain database", "range", fmt.Sprintf("0x%0.2X-0x%0.2X", b, b+1))
		err := ldbPtr.LDB().CompactRange(util.Range{Start: []byte{b}, Limit: []byte{b + 1}})
		if err != nil {
			bgmlogs.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (apiPtr *PrivateDebugAPI) SetHead(number hexutil.Uint64) {
	apiPtr.bPtr.SetHead(Uint64(number))
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion Uint64
}

// NewPublicNetAPI creates a new net apiPtr instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion Uint64) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(s.net.PeerCount())
}

// Version returns the current bgmchain protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%-d", s.networkVersion)
}
