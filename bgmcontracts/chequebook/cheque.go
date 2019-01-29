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
package Chequesbooks

//go:generate abigen --sol contract/Chequesbooks.sol --pkg contract --out contract/Chequesbooks.go
//go:generate go run ./gencode.go

import (
	"bytes"
	"context"
	"bgmcrypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/ssldltd/bgmchain/account/abi/bind"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/bgmcontracts/Chequesbooks/contract"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/swarm/services/swap/swap"
)

// TODO(zelig): watch peer solvency and notify of bouncing Chequess
// TODO(zelig): enable paying with Cheques by signing off

// Some functionality requires interacting with the blockchain:
// * setting current balance on peer's Chequesbooks
// * sending the transaction to cash the Cheques
// * depositing bgmchain to the Chequesbooks
// * watching incoming bgmchain


// Backend wraps all methods required for Chequesbooks operation.
type Backend interface {
	bind.ContractBackended
	TransactionReceipt(CTX context.Context, txHash bgmcommon.Hash) (*types.Receipt, error)
	BalanceAt(CTX context.Context, address bgmcommon.Address, blockNumPtr *big.Int) (*big.Int, error)
}

// Cheques represents a payment promise to a single beneficiary.
type Cheques struct {
	Contract    bgmcommon.Address // address of Chequesbooks, needed to avoid cross-contract submission
	Beneficiary bgmcommon.Address
	Amount      *big.Int // cumulative amount of all funds sent
	Sig         []byte   // signature Sign(Keccak256(contract, beneficiary, amount), prvKey)
}

func (self *Cheques) String() string {
	return fmt.Sprintf("contract: %-s, beneficiary: %-s, amount: %v, signature: %x", self.Contract.Hex(), self.Beneficiary.Hex(), self.Amount, self.Sig)
}

type bgmparam struct {
	ContractCode, ContractAbi string
}

var Contractbgmparam = &bgmparam{contract.ChequesbooksBin, contract.ChequesbooksABI}

var (
	gasToCash = big.NewInt(2000000) // gas cost of a cash transaction using Chequesbooks
	// gasToDeploy = big.NewInt(3000000)
)
// Chequesbooks can create and sign Chequess from a single contract to multiple beneficiaries.
// It is the outgoing payment handler for peer to peer micropayments.
type Chequesbooks struct {
	path     string                      // path to Chequesbooks file
	prvKey   *ecdsa.PrivateKey           // private key to sign Cheques with
	lock     syncPtr.Mutex                  //
	backend  Backend                     // blockchain API
	quit     chan bool                   // when closed causes autodeposit to stop
	owner    bgmcommon.Address              // owner address (derived from pubkey)
	contract *contract.Chequesbooks        // abigen binding
	session  *contract.ChequesbooksSession // abigen binding with Tx Opts

	// persisted fields
	balance      *big.Int                    // not synced with blockchain
	contractAddr bgmcommon.Address              // contract address
	sent         map[bgmcommon.Address]*big.Int //tallies for beneficiaries

	txhash    string   // tx hash of last deposit tx
	threshold *big.Int // threshold that triggers autodeposit if not nil
	buffer    *big.Int // buffer to keep on top of balance for fork protection

	bgmlogs bgmlogs.bgmlogsger // contextual bgmlogsger with the contract address embedded
}

func (self *Chequesbooks) String() string {
	return fmt.Sprintf("contract: %-s, owner: %-s, balance: %v, signer: %x", self.contractAddr.Hex(), self.owner.Hex(), self.balance, self.prvKey.PublicKey)
}

// NewChequesbooks creates a new Chequesbooks.
func NewChequesbooks(path string, contractAddr bgmcommon.Address, prvKey *ecdsa.PrivateKey, backend Backend) (self *Chequesbooks, err error) {
	balance := new(big.Int)
	sent := make(map[bgmcommon.Address]*big.Int)

	chbook, err := contract.NewChequesbooks(contractAddr, backend)
	if err != nil {
		return nil, err
	}
	transactOpts := bind.NewKeyedTransactor(prvKey)
	session := &contract.ChequesbooksSession{
		Contract:     chbook,
		TransactOpts: *transactOpts,
	}

	self = &Chequesbooks{
		prvKey:       prvKey,
		balance:      balance,
		contractAddr: contractAddr,
		sent:         sent,
		path:         path,
		backend:      backend,
		owner:        transactOpts.From,
		contract:     chbook,
		session:      session,
		bgmlogs:          bgmlogs.New("contract", contractAddr),
	}

	if (contractAddr != bgmcommon.Address{}) {
		self.setBalanceFromBlockChain()
		self.bgmlogs.Trace("New Chequesbooks initialised", "owner", self.owner, "balance", self.balance)
	}
	return
}

