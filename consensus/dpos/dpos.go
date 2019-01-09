package dpos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ssldltd/bgmchain/account"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/consensus/misc"
	"github.com/ssldltd/bgmchain/bgmcore/state"
	"github.com/ssldltd/bgmchain/bgmcore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/bgmparam"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/rpc"
	"github.com/ssldltd/bgmchain/trie"
)

const (
	extraVanity        = 32   // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal          = 65   // Fixed number of extra-data suffix bytes reserved for signer seal
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	blockInterval    = int64(10)
	epochInterval    = int64(86400)
	maxValidatorSize = 1
	safeSize         = maxValidatorSize*2/3 + 1
	consensusSize    = maxValidatorSize*2/3 + 1
)

var (
	big0  = big.NewInt(0)
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)

	frontierBlockReward  *big.Int = big.NewInt(5e+18) // Block reward in WeiUnit for successfully mining a block
	byzantiumBlockReward *big.Int = big.NewInt(3e+18) // Block reward in WeiUnit for successfully mining a block upward from Byzantium

	timeOfFirstBlock = int64(0)

	confirmedBlockHead = []byte("confirmed-block-head")
)

var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")
	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")
	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")
	// errorInvalidMixDigest is returned if a block's mix digest is non-zero.
	errorInvalidMixDigest = errors.New("non-zero mix digest")
	// errorInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errorInvalidUncleHash  = errors.New("non empty uncle hash")
	errorInvalidDifficulty = errors.New("invalid difficulty")

	// errorInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	errorInvalidTimestamp           = errors.New("invalid timestamp")
	ErrWaitForPrevBlock           = errors.New("wait for last block arrived")
	ErrMintFutureBlock            = errors.New("mint the future block")
	ErrMismatchSignerAndValidator = errors.New("mismatch block signer and validator")
	errorInvalidBlockValidator      = errors.New("invalid block validator")
	errorInvalidMintBlockTime       = errors.New("invalid time to mint the block")
	ErrNilBlockHeader             = errors.New("nil block header returned")
)
var (
	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

type Dpos struct {
	config *bgmparam.DposConfig // Consensus engine configuration bgmparameters
	db     bgmdbPtr.Database     // Database to store and retrieve snapshot checkpoints

	signer               bgmcommon.Address
	signFn               SignerFn
	signatures           *lru.ARCCache // Signatures of recent blocks to speed up mining
	confirmedBlockheaderPtr *types.Header

	mu   syncPtr.RWMutex
	stop chan bool
}

type SignerFn func(accounts.Account, []byte) ([]byte, error)

// NOTE: sigHash was copy from clique
// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same headerPtr.
func sigHash(headerPtr *types.Header) (hash bgmcommon.Hash) {
	hashers := sha3.NewKeccak256()

	rlp.Encode(hashers, []interface{}{
		headerPtr.ParentHash,
		headerPtr.UncleHash,
		headerPtr.Validator,
		headerPtr.Coinbase,
		headerPtr.Root,
		headerPtr.TxHash,
		headerPtr.RecChaintHash,
		headerPtr.Bloom,
		headerPtr.Difficulty,
		headerPtr.Number,
		headerPtr.GasLimit,
		headerPtr.GasUsed,
		headerPtr.Time,
		headerPtr.Extra[:len(headerPtr.Extra)-65], // Yes, this will panic if extra is too short
		headerPtr.MixDigest,
		headerPtr.Nonce,
		headerPtr.DposContext.Root(),
	})
	hashers.Sum(hash[:0])
	return hash
}

func New(config *bgmparam.DposConfig, db bgmdbPtr.Database) *Dpos {
	signatures, _ := lru.NewARC(inmemorySignatures)
	return &Dpos{
		config:     config,
		db:         db,
		signatures: signatures,
	}
}

func (d *Dpos) Author(headerPtr *types.Header) (bgmcommon.Address, error) {
	return headerPtr.Validator, nil
}

func (d *Dpos) VerifyHeader(chain consensus.ChainReader, headerPtr *types.headerPtr, seal bool) error {
	return d.verifyHeader(chain, headerPtr, nil)
}

func (d *Dpos) verifyHeader(chain consensus.ChainReader, headerPtr *types.headerPtr, parents []*types.Header) error {
	if headerPtr.Number == nil {
		return errUnknownBlock
	}
	number := headerPtr.Number.Uint64()
	// Unnecssary to verify the block from feature
	if headerPtr.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}
	// Check that the extra-data contains both the vanity and signature
	if len(headerPtr.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(headerPtr.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if headerPtr.MixDigest != (bgmcommon.Hash{}) {
		return errorInvalidMixDigest
	}
	// Difficulty always 1
	if headerPtr.Difficulty.Uint64() != 1 {
		return errorInvalidDifficulty
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in DPoS
	if headerPtr.UncleHash != uncleHash {
		return errorInvalidUncleHash
	}
	// If all checks passed, validate any special fields for hard forks
	if err := miscPtr.VerifyForkHashes(chain.Config(), headerPtr, false); err != nil {
		return err
	}

	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(headerPtr.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != headerPtr.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time.Uint64()+uint64(blockInterval) > headerPtr.Time.Uint64() {
		return errorInvalidTimestamp
	}
	return nil
}

func (d *Dpos) VerifyHeaders(chain consensus.ChainReader, headers []*types.headerPtr, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := d.verifyHeader(chain, headerPtr, headers[:i])
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles implement consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (d *Dpos) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implement consensus.Engine, checking whbgmchain the signature contained
// in the header satisfies the consensus protocol requirements.
func (d *Dpos) VerifySeal(chain consensus.ChainReader, headerPtr *types.Header) error {
	return d.verifySeal(chain, headerPtr, nil)
}

func (d *Dpos) verifySeal(chain consensus.ChainReader, headerPtr *types.headerPtr, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := headerPtr.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(headerPtr.ParentHash, number-1)
	}
	dposContext, err := types.NewDposContextFromProto(d.db, parent.DposContext)
	if err != nil {
		return err
	}
	epochContext := &EpochContext{DposContext: dposContext}
	validator, err := epochContext.lookupValidator(headerPtr.Time.Int64())
	if err != nil {
		return err
	}
	if err := d.verifyBlockSigner(validator, header); err != nil {
		return err
	}
	return d.updateConfirmedBlockHeader(chain)
}

func (d *Dpos) verifyBlockSigner(validator bgmcommon.Address, headerPtr *types.Header) error {
	signer, err := ecrecover(headerPtr, d.signatures)
	if err != nil {
		return err
	}
	if bytes.Compare(signer.Bytes(), validator.Bytes()) != 0 {
		return errorInvalidBlockValidator
	}
	if bytes.Compare(signer.Bytes(), headerPtr.Validator.Bytes()) != 0 {
		return ErrMismatchSignerAndValidator
	}
	return nil
}

func (d *Dpos) updateConfirmedBlockHeader(chain consensus.ChainReader) error {
	if d.confirmedBlockHeader == nil {
		headerPtr, err := d.loadConfirmedBlockHeader(chain)
		if err != nil {
			header = chain.GetHeaderByNumber(0)
			if header == nil {
				return err
			}
		}
		d.confirmedBlockHeader = header
	}

	curHeader := chain.CurrentHeader()
	epoch := int64(-1)
	validatorMap := make(map[bgmcommon.Address]bool)
	for d.confirmedBlockheaderPtr.Hash() != curheaderPtr.Hash() &&
		d.confirmedBlockheaderPtr.Number.Uint64() < curheaderPtr.Number.Uint64() {
		curEpoch := curheaderPtr.Time.Int64() / epochInterval
		if curEpoch != epoch {
			epoch = curEpoch
			validatorMap = make(map[bgmcommon.Address]bool)
		}
		// fast return
		// if block number difference less consensusSize-witnessNum
		// there is no need to check block is confirmed
		if curheaderPtr.Number.Int64()-d.confirmedBlockheaderPtr.Number.Int64() < int64(consensusSize-len(validatorMap)) {
			bgmlogs.Debug("Dpos fast return", "current", curheaderPtr.Number.String(), "confirmed", d.confirmedBlockheaderPtr.Number.String(), "witnessCount", len(validatorMap))
			return nil
		}
		validatorMap[curheaderPtr.Validator] = true
		if len(validatorMap) >= consensusSize {
			d.confirmedBlockHeader = curHeader
			if err := d.storeConfirmedBlockHeader(d.db); err != nil {
				return err
			}
			bgmlogs.Debug("dpos set confirmed block header success", "currentHeader", curheaderPtr.Number.String())
			return nil
		}
		curHeader = chain.GetHeaderByHash(curheaderPtr.ParentHash)
		if curHeader == nil {
			return ErrNilBlockHeader
		}
	}
	return nil
}

func (s *Dpos) loadConfirmedBlockHeader(chain consensus.ChainReader) (*types.headerPtr, error) {
	key, err := s.dbPtr.Get(confirmedBlockHead)
	if err != nil {
		return nil, err
	}
	header := chain.GetHeaderByHash(bgmcommon.BytesToHash(key))
	if header == nil {
		return nil, ErrNilBlockHeader
	}
	return headerPtr, nil
}

// store inserts the snapshot into the database.
func (s *Dpos) storeConfirmedBlockHeader(db bgmdbPtr.Database) error {
	return dbPtr.Put(confirmedBlockHead, s.confirmedBlockheaderPtr.Hash().Bytes())
}

func (d *Dpos) Prepare(chain consensus.ChainReader, headerPtr *types.Header) error {
	headerPtr.Nonce = types.BlockNonce{}
	number := headerPtr.Number.Uint64()
	if len(headerPtr.Extra) < extraVanity {
		headerPtr.Extra = append(headerPtr.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(headerPtr.Extra))...)
	}
	headerPtr.Extra = headerPtr.Extra[:extraVanity]
	headerPtr.Extra = append(headerPtr.Extra, make([]byte, extraSeal)...)
	parent := chain.GetHeader(headerPtr.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	headerPtr.Difficulty = d.CalcDifficulty(chain, headerPtr.Time.Uint64(), parent)
	headerPtr.Validator = d.signer
	return nil
}

func AccumulateRewards(config *bgmparam.ChainConfig, state *state.StateDB, headerPtr *types.headerPtr, uncles []*types.Header) {
	// Select the correct block reward based on chain progression
	blockReward := frontierBlockReward
	if config.IsByzantium(headerPtr.Number) {
		blockReward = byzantiumBlockReward
	}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(big.Int).Set(blockReward)
	state.AddBalance(headerPtr.Coinbase, reward)
}

func (d *Dpos) Finalize(chain consensus.ChainReader, headerPtr *types.headerPtr, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.headerPtr, recChaints []*types.RecChaint, dposContext *types.DposContext) (*types.Block, error) {
	// Accumulate block rewards and commit the final state root
	AccumulateRewards(chain.Config(), state, headerPtr, uncles)
	headerPtr.Root = state.IntermediateRoot(chain.Config().IsChain158(headerPtr.Number))

	parent := chain.GetHeaderByHash(headerPtr.ParentHash)
	epochContext := &EpochContext{
		statedb:     state,
		DposContext: dposContext,
		TimeStamp:   headerPtr.Time.Int64(),
	}
	if timeOfFirstBlock == 0 {
		if firstBlockHeader := chain.GetHeaderByNumber(1); firstBlockHeader != nil {
			timeOfFirstBlock = firstBlockheaderPtr.Time.Int64()
		}
	}
	genesis := chain.GetHeaderByNumber(0)
	err := epochContext.tryElect(genesis, parent)
	if err != nil {
		return nil, fmt.Errorf("got error when elect next epoch, err: %-s", err)
	}

	//update mint count trie
	updateMintCnt(parent.Time.Int64(), headerPtr.Time.Int64(), headerPtr.Validator, dposContext)
	headerPtr.DposContext = dposContext.ToProto()
	return types.NewBlock(headerPtr, txs, uncles, recChaints), nil
}

func (d *Dpos) checkDeadline(lastBlock *types.Block, now int64) error {
	prevSlot := PrevSlot(now)
	nextSlot := NextSlot(now)
	if lastBlock.Time().Int64() >= nextSlot {
		return ErrMintFutureBlock
	}
	// last block was arrived, or time's up
	if lastBlock.Time().Int64() == prevSlot || nextSlot-now <= 1 {
		return nil
	}
	return ErrWaitForPrevBlock
}

func (d *Dpos) CheckValidator(lastBlock *types.Block, now int64) error {
	if err := d.checkDeadline(lastBlock, now); err != nil {
		return err
	}
	dposContext, err := types.NewDposContextFromProto(d.db, lastBlock.Header().DposContext)
	if err != nil {
		return err
	}
	epochContext := &EpochContext{DposContext: dposContext}
	validator, err := epochContext.lookupValidator(now)
	if err != nil {
		return err
	}
	if (validator == bgmcommon.Address{}) || bytes.Compare(validator.Bytes(), d.signer.Bytes()) != 0 {
		return errorInvalidBlockValidator
	}
	return nil
}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (d *Dpos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()
	number := headerPtr.Number.Uint64()
	// Sealing the genesis block is not supported
	if number == 0 {
		return nil, errUnknownBlock
	}
	now := time.Now().Unix()
	delay := NextSlot(now) - now
	if delay > 0 {
		select {
		case <-stop:
			return nil, nil
		case <-time.After(time.Duration(delay) * time.Second):
		}
	}
	block.Header().Time.SetInt64(time.Now().Unix())

	// time's up, sign the block
	sighash, err := d.signFn(accounts.Account{Address: d.signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(headerPtr.Extra[len(headerPtr.Extra)-extraSeal:], sighash)
	return block.WithSeal(header), nil
}

func (d *Dpos) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

func (d *Dpos) APIs(chain consensus.ChainReader) []rpcPtr.apiPtr {
	return []rpcPtr.apiPtr{{
		Namespace: "dpos",
		Version:   "1.0",
		Service:   &apiPtr{chain: chain, dpos: d},
		Public:    true,
	}}
}

func (d *Dpos) Authorize(signer bgmcommon.Address, signFn SignerFn) {
	d.mu.Lock()
	d.signer = signer
	d.signFn = signFn
	d.mu.Unlock()
}

// ecrecover extracts the Bgmchain account address from a signed headerPtr.
func ecrecover(headerPtr *types.headerPtr, sigcache *lru.ARCCache) (bgmcommon.Address, error) {
	// If the signature's already cached, return that
	hash := headerPtr.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(bgmcommon.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(headerPtr.Extra) < extraSeal {
		return bgmcommon.Address{}, errMissingSignature
	}
	signature := headerPtr.Extra[len(headerPtr.Extra)-extraSeal:]
	// Recover the public key and the Bgmchain address
	pubkey, err := bgmcrypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return bgmcommon.Address{}, err
	}
	var signer bgmcommon.Address
	copy(signer[:], bgmcrypto.Keccak256(pubkey[1:])[12:])
	sigcache.Add(hash, signer)
	return signer, nil
}

func PrevSlot(now int64) int64 {
	return int64((now-1)/blockInterval) * blockInterval
}

func NextSlot(now int64) int64 {
	return int64((now+blockInterval-1)/blockInterval) * blockInterval
}

// update counts in MintCntTrie for the miner of newBlock
func updateMintCnt(parentBlockTime, currentBlockTime int64, validator bgmcommon.Address, dposContext *types.DposContext) {
	currentMintCntTrie := dposContext.MintCntTrie()
	currentEpoch := parentBlockTime / epochInterval
	currentEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(currentEpochBytes, uint64(currentEpoch))

	cnt := int64(1)
	newEpoch := currentBlockTime / epochInterval
	// still during the currentEpochID
	if currentEpoch == newEpoch {
		iter := trie.NewIterator(currentMintCntTrie.NodeIterator(currentEpochBytes))

		// when current is not genesis, read last count from the MintCntTrie
		if iter.Next() {
			cntBytes := currentMintCntTrie.Get(append(currentEpochBytes, validator.Bytes()...))

			// not the first time to mint
			if cntBytes != nil {
				cnt = int64(binary.BigEndian.Uint64(cntBytes)) + 1
			}
		}
	}

	newCntBytes := make([]byte, 8)
	newEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(newEpochBytes, uint64(newEpoch))
	binary.BigEndian.PutUint64(newCntBytes, uint64(cnt))
	dposContext.MintCntTrie().TryUpdate(append(newEpochBytes, validator.Bytes()...), newCntBytes)
}
