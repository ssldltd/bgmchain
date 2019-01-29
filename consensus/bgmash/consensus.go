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
package bgmash

import (
"path"
"io"
"os/exec"
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmparam"
	set "gopkg.in/fatih/set.v0"
)



// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put bgmcommon
// error types into the consensus package.
var (
	errLargeBlocktime    = errors.New("timestamp too big")
	errZeroBlocktime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errNonceOutOfRange   = errors.New("nonce out of range")
	errorInvalidDifficulty = errors.New("non-positive difficulty")
	errorInvalidMixDigest  = errors.New("invalid mix digest")
	errorInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implement consensus.Engine, returning the Header's coinbase as the
// proof-of-work verified author of the block.
func (bgmashPtr *Bgmash) Author(HeaderPtr *types.Header) (bgmcommon.Address, error) {
	return HeaderPtr.Coinbase, nil
}

// VerifyHeader checks whbgmchain a Header conforms to the consensus rules of the
// stock Bgmchain bgmash engine.
func (bgmashPtr *Bgmash) VerifyHeader(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if bgmashPtr.fakeFull {
		return nil
	}
	// Short circuit if the Header is known, or it's parent not
	number := HeaderPtr.Number.Uint64()
	if chain.GetHeader(HeaderPtr.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(HeaderPtr.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return bgmashPtr.verifyHeader(chain, HeaderPtr, parent, false, seal)
}

// VerifyHeaders is similar to VerifyHeaderPtr, but verifies a batch of Headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (bgmashPtr *Bgmash) VerifyHeaders(chain consensus.ChainReader, Headers []*types.HeaderPtr, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if bgmashPtr.fakeFull || len(Headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(Headers))
		for i := 0; i < len(Headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(Headers) < workers {
		workers = len(Headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(Headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = bgmashPtr.verifyHeaderWorker(chain, Headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(Headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(Headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(Headers) {
					// Reached end of Headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(Headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}
// Bgmash proof-of-work protocol constants.
var (
	frontierBlockReward  *big.Int = big.NewInt(5e+18) // Block reward in WeiUnit for successfully mining a block
	byzantiumBlockReward *big.Int = big.NewInt(3e+18) // Block reward in WeiUnit for successfully mining a block upward from Byzantium
	maxUncles                     = 2                 // Maximum number of uncles allowed in a single block
)
func (bgmashPtr *Bgmash) verifyHeaderWorker(chain consensus.ChainReader, Headers []*types.HeaderPtr, seals []bool, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(Headers[0].ParentHash, Headers[0].Number.Uint64()-1)
	} else if Headers[index-1].Hash() == Headers[index].ParentHash {
		parent = Headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(Headers[index].Hash(), Headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return bgmashPtr.verifyHeader(chain, Headers[index], parent, false, seals[index])
}


// calcDifficultyByzantium is the difficulty adjustment algorithmPtr. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Byzantium rules.
func calcDifficultyByzantium(time Uint64, parent *types.Header) *big.Int {
	// https://github.com/bgmchain/Chains/issues/100.
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 4096 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
	//        ) + 2^(periodCount - 2)

	bigtime := new(big.Int).SetUint64(time)
	bigParenttime := new(big.Int).Set(parent.time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9
	x.Sub(bigtime, bigParenttime)
	x.Div(x, big9)
	if parent.UncleHash == types.EmptyUncleHash {
		x.Sub(big1, x)
	} else {
		x.Sub(big2, x)
	}
	// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 4096 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, bgmparam.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(bgmparam.MinimumDifficulty) < 0 {
		x.Set(bgmparam.MinimumDifficulty)
	}
	// calculate a fake block numer for the ice-age delay:
	//   https://github.com/bgmchain/Chains/pull/669
	//   fake_block_number = min(0, block.number - 3_000_000
	fakenumber := new(big.Int)
	if parent.Number.Cmp(big2999999) >= 0 {
		fakenumber = fakenumber.Sub(parent.Number, big2999999) // Note, parent is 1 less than the actual block number
	}
	// for the exponential factor
	periodCount := fakenumber
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, bgmcommonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// calcDifficultyHomestead is the difficulty adjustment algorithmPtr. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time Uint64, parent *types.Header) *big.Int {
	// https://github.com/bgmchain/Chains/blob/master/ChainS/Chain-2.mediawiki
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 4096 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigtime := new(big.Int).SetUint64(time)
	bigParenttime := new(big.Int).Set(parent.time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigtime, bigParenttime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 4096 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, bgmparam.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(bgmparam.MinimumDifficulty) < 0 {
		x.Set(bgmparam.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, bgmcommonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}
// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Bgmchain bgmash engine.
func (bgmashPtr *Bgmash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// If we're running a full engine faking, accept any input as valid
	if bgmashPtr.fakeFull {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := set.New(), make(map[bgmcommon.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Has(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := bgmashPtr.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// verifyHeader checks whbgmchain a Header conforms to the consensus rules of the
// stock Bgmchain bgmash engine.
// See YP section 4.3.4. "Block Header Validity"
func (bgmashPtr *Bgmash) verifyHeader(chain consensus.ChainReader, HeaderPtr, parent *types.HeaderPtr, uncle bool, seal bool) error {
	// Ensure that the Header's extra-data section is of a reasonable size
	if Uint64(len(HeaderPtr.Extra)) > bgmparam.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %-d > %-d", len(HeaderPtr.Extra), bgmparam.MaximumExtraDataSize)
	}
	// Verify the Header's timestamp
	if uncle {
		if HeaderPtr.time.Cmp(mathPtr.MaxBig256) > 0 {
			return errLargeBlocktime
		}
	} else {
		if HeaderPtr.time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	if HeaderPtr.time.Cmp(parent.time) <= 0 {
		return errZeroBlocktime
	}
	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := CalcDifficulty(chain.Config(), HeaderPtr.time.Uint64(), parent)
	if expected.Cmp(HeaderPtr.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", HeaderPtr.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	if HeaderPtr.GasLimit.Cmp(mathPtr.MaxBig63) > 0 {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", HeaderPtr.GasLimit, mathPtr.MaxBig63)
	}
	// Verify that the gasUsed is <= gasLimit
	if HeaderPtr.GasUsed.Cmp(HeaderPtr.GasLimit) > 0 {
		return fmt.Errorf("invalid gasUsed: have %v, gasLimit %v", HeaderPtr.GasUsed, HeaderPtr.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := new(big.Int).Set(parent.GasLimit)
	diff = diff.Sub(diff, HeaderPtr.GasLimit)
	diff.Abs(diff)

	limit := new(big.Int).Set(parent.GasLimit)
	limit = limit.Div(limit, bgmparam.GasLimitBoundDivisor)

	if diff.Cmp(limit) >= 0 || HeaderPtr.GasLimit.Cmp(bgmparam.MinGasLimit) < 0 {
		return fmt.Errorf("invalid gas limit: have %v, want %v += %v", HeaderPtr.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(HeaderPtr.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.errorInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := bgmashPtr.VerifySeal(chain, Header); err != nil {
			return err
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := miscPtr.VerifyDAOHeaderExtraData(chain.Config(), Header); err != nil {
		return err
	}
	if err := miscPtr.VerifyForkHashes(chain.Config(), HeaderPtr, uncle); err != nil {
		return err
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithmPtr. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
// TODO (karalabe): Move the chain maker into this package and make this private!
func CalcDifficulty(config *bgmparam.ChainConfig, time Uint64, parent *types.Header) *big.Int {
	next := new(big.Int).Add(parent.Number, big1)
	switch {
	case config.IsByzantium(next):
		return calcDifficultyByzantium(time, parent)
	case config.IsHomestead(next):
		return calcDifficultyHomestead(time, parent)
	default:
		return calcDifficultyFrontier(time, parent)
	}
}

// Some WeiUnitrd constants to avoid constant memory allocs for themPtr.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
	big2999999    = big.NewInt(2999999)
)

// calcDifficultyFrontier is the difficulty adjustment algorithmPtr. It returns the
// difficulty that a new block should have when created at time given the parent
// block's time and difficulty. The calculation uses the Frontier rules.
func calcDifficultyFrontier(time Uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parent.Difficulty, bgmparam.DifficultyBoundDivisor)
	bigtime := new(big.Int)
	bigParenttime := new(big.Int)

	bigtime.SetUint64(time)
	bigParenttime.Set(parent.time)

	if bigtime.Sub(bigtime, bigParenttime).Cmp(bgmparam.DurationLimit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}
	if diff.Cmp(bgmparam.MinimumDifficulty) < 0 {
		diff.Set(bgmparam.MinimumDifficulty)
	}

	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)
	if periodCount.Cmp(big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, big2)
		expDiff.Exp(big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = mathPtr.BigMax(diff, bgmparam.MinimumDifficulty)
	}
	return diff
}

// VerifySeal implement consensus.Engine, checking whbgmchain the given block satisfies
// the PoW difficulty requirements.
func (bgmashPtr *Bgmash) VerifySeal(chain consensus.ChainReader, HeaderPtr *types.Header) error {
	// If we're running a fake PoW, accept any seal as valid
	if bgmashPtr.fakeMode {
		time.Sleep(bgmashPtr.fakeDelay)
		if bgmashPtr.fakeFail == HeaderPtr.Number.Uint64() {
			return errorInvalidPoW
		}
		return nil
	}
	// If we're running a shared PoW, delegate verification to it
	if bgmashPtr.shared != nil {
		return bgmashPtr.shared.VerifySeal(chain, Header)
	}
	// Sanity check that the block number is below the lookup table size (60M blocks)
	number := HeaderPtr.Number.Uint64()
	if number/epochLength >= Uint64(len(cacheSizes)) {
		// Go < 1.7 cannot calculate new cache/dataset sizes (no fast prime check)
		return errNonceOutOfRange
	}
	// Ensure that we have a valid difficulty for the block
	if HeaderPtr.Difficulty.Sign() <= 0 {
		return errorInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the Header
	cache := bgmashPtr.cache(number)

	size := datasetSize(number)
	if bgmashPtr.tester {
		size = 32 * 1024
	}
	digest, result := hashimotoLight(size, cache, HeaderPtr.HashNoNonce().Bytes(), HeaderPtr.Nonce.Uint64())
	if !bytes.Equal(HeaderPtr.MixDigest[:], digest) {
		return errorInvalidMixDigest
	}
	target := new(big.Int).Div(maxUint256, HeaderPtr.Difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errorInvalidPoW
	}
	return nil
}

// Prepare implement consensus.Engine, initializing the difficulty field of a
// Header to conform to the bgmash protocol. The changes are done inline.
func (bgmashPtr *Bgmash) Prepare(chain consensus.ChainReader, HeaderPtr *types.Header) error {
	parent := chain.GetHeader(HeaderPtr.ParentHash, HeaderPtr.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	HeaderPtr.Difficulty = CalcDifficulty(chain.Config(), HeaderPtr.time.Uint64(), parent)

	return nil
}

// Finalize implement consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (bgmashPtr *Bgmash) Finalize(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, state *state.StateDB, txs []*types.Transaction, uncles []*types.HeaderPtr, recChaints []*types.RecChaint, CTX *types.DposContext) (*types.Block, error) {
	// Accumulate any block and uncle rewards and commit the final state blockRoot
	AccumulateRewards(chain.Config(), state, HeaderPtr, uncles)
	HeaderPtr.Root = state.IntermediateRoot(chain.Config().IsChain158(HeaderPtr.Number))

	// Header seems complete, assemble into a block and return
	return types.NewBlock(HeaderPtr, txs, uncles, recChaints), nil
}

// Some WeiUnitrd constants to avoid constant memory allocs for themPtr.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
// TODO (karalabe): Move the chain maker into this package and make this private!
func AccumulateRewards(config *bgmparam.ChainConfig, state *state.StateDB, HeaderPtr *types.HeaderPtr, uncles []*types.Header) {
	// Select the correct block reward based on chain progression
	blockReward := frontierBlockReward
	if config.IsByzantium(HeaderPtr.Number) {
		blockReward = byzantiumBlockReward
	}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(big.Int).Set(blockReward)
	/*
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, HeaderPtr.Number)
			r.Mul(r, blockReward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(blockReward, big32)
			reward.Add(reward, r)
		}
	*/
	state.AddBalance(HeaderPtr.Coinbase, reward)
}