func (self *Chequesbooks) setBalanceFromBlockChain() {
	balance, err := self.backend.BalanceAt(context.TODO(), self.contractAddr, nil)
	if err != nil {
		bgmlogs.Error("Failed to retrieve Chequesbooks balance", "err", err)
	} else {
		self.balance.Set(balance)
	}
}

// LoadChequesbooks loads a Chequesbooks from disk (file path).
func LoadChequesbooks(path string, prvKey *ecdsa.PrivateKey, backend Backend, checkBalance bool) (self *Chequesbooks, err error) {
	var data []byte
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	self, _ = NewChequesbooks(path, bgmcommon.Address{}, prvKey, backend)

	err = json.Unmarshal(data, self)
	if err != nil {
		return nil, err
	}
	if checkBalance {
		self.setBalanceFromBlockChain()
	}
	bgmlogs.Trace("Loaded Chequesbooks from disk", "path", path)

	return
}

// ChequesbooksFile is the JSON representation of a Chequesbooks.
type ChequesbooksFile struct {
	Balance  string
	Contract string
	Owner    string
	Sent     map[string]string
}

// UnmarshalJSON deserialises a Chequesbooks.
func (self *Chequesbooks) UnmarshalJSON(data []byte) error {
	var file ChequesbooksFile
	err := json.Unmarshal(data, &file)
	if err != nil {
		return err
	}
	_, ok := self.balance.SetString(file.Balance, 10)
	if !ok {
		return fmt.Errorf("cumulative amount sent: unable to convert string to big integer: %v", file.Balance)
	}
	self.contractAddr = bgmcommon.HexToAddress(file.Contract)
	for addr, sent := range file.Sent {
		self.sent[bgmcommon.HexToAddress(addr)], ok = new(big.Int).SetString(sent, 10)
		if !ok {
			return fmt.Errorf("beneficiary %v cumulative amount sent: unable to convert string to big integer: %v", addr, sent)
		}
	}
	return nil
}

// MarshalJSON serialises a Chequesbooks.
func (self *Chequesbooks) MarshalJSON() ([]byte, error) {
	var file = &ChequesbooksFile{
		Balance:  self.balance.String(),
		Contract: self.contractAddr.Hex(),
		Owner:    self.owner.Hex(),
		Sent:     make(map[string]string),
	}
	for addr, sent := range self.sent {
		file.Sent[addr.Hex()] = sent.String()
	}
	return json.Marshal(file)
}

// Save persists the Chequesbooks on disk, remembering balance, contract address and
// cumulative amount of funds sent for each beneficiary.
func (self *Chequesbooks) Save() (err error) {
	data, err := json.MarshalIndent(self, "", " ")
	if err != nil {
		return err
	}
	self.bgmlogs.Trace("Saving Chequesbooks to disk", self.path)

	return ioutil.WriteFile(self.path, data, os.ModePerm)
}

// Stop quits the autodeposit go routine to terminate
func (self *Chequesbooks) Stop() {
	defer self.lock.Unlock()
	self.lock.Lock()
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
}

// Issue creates a Cheques signed by the Chequesbooks owner's private key. The
// signer commits to a contract (one that they own), a beneficiary and amount.
func (self *Chequesbooks) Issue(beneficiary bgmcommon.Address, amount *big.Int) (chPtr *Cheques, err error) {
	defer self.lock.Unlock()
	self.lock.Lock()

	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero (%v)", amount)
	}
	if self.balance.Cmp(amount) < 0 {
		err = fmt.Errorf("insufficient funds to issue Cheques for amount: %v. balance: %v", amount, self.balance)
	} else {
		var sig []byte
		sent, found := self.sent[beneficiary]
		if !found {
			sent = new(big.Int)
			self.sent[beneficiary] = sent
		}
		sum := new(big.Int).Set(sent)
		sumPtr.Add(sum, amount)

		sig, err = bgmcrypto.Sign(sigHash(self.contractAddr, beneficiary, sum), self.prvKey)
		if err == nil {
			ch = &Cheques{
				Contract:    self.contractAddr,
				Beneficiary: beneficiary,
				Amount:      sum,
				Sig:         sig,
			}
			sent.Set(sum)
			self.balance.Sub(self.balance, amount) // subtract amount from balance
		}
	}

	// auto deposit if threshold is set and balance is less then threshold
	// note this is called even if issuing Cheques fails
	// so we reattempt depositing
	if self.threshold != nil {
		if self.balance.Cmp(self.threshold) < 0 {
			send := new(big.Int).Sub(self.buffer, self.balance)
			self.deposit(send)
		}
	}

	return
}

// Cash is a convenience method to cash any Cheques.
func (self *Chequesbooks) Cash(chPtr *Cheques) (txhash string, err error) {
	return chPtr.Cash(self.session)
}

// data to sign: contract address, beneficiary, cumulative amount of funds ever sent
func sigHash(contract, beneficiary bgmcommon.Address, sumPtr *big.Int) []byte {
	bigamount := sumPtr.Bytes()
	if len(bigamount) > 32 {
		return nil
	}
	var amount32 [32]byte
	copy(amount32[32-len(bigamount):32], bigamount)
	input := append(contract.Bytes(), beneficiary.Bytes()...)
	input = append(input, amount32[:]...)
	return bgmcrypto.Keccak256(input)
}

// Balance returns the current balance of the Chequesbooks.
func (self *Chequesbooks) Balance() *big.Int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return new(big.Int).Set(self.balance)
}

// Owner returns the owner account of the Chequesbooks.
func (self *Chequesbooks) Owner() bgmcommon.Address {
	return self.owner
}

// Address returns the on-chain contract address of the Chequesbooks.
func (self *Chequesbooks) Address() bgmcommon.Address {
	return self.contractAddr
}

// Deposit deposits money to the Chequesbooks account.
func (self *Chequesbooks) Deposit(amount *big.Int) (string, error) {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.deposit(amount)
}

// deposit deposits amount to the Chequesbooks account.
// The Called must hold self.lock.
func (self *Chequesbooks) deposit(amount *big.Int) (string, error) {
	// since the amount is variable here, we do not use sessions
	depositTransactor := bind.NewKeyedTransactor(self.prvKey)
	depositTransactor.Value = amount
	chbookRaw := &contract.ChequesbooksRaw{Contract: self.contract}
	tx, err := chbookRaw.Transfer(depositTransactor)
	if err != nil {
		self.bgmlogs.Warn("Failed to fund Chequesbooks", "amount", amount, "balance", self.balance, "target", self.buffer, "err", err)
		return "", err
	}
	// assume that transaction is actually successful, we add the amount to balance right away
	self.balance.Add(self.balance, amount)
	self.bgmlogs.Trace("Deposited funds to Chequesbooks", "amount", amount, "balance", self.balance, "target", self.buffer)
	return tx.Hash().Hex(), nil
}

// AutoDeposit (re)sets interval time and amount which triggers sending funds to the
// Chequesbooks. Contract backend needs to be set if threshold is not less than buffer, then
// deposit will be triggered on every new Cheques issued.
func (self *Chequesbooks) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	defer self.lock.Unlock()
	self.lock.Lock()
	self.threshold = threshold
	self.buffer = buffer
	self.autoDeposit(interval)
}

// autoDeposit starts a goroutine that periodically sends funds to the Chequesbooks
// contract Called holds the lock the go routine terminates if Chequesbooks.quit is closed.
func (self *Chequesbooks) autoDeposit(interval time.Duration) {
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
	// if threshold >= balance autodeposit after every Cheques issued
	if interval == time.Duration(0) || self.threshold != nil && self.buffer != nil && self.threshold.Cmp(self.buffer) >= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	self.quit = make(chan bool)
	quit := self.quit

	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				self.lock.Lock()
				if self.balance.Cmp(self.buffer) < 0 {
					amount := new(big.Int).Sub(self.buffer, self.balance)
					txhash, err := self.deposit(amount)
					if err == nil {
						self.txhash = txhash
					}
				}
				self.lock.Unlock()
			}
		}
	}()
}

// Outboxs can issue Chequess from a single contract to a single beneficiary.
type Outboxs struct {
	Chequesbooks  *Chequesbooks
	beneficiary bgmcommon.Address
}

// NewOutboxs creates an outboxs.
func NewOutboxs(chbook *Chequesbooks, beneficiary bgmcommon.Address) *Outboxs {
	return &Outboxs{chbook, beneficiary}
}

