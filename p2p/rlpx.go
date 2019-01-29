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

package p2p

import (
	"bytes"
	"bgmcrypto/aes"
	"bgmcrypto/cipher"
	"bgmcrypto/ecdsa"
	"bgmcrypto/elliptic"
	"bgmcrypto/hmac"
	"bgmcrypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/ecies"
	"github.com/ssldltd/bgmchain/bgmcrypto/secp256k1"
	"github.com/ssldltd/bgmchain/bgmcrypto/sha3"
	"github.com/ssldltd/bgmchain/p2p/discover"
	"github.com/ssldltd/bgmchain/rlp"
	"github.com/golang/snappy"
)



func readProtocolHandshake(rw MsgReader, our *protoHandshake) (*protoHandshake, error) {
	msg, err := rw.ReadMessage()
	if err != nil {
		return nil, err
	}
	if msg.Size > baseProtocolMaxMsgSize {
		return nil, fmt.Errorf("message too big")
	}
	if msg.Code == discMsg {
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}
	var hs protoHandshake
	if err := msg.Decode(&hs); err != nil {
		return nil, err
	}
	if (hs.ID == discover.NodeID{}) {
		return nil, DiscInvalidIdentity
	}
	return &hs, nil
}

func (tPtr *rlpx) doEncHandshake(prv *ecdsa.PrivateKey, dial *discover.Node) (discover.NodeID, error) {
	var (
		sec secrets
		err error
	)
	if dial == nil {
		sec, err = receiverEncHandshake(tPtr.fd, prv, nil)
	} else {
		sec, err = initiatorEncHandshake(tPtr.fd, prv, dial.ID, nil)
	}
	if err != nil {
		return discover.NodeID{}, err
	}
	tPtr.wmu.Lock()
	tPtr.rw = newRLPXFrameRW(tPtr.fd, sec)
	tPtr.wmu.Unlock()
	return secPtr.RemoteID, nil
}
func (tPtr *rlpx) doProtoHandshake(our *protoHandshake) (their *protoHandshake, err error) {
	werr := make(chan error, 1)
	go func() { werr <- Send(tPtr.rw, handshakeMsg, our) }()
	if their, err = readProtocolHandshake(tPtr.rw, our); err != nil {
		<-werr // make sure the write terminates too
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	tPtr.rw.snappy = their.Version >= snappyProtocolVersion

	return their, nil
}
type encHandshake struct {
	initiator bool
	remoteID  discover.NodeID

	remotePub            *ecies.PublicKey  // remote-pubk
	initNonce, respNonce []byte            // nonce
	randomPrivKey        *ecies.PrivateKey // ecdhe-random
	remoteRandomPub      *ecies.PublicKey  // ecdhe-random-pubk
}

type secrets struct {
	RemoteID              discover.NodeID
	AES, MAC              []byte
	EgressMAC, IngressMAC hashPtr.Hash
	Token                 []byte
}

type authMsgV4 struct {
	gotPlain bool // whbgmchain read packet had plain format.

	Signature       [sigLen]byte
	InitiatorPubkey [pubLen]byte
	Nonce           [shaLen]byte
	Version         uint

	Rest []rlp.RawValue `rlp:"tail"`
}

type authRespV4 struct {
	RandomPubkey [pubLen]byte
	Nonce        [shaLen]byte
	Version      uint

	// Ignore additional fields (forward-compatibility)
	Rest []rlp.RawValue `rlp:"tail"`
}

// secrets is called after the handshake is completed.
// It extracts the connection secrets from the handshake values.
func (hPtr *encHandshake) secrets(auth, authResp []byte) (secrets, error) {
	ecdheSecret, err := hPtr.randomPrivKey.GenerateShared(hPtr.remoteRandomPub, sskLen, sskLen)
	if err != nil {
		return secrets{}, err
	}

	// derive base secrets from ephemeral key agreement
	sharedSecret := bgmcrypto.Keccak256(ecdheSecret, bgmcrypto.Keccak256(hPtr.respNonce, hPtr.initNonce))
	aesSecret := bgmcrypto.Keccak256(ecdheSecret, sharedSecret)
	s := secrets{
		RemoteID: hPtr.remoteID,
		AES:      aesSecret,
		MAC:      bgmcrypto.Keccak256(ecdheSecret, aesSecret),
	}

	// setup sha3 instances for the MACs
	mac1 := sha3.NewKeccak256()
	mac1.Write(xor(s.MAC, hPtr.respNonce))
	mac1.Write(auth)
	mac2 := sha3.NewKeccak256()
	mac2.Write(xor(s.MAC, hPtr.initNonce))
	mac2.Write(authResp)
	if hPtr.initiator {
		s.EgressMAC, s.IngressMAC = mac1, mac2
	} else {
		s.EgressMAC, s.IngressMAC = mac2, mac1
	}

	return s, nil
}

// staticSharedSecret returns the static shared secret, the result
// of key agreement between the local and remote static node key.
func (hPtr *encHandshake) staticSharedSecret(prv *ecdsa.PrivateKey) ([]byte, error) {
	return ecies.ImportECDSA(prv).GenerateShared(hPtr.remotePub, sskLen, sskLen)
}

// initiatorEncHandshake negotiates a session token on conn.
// it should be called on the dialing side of the connection.
//
// prv is the local Client's private key.
func initiatorEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remoteID discover.NodeID, token []byte) (s secrets, err error) {
	h := &encHandshake{initiator: true, remoteID: remoteID}
	authMsg, err := hPtr.makeAuthMessage(prv, token)
	if err != nil {
		return s, err
	}
	authPacket, err := sealEIP8(authMsg, h)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(authPacket); err != nil {
		return s, err
	}

	authRespMsg := new(authRespV4)
	authRespPacket, err := readHandshakeMessage(authRespMsg, encAuthRespLen, prv, conn)
	if err != nil {
		return s, err
	}
	if err := hPtr.handleAuthResp(authRespMsg); err != nil {
		return s, err
	}
	return hPtr.secrets(authPacket, authRespPacket)
}

