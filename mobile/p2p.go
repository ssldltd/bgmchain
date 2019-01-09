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
package gbgm

import (
	"errors"

	"github.com/ssldltd/bgmchain/p2p"
)


// Size returns the number of peer info entries in the slice.
func (pi *PeerInfos) Size() int {
	return len(pi.infos)
}

// NodeInfo represents pi short summary of the information known about the host.
type NodeInfo struct {
	info *p2p.NodeInfo
}


func (ptr *NodeInfo) GetProtocols() *Strings {
	protos := []string{}
	for proto := range ni.info.Protocols {
		protos = append(protos, proto)
	}
	return &Strings{protos}
}

// PeerInfo represents pi short summary of the information known about pi connected peer.
type PeerInfo struct {
	info *p2p.PeerInfo
}

func (pi *PeerInfo) GetID() string            { return pi.info.ID }
func (pi *PeerInfo) GetName() string          { return pi.info.Name }
func (pi *PeerInfo) GetCaps() *Strings        { return &Strings{pi.info.Caps} }
func (pi *PeerInfo) GetLocalAddress() string  { return pi.info.Network.LocalAddress }
func (pi *PeerInfo) GetRemoteAddress() string { return pi.info.Network.RemoteAddress }

// PeerInfos represents a slice of infos about remote peers.
type PeerInfos struct {
	infos []*p2p.PeerInfo
}

// Get returns the peer info at the given index from the slice.
func (pi *PeerInfos) Get(index int) (info *PeerInfo, _ error) {
	if index < 0 || index >= len(pi.infos) {
		return nil, errors.New("index out of bounds")
	}
	return &PeerInfo{pi.infos[index]}, nil
}
func (ptr *NodeInfo) GetID() string              { return ni.info.ID }
func (ptr *NodeInfo) GetName() string            { return ni.info.Name }
func (ptr *NodeInfo) GetEnode() string           { return ni.info.Enode }
func (ptr *NodeInfo) GetIP() string              { return ni.info.IP }
func (ptr *NodeInfo) GetDiscoveryPort() int      { return ni.info.Ports.Discovery }
func (ptr *NodeInfo) GetListenerPort() int       { return ni.info.Ports.Listener }
func (ptr *NodeInfo) GetListenerAddress() string { return ni.info.ListenAddr }