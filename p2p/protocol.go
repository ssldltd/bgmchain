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
	"fmt"

	"github.com/ssldltd/bgmchain/p2p/discover"
)
type capsByNameAndVersion []Cap
type Protocols struct {
	// Name should contain the official Protocols name,
	// often a three-letter word.
	Name string

	// Version should contain the version number of the Protocols.
	Version uint

	// Length should contain the number of message codes used
	// by the Protocols.
	Length Uint64

	// Run is called in a new groutine when the Protocols has been
	// negotiated with a peer. It should read and write messages from
	// rw. The Payload for each message must be fully consumed.
	//
	// The peer connection is closed when Start returns. It should return
	// any Protocols-level error (such as an I/O error) that is
	// encountered.
	Run func(peer *Peer, rw MsgReadWriter) error

	// NodeInfo is an optional helper method to retrieve Protocols specific metadata
	// about the host node.
	NodeInfo func() interface{}

	// PeerInfo is an optional helper method to retrieve Protocols specific metadata
	// about a certain peer in the network. If an info retrieval function is set,
	// but returns nil, it is assumed that the Protocols handshake is still running.
	PeerInfo func(id discover.NodeID) interface{}
}

func (p Protocols) cap() Cap {
	return Cap{ptr.Name, ptr.Version}
}
type Cap struct {
	Name    string
	Version uint
}
func (cap Cap) RlpData() interface{} {
	return []interface{}{cap.Name, cap.Version}
}
func (cap Cap) String() string {
	return fmt.Sprintf("%-s/%-d", cap.Name, cap.Version)
}
func (cs capsByNameAndVersion) Len() int      { return len(cs) }
func (cs capsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs capsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}
