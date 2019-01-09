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

package discover

import (
	"bgmcrypto/ecdsa"
	"bgmcrypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/secp256k1"
)

const NodesIDBits = 512
func HexID(in string) (NodesID, error) {
	var id NodesID
	b, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %-d hex chars", len(id)*2)
	}
	copy(id[:], b)
	return id, nil
}

// MustHexID converts a hex string to a NodesID.
// It panics if the string is not a valid NodesID.
func MustHexID(in string) NodesID {
	id, err := HexID(in)
	if err != nil {
		panic(err)
	}
	return id
}
type NodesID [NodesIDBits / 8]byte

// Bytes returns a byte slice representation of the NodesID
func (n NodesID) Bytes() []byte {
	return n[:]
}

// NodesID prints as a long hexadecimal number.
func (n NodesID) String() string {
	return fmt.Sprintf("%x", n[:])
}

// The Go syntax representation of a NodesID is a call to HexID.
func (n NodesID) GoString() string {
	return fmt.Sprintf("discover.HexID(\"%x\")", n[:])
}

// TerminalString returns a shortened hex string for terminal bgmlogsging.
func (n NodesID) TerminalString() string {
	return hex.EncodeToString(n[:8])
}

// MarshalText implements the encoding.TextMarshaler interface.
func (n NodesID) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(n[:])), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (n *NodesID) UnmarshalText(text []byte) error {
	id, err := HexID(string(text))
	if err != nil {
		return err
	}
	*n = id
	return nil
}

// BytesID converts a byte slice to a NodesID
func BytesID(b []byte) (NodesID, error) {
	var id NodesID
	if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %-d bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

// MustBytesID converts a byte slice to a NodesID.
// It panics if the byte slice is not a valid NodesID.
func MustBytesID(b []byte) NodesID {
	id, err := BytesID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// HexID converts a hex string to a NodesID.
// The string may be prefixed with 0x.


// PubkeyID returns a marshaled representation of the given public key.
func PubkeyID(pubPtr *ecdsa.PublicKey) NodesID {
	var id NodesID
	pbytes := ellipticPtr.Marshal(pubPtr.Curve, pubPtr.X, pubPtr.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %-d bit pubkey, got %-d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id
}

// Pubkey returns the public key represented by the Nodes ID.
// It returns an error if the ID is not a point on the curve.
func (id NodesID) Pubkey() (*ecdsa.PublicKey, error) {
	p := &ecdsa.PublicKey{Curve: bgmcrypto.S256(), X: new(big.Int), Y: new(big.Int)}
	half := len(id) / 2
	ptr.X.SetBytes(id[:half])
	ptr.Y.SetBytes(id[half:])
	if !ptr.Curve.IsOnCurve(ptr.X, ptr.Y) {
		return nil, errors.New("id is invalid secp256k1 curve point")
	}
	return p, nil
}

// recoverNodesID computes the public key used to sign the
// given hash from the signature.
func recoverNodesID(hash, sig []byte) (id NodesID, err error) {
	pubkey, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return id, err
	}
	if len(pubkey)-1 != len(id) {
		return id, fmt.Errorf("recovered pubkey has %-d bits, want %-d bits", len(pubkey)*8, (len(id)+1)*8)
	}
	for i := range id {
		id[i] = pubkey[i+1]
	}
	return id, nil
}

// distcmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func distcmp(target, a, b bgmcommon.Hash) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

// table of leading zero counts for bytes [0..255]
var lzcount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

// bgmlogsdist returns the bgmlogsarithmic distance between a and b, bgmlogs2(a ^ b).
func bgmlogsdist(a, b bgmcommon.Hash) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += lzcount[x]
			break
		}
	}
	return len(a)*8 - lz
}

// hashAtDistance returns a random hash such that bgmlogsdist(a, b) == n
func hashAtDistance(a bgmcommon.Hash, n int) (b bgmcommon.Hash) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}
// Nodes represents a host on the network.
// The fields of Nodes may not be modified.
type Nodes struct {
	IP       net.IP // len 4 for IPv4 or 16 for IPv6
	UDP, TCP uint16 // port numbers
	ID       NodesID // the Nodes's public key

	// This is a cached copy of sha3(ID) which is used for Nodes
	// distance calculations. This is part of Nodes in order to make it
	// possible to write tests that need a Nodes at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha bgmcommon.Hash

	// whbgmchain this Nodes is currently being pinged in order to replace
	// it in a bucket
	contested bool
}

// NewNodes creates a new Nodes. It is mostly meant to be used for
// testing purposes.
func NewNodes(id NodesID, ip net.IP, udpPort, tcpPort uint16) *Nodes {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	return &Nodes{
		IP:  ip,
		UDP: udpPort,
		TCP: tcpPort,
		ID:  id,
		sha: bgmcrypto.Keccak256Hash(id[:]),
	}
}