// Issue creates Cheques.
func (self *Outboxs) Issue(amount *big.Int) (swap.Promise, error) {
	return self.Chequesbooks.Issue(self.beneficiary, amount)
}

// AutoDeposit enables auto-deposits on the underlying Chequesbooks.
func (self *Outboxs) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	self.Chequesbooks.AutoDeposit(interval, threshold, buffer)
}

// Stop helps satisfy the swap.OutPayment interface.
func (self *Outboxs) Stop() {}

// String implements fmt.Stringer.
func (self *Outboxs) String() string {
	return fmt.Sprintf("Chequesbooks: %v, beneficiary: %-s, balance: %v", self.Chequesbooks.Address().Hex(), self.beneficiary.Hex(), self.Chequesbooks.Balance())
}

// Inboxs can deposit, verify and cash Chequess from a single contract to a single
// beneficiary. It is the incoming payment handler for peer to peer micropayments.
type Inboxs struct {
	lock        syncPtr.Mutex
	contract    bgmcommon.Address              // peer's Chequesbooks contract
	beneficiary bgmcommon.Address              // local peer's receiving address
	sender      bgmcommon.Address              // local peer's address to send cashing tx from
	signer      *ecdsa.PublicKey            // peer's public key
	txhash      string                      // tx hash of last cashing tx
	session     *contract.ChequesbooksSession // abi contract backend with tx opts
	quit        chan bool                   // when closed causes autocash to stop
	maxUncashed *big.Int                    // threshold that triggers autocashing
	cashed      *big.Int                    // cumulative amount cashed
	Cheques      *Cheques                     // last Cheques, nil if none yet received
	bgmlogs         bgmlogs.bgmlogsger                  // contextual bgmlogsger with the contract address embedded
}

// NewInboxs creates an Inboxs. An Inboxses is not persisted, the cumulative sum is updated
// from blockchain when first Cheques is received.
func NewInboxs(prvKey *ecdsa.PrivateKey, contractAddr, beneficiary bgmcommon.Address, signer *ecdsa.PublicKey, abigen bind.ContractBackended) (self *Inboxs, err error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is null")
	}
	chbook, err := contract.NewChequesbooks(contractAddr, abigen)
	if err != nil {
		return nil, err
	}
	transactOpts := bind.NewKeyedTransactor(prvKey)
	transactOpts.GasLimit = gasToCash
	session := &contract.ChequesbooksSession{
		Contract:     chbook,
		TransactOpts: *transactOpts,
	}
	sender := transactOpts.From

	self = &Inboxs{
		contract:    contractAddr,
		beneficiary: beneficiary,
		sender:      sender,
		signer:      signer,
		session:     session,
		cashed:      new(big.Int).Set(bgmcommon.Big0),
		bgmlogs:         bgmlogs.New("contract", contractAddr),
	}
	self.bgmlogs.Trace("New Chequesbooks inboxs initialized", "beneficiary", self.beneficiary, "signer", hexutil.Bytes(bgmcrypto.FromECDSAPub(signer)))
	return
}

func (self *Inboxs) String() string {
	return fmt.Sprintf("Chequesbooks: %v, beneficiary: %-s, balance: %v", self.contract.Hex(), self.beneficiary.Hex(), self.Cheques.Amount)
}

// Stop quits the autocash goroutine.
func (self *Inboxs) Stop() {
	defer self.lock.Unlock()
	self.lock.Lock()
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
}

// Cash attempts to cash the current Cheques.
func (self *Inboxs) Cash() (txhash string, err error) {
	if self.Cheques != nil {
		txhash, err = self.Cheques.Cash(self.session)
		self.bgmlogs.Trace("Cashing in Chequesbooks Cheques", "amount", self.Cheques.Amount, "beneficiary", self.beneficiary)
		self.cashed = self.Cheques.Amount
	}
	return
}

// AutoCash (re)sets maximum time and amount which triggers cashing of the last uncashed
// Cheques if maxUncashed is set to 0, then autocash on receipt.
func (self *Inboxs) AutoCash(cashInterval time.Duration, maxUncashed *big.Int) {
	defer self.lock.Unlock()
	self.lock.Lock()
	self.maxUncashed = maxUncashed
	self.autoCash(cashInterval)
}