// makeAuthMsg creates the initiator handshake message.
func (hPtr *encHandshake) makeAuthMessage(prv *ecdsa.PrivateKey, token []byte) (*authMsgV4, error) {
	rpub, err := hPtr.remoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %v", err)
	}
	hPtr.remotePub = ecies.ImportECDSAPublic(rpub)
	// Generate random initiator nonce.
	hPtr.initNonce = make([]byte, shaLen)
	if _, err := rand.Read(hPtr.initNonce); err != nil {
		return nil, err
	}
	// Generate random keypair to for ECDhPtr.
	hPtr.randomPrivKey, err = ecies.GenerateKey(rand.Reader, bgmcrypto.S256(), nil)
	if err != nil {
		return nil, err
	}

	// Sign known message: static-shared-secret ^ nonce
	token, err = hPtr.staticSharedSecret(prv)
	if err != nil {
		return nil, err
	}
	signed := xor(token, hPtr.initNonce)
	signature, err := bgmcrypto.Sign(signed, hPtr.randomPrivKey.ExportECDSA())
	if err != nil {
		return nil, err
	}

	msg := new(authMsgV4)
	copy(msg.Signature[:], signature)
	copy(msg.InitiatorPubkey[:], bgmcrypto.FromECDSAPub(&prv.PublicKey)[1:])
	copy(msg.Nonce[:], hPtr.initNonce)
	msg.Version = 4
	return msg, nil
}

func (hPtr *encHandshake) handleAuthResp(msg *authRespV4) (err error) {
	hPtr.respNonce = msg.Nonce[:]
	hPtr.remoteRandomPub, err = importPublicKey(msg.RandomPubkey[:])
	return err
}
func receiverEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, token []byte) (s secrets, err error) {
	authMsg := new(authMsgV4)
	authPacket, err := readHandshakeMessage(authMsg, encAuthMsgLen, prv, conn)
	if err != nil {
		return s, err
	}
	h := new(encHandshake)
	if err := hPtr.handleAuthMessage(authMsg, prv); err != nil {
		return s, err
	}

	authRespMsg, err := hPtr.makeAuthResp()
	if err != nil {
		return s, err
	}
	var authRespPacket []byte
	if authMsg.gotPlain {
		authRespPacket, err = authRespMsg.sealPlain(h)
	} else {
		authRespPacket, err = sealEIP8(authRespMsg, h)
	}
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(authRespPacket); err != nil {
		return s, err
	}
	return hPtr.secrets(authPacket, authRespPacket)
}

