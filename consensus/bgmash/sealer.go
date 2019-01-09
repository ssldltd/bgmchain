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
	crand "bgmcrypto/rand"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

// Seal implement consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (bgmashPtr *Bgmash) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// If we're running a fake PoW, simply return a 0 nonce immediately
	if bgmashPtr.fakeMode {
		header := block.Header()
		headerPtr.Nonce, headerPtr.MixDigest = types.BlockNonce{}, bgmcommon.Hash{}
		return block.WithSeal(header), nil
	}
	// If we're running a shared PoW, delegate sealing to it
	if bgmashPtr.shared != nil {
		return bgmashPtr.shared.Seal(chain, block, stop)
	}
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})
	found := make(chan *types.Block)

	bgmashPtr.lock.Lock()
	threads := bgmashPtr.threads
	if bgmashPtr.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(mathPtr.MaxInt64))
		if err != nil {
			bgmashPtr.lock.Unlock()
			return nil, err
		}
		bgmashPtr.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	bgmashPtr.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra bgmlogsic around local/remote
	}
	var pend syncPtr.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			bgmashPtr.mine(block, id, nonce, abort, found)
		}(i, uint64(bgmashPtr.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	var result *types.Block
	select {
	case <-stop:
		// Outside abort, stop all miner threads
		close(abort)
	case result = <-found:
		// One of the threads found a block, abort all others
		close(abort)
	case <-bgmashPtr.update:
		// Thread count was changed on user request, restart
		close(abort)
		pend.Wait()
		return bgmashPtr.Seal(chain, block, stop)
	}
	// Wait for all miners to terminate and return the block
	pend.Wait()
	return result, nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final block difficulty.
func (bgmashPtr *Bgmash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	// Extract some data from the header
	var (
		header = block.Header()
		hash   = headerPtr.HashNoNonce().Bytes()
		target = new(big.Int).Div(maxUint256, headerPtr.Difficulty)

		number  = headerPtr.Number.Uint64()
		dataset = bgmashPtr.dataset(number)
	)
	// Start generating random nonces until we abort or find a good one
	var (
		attempts = int64(0)
		nonce    = seed
	)
	bgmlogsger := bgmlogs.New("miner", id)
	bgmlogsger.Trace("Started bgmash search for new nonces", "seed", seed)
	for {
		select {
		case <-abort:
			// Mining terminated, update stats and abort
			bgmlogsger.Trace("Bgmash nonce search aborted", "attempts", nonce-seed)
			bgmashPtr.hashrate.Mark(attempts)
			return

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				bgmashPtr.hashrate.Mark(attempts)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			digest, result := hashimotoFull(dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				header = types.CopyHeader(header)
				headerPtr.Nonce = types.EncodeNonce(nonce)
				headerPtr.MixDigest = bgmcommon.BytesToHash(digest)

				// Seal and return a block (if still needed)
				select {
				case found <- block.WithSeal(header):
					bgmlogsger.Trace("Bgmash nonce found and reported", "attempts", nonce-seed, "nonce", nonce)
				case <-abort:
					bgmlogsger.Trace("Bgmash nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				return
			}
			nonce++
		}
	}
}
