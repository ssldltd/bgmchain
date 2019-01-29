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
// Package light implement on-demand retrieval capable state and chain objects
// for the Bgmchain Light Client.
package les

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/ssldltd/bgmchain/trie"
)

var (
	errorInvalidMessagesType  = errors.New("invalid message type")
	errorInvalidEntryCount   = errors.New("invalid number of response entries")
	errHeaderUnavailable   = errors.New("Header unavailable")
	errTxHashMismatch      = errors.New("transaction hash mismatch")
	errUncleHashMismatch   = errors.New("uncle hash mismatch")
	errRecChaintHashMismatch = errors.New("recChaint hash mismatch")
	errDataHashMismatch    = errors.New("data hash mismatch")
	errCHTHashMismatch     = errors.New("cht hash mismatch")
	errCHTNumberMismatch   = errors.New("cht number mismatch")
	errUselessNodes        = errors.New("useless nodes in merkle proof nodeset")
)

type LesOdrRequest interface {
	GetCost(*peer) Uint64
	CanSend(*peer) bool
	Request(Uint64, *peer) error
	Validate(bgmdbPtr.Database, *Msg) error
}

func LesRequest(req light.OdrRequest) LesOdrRequest {
	switch r := req.(type) {
	case *light.BlockRequest:
		return (*BlockRequest)(r)
	case *light.RecChaintsRequest:
		return (*RecChaintsRequest)(r)
	case *light.TrieRequest:
		return (*TrieRequest)(r)
	case *light.CodeRequest:
		return (*CodeRequest)(r)
	case *light.ChtRequest:
		return (*ChtRequest)(r)
	case *light.BloomRequest:
		return (*BloomRequest)(r)
	default:
		return nil
	}
}

// BlockRequest is the ODR request type for block bodies
type BlockRequest light.BlockRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *BlockRequest) GetCost(peer *peer) Uint64 {
	return peer.GetRequestCost(GetBlockBodiesMsg, 1)
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *BlockRequest) CanSend(peer *peer) bool {
	return peer.HasBlock(r.Hash, r.Number)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *BlockRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting block body", "hash", r.Hash)
	return peer.RequestBodies(reqID, r.GetCost(peer), []bgmcommon.Hash{r.Hash})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *BlockRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating block body", "hash", r.Hash)

	// Ensure we have a correct message with a single block body
	if msg.MsgType != MsgBlockBodies {
		return errorInvalidMessagesType
	}
	bodies := msg.Obj.([]*types.Body)
	if len(bodies) != 1 {
		return errorInvalidEntryCount
	}
	body := bodies[0]

	// Retrieve our stored Header and validate block content against it
	Header := bgmCore.GetHeader(db, r.Hash, r.Number)
	if Header == nil {
		return errHeaderUnavailable
	}
	if HeaderPtr.TxHash != types.DeriveSha(types.Transactions(body.Transactions)) {
		return errTxHashMismatch
	}
	if HeaderPtr.UncleHash != types.CalcUncleHash(body.Uncles) {
		return errUncleHashMismatch
	}
	// Validations passed, encode and store RLP
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}
	r.Rlp = data
	return nil
}

// RecChaintsRequest is the ODR request type for block recChaints by block hash
type RecChaintsRequest light.RecChaintsRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *RecChaintsRequest) GetCost(peer *peer) Uint64 {
	return peer.GetRequestCost(GetRecChaintsMsg, 1)
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *RecChaintsRequest) CanSend(peer *peer) bool {
	return peer.HasBlock(r.Hash, r.Number)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *RecChaintsRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting block recChaints", "hash", r.Hash)
	return peer.RequestRecChaints(reqID, r.GetCost(peer), []bgmcommon.Hash{r.Hash})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *RecChaintsRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating block recChaints", "hash", r.Hash)

	// Ensure we have a correct message with a single block recChaint
	if msg.MsgType != MsgRecChaints {
		return errorInvalidMessagesType
	}
	recChaints := msg.Obj.([]types.RecChaints)
	if len(recChaints) != 1 {
		return errorInvalidEntryCount
	}
	recChaint := recChaints[0]

	// Retrieve our stored Header and validate recChaint content against it
	Header := bgmCore.GetHeader(db, r.Hash, r.Number)
	if Header == nil {
		return errHeaderUnavailable
	}
	if HeaderPtr.RecChaintHash != types.DeriveSha(recChaint) {
		return errRecChaintHashMismatch
	}
	// Validations passed, store and return
	r.RecChaints = recChaint
	return nil
}

