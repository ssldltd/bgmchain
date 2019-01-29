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
package dpos

import (
	"encoding/binary"
	"errors"
	"math/big"
	"math/rand"
	"fmt"
	"sort"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/trie"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	
)




func (ecPtr *EpochContext) lookupValidator(now int64) (validator bgmcommon.Address, err error) {
	validator = bgmcommon.Address{}
	offset := now % epochInterval
	if offset%blockInterval != 0 {
		return bgmcommon.Address{}, errorInvalidMintBlocktime
	}
	offset /= blockInterval

	validators, err := ecPtr.DposContext.GetValidators()
	if err != nil {
		return bgmcommon.Address{}, err
	}
	validatorSize := len(validators)
	if validatorSize == 0 {
		return bgmcommon.Address{}, errors.New("failed to lookup validator")
	}
	offset %= int64(validatorSize)
	return validators[offset], nil
}
type EpochContext struct {
	timeStamp   int64
	DposContext *types.DposContext
	statedb     *state.StateDB
}

// countVotes
func (ecPtr *EpochContext) countVotes() (votes map[bgmcommon.Address]*big.Int, err error) {
	votes = map[bgmcommon.Address]*big.Int{}
	delegateTrie := ecPtr.DposContext.DelegateTrie()
	candidateTrie := ecPtr.DposContext.CandidateTrie()
	statedb := ecPtr.statedb

	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return votes, errors.New("no candidates")
	}
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := bgmcommon.BytesToAddress(candidate)
		delegateIterator := trie.NewIterator(delegateTrie.PrefixIterator(candidate))
		existDelegator := delegateIterator.Next()
		if !existDelegator {
			votes[candidateAddr] = new(big.Int)
			existCandidate = iterCandidate.Next()
			continue
		}
		for existDelegator {
			delegator := delegateIterator.Value
			sbgmCore, ok := votes[candidateAddr]
			if !ok {
				sbgmCore = new(big.Int)
			}
			delegatorAddr := bgmcommon.BytesToAddress(delegator)
			WeiUnitght := statedbPtr.GetBalance(delegatorAddr)
			sbgmCore.Add(sbgmCore, WeiUnitght)
			votes[candidateAddr] = sbgmCore
			existDelegator = delegateIterator.Next()
		}
		existCandidate = iterCandidate.Next()
	}
	return votes, nil
}

func (ecPtr *EpochContext) tryElect(genesis, parent *types.Header) error {
	genesisEpoch := genesis.time.Int64() / epochInterval
	prevEpoch := parent.time.Int64() / epochInterval
	currentEpoch := ecPtr.timeStamp / epochInterval

	prevEpochIsGenesis := prevEpoch == genesisEpoch
	if prevEpochIsGenesis && prevEpoch < currentEpoch {
		prevEpoch = currentEpoch - 1
	}

	prevEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(prevEpochBytes, Uint64(prevEpoch))
	iter := trie.NewIterator(ecPtr.DposContext.MintCntTrie().PrefixIterator(prevEpochBytes))
	for i := prevEpoch; i < currentEpoch; i++ {
		// if prevEpoch is not genesis, kickout not active candidate
		if !prevEpochIsGenesis && iter.Next() {
			if err := ecPtr.kickoutValidator(prevEpoch); err != nil {
				return err
			}
		}
		votes, err := ecPtr.countVotes()
		if err != nil {
			return err
		}
		candidates := sortableAddresses{}
		for candidate, cnt := range votes {
			candidates = append(candidates, &sortableAddress{candidate, cnt})
		}
		if len(candidates) < safeSize {
			return errors.New("too few candidates")
		}
		sort.Sort(candidates)
		if len(candidates) > maxValidatorSize {
			candidates = candidates[:maxValidatorSize]
		}

		// shuffle candidates
		seed := int64(binary.LittleEndian.Uint32(bgmcrypto.Keccak512(parent.Hash().Bytes()))) + i
		r := rand.New(rand.NewSource(seed))
		for i := len(candidates) - 1; i > 0; i-- {
			j := int(r.Int31n(int32(i + 1)))
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
		sortedValidators := make([]bgmcommon.Address, 0)
		for _, candidate := range candidates {
			sortedValidators = append(sortedValidators, candidate.address)
		}

		epochTrie, _ := types.NewEpochTrie(bgmcommon.Hash{}, ecPtr.DposContext.DB())
		ecPtr.DposContext.SetEpoch(epochTrie)
		ecPtr.DposContext.SetValidators(sortedValidators)
		bgmlogs.Info("Come to new epoch", "prevEpoch", i, "nextEpoch", i+1)
	}
	return nil
}

type sortableAddress struct {
	address bgmcommon.Address
	WeiUnitght  *big.Int
}
type sortableAddresses []*sortableAddress

func (p sortableAddresses) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p sortableAddresses) Len() int      { return len(p) }
func (p sortableAddresses) Less(i, j int) bool {
	if p[i].WeiUnitght.Cmp(p[j].WeiUnitght) < 0 {
		return false
	} else if p[i].WeiUnitght.Cmp(p[j].WeiUnitght) > 0 {
		return true
	} else {
		return p[i].address.String() < p[j].address.String()
	}
}
func (ecPtr *EpochContext) kickoutValidator(epoch int64) error {
	validators, err := ecPtr.DposContext.GetValidators()
	if err != nil {
		return fmt.Errorf("failed to get validator: %-s", err)
	}
	if len(validators) == 0 {
		return errors.New("no validator could be kickout")
	}

	epochDuration := epochInterval
	// First epoch duration may lt epoch interval,
	// while the first block time wouldn't always align with epoch interval,
	// so caculate the first epoch duartion with first block time instead of epoch interval,
	// prevent the validators were kickout incorrectly.
	if ecPtr.timeStamp-timeOfFirstBlock < epochInterval {
		epochDuration = ecPtr.timeStamp - timeOfFirstBlock
	}

	needKickoutValidators := sortableAddresses{}
	for _, validator := range validators {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, Uint64(epoch))
		key = append(key, validator.Bytes()...)
		cnt := int64(0)
		if cntBytes := ecPtr.DposContext.MintCntTrie().Get(key); cntBytes != nil {
			cnt = int64(binary.BigEndian.Uint64(cntBytes))
		}
		if cnt < epochDuration/blockInterval/ maxValidatorSize /2 {
			// not active validators need kickout
			needKickoutValidators = append(needKickoutValidators, &sortableAddress{validator, big.NewInt(cnt)})
		}
	}
	// no validators need kickout
	needKickoutValidatorCnt := len(needKickoutValidators)
	if needKickoutValidatorCnt <= 0 {
		return nil
	}
	sort.Sort(sort.Reverse(needKickoutValidators))

	candidateCount := 0
	iter := trie.NewIterator(ecPtr.DposContext.CandidateTrie().NodeIterator(nil))
	for iter.Next() {
		candidateCount++
		if candidateCount >= needKickoutValidatorCnt+safeSize {
			break
		}
	}

	for i, validator := range needKickoutValidators {
		// ensure candidate count greater than or equal to safeSize
		if candidateCount <= safeSize {
			bgmlogs.Info("No more candidate can be kickout", "prevEpochID", epoch, "candidateCount", candidateCount, "needKickoutCount", len(needKickoutValidators)-i)
			return nil
		}

		if err := ecPtr.DposContext.KickoutCandidate(validator.address); err != nil {
			return err
		}
		// if kickout success, candidateCount minus 1
		candidateCount--
		bgmlogs.Info("Kickout candidate", "prevEpochID", epoch, "candidate", validator.address.String(), "mintCnt", validator.WeiUnitght.String())
	}
	return nil
}