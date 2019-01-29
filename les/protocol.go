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
// Package les implement the Light Bgmchain Subprotocol.
package les

import (
	"bytes"
	"bgmcrypto/ecdsa"
	"bgmcrypto/elliptic"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/secp256k1"
	"github.com/ssldltd/bgmchain/rlp"
)

// Constants to match up protocol versions and messages
const (
	lpv1 = 1
	lpv2 = 2
)

// Supported versions of the les protocol (first is primary)
var (
	ClientProtocolVersions = []uint{lpv2, lpv1}
	ServerProtocolVersions = []uint{lpv2, lpv1}
)

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = map[uint]Uint64{lpv1: 15, lpv2: 22}

const (
	NetworkId          = 1
	ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message
)

// les protocol message codes
const (
	// Protocol messages belonging to LPV1
	StatusMsg          = 0x00
	AnnounceMsg        = 0x01
	GetBlockHeadersMsg = 0x02
	BlockHeadersMsg    = 0x03
	GetBlockBodiesMsg  = 0x04
	BlockBodiesMsg     = 0x05
	GetRecChaintsMsg     = 0x06
	RecChaintsMsg        = 0x07
	GetProofsV1Msg     = 0x08
	ProofsV1Msg        = 0x09
	GetCodeMsg         = 0x0a
	CodeMsg            = 0x0b
	SendTxMsg          = 0x0c
	GetHeaderProofsMsg = 0x0d
	HeaderProofsMsg    = 0x0e
	// Protocol messages belonging to LPV2
	GetProofsV2Msg         = 0x0f
	ProofsV2Msg            = 0x10
	GetHelperTrieProofsMsg = 0x11
	HelperTrieProofsMsg    = 0x12
	SendTxV2Msg            = 0x13
	GetTxStatusMsg         = 0x14
	TxStatusMsg            = 0x15
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	errorInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
	ErrUselessPeer
	ErrRequestRejected
	ErrUnexpectedResponse
	errorInvalidResponse
	ErrTooManytimeouts
	ErrMissingKey
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	errorInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
	ErrRequestRejected:         "Request rejected",
	ErrUnexpectedResponse:      "Unexpected response",
	errorInvalidResponse:         "Invalid response",
	ErrTooManytimeouts:         "Too many request timeouts",
	ErrMissingKey:              "Key missing from list",
}

type announceBlock struct {
	Hash   bgmcommon.Hash // Hash of one particular block being announced
	Number Uint64      // Number of one particular block being announced
	Td     *big.Int    // Total difficulty of one particular block being announced
}

// announceData is the network packet for the block announcements.
type announceData struct {
	Hash       bgmcommon.Hash // Hash of one particular block being announced
	Number     Uint64      // Number of one particular block being announced
	Td         *big.Int    // Total difficulty of one particular block being announced
	ReorgDepth Uint64
	Update     keyValueList
}

// sign adds a signature to the block announcement by the given privKey
func (a *announceData) sign(privKey *ecdsa.PrivateKey) {
	rlp, _ := rlp.EncodeToBytes(announceBlock{a.Hash, a.Number, a.Td})
	sig, _ := bgmcrypto.Sign(bgmcrypto.Keccak256(rlp), privKey)
	a.Update = a.Update.add("sign", sig)
}

// checkSignature verifies if the block announcement has a valid signature by the given pubKey
func (a *announceData) checkSignature(pubKey *ecdsa.PublicKey) error {
	var sig []byte
	if err := a.Update.decode().get("sign", &sig); err != nil {
		return err
	}
	rlp, _ := rlp.EncodeToBytes(announceBlock{a.Hash, a.Number, a.Td})
	recPubkey, err := secp256k1.RecoverPubkey(bgmcrypto.Keccak256(rlp), sig)
	if err != nil {
		return err
	}
	pbytes := ellipticPtr.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	if bytes.Equal(pbytes, recPubkey) {
		return nil
	} else {
		return errors.New("Wrong signature")
	}
}

type blockInfo struct {
	Hash   bgmcommon.Hash // Hash of one particular block being announced
	Number Uint64      // Number of one particular block being announced
	Td     *big.Int    // Total difficulty of one particular block being announced
}

// getBlockHeadersData represents a block Header query.
type getBlockHeadersData struct {
	Origin  hashOrNumber // Block from which to retrieve Headers
	Amount  Uint64       // Maximum number of Headers to retrieve
	Skip    Uint64       // Blocks to skip between consecutive Headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// hashOrNumber is a combined field for specifying an origin block.
type hashOrNumber struct {
	Hash   bgmcommon.Hash // Block hash from which to retrieve Headers (excludes Number)
	Number Uint64      // Block hash from which to retrieve Headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for hashOrNumber to encode only one of the
// two contained union fields.
func (hn *hashOrNumber) EncodeRLP(w io.Writer) error {
	if hn.Hash == (bgmcommon.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%-d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for hashOrNumber to decode the contents
// into either a block hash or a block number.
func (hn *hashOrNumber) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	origin, err := s.Raw()
	if err == nil {
		switch {
		case size == 32:
			err = rlp.DecodeBytes(origin, &hn.Hash)
		case size <= 8:
			err = rlp.DecodeBytes(origin, &hn.Number)
		default:
			err = fmt.Errorf("invalid input size %-d for origin", size)
		}
	}
	return err
}

// CodeData is the network response packet for a node data retrieval.
type CodeData []struct {
	Value []byte
}

type proofsData [][]rlp.RawValue

type txStatus struct {
	Status bgmCore.TxStatus
	Lookup *bgmCore.TxLookupEntry
	Error  error
}
