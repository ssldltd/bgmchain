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
package clique

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/hexutil"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmCore/state"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
	lru "github.com/hashicorp/golang-lru"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	wiggletime = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Clique proof-of-authority protocol constants.
var (
	epochLength = Uint64(30000) // Default number of blocks after which to checkpoint and reset the pending votes
	blockPeriod = Uint64(15)    // Default minimum difference between two consecutive block's timestamps

	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put bgmcommon
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errorInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errorInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errorInvalidVote is returned if a nonce value is sombgming else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errorInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errorInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errorInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errorInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes, or not the correct
	// ones).
	errorInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	// errorInvalidMixDigest is returned if a block's mix digest is non-zero.
	errorInvalidMixDigest = errors.New("non-zero mix digest")

	// errorInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errorInvalidUncleHash = errors.New("non empty uncle hash")

	// errorInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errorInvalidDifficulty = errors.New("invalid difficulty")

	// errorInvalidtimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	errorInvalidtimestamp = errors.New("invalid timestamp")

	// errorInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous Headers.
	errorInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorized is returned if a Header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	errWaitTransactions = errors.New("waiting for transactions")
)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire Header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same HeaderPtr.
func sigHash(HeaderPtr *types.Header) (hash bgmcommon.Hash) {
	hashers := sha3.NewKeccak256()

	rlp.Encode(hashers, []interface{}{
		HeaderPtr.ParentHash,
		HeaderPtr.UncleHash,
		HeaderPtr.Coinbase,
		HeaderPtr.Root,
		HeaderPtr.TxHash,
		HeaderPtr.RecChaintHash,
		HeaderPtr.Bloom,
		HeaderPtr.Difficulty,
		HeaderPtr.Number,
		HeaderPtr.GasLimit,
		HeaderPtr.GasUsed,
		HeaderPtr.time,
		HeaderPtr.Extra[:len(HeaderPtr.Extra)-65], // Yes, this will panic if extra is too short
		HeaderPtr.MixDigest,
		HeaderPtr.Nonce,
	})
	hashers.Sum(hash[:0])
	return hash
}

// ecrecover extracts the Bgmchain account address from a signed HeaderPtr.
func ecrecover(HeaderPtr *types.HeaderPtr, sigcache *lru.ARCCache) (bgmcommon.Address, error) {
	// If the signature's already cached, return that
	hash := HeaderPtr.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(bgmcommon.Address), nil
	}
	// Retrieve the signature from the Header extra-data
	if len(HeaderPtr.Extra) < extraSeal {
		return bgmcommon.Address{}, errMissingSignature
	}
	signature := HeaderPtr.Extra[len(HeaderPtr.Extra)-extraSeal:]

	// Recover the public key and the Bgmchain address
	pubkey, err := bgmcrypto.Ecrecover(sigHash(Header).Bytes(), signature)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	var signer bgmcommon.Address
	copy(signer[:], bgmcrypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// Clique is the proof-of-authority consensus engine proposed to support the
// Bgmchain testnet following the Ropsten attacks.
type Clique struct {
	config *bgmparam.CliqueConfig // Consensus engine configuration bgmparameters
	db     bgmdbPtr.Database       // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	proposals map[bgmcommon.Address]bool // Current list of proposals we are pushing

	signer bgmcommon.Address // Bgmchain address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   syncPtr.RWMutex   // Protects the signer fields
}

// New creates a Clique proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *bgmparam.CliqueConfig, db bgmdbPtr.Database) *Clique {
	// Set any missing consensus bgmparameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &Clique{
		config:     &conf,
		db:         db,
		recents:    recents,
		signatures: signatures,
		proposals:  make(map[bgmcommon.Address]bool),
	}
}

// Author implement consensus.Engine, returning the Bgmchain address recovered
// from the signature in the Header's extra-data section.
func (cPtr *Clique) Author(HeaderPtr *types.Header) (bgmcommon.Address, error) {
	return ecrecover(HeaderPtr, cPtr.signatures)
}