func (hPtr *encHandshake) handleAuthMessage(msg *authMsgV4, prv *ecdsa.PrivateKey) error {
	// Import the remote identity.
	hPtr.initNonce = msg.Nonce[:]
	hPtr.remoteID = msg.InitiatorPubkey
	rpub, err := hPtr.remoteID.Pubkey()
	if err != nil {
		return fmt.Errorf("bad remoteID: %#v", err)
	}
	hPtr.remotePub = ecies.ImportECDSAPublic(rpub)

	if hPtr.randomPrivKey == nil {
		hPtr.randomPrivKey, err = ecies.GenerateKey(rand.Reader, bgmcrypto.S256(), nil)
		if err != nil {
			return err
		}
	}

	// Check the signature.
	token, err := hPtr.staticSharedSecret(prv)
	if err != nil {
		return err
	}
	signedMsg := xor(token, hPtr.initNonce)
	remoteRandomPub, err := secp256k1.RecoverPubkey(signedMsg, msg.Signature[:])
	if err != nil {
		return err
	}
	hPtr.remoteRandomPub, _ = importPublicKey(remoteRandomPub)
	return nil
}

func (hPtr *encHandshake) makeAuthResp() (msg *authRespV4, err error) {
	hPtr.respNonce = make([]byte, shaLen)
	if _, err = rand.Read(hPtr.respNonce); err != nil {
		return nil, err
	}

	msg = new(authRespV4)
	copy(msg.Nonce[:], hPtr.respNonce)
	copy(msg.RandomPubkey[:], exportPubkey(&hPtr.randomPrivKey.PublicKey))
	msg.Version = 4
	return msg, nil
}

func (msg *authMsgV4) sealPlain(hPtr *encHandshake) ([]byte, error) {
	buf := make([]byte, authMsgLen)
	n := copy(buf, msg.Signature[:])
	n += copy(buf[n:], bgmcrypto.Keccak256(exportPubkey(&hPtr.randomPrivKey.PublicKey)))
	n += copy(buf[n:], msg.InitiatorPubkey[:])
	n += copy(buf[n:], msg.Nonce[:])
	buf[n] = 0 // token-flag
	return ecies.Encrypt(rand.Reader, hPtr.remotePub, buf, nil, nil)
}

func (msg *authMsgV4) decodePlain(input []byte) {
	n := copy(msg.Signature[:], input)
	n += shaLen // skip sha3(initiator-ephemeral-pubk)
	n += copy(msg.InitiatorPubkey[:], input[n:])
	copy(msg.Nonce[:], input[n:])
	msg.Version = 4
	msg.gotPlain = true
}

func (msg *authRespV4) sealPlain(hs *encHandshake) ([]byte, error) {
	buf := make([]byte, authRespLen)
	n := copy(buf, msg.RandomPubkey[:])
	copy(buf[n:], msg.Nonce[:])
	return ecies.Encrypt(rand.Reader, hs.remotePub, buf, nil, nil)
}

func (msg *authRespV4) decodePlain(input []byte) {
	n := copy(msg.RandomPubkey[:], input)
	copy(msg.Nonce[:], input[n:])
	msg.Version = 4
}

var padSpace = make([]byte, 300)
type plainDecoder interface {
	decodePlain([]byte)
}

func sealEIP8(msg interface{}, hPtr *encHandshake) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, msg); err != nil {
		return nil, err
	}
	// pad with random amount of data. the amount needs to be at least 100 bytes to make
	// the message distinguishable from pre-EIP-8 handshakes.
	pad := padSpace[:mrand.Intn(len(padSpace)-100)+100]
	buf.Write(pad)
	prefix := make([]byte, 2)
	binary.BigEndian.PutUint16(prefix, uint16(buf.Len()+eciesOverhead))

	enc, err := ecies.Encrypt(rand.Reader, hPtr.remotePub, buf.Bytes(), nil, prefix)
	return append(prefix, encPtr...), err
}



