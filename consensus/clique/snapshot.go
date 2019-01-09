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
// along with the bgmchain library. If not, see <http://www.gnu.org/licenses/>.

package clique

import (
	"bytes"
	

	"github.com/ssldltd/bgmchain/bgmcommon"
	
	"github.com/ssldltd/bgmchain/bgmparam"
	lru "github.com/hashicorp/golang-lru"
)
// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errorInvalidVotingChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errorInvalidVotingChain
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()

	for _, header := range headers {
		// Remove any votes on checkpoint blocks
		number := headerPtr.Number.Uint64()
		if number%-s.config.Epoch == 0 {
			snap.Votes = nil
			snap.Tally = make(map[bgmcommon.Address]Tally)
		}
		// Delete the oldest signer from the recent list to allow it signing again
		if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
			delete(snap.Recents, number-limit)
		}
		// Resolve the authorization key and check against signers
		signer, err := ecrecover(headerPtr, s.sigcache)
		if err != nil {
			return nil, err
		}
		if _, ok := snap.Signers[signer]; !ok {
			return nil, errUnauthorized
		}
		for _, recent := range snap.Recents {
			if recent == signer {
				return nil, errUnauthorized
			}
		}
		snap.Recents[number] = signer

		// Header authorized, discard any previous votes from the signer
		for i, vote := range snap.Votes {
			if vote.Signer == signer && vote.Address == headerPtr.Coinbase {
				// Uncast the vote from the cached tally
				snap.uncast(vote.Address, vote.Authorize)

				// Uncast the vote from the chronobgmlogsical list
				snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
				break // only one vote allowed
			}
		}
		// Tally up the new vote from the signer
		var authorize bool
		switch {
		case bytes.Equal(headerPtr.Nonce[:], nonceAuthVote):
			authorize = true
		case bytes.Equal(headerPtr.Nonce[:], nonceDropVote):
			authorize = false
		default:
			return nil, errorInvalidVote
		}
		if snap.cast(headerPtr.Coinbase, authorize) {
			snap.Votes = append(snap.Votes, &Vote{
				Signer:    signer,
				Block:     number,
				Address:   headerPtr.Coinbase,
				Authorize: authorize,
			})
		}
		// If the vote passed, update the list of signers
		if tally := snap.Tally[headerPtr.Coinbase]; tally.Votes > len(snap.Signers)/2 {
			if tally.Authorize {
				snap.Signers[headerPtr.Coinbase] = struct{}{}
			} else {
				delete(snap.Signers, headerPtr.Coinbase)

				// Signer list shrunk, delete any leftover recent caches
				if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
					delete(snap.Recents, number-limit)
				}
				// Discard any previous votes the deauthorized signer cast
				for i := 0; i < len(snap.Votes); i++ {
					if snap.Votes[i].Signer == headerPtr.Coinbase {
						// Uncast the vote from the cached tally
						snap.uncast(snap.Votes[i].Address, snap.Votes[i].Authorize)

						// Uncast the vote from the chronobgmlogsical list
						snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)

						i--
					}
				}
			}
			// Discard any previous votes around the just changed account
			for i := 0; i < len(snap.Votes); i++ {
				if snap.Votes[i].Address == headerPtr.Coinbase {
					snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
					i--
				}
			}
			delete(snap.Tally, headerPtr.Coinbase)
		}
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) signers() []bgmcommon.Address {
	signers := make([]bgmcommon.Address, 0, len(s.Signers))
	for signer := range s.Signers {
		signers = append(signers, signer)
	}
	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}
	return signers
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(number uint64, signer bgmcommon.Address) bool {
	signers, offset := s.signers(), 0
	for offset < len(signers) && signers[offset] != signer {
		offset++
	}
	return (number % uint64(len(signers))) == uint64(offset)
}


// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *bgmparam.CliqueConfig // Consensus engine bgmparameters to fine tune behavior
	sigcache *lru.ARCCache        // Cache of recent block signatures to speed up ecrecover

	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    bgmcommon.Hash                 `json:"hash"`    // Block hash where the snapshot was created
	Signers map[bgmcommon.Address]struct{} `json:"signers"` // Set of authorized signers at this moment
	Recents map[uint64]bgmcommon.Address   `json:"recents"` // Set of recent signers for spam protections
	Votes   []*Vote                     `json:"votes"`   // List of votes cast in chronobgmlogsical order
	Tally   map[bgmcommon.Address]Tally    `json:"tally"`   // Current vote tally to avoid recalculating
}

// newSnapshot creates a new snapshot with the specified startup bgmparameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *bgmparam.CliqueConfig, sigcache *lru.ARCCache, number uint64, hash bgmcommon.Hash, signers []bgmcommon.Address) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,
		Signers:  make(map[bgmcommon.Address]struct{}),
		Recents:  make(map[uint64]bgmcommon.Address),
		Tally:    make(map[bgmcommon.Address]Tally),
	}
	for _, signer := range signers {
		snap.Signers[signer] = struct{}{}
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *bgmparam.CliqueConfig, sigcache *lru.ARCCache, db bgmdbPtr.Database, hash bgmcommon.Hash) (*Snapshot, error) {
	blob, err := dbPtr.Get(append([]byte("clique-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache

	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db bgmdbPtr.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return dbPtr.Put(append([]byte("clique-"), s.Hash[:]...), blob)
}
// Vote represents a single vote that an authorized signer made to modify the
// list of authorizations.
type Vote struct {
	Signer    bgmcommon.Address `json:"signer"`    // Authorized signer that cast this vote
	Block     uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
	Address   bgmcommon.Address `json:"address"`   // Account being voted on to change its authorization
	Authorize bool           `json:"authorize"` // Whbgmchain to authorize or deauthorize the voted account
}

// Tally is a simple vote tally to keep the current sbgmcore of votes. Votes that
// go against the proposal aren't counted since it's equivalent to not voting.
type Tally struct {
	Authorize bool `json:"authorize"` // Whbgmchain the vote is about authorizing or kicking someone
	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
}
// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:   s.config,
		sigcache: s.sigcache,
		Number:   s.Number,
		Hash:     s.Hash,
		Signers:  make(map[bgmcommon.Address]struct{}),
		Recents:  make(map[uint64]bgmcommon.Address),
		Votes:    make([]*Vote, len(s.Votes)),
		Tally:    make(map[bgmcommon.Address]Tally),
	}
	for signer := range s.Signers {
		cpy.Signers[signer] = struct{}{}
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}
	for address, tally := range s.Tally {
		cpy.Tally[address] = tally
	}
	copy(cpy.Votes, s.Votes)

	return cpy
}

// validVote returns whbgmchain it makes sense to cast the specified vote in the
// given snapshot context (e.g. don't try to add an already authorized signer).
func (s *Snapshot) validVote(address bgmcommon.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}

// cast adds a new vote into the tally.
func (s *Snapshot) cast(address bgmcommon.Address, authorize bool) bool {
	// Ensure the vote is meaningful
	if !s.validVote(address, authorize) {
		return false
	}
	// Cast the vote into an existing or new tally
	if old, ok := s.Tally[address]; ok {
		old.Votes++
		s.Tally[address] = old
	} else {
		s.Tally[address] = Tally{Authorize: authorize, Votes: 1}
	}
	return true
}

// uncast removes a previously cast vote from the tally.
func (s *Snapshot) uncast(address bgmcommon.Address, authorize bool) bool {
	// If there's no tally, it's a dangling vote, just drop
	tally, ok := s.Tally[address]
	if !ok {
		return false
	}
	// Ensure we only revert counted votes
	if tally.Authorize != authorize {
		return false
	}
	// Otherwise revert the vote
	if tally.Votes > 1 {
		tally.Votes--
		s.Tally[address] = tally
	} else {
		delete(s.Tally, address)
	}
	return true
}


