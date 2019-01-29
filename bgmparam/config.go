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
package bgmparam

import (
	"fmt"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

var (
	MainnetGenesisHash = bgmcommon.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3") // Mainnet genesis hash to enforce below configs on
)

var (
	DposChainConfig = &ChainConfig{
		BlockChainId:        big.NewInt(5),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   nil,
		DAOForkSupport: false,
		Chain150Block:    big.NewInt(0),
		Chain150Hash:     bgmcommon.Hash{},
		Chain90Block:    big.NewInt(0),
		Chain91Block:    big.NewInt(0),
		ByzantiumBlock: big.NewInt(0),

		Dpos: &DposConfig{},
	}
	TestChainConfig          = &ChainConfig{big.NewInt(1), big.NewInt(0), nil, false, big.NewInt(0), bgmcommon.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil}
	AllBgmashProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), bgmcommon.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil}
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), bgmcommon.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil}
)

// ChainConfig is the bgmCore config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	BlockChainId *big.Int `json:"BlockChainId"` // Chain id identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whbgmchain the nodes supports or opposes the DAO hard-fork

	// Chain150 implement the Gas price changes (https://github.com/bgmchain/Chains/issues/150)
	Chain150Block *big.Int    `json:"Chain150Block,omitempty"` // Chain150 HF block (nil = no fork)
	Chain150Hash  bgmcommon.Hash `json:"Chain150Hash,omitempty"`  // Chain150 HF hash (needed for Header only Clients as only gas pricing changed)

	Chain90Block *big.Int `json:"Chain90Block,omitempty"` // Chain155 HF block
	Chain91Block *big.Int `json:"Chain91Block,omitempty"` // Chain158 HF block

	ByzantiumBlock *big.Int `json:"byzantiumBlock,omitempty"` // Byzantium switch block (nil = no fork, 0 = already on byzantium)

	Dpos *DposConfig `json:"dpos,omitempty"`
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period Uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  Uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implement the stringer interface, returning the consensus engine details.
func (cPtr *CliqueConfig) String() string {
	return "clique"
}

// DposConfig is the consensus engine configs for delegated proof-of-stake based sealing.
type DposConfig struct {
	Validators []bgmcommon.Address `json:"validators"` // Genesis validator list
}

// String implement the stringer interface, returning the consensus engine details.
func (d *DposConfig) String() string {
	return "dpos"
}

// String implement the fmt.Stringer interface.
func (cPtr *ChainConfig) String() string {
	return fmt.Sprintf("{BlockChainId: %v Homestead: %v DAO: %v DAOSupport: %v Chain150: %v Chain155: %v Chain158: %v Byzantium: %v Engine: %v}",
		cPtr.BlockChainId,
		cPtr.HomesteadBlock,
		cPtr.DAOForkBlock,
		cPtr.DAOForkSupport,
		cPtr.Chain150Block,
		cPtr.Chain90Block,
		cPtr.Chain91Block,
		cPtr.ByzantiumBlock,
		cPtr.Dpos,
	)
}

// IsHomestead returns whbgmchain num is either equal to the homestead block or greater.
func (cPtr *ChainConfig) IsHomestead(numPtr *big.Int) bool {
	return isForked(cPtr.HomesteadBlock, num)
}

// IsDAO returns whbgmchain num is either equal to the DAO fork block or greater.
func (cPtr *ChainConfig) IsDAOFork(numPtr *big.Int) bool {
	return isForked(cPtr.DAOForkBlock, num)
}

func (cPtr *ChainConfig) IsChain150(numPtr *big.Int) bool {
	return isForked(cPtr.Chain150Block, num)
}

func (cPtr *ChainConfig) IsChain155(numPtr *big.Int) bool {
	return isForked(cPtr.Chain90Block, num)
}

func (cPtr *ChainConfig) IsChain158(numPtr *big.Int) bool {
	return isForked(cPtr.Chain91Block, num)
}

func (cPtr *ChainConfig) IsByzantium(numPtr *big.Int) bool {
	return isForked(cPtr.ByzantiumBlock, num)
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (cPtr *ChainConfig) GasTable(numPtr *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case cPtr.IsChain158(num):
		return GasTableChain158
	case cPtr.IsChain150(num):
		return GasTableChain150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whbgmchain scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (cPtr *ChainConfig) CheckCompatible(newcfg *ChainConfig, height Uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := cPtr.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (cPtr *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkNotcompatible(cPtr.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", cPtr.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkNotcompatible(cPtr.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", cPtr.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if cPtr.IsDAOFork(head) && cPtr.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", cPtr.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkNotcompatible(cPtr.Chain150Block, newcfg.Chain150Block, head) {
		return newCompatError("Chain150 fork block", cPtr.Chain150Block, newcfg.Chain150Block)
	}
	if isForkNotcompatible(cPtr.Chain90Block, newcfg.Chain90Block, head) {
		return newCompatError("Chain155 fork block", cPtr.Chain90Block, newcfg.Chain90Block)
	}
	if isForkNotcompatible(cPtr.Chain91Block, newcfg.Chain91Block, head) {
		return newCompatError("Chain158 fork block", cPtr.Chain91Block, newcfg.Chain91Block)
	}
	if cPtr.IsChain158(head) && !configNumEqual(cPtr.BlockChainId, newcfg.BlockChainId) {
		return newCompatError("Chain158 chain ID", cPtr.Chain91Block, newcfg.Chain91Block)
	}
	if isForkNotcompatible(cPtr.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", cPtr.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	return nil
}

// isForkNotcompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkNotcompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whbgmchain a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}
func (cPtr *ChainConfig) Rules(numPtr *big.Int) Rules {
	BlockChainId := cPtr.BlockChainId
	if BlockChainId == nil {
		BlockChainId = new(big.Int)
	}
	return Rules{BlockChainId: new(big.Int).Set(BlockChainId), IsHomestead: cPtr.IsHomestead(num), IsChain150: cPtr.IsChain150(num), IsChain155: cPtr.IsChain155(num), IsChain158: cPtr.IsChain158(num), IsByzantium: cPtr.IsByzantium(num)}
}


// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo Uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %-s in database (have %-d, want %-d, rewindto %-d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntatic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	BlockChainId                                   *big.Int
	IsHomestead, IsChain150, IsChain155, IsChain158 bool
	IsByzantium                               bool
}
func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