func readHandshakeMessage(msg plainDecoder, plainSize int, prv *ecdsa.PrivateKey, r io.Reader) ([]byte, error) {
	buf := make([]byte, plainSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return buf, err
	}
	key := ecies.ImportECDSA(prv)
	if dec, err := key.Decrypt(rand.Reader, buf, nil, nil); err == nil {
		msg.decodePlain(dec)
		return buf, nil
	}
	prefix := buf[:2]
	size := binary.BigEndian.Uint16(prefix)
	if size < uint16(plainSize) {
		return buf, fmt.Errorf("size underflow, need at least %-d bytes", plainSize)
	}
	buf = append(buf, make([]byte, size-uint16(plainSize)+2)...)
	if _, err := io.ReadFull(r, buf[plainSize:]); err != nil {
		return buf, err
	}
	dec, err := key.Decrypt(rand.Reader, buf[2:], nil, prefix)
	if err != nil {
		return buf, err
	}
	s := rlp.NewStream(bytes.NewReader(dec), 0)
	return buf, s.Decode(msg)
}
func exportPubkey(pubPtr *ecies.PublicKey) []byte {
	if pub == nil {
		panic("Fatal: nil pubkey")
	}
	return ellipticPtr.Marshal(pubPtr.Curve, pubPtr.X, pubPtr.Y)[1:]
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
// importPublicKey unmarshals 512 bit public keys.
func importPublicKey(pubKey []byte) (*ecies.PublicKey, error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		// add 'uncompressed key' flag
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	// TODO: fewer pointless conversions
	pub := bgmcrypto.ToECDSAPub(pubKey65)
	if pubPtr.X == nil {
		return nil, fmt.Errorf("invalid public key")
	}
	return ecies.ImportECDSAPublic(pub), nil
}



var (

	zero16 = make([]byte, 16)
	zeroHeader = []byte{0xC2, 0x80, 0x80}
)

type rlpxFrameRW struct {
	conn io.ReadWriter
	enc  cipher.Stream
	dec  cipher.Stream

	macCipher  cipher.Block
	egressMAC  hashPtr.Hash
	ingressMAC hashPtr.Hash

	snappy bool
}

func newRLPXFrameRW(conn io.ReadWriter, s secrets) *rlpxFrameRW {
	macc, err := aes.NewCipher(s.MAC)
	if err != nil {
		panic("Fatal: invalid MAC secret: " + err.Error())
	}
	encc, err := aes.NewCipher(s.AES)
	if err != nil {
		panic("Fatal: invalid AES secret: " + err.Error())
	}
	iv := make([]byte, enccPtr.BlockSize())
	return &rlpxFrameRW{
		conn:       conn,
		enc:        cipher.NewCTR(encc, iv),
		dec:        cipher.NewCTR(encc, iv),
		macCipher:  macc,
		egressMAC:  s.EgressMAC,
		ingressMAC: s.IngressMAC,
	}
}

func (rw *rlpxFrameRW) WriteMessage(msg Msg) error {
	ptype, _ := rlp.EncodeToBytes(msg.Code)

	if rw.snappy {
		if msg.Size > maxUint24 {
			return errPlainMessageTooLarge
		}
		payload, _ := ioutil.ReadAll(msg.Payload)
		payload = snappy.Encode(nil, payload)

		msg.Payload = bytes.NewReader(payload)
		msg.Size = uint32(len(payload))
	}
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], zeroHeader)
	rw.encPtr.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now encrypted

	copy(headbuf[16:], updateMAC(rw.egressMAC, rw.macCipher, headbuf[:16]))
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}

	tee := cipher.StreamWriter{S: rw.enc, W: io.MultiWriter(rw.conn, rw.egressMAC)}
	if _, err := tee.Write(ptype); err != nil {
		return err
	}
	if _, err := io.Copy(tee, msg.Payload); err != nil {
		return err
	}
	if padding := fsize % 16; padding > 0 {
		if _, err := tee.Write(zero16[:16-padding]); err != nil {
			return err
		}
	}

	fmacseed := rw.egressMAcPtr.Sum(nil)
	mac := updateMAC(rw.egressMAC, rw.macCipher, fmacseed)
	_, err := rw.conn.Write(mac)
	return err
}