// VerifyHeader checks whbgmchain a Header conforms to the consensus rules.
func (cPtr *Clique) VerifyHeader(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, seal bool) error {
	return cPtr.verifyHeader(chain, HeaderPtr, nil)
}

// VerifyHeaders is similar to VerifyHeaderPtr, but verifies a batch of Headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (cPtr *Clique) VerifyHeaders(chain consensus.ChainReader, Headers []*types.HeaderPtr, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(Headers))

	go func() {
		for i, Header := range Headers {
			err := cPtr.verifyHeader(chain, HeaderPtr, Headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whbgmchain a Header conforms to the consensus rules.The
// Called may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new Headers.
func (cPtr *Clique) verifyHeader(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, parents []*types.Header) error {
	if HeaderPtr.Number == nil {
		return errUnknownBlock
	}
	number := HeaderPtr.Number.Uint64()

	// Don't waste time checking blocks from the future
	if HeaderPtr.time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}
	// Checkpoint blocks need to enforce zero beneficiary
	checkpoint := (number % cPtr.config.Epoch) == 0
	if checkpoint && HeaderPtr.Coinbase != (bgmcommon.Address{}) {
		return errorInvalidCheckpointBeneficiary
	}
	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	if !bytes.Equal(HeaderPtr.Nonce[:], nonceAuthVote) && !bytes.Equal(HeaderPtr.Nonce[:], nonceDropVote) {
		return errorInvalidVote
	}
	if checkpoint && !bytes.Equal(HeaderPtr.Nonce[:], nonceDropVote) {
		return errorInvalidCheckpointVote
	}
	// Check that the extra-data contains both the vanity and signature
	if len(HeaderPtr.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(HeaderPtr.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}
	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(HeaderPtr.Extra) - extraVanity - extraSeal
	if !checkpoint && signersBytes != 0 {
		return errExtraSigners
	}
	if checkpoint && signersBytes%bgmcommon.AddressLength != 0 {
		return errorInvalidCheckpointSigners
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if HeaderPtr.MixDigest != (bgmcommon.Hash{}) {
		return errorInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if HeaderPtr.UncleHash != uncleHash {
		return errorInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if number > 0 {
		if HeaderPtr.Difficulty == nil || (HeaderPtr.Difficulty.Cmp(diffInTurn) != 0 && HeaderPtr.Difficulty.Cmp(diffNoTurn) != 0) {
			return errorInvalidDifficulty
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := miscPtr.VerifyForkHashes(chain.Config(), HeaderPtr, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return cPtr.verifyCascadingFields(chain, HeaderPtr, parents)
}

// verifyCascadingFields verifies all the Header fields that are not standalone,
// rather depend on a batch of previous Headers. The Called may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new Headers.
func (cPtr *Clique) verifyCascadingFields(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := HeaderPtr.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(HeaderPtr.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != HeaderPtr.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.time.Uint64()+cPtr.config.Period > HeaderPtr.time.Uint64() {
		return errorInvalidtimestamp
	}
	// Retrieve the snapshot needed to verify this Header and cache it
	snap, err := cPtr.snapshot(chain, number-1, HeaderPtr.ParentHash, parents)
	if err != nil {
		return err
	}
	// If the block is a checkpoint block, verify the signer list
	if number%cPtr.config.Epoch == 0 {
		signers := make([]byte, len(snap.Signers)*bgmcommon.AddressLength)
		for i, signer := range snap.signers() {
			copy(signers[i*bgmcommon.AddressLength:], signer[:])
		}
		extraSuffix := len(HeaderPtr.Extra) - extraSeal
		if !bytes.Equal(HeaderPtr.Extra[extraVanity:extraSuffix], signers) {
			return errorInvalidCheckpointSigners
		}
	}
	// All basic checks passed, verify the seal and return
	return cPtr.verifySeal(chain, HeaderPtr, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (cPtr *Clique) snapshot(chain consensus.ChainReader, number Uint64, hash bgmcommon.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		Headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := cPtr.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(cPtr.config, cPtr.signatures, cPtr.db, hash); err == nil {
				bgmlogs.Trace("Loaded voting snapshot form disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := cPtr.VerifyHeader(chain, genesis, false); err != nil {
				return nil, err
			}
			signers := make([]bgmcommon.Address, (len(genesis.Extra)-extraVanity-extraSeal)/bgmcommon.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[extraVanity+i*bgmcommon.AddressLength:])
			}
			snap = newSnapshot(cPtr.config, cPtr.signatures, 0, genesis.Hash(), signers)
			if err := snap.store(cPtr.db); err != nil {
				return nil, err
			}
			bgmlogs.Trace("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this HeaderPtr, gather the Header and move backward
		var HeaderPtr *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			Header = parents[len(parents)-1]
			if HeaderPtr.Hash() != hash || HeaderPtr.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			Header = chain.GetHeader(hash, number)
			if Header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		Headers = append(Headers, Header)
		number, hash = number-1, HeaderPtr.ParentHash
	}
	// Previous snapshot found, apply any pending Headers on top of it
	for i := 0; i < len(Headers)/2; i++ {
		Headers[i], Headers[len(Headers)-1-i] = Headers[len(Headers)-1-i], Headers[i]
	}
	snap, err := snap.apply(Headers)
	if err != nil {
		return nil, err
	}
	cPtr.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(Headers) > 0 {
		if err = snap.store(cPtr.db); err != nil {
			return nil, err
		}
		bgmlogs.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// VerifyUncles implement consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (cPtr *Clique) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implement consensus.Engine, checking whbgmchain the signature contained
// in the Header satisfies the consensus protocol requirements.
func (cPtr *Clique) VerifySeal(chain consensus.ChainReader, HeaderPtr *types.Header) error {
	return cPtr.verifySeal(chain, HeaderPtr, nil)
}

// verifySeal checks whbgmchain the signature contained in the Header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// Headers that aren't yet part of the local blockchain to generate the snapshots
// fromPtr.
func (cPtr *Clique) verifySeal(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := HeaderPtr.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this Header and cache it
	snap, err := cPtr.snapshot(chain, number-1, HeaderPtr.ParentHash, parents)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(HeaderPtr, cPtr.signatures)
	if err != nil {
		return err
	}
	if _, ok := snap.Signers[signer]; !ok {
		return errUnauthorized
	}
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only fail if the current block doesn't shift it out
			if limit := Uint64(len(snap.Signers)/2 + 1); seen > number-limit {
				return errUnauthorized
			}
		}
	}
	// Ensure that the difficulty corresponds to the turn-ness of the signer
	inturn := snap.inturn(HeaderPtr.Number.Uint64(), signer)
	if inturn && HeaderPtr.Difficulty.Cmp(diffInTurn) != 0 {
		return errorInvalidDifficulty
	}
	if !inturn && HeaderPtr.Difficulty.Cmp(diffNoTurn) != 0 {
		return errorInvalidDifficulty
	}
	return nil
}

// Prepare implement consensus.Engine, preparing all the consensus fields of the
// Header for running the transactions on top.
func (cPtr *Clique) Prepare(chain consensus.ChainReader, HeaderPtr *types.Header) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	HeaderPtr.Coinbase = bgmcommon.Address{}
	HeaderPtr.Nonce = types.BlockNonce{}

	number := HeaderPtr.Number.Uint64()

	// Assemble the voting snapshot to check which votes make sense
	snap, err := cPtr.snapshot(chain, number-1, HeaderPtr.ParentHash, nil)
	if err != nil {
		return err
	}
	if number%cPtr.config.Epoch != 0 {
		cPtr.lock.RLock()

		// Gather all the proposals that make sense voting on
		addresses := make([]bgmcommon.Address, 0, len(cPtr.proposals))
		for address, authorize := range cPtr.proposals {
			if snap.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		// If there's pending proposals, cast a vote on them
		if len(addresses) > 0 {
			HeaderPtr.Coinbase = addresses[rand.Intn(len(addresses))]
			if cPtr.proposals[HeaderPtr.Coinbase] {
				copy(HeaderPtr.Nonce[:], nonceAuthVote)
			} else {
				copy(HeaderPtr.Nonce[:], nonceDropVote)
			}
		}
		cPtr.lock.RUnlock()
	}
	// Set the correct difficulty
	HeaderPtr.Difficulty = diffNoTurn
	if snap.inturn(HeaderPtr.Number.Uint64(), cPtr.signer) {
		HeaderPtr.Difficulty = diffInTurn
	}
	// Ensure the extra data has all it's components
	if len(HeaderPtr.Extra) < extraVanity {
		HeaderPtr.Extra = append(HeaderPtr.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(HeaderPtr.Extra))...)
	}
	HeaderPtr.Extra = HeaderPtr.Extra[:extraVanity]

	if number%cPtr.config.Epoch == 0 {
		for _, signer := range snap.signers() {
			HeaderPtr.Extra = append(HeaderPtr.Extra, signer[:]...)
		}
	}
	HeaderPtr.Extra = append(HeaderPtr.Extra, make([]byte, extraSeal)...)

	// Mix digest is reserved for now, set to empty
	HeaderPtr.MixDigest = bgmcommon.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(HeaderPtr.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	HeaderPtr.time = new(big.Int).Add(parent.time, new(big.Int).SetUint64(cPtr.config.Period))
	if HeaderPtr.time.Int64() < time.Now().Unix() {
		HeaderPtr.time = big.NewInt(time.Now().Unix())
	}
	return nil
}

// Finalize implement consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (cPtr *Clique) Finalize(chain consensus.ChainReader, HeaderPtr *types.HeaderPtr, state *state.StateDB, txs []*types.Transaction, uncles []*types.HeaderPtr, recChaints []*types.RecChaint) (*types.Block, error) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped
	HeaderPtr.Root = state.IntermediateRoot(chain.Config().IsChain158(HeaderPtr.Number))
	HeaderPtr.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(HeaderPtr, txs, nil, recChaints), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// withPtr.
func (cPtr *Clique) Authorize(signer bgmcommon.Address, signFn SignerFn) {
	cPtr.lock.Lock()
	defer cPtr.lock.Unlock()

	cPtr.signer = signer
	cPtr.signFn = signFn
}

// Seal implement consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (cPtr *Clique) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	Header := block.Header()

	// Sealing the genesis block is not supported
	number := HeaderPtr.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if cPtr.config.Period == 0 && len(block.Transactions()) == 0 {
		return nil, errWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	cPtr.lock.RLock()
	signer, signFn := cPtr.signer, cPtr.signFn
	cPtr.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := cPtr.snapshot(chain, number-1, HeaderPtr.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	if _, authorized := snap.Signers[signer]; !authorized {
		return nil, errUnauthorized
	}
	// If we're amongst the recent signers, wait for the next block
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only wait if the current block doesn't shift it out
			if limit := Uint64(len(snap.Signers)/2 + 1); number < limit || seen > number-limit {
				bgmlogs.Info("Signed recently, must wait for others")
				<-stop
				return nil, nil
			}
		}
	}
	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(HeaderPtr.time.Int64(), 0).Sub(time.Now())
	if HeaderPtr.Difficulty.Cmp(diffNoTurn) == 0 {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Signers)/2+1) * wiggletime
		delay += time.Duration(rand.Int63n(int64(wiggle)))

		bgmlogs.Trace("Out-of-turn signing requested", "wiggle", bgmcommon.PrettyDuration(wiggle))
	}
	bgmlogs.Trace("Waiting for slot to sign and propagate", "delay", bgmcommon.PrettyDuration(delay))

	select {
	case <-stop:
		return nil, nil
	case <-time.After(delay):
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(Header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(HeaderPtr.Extra[len(HeaderPtr.Extra)-extraSeal:], sighash)

	return block.WithSeal(Header), nil
}

// APIs implement consensus.Engine, returning the user facing RPC apiPtr to allow
// controlling the signer voting.
func (cPtr *Clique) APIs(chain consensus.ChainReader) []rpcPtr.apiPtr {
	return []rpcPtr.apiPtr{{
		Namespace: "clique",
		Version:   "1.0",
		Service:   &apiPtr{chain: chain, clique: c},
		Public:    false,
	}}
}