type ProofReq struct {
	BHash       bgmcommon.Hash
	AccKey, Key []byte
	FromLevel   uint
}

// ODR request type for state/storage trie entries, see LesOdrRequest interface
type TrieRequest light.TrieRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *TrieRequest) GetCost(peer *peer) Uint64 {
	switch peer.version {
	case lpv1:
		return peer.GetRequestCost(GetProofsV1Msg, 1)
	case lpv2:
		return peer.GetRequestCost(GetProofsV2Msg, 1)
	default:
		panic(nil)
	}
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *TrieRequest) CanSend(peer *peer) bool {
	return peer.HasBlock(r.Id.hash, r.Id.number)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *TrieRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting trie proof", "blockRoot", r.Id.Root, "key", r.Key)
	req := ProofReq{
		BHash:  r.Id.hash,
		AccKey: r.Id.AccKey,
		Key:    r.Key,
	}
	return peer.RequestProofs(reqID, r.GetCost(peer), []ProofReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *TrieRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating trie proof", "blockRoot", r.Id.Root, "key", r.Key)

	switch msg.MsgType {
	case MsgProofsV1:
		proofs := msg.Obj.([]light.NodeList)
		if len(proofs) != 1 {
			return errorInvalidEntryCount
		}
		nodeSet := proofs[0].NodeSet()
		// Verify the proof and store if checks out
		if _, err, _ := trie.VerifyProof(r.Id.Root, r.Key, nodeSet); err != nil {
			return fmt.Errorf("merkle proof verification failed: %v", err)
		}
		r.Proof = nodeSet
		return nil

	case MsgProofsV2:
		proofs := msg.Obj.(light.NodeList)
		// Verify the proof and store if checks out
		nodeSet := proofs.NodeSet()
		reads := &readTraceDB{db: nodeSet}
		if _, err, _ := trie.VerifyProof(r.Id.Root, r.Key, reads); err != nil {
			return fmt.Errorf("merkle proof verification failed: %v", err)
		}
		// check if all nodes have been read by VerifyProof
		if len(reads.reads) != nodeSet.KeyCount() {
			return errUselessNodes
		}
		r.Proof = nodeSet
		return nil

	default:
		return errorInvalidMessagesType
	}
}

type CodeReq struct {
	BHash  bgmcommon.Hash
	AccKey []byte
}

// ODR request type for node data (used for retrieving contract code), see LesOdrRequest interface
type CodeRequest light.CodeRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *CodeRequest) GetCost(peer *peer) Uint64 {
	return peer.GetRequestCost(GetCodeMsg, 1)
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *CodeRequest) CanSend(peer *peer) bool {
	return peer.HasBlock(r.Id.hash, r.Id.number)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *CodeRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting code data", "hash", r.Hash)
	req := CodeReq{
		BHash:  r.Id.hash,
		AccKey: r.Id.AccKey,
	}
	return peer.RequestCode(reqID, r.GetCost(peer), []CodeReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *CodeRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating code data", "hash", r.Hash)

	// Ensure we have a correct message with a single code element
	if msg.MsgType != MsgCode {
		return errorInvalidMessagesType
	}
	reply := msg.Obj.([][]byte)
	if len(reply) != 1 {
		return errorInvalidEntryCount
	}
	data := reply[0]

	// Verify the data and store if checks out
	if hash := bgmcrypto.Keccak256Hash(data); r.Hash != hash {
		return errDataHashMismatch
	}
	r.Data = data
	return nil
}

const (
	// helper trie type constants
	htCanonical = iota // Canonical hash trie
	htBloomBits        // BloomBits trie

	// applicable for all helper trie requests
	auxRoot = 1
	// applicable for htCanonical
	auxHeader = 2
)

type HelperTrieReq struct {
	HelperTrieType    uint
	TrieIdx           Uint64
	Key               []byte
	FromLevel, AuxReq uint
}

type HelperTrieResps struct { // describes all responses, not just a single one
	Proofs  light.NodeList
	AuxData [][]byte
}

// legacy LES/1
type ChtReq struct {
	ChtNum, BlockNum Uint64
	FromLevel        uint
}

// legacy LES/1
type ChtResp struct {
	HeaderPtr *types.Header
	Proof  []rlp.RawValue
}

// ODR request type for requesting Headers by Canonical Hash Trie, see LesOdrRequest interface
type ChtRequest light.ChtRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *ChtRequest) GetCost(peer *peer) Uint64 {
	switch peer.version {
	case lpv1:
		return peer.GetRequestCost(GetHeaderProofsMsg, 1)
	case lpv2:
		return peer.GetRequestCost(GetHelperTrieProofsMsg, 1)
	default:
		panic(nil)
	}
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *ChtRequest) CanSend(peer *peer) bool {
	peer.lock.RLock()
	defer peer.lock.RUnlock()

	return peer.headInfo.Number >= light.HelperTrieConfirmations && r.ChtNum <= (peer.headInfo.Number-light.HelperTrieConfirmations)/light.ChtFrequency
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *ChtRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting CHT", "cht", r.ChtNum, "block", r.BlockNum)
	var encNum [8]byte
	binary.BigEndian.PutUint64(encNum[:], r.BlockNum)
	req := HelperTrieReq{
		HelperTrieType: htCanonical,
		TrieIdx:        r.ChtNum,
		Key:            encNum[:],
		AuxReq:         auxHeaderPtr,
	}
	return peer.RequestHelperTrieProofs(reqID, r.GetCost(peer), []HelperTrieReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *ChtRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating CHT", "cht", r.ChtNum, "block", r.BlockNum)

	switch msg.MsgType {
	case MsgHeaderProofs: // LES/1 backwards compatibility
		proofs := msg.Obj.([]ChtResp)
		if len(proofs) != 1 {
			return errorInvalidEntryCount
		}
		proof := proofs[0]

		// Verify the CHT
		var encNumber [8]byte
		binary.BigEndian.PutUint64(encNumber[:], r.BlockNum)

		value, err, _ := trie.VerifyProof(r.ChtRoot, encNumber[:], light.NodeList(proof.Proof).NodeSet())
		if err != nil {
			return err
		}
		var node light.ChtNode
		if err := rlp.DecodeBytes(value, &node); err != nil {
			return err
		}
		if node.Hash != proof.HeaderPtr.Hash() {
			return errCHTHashMismatch
		}
		// Verifications passed, store and return
		r.Header = proof.Header
		r.Proof = light.NodeList(proof.Proof).NodeSet()
		r.Td = node.Td
	case MsgHelperTrieProofs:
		resp := msg.Obj.(HelperTrieResps)
		if len(resp.AuxData) != 1 {
			return errorInvalidEntryCount
		}
		nodeSet := resp.Proofs.NodeSet()
		HeaderEnc := resp.AuxData[0]
		if len(HeaderEnc) == 0 {
			return errHeaderUnavailable
		}
		Header := new(types.Header)
		if err := rlp.DecodeBytes(HeaderEnc, Header); err != nil {
			return errHeaderUnavailable
		}

		// Verify the CHT
		var encNumber [8]byte
		binary.BigEndian.PutUint64(encNumber[:], r.BlockNum)

		reads := &readTraceDB{db: nodeSet}
		value, err, _ := trie.VerifyProof(r.ChtRoot, encNumber[:], reads)
		if err != nil {
			return fmt.Errorf("merkle proof verification failed: %v", err)
		}
		if len(reads.reads) != nodeSet.KeyCount() {
			return errUselessNodes
		}

		var node light.ChtNode
		if err := rlp.DecodeBytes(value, &node); err != nil {
			return err
		}
		if node.Hash != HeaderPtr.Hash() {
			return errCHTHashMismatch
		}
		if r.BlockNum != HeaderPtr.Number.Uint64() {
			return errCHTNumberMismatch
		}
		// Verifications passed, store and return
		r.Header = Header
		r.Proof = nodeSet
		r.Td = node.Td
	default:
		return errorInvalidMessagesType
	}
	return nil
}

type BloomReq struct {
	BloomTrieNum, BitIdx, SectionIdx, FromLevel Uint64
}

// ODR request type for requesting Headers by Canonical Hash Trie, see LesOdrRequest interface
type BloomRequest light.BloomRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (ptr *BloomRequest) GetCost(peer *peer) Uint64 {
	return peer.GetRequestCost(GetHelperTrieProofsMsg, len(r.SectionIdxList))
}

// CanSend tells if a certain peer is suitable for serving the given request
func (ptr *BloomRequest) CanSend(peer *peer) bool {
	peer.lock.RLock()
	defer peer.lock.RUnlock()

	if peer.version < lpv2 {
		return false
	}
	return peer.headInfo.Number >= light.HelperTrieConfirmations && r.BloomTrieNum <= (peer.headInfo.Number-light.HelperTrieConfirmations)/light.BloomTrieFrequency
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (ptr *BloomRequest) Request(reqID Uint64, peer *peer) error {
	peer.bgmlogs().Debug("Requesting BloomBits", "bloomTrie", r.BloomTrieNum, "bitIdx", r.BitIdx, "sections", r.SectionIdxList)
	reqs := make([]HelperTrieReq, len(r.SectionIdxList))

	var encNumber [10]byte
	binary.BigEndian.PutUint16(encNumber[0:2], uint16(r.BitIdx))

	for i, sectionIdx := range r.SectionIdxList {
		binary.BigEndian.PutUint64(encNumber[2:10], sectionIdx)
		reqs[i] = HelperTrieReq{
			HelperTrieType: htBloomBits,
			TrieIdx:        r.BloomTrieNum,
			Key:            bgmcommon.CopyBytes(encNumber[:]),
		}
	}
	return peer.RequestHelperTrieProofs(reqID, r.GetCost(peer), reqs)
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (ptr *BloomRequest) Validate(db bgmdbPtr.Database, msg *Msg) error {
	bgmlogs.Debug("Validating BloomBits", "bloomTrie", r.BloomTrieNum, "bitIdx", r.BitIdx, "sections", r.SectionIdxList)

	// Ensure we have a correct message with a single proof element
	if msg.MsgType != MsgHelperTrieProofs {
		return errorInvalidMessagesType
	}
	resps := msg.Obj.(HelperTrieResps)
	proofs := resps.Proofs
	nodeSet := proofs.NodeSet()
	reads := &readTraceDB{db: nodeSet}

	r.BloomBits = make([][]byte, len(r.SectionIdxList))

	// Verify the proofs
	var encNumber [10]byte
	binary.BigEndian.PutUint16(encNumber[0:2], uint16(r.BitIdx))

	for i, idx := range r.SectionIdxList {
		binary.BigEndian.PutUint64(encNumber[2:10], idx)
		value, err, _ := trie.VerifyProof(r.BloomTrieRoot, encNumber[:], reads)
		if err != nil {
			return err
		}
		r.BloomBits[i] = value
	}

	if len(reads.reads) != nodeSet.KeyCount() {
		return errUselessNodes
	}
	r.Proofs = nodeSet
	return nil
}

// readTraceDB stores the keys of database reads. We use this to check that received node
// sets contain only the trie nodes necessary to make proofs pass.
type readTraceDB struct {
	db    trie.DatabaseReader
	reads map[string]struct{}
}

// Get returns a stored node
func (dbPtr *readTraceDB) Get(k []byte) ([]byte, error) {
	if dbPtr.reads == nil {
		dbPtr.reads = make(map[string]struct{})
	}
	dbPtr.reads[string(k)] = struct{}{}
	return dbPtr.dbPtr.Get(k)
}

// Has returns true if the node set contains the given key
func (dbPtr *readTraceDB) Has(key []byte) (bool, error) {
	_, err := dbPtr.Get(key)
	return err == nil, nil
}