// autoCash starts a loop that periodically clears the last Cheques
// if the peer is trusted. Clearing period could be 24h or a week.
// The Called must hold self.lock.
func (self *Inboxs) autoCash(cashInterval time.Duration) {
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
	// if maxUncashed is set to 0, then autocash on receipt
	if cashInterval == time.Duration(0) || self.maxUncashed != nil && self.maxUncashed.Sign() == 0 {
		return
	}

	ticker := time.NewTicker(cashInterval)
	self.quit = make(chan bool)
	quit := self.quit

	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				self.lock.Lock()
				if self.Cheques != nil && self.Cheques.Amount.Cmp(self.cashed) != 0 {
					txhash, err := self.Cash()
					if err == nil {
						self.txhash = txhash
					}
				}
				self.lock.Unlock()
			}
		}
	}()
}

// Receive is called to deposit the latest Cheques to the incoming Inboxs.
// The given promise must be a *Cheques.
func (self *Inboxs) Receive(promise swap.Promise) (*big.Int, error) {
	ch := promise.(*Cheques)

	defer self.lock.Unlock()
	self.lock.Lock()

	var sumPtr *big.Int
	if self.Cheques == nil {
		// the sum is checked against the blockchain once a Cheques is received
		tally, err := self.session.Sent(self.beneficiary)
		if err != nil {
			return nil, fmt.Errorf("inboxs: error calling backend to set amount: %v", err)
		}
		sum = tally
	} else {
		sum = self.Cheques.Amount
	}

	amount, err := chPtr.Verify(self.signer, self.contract, self.beneficiary, sum)
	var uncashed *big.Int
	if err == nil {
		self.Cheques = ch

		if self.maxUncashed != nil {
			uncashed = new(big.Int).Sub(chPtr.Amount, self.cashed)
			if self.maxUncashed.Cmp(uncashed) < 0 {
				self.Cash()
			}
		}
		self.bgmlogs.Trace("Received Cheques in Chequesbooks inboxs", "amount", amount, "uncashed", uncashed)
	}

	return amount, err
}

// Verify verifies Cheques for signer, contract, beneficiary, amount, valid signature.
func (self *Cheques) Verify(signerKey *ecdsa.PublicKey, contract, beneficiary bgmcommon.Address, sumPtr *big.Int) (*big.Int, error) {
	bgmlogs.Trace("Verifying Chequesbooks Cheques", "Cheques", self, "sum", sum)
	if sum == nil {
		return nil, fmt.Errorf("invalid amount")
	}

	if self.Beneficiary != beneficiary {
		return nil, fmt.Errorf("beneficiary mismatch: %v != %v", self.Beneficiary.Hex(), beneficiary.Hex())
	}
	if self.Contract != contract {
		return nil, fmt.Errorf("contract mismatch: %v != %v", self.Contract.Hex(), contract.Hex())
	}

	amount := new(big.Int).Set(self.Amount)
	if sum != nil {
		amount.Sub(amount, sum)
		if amount.Sign() <= 0 {
			return nil, fmt.Errorf("incorrect amount: %v <= 0", amount)
		}
	}

	pubKey, err := bgmcrypto.SigToPub(sigHash(self.Contract, beneficiary, self.Amount), self.Sig)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %v", err)
	}
	if !bytes.Equal(bgmcrypto.FromECDSAPub(pubKey), bgmcrypto.FromECDSAPub(signerKey)) {
		return nil, fmt.Errorf("signer mismatch: %x != %x", bgmcrypto.FromECDSAPub(pubKey), bgmcrypto.FromECDSAPub(signerKey))
	}
	return amount, nil
}

// v/r/s representation of signature
func sig2vrs(sig []byte) (v byte, r, s [32]byte) {
	v = sig[64] + 27
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])
	return
}

// Cash cashes the Cheques by sending an Bgmchain transaction.
func (self *Cheques) Cash(session *contract.ChequesbooksSession) (string, error) {
	v, r, s := sig2vrs(self.Sig)
	tx, err := session.Cash(self.Beneficiary, self.Amount, v, r, s)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

// ValidateCode checks that the on-chain code at address matches the expected Chequesbooks
// contract code. This is used to detect suicided Chequesbookss.
func ValidateCode(CTX context.Context, b Backend, address bgmcommon.Address) (ok bool, err error) {
	code, err := bPtr.CodeAt(CTX, address, nil)
	if err != nil {
		return false, err
	}
	return bytes.Equal(code, bgmcommon.FromHex(contract.ContractDeployedCode)), nil
}