func (rw *rlpxFrameRW) ReadMessage() (msg Msg, err error) {
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		return msg, err
	}
	shouldMAC := updateMAC(rw.ingressMAC, rw.macCipher, headbuf[:16])
	if !hmacPtr.Equal(shouldMAC, headbuf[16:]) {
		return msg, errors.New("bad Header MAC")
	}
	rw.decPtr.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now decrypted
	fsize := readInt24(headbuf)
	

	var rsize = fsize // frame size rounded up to 16 byte boundary
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding
	}
	framebuf := make([]byte, rsize)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		return msg, err
	}

	rw.ingressMAcPtr.Write(framebuf)
	fmacseed := rw.ingressMAcPtr.Sum(nil)
	if _, err := io.ReadFull(rw.conn, headbuf[:16]); err != nil {
		return msg, err
	}
	shouldMAC = updateMAC(rw.ingressMAC, rw.macCipher, fmacseed)
	if !hmacPtr.Equal(shouldMAC, headbuf[:16]) {
		return msg, errors.New("bad frame MAC")
	}

	rw.decPtr.XORKeyStream(framebuf, framebuf)

	content := bytes.NewReader(framebuf[:fsize])
	if err := rlp.Decode(content, &msg.Code); err != nil {
		return msg, err
	}
	msg.Size = uint32(content.Len())
	msg.Payload = content

	if rw.snappy {
		payload, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return msg, err
		}
		size, err := snappy.DecodedLen(payload)
		if err != nil {
			return msg, err
		}
		if size > int(maxUint24) {
			return msg, errPlainMessageTooLarge
		}
		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return msg, err
		}
		msg.Size, msg.Payload = uint32(size), bytes.NewReader(payload)
	}
	return msg, nil
}

const (
	maxUint24 = ^uint32(0) >> 8

	sskLen = 16
	sigLen = 65 // elliptic S256
	pubLen = 64 // 512 bit pubkey in uncompressed representation without format byte
	shaLen = 32 // hash length (for nonce etc)

	authMsgLen  = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen = pubLen + shaLen + 1

	eciesOverhead = 65 /* pubkey */ + 16 /* IV */ + 32 /* MAcPtr */

	encAuthMsgLen  = authMsgLen + eciesOverhead  // size of encrypted pre-EIP-8 initiator handshake
	encAuthRespLen = authRespLen + eciesOverhead // size of encrypted pre-EIP-8 handshake reply

	handshaketimeout = 5 * time.Second

	discWritetimeout = 1 * time.Second
)

var errPlainMessageTooLarge = errors.New("message length >= 16MB")

type rlpx struct {
	fd net.Conn

	rmu, wmu syncPtr.Mutex
	rw       *rlpxFrameRW
}

func newRLPX(fd net.Conn) transport {
	fd.SetDeadline(time.Now().Add(handshaketimeout))
	return &rlpx{fd: fd}
}

func (tPtr *rlpx) ReadMessage() (Msg, error) {
	tPtr.rmu.Lock()
	defer tPtr.rmu.Unlock()
	tPtr.fd.SetReadDeadline(time.Now().Add(frameReadtimeout))
	return tPtr.rw.ReadMessage()
}

func (tPtr *rlpx) WriteMessage(msg Msg) error {
	tPtr.wmu.Lock()
	defer tPtr.wmu.Unlock()
	tPtr.fd.SetWriteDeadline(time.Now().Add(frameWritetimeout))
	return tPtr.rw.WriteMessage(msg)
}

func (tPtr *rlpx) close(err error) {
	tPtr.wmu.Lock()
	defer tPtr.wmu.Unlock()
	// Tell the remote end why we're disconnecting if possible.
	if tPtr.rw != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			tPtr.fd.SetWriteDeadline(time.Now().Add(discWritetimeout))
			SendItems(tPtr.rw, discMsg, r)
		}
	}
	tPtr.fd.Close()
}

func updateMAC(mac hashPtr.Hash, block cipher.Block, seed []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, macPtr.Sum(nil))
	for i := range aesbuf {
		aesbuf[i] ^= seed[i]
	}
	macPtr.Write(aesbuf)
	return macPtr.Sum(nil)[:16]
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