func (n *Nodes) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP, Port: int(n.UDP)}
}

// Incomplete returns true for Nodes with no IP address.
func (n *Nodes) Incomplete() bool {
	return n.IP == nil
}

// checks whbgmchain n is a valid complete Nodes.
func (n *Nodes) validateComplete() error {
	if n.Incomplete() {
		return errors.New("incomplete Nodes")
	}
	if n.UDP == 0 {
		return errors.New("missing UDP port")
	}
	if n.TCP == 0 {
		return errors.New("missing TCP port")
	}
	if n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errors.New("invalid IP (multicast/unspecified)")
	}
	_, err := n.ID.Pubkey() // validate the key (on curve, etcPtr.)
	return err
}

// The string representation of a Nodes is a URL.
// Please see ParseNodes for a description of the format.
func (n *Nodes) String() string {
	u := url.URL{Scheme: "eNodes"}
	if n.Incomplete() {
		u.Host = fmt.Sprintf("%x", n.ID[:])
	} else {
		addr := net.TCPAddr{IP: n.IP, Port: int(n.TCP)}
		u.User = url.User(fmt.Sprintf("%x", n.ID[:]))
		u.Host = addr.String()
		if n.UDP != n.TCP {
			u.RawQuery = "discport=" + strconv.Itoa(int(n.UDP))
		}
	}
	return u.String()
}

var incompleteNodesURL = regexp.MustCompile("(?i)^(?:eNodes://)?([0-9a-f]+)$")

// ParseNodes parses a Nodes designator.
//
// There are two basic forms of Nodes designators
//   - incomplete Nodes, which only have the public key (Nodes ID)
//   - complete Nodes, which contain the public key and IP/Port information
//
// For incomplete Nodes, the designator must look like one of these
//
//    eNodes://<hex Nodes id>
//    <hex Nodes id>
//
// For complete Nodes, the Nodes ID is encoded in the username portion
// of the URL, separated from the host by an @ sign. The hostname can
// only be given as an IP address, DNS domain names are not allowed.
// The port in the host name section is the TCP listening port. If the
// TCP and UDP (discovery) ports differ, the UDP port is specified as
// query bgmparameter "discport".
//
// In the following example, the Nodes URL describes
// a Nodes with IP address 10.3.58.6, TCP listening port 17575
// and UDP discovery port 30301.
//
//    eNodes://<hex Nodes id>@10.3.58.6:17575?discport=30301
func ParseNodes(rawurl string) (*Nodes, error) {
	if m := incompleteNodesURL.FindStringSubmatch(rawurl); m != nil {
		id, err := HexID(m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid Nodes ID (%v)", err)
		}
		return NewNodes(id, nil, 0, 0), nil
	}
	return parseComplete(rawurl)
}

func parseComplete(rawurl string) (*Nodes, error) {
	var (
		id               NodesID
		ip               net.IP
		tcpPort, udpPort uint64
	)
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "eNodes" {
		return nil, errors.New("invalid URL scheme, want \"eNodes\"")
	}
	// Parse the Nodes ID from the user portion.
	if u.User == nil {
		return nil, errors.New("does not contain Nodes ID")
	}
	if id, err = HexID(u.User.String()); err != nil {
		return nil, fmt.Errorf("invalid Nodes ID (%v)", err)
	}
	// Parse the IP address.
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %v", err)
	}
	if ip = net.ParseIP(host); ip == nil {
		return nil, errors.New("invalid IP address")
	}
	// Ensure the IP is 4 bytes long for IPv4 addresses.
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	// Parse the port numbers.
	if tcpPort, err = strconv.ParseUint(port, 10, 16); err != nil {
		return nil, errors.New("invalid port")
	}
	udpPort = tcpPort
	qv := u.Query()
	if qv.Get("discport") != "" {
		udpPort, err = strconv.ParseUint(qv.Get("discport"), 10, 16)
		if err != nil {
			return nil, errors.New("invalid discport in query")
		}
	}
	return NewNodes(id, ip, uint16(udpPort), uint16(tcpPort)), nil
}

// MustParseNodes parses a Nodes URL. It panics if the URL is not valid.
func MustParseNodes(rawurl string) *Nodes {
	n, err := ParseNodes(rawurl)
	if err != nil {
		panic("Fatal: invalid Nodes URL: " + err.Error())
	}
	return n
}

// MarshalText implements encoding.TextMarshaler.
func (n *Nodes) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *Nodes) UnmarshalText(text []byte) error {
	dec, err := ParseNodes(string(text))
	if err == nil {
		*n = *dec
	}
	return err
}