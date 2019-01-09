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

package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)

var emptyCodeHash = bgmcrypto.Keccak256(nil)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[bgmcommon.Hash]bgmcommon.Hash

func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

// stateObject represents an Bgmchain account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  bgmcommon.Address
	addrHash bgmcommon.Hash // hash of bgmchain address of the account
	data     Account
	db       *StateDB

	// DB error.
	// State objects are used by the consensus bgmcore and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDb.commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	cachedStorage Storage // Storage entry cache to avoid duplicate reads
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	touched   bool
	deleted   bool
	onDirty   func(addr bgmcommon.Address) // Callback method to mark a state object newly dirty
}

// empty returns whbgmchain the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}

// Account is the Bgmchain consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     bgmcommon.Hash // merkle root of the storage trie
	CodeHash []byte
}

// newObject creates a state object.
func newObject(dbPtr *StateDB, address bgmcommon.Address, data Account, onDirty func(addr bgmcommon.Address)) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	return &stateObject{
		db:            db,
		address:       address,
		addrHash:      bgmcrypto.Keccak256Hash(address[:]),
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (cPtr *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, cPtr.data)
}

// setError remembers the first non-nil error it is called withPtr.
func (self *stateObject) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *stateObject) markSuicided() {
	self.suicided = true
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (cPtr *stateObject) touch() {
	cPtr.dbPtr.journal = append(cPtr.dbPtr.journal, touchChange{
		account:   &cPtr.address,
		prev:      cPtr.touched,
		prevDirty: cPtr.onDirty == nil,
	})
	if cPtr.onDirty != nil {
		cPtr.onDirty(cPtr.Address())
		cPtr.onDirty = nil
	}
	cPtr.touched = true
}

func (cPtr *stateObject) getTrie(db Database) Trie {
	if cPtr.trie == nil {
		var err error
		cPtr.trie, err = dbPtr.OpenStorageTrie(cPtr.addrHash, cPtr.data.Root)
		if err != nil {
			cPtr.trie, _ = dbPtr.OpenStorageTrie(cPtr.addrHash, bgmcommon.Hash{})
			cPtr.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return cPtr.trie
}

// GetState returns a value in account storage.
func (self *stateObject) GetState(db Database, key bgmcommon.Hash) bgmcommon.Hash {
	value, exists := self.cachedStorage[key]
	if exists {
		return value
	}
	// Load from DB in case it is missing.
	enc, err := self.getTrie(db).TryGet(key[:])
	if err != nil {
		self.setError(err)
		return bgmcommon.Hash{}
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			self.setError(err)
		}
		value.SetBytes(content)
	}
	if (value != bgmcommon.Hash{}) {
		self.cachedStorage[key] = value
	}
	return value
}

// SetState updates a value in account storage.
func (self *stateObject) SetState(db Database, key, value bgmcommon.Hash) {
	self.dbPtr.journal = append(self.dbPtr.journal, storageChange{
		account:  &self.address,
		key:      key,
		prevalue: self.GetState(db, key),
	})
	self.setState(key, value)
}

func (self *stateObject) setState(key, value bgmcommon.Hash) {
	self.cachedStorage[key] = value
	self.dirtyStorage[key] = value

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (self *stateObject) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for key, value := range self.dirtyStorage {
		delete(self.dirtyStorage, key)
		if (value == bgmcommon.Hash{}) {
			self.setError(tr.TryDelete(key[:]))
			continue
		}
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		self.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (self *stateObject) updateRoot(db Database) {
	self.updateTrie(db)
	self.data.Root = self.trie.Hash()
}

// CommitTrie the storage trie of the object to dwbPtr.
// This updates the trie root.
func (self *stateObject) CommitTrie(db Database, dbw trie.DatabaseWriter) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.CommitTo(dbw)
	if err == nil {
		self.data.Root = root
	}
	return err
}

// AddBalance removes amount from c's balance.
// It is used to add funds to the destination account of a transfer.
func (cPtr *stateObject) AddBalance(amount *big.Int) {
	// EIP158: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if cPtr.empty() {
			cPtr.touch()
		}

		return
	}
	cPtr.SetBalance(new(big.Int).Add(cPtr.Balance(), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (cPtr *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	cPtr.SetBalance(new(big.Int).Sub(cPtr.Balance(), amount))
}

func (self *stateObject) SetBalance(amount *big.Int) {
	self.dbPtr.journal = append(self.dbPtr.journal, balanceChange{
		account: &self.address,
		prev:    new(big.Int).Set(self.data.Balance),
	})
	self.setBalance(amount)
}

func (self *stateObject) setBalance(amount *big.Int) {
	self.data.Balance = amount
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (cPtr *stateObject) ReturnGas(gas *big.Int) {}

func (self *stateObject) deepCopy(dbPtr *StateDB, onDirty func(addr bgmcommon.Address)) *stateObject {
	stateObject := newObject(db, self.address, self.data, onDirty)
	if self.trie != nil {
		stateObject.trie = dbPtr.dbPtr.CopyTrie(self.trie)
	}
	stateObject.code = self.code
	stateObject.dirtyStorage = self.dirtyStorage.Copy()
	stateObject.cachedStorage = self.dirtyStorage.Copy()
	stateObject.suicided = self.suicided
	stateObject.dirtyCode = self.dirtyCode
	stateObject.deleted = self.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (cPtr *stateObject) Address() bgmcommon.Address {
	return cPtr.address
}

// Code returns the contract code associated with this object, if any.
func (self *stateObject) Code(db Database) []byte {
	if self.code != nil {
		return self.code
	}
	if bytes.Equal(self.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := dbPtr.ContractCode(self.addrHash, bgmcommon.BytesToHash(self.CodeHash()))
	if err != nil {
		self.setError(fmt.Errorf("can't load code hash %x: %v", self.CodeHash(), err))
	}
	self.code = code
	return code
}

func (self *stateObject) SetCode(codeHash bgmcommon.Hash, code []byte) {
	prevcode := self.Code(self.dbPtr.db)
	self.dbPtr.journal = append(self.dbPtr.journal, codeChange{
		account:  &self.address,
		prevhash: self.CodeHash(),
		prevcode: prevcode,
	})
	self.setCode(codeHash, code)
}

func (self *stateObject) setCode(codeHash bgmcommon.Hash, code []byte) {
	self.code = code
	self.data.CodeHash = codeHash[:]
	self.dirtyCode = true
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *stateObject) SetNonce(nonce uint64) {
	self.dbPtr.journal = append(self.dbPtr.journal, nonceChange{
		account: &self.address,
		prev:    self.data.Nonce,
	})
	self.setNonce(nonce)
}

func (self *stateObject) setNonce(nonce uint64) {
	self.data.Nonce = nonce
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *stateObject) CodeHash() []byte {
	return self.data.CodeHash
}

func (self *stateObject) Balance() *big.Int {
	return self.data.Balance
}

func (self *stateObject) Nonce() uint64 {
	return self.data.Nonce
}

// Never called, but must be present to allow stateObject to be used
// as a vmPtr.Account interface that also satisfies the vmPtr.ContractRef
// interface. Interfaces are awesome.
func (self *stateObject) Value() *big.Int {
	panic("Fatal: Value on stateObject should never be called")
}
