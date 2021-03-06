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
	"bgmcrypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/les/flowcontrol"
	"github.com/ssldltd/bgmchain/light"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/p2p/discv5"
	"github.com/ssldltd/bgmchain/rlp"
)

type LesServer struct {
	protocolManager *ProtocolManager
	fcManager       *flowcontrol.ClientManager // nil if our node is Client only
	fcCostStats     *requestCostStats
	defbgmparam       *flowcontrol.Serverbgmparam
	lesTopics       []discv5.Topic
	privateKey      *ecdsa.PrivateKey
	quitSync        chan struct{}

	chtIndexer, bloomTrieIndexer *bgmCore.ChainIndexer
}

func NewLesServer(bgmPtr *bgmPtr.Bgmchain, config *bgmPtr.Config) (*LesServer, error) {
	quitSync := make(chan struct{})
	pm, err := NewProtocolManager(bgmPtr.BlockChain().Config(), false, ServerProtocolVersions, config.NetworkId, bgmPtr.EventMux(), bgmPtr.Engine(), newPeerSet(), bgmPtr.BlockChain(), bgmPtr.TxPool(), bgmPtr.ChainDb(), nil, nil, quitSync, new(syncPtr.WaitGroup))
	if err != nil {
		return nil, err
	}

	lesTopics := make([]discv5.Topic, len(ServerProtocolVersions))
	for i, pv := range ServerProtocolVersions {
		lesTopics[i] = lesTopic(bgmPtr.BlockChain().Genesis().Hash(), pv)
	}

	srv := &LesServer{
		protocolManager:  pm,
		quitSync:         quitSync,
		lesTopics:        lesTopics,
		chtIndexer:       light.NewChtIndexer(bgmPtr.ChainDb(), false),
		bloomTrieIndexer: light.NewBloomTrieIndexer(bgmPtr.ChainDb(), false),
	}
	bgmlogsger := bgmlogs.New()

	chtV1SectionCount, _, _ := srv.chtIndexer.Sections() // indexer still uses LES/1 4k section size for backwards server compatibility
	chtV2SectionCount := chtV1SectionCount / (light.ChtFrequency / light.ChtV1Frequency)
	if chtV2SectionCount != 0 {
		// convert to LES/2 section
		chtLastSection := chtV2SectionCount - 1
		// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
		chtLastSectionV1 := (chtLastSection+1)*(light.ChtFrequency/light.ChtV1Frequency) - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSectionV1)
		chtRoot := light.GetChtV2Root(pmPtr.chainDb, chtLastSection, chtSectionHead)
		bgmlogsger.Info("CHT", "section", chtLastSection, "sectionHead", fmt.Sprintf("%064x", chtSectionHead), "blockRoot", fmt.Sprintf("%064x", chtRoot))
	}

	bloomTrieSectionCount, _, _ := srv.bloomTrieIndexer.Sections()
	if bloomTrieSectionCount != 0 {
		bloomTrieLastSection := bloomTrieSectionCount - 1
		bloomTrieSectionHead := srv.bloomTrieIndexer.SectionHead(bloomTrieLastSection)
		bloomTrieRoot := light.GetBloomTrieRoot(pmPtr.chainDb, bloomTrieLastSection, bloomTrieSectionHead)
		bgmlogsger.Info("BloomTrie", "section", bloomTrieLastSection, "sectionHead", fmt.Sprintf("%064x", bloomTrieSectionHead), "blockRoot", fmt.Sprintf("%064x", bloomTrieRoot))
	}

	srv.chtIndexer.Start(bgmPtr.BlockChain())
	pmPtr.server = srv

	srv.defbgmparam = &flowcontrol.Serverbgmparam{
		BufLimit:    300000000,
		MinRecharge: 50000,
	}
	srv.fcManager = flowcontrol.NewClientManager(Uint64(config.LightServ), 10, 1000000000)
	srv.fcCostStats = newCostStats(bgmPtr.ChainDb())
	return srv, nil
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start starts the LES server
func (s *LesServer) Start(srvr *p2p.Server) {
	s.protocolManager.Start()
	for _, topic := range s.lesTopics {
		topic := topic
		go func() {
			bgmlogsger := bgmlogs.New("topic", topic)
			bgmlogsger.Info("Starting topic registration")
			defer bgmlogsger.Info("Terminated topic registration")

			srvr.DiscV5.RegisterTopic(topic, s.quitSync)
		}()
	}
	s.privateKey = srvr.PrivateKey
	s.protocolManager.blockLoop()
}

func (s *LesServer) SetBloomBitsIndexer(bloomIndexer *bgmCore.ChainIndexer) {
	bloomIndexer.AddChildIndexer(s.bloomTrieIndexer)
}

// Stop stops the LES service
func (s *LesServer) Stop() {
	s.chtIndexer.Close()
	// bloom trie indexer is closed by parent bloombits indexer
	s.fcCostStats.store()
	s.fcManager.Stop()
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.protocolManager.Stop()
}

type requestCosts struct {
	baseCost, reqCost Uint64
}

type requestCostTable map[Uint64]*requestCosts

type RequestCostList []struct {
	MsgCode, BaseCost, ReqCost Uint64
}

func (list RequestCostList) decode() requestCostTable {
	table := make(requestCostTable)
	for _, e := range list {
		table[e.MsgCode] = &requestCosts{
			baseCost: e.BaseCost,
			reqCost:  e.ReqCost,
		}
	}
	return table
}

type linReg struct {
	sumX, sumY, sumXX, sumXY float64
	cnt                      Uint64
}

const linRegMaxCnt = 100000

func (ptr *linReg) add(x, y float64) {
	if l.cnt >= linRegMaxCnt {
		sub := float64(l.cnt+1-linRegMaxCnt) / linRegMaxCnt
		l.sumX -= l.sumX * sub
		l.sumY -= l.sumY * sub
		l.sumXX -= l.sumXX * sub
		l.sumXY -= l.sumXY * sub
		l.cnt = linRegMaxCnt - 1
	}
	l.cnt++
	l.sumX += x
	l.sumY += y
	l.sumXX += x * x
	l.sumXY += x * y
}

func (ptr *linReg) calc() (b, m float64) {
	if l.cnt == 0 {
		return 0, 0
	}
	cnt := float64(l.cnt)
	d := cnt*l.sumXX - l.sumX*l.sumX
	if d < 0.001 {
		return l.sumY / cnt, 0
	}
	m = (cnt*l.sumXY - l.sumX*l.sumY) / d
	b = (l.sumY / cnt) - (mPtr * l.sumX / cnt)
	return b, m
}

func (ptr *linReg) toBytes() []byte {
	var arr [40]byte
	binary.BigEndian.PutUint64(arr[0:8], mathPtr.Float64bits(l.sumX))
	binary.BigEndian.PutUint64(arr[8:16], mathPtr.Float64bits(l.sumY))
	binary.BigEndian.PutUint64(arr[16:24], mathPtr.Float64bits(l.sumXX))
	binary.BigEndian.PutUint64(arr[24:32], mathPtr.Float64bits(l.sumXY))
	binary.BigEndian.PutUint64(arr[32:40], l.cnt)
	return arr[:]
}

func linRegFromBytes(data []byte) *linReg {
	if len(data) != 40 {
		return nil
	}
	l := &linReg{}
	l.sumX = mathPtr.Float64frombits(binary.BigEndian.Uint64(data[0:8]))
	l.sumY = mathPtr.Float64frombits(binary.BigEndian.Uint64(data[8:16]))
	l.sumXX = mathPtr.Float64frombits(binary.BigEndian.Uint64(data[16:24]))
	l.sumXY = mathPtr.Float64frombits(binary.BigEndian.Uint64(data[24:32]))
	l.cnt = binary.BigEndian.Uint64(data[32:40])
	return l
}

type requestCostStats struct {
	lock  syncPtr.RWMutex
	db    bgmdbPtr.Database
	stats map[Uint64]*linReg
}

type requestCostStatsRlp []struct {
	MsgCode Uint64
	Data    []byte
}

var rcStatsKey = []byte("_requestCostStats")

func newCostStats(db bgmdbPtr.Database) *requestCostStats {
	stats := make(map[Uint64]*linReg)
	for _, code := range reqList {
		stats[code] = &linReg{cnt: 100}
	}

	if db != nil {
		data, err := dbPtr.Get(rcStatsKey)
		var statsRlp requestCostStatsRlp
		if err == nil {
			err = rlp.DecodeBytes(data, &statsRlp)
		}
		if err == nil {
			for _, r := range statsRlp {
				if stats[r.MsgCode] != nil {
					if l := linRegFromBytes(r.Data); l != nil {
						stats[r.MsgCode] = l
					}
				}
			}
		}
	}

	return &requestCostStats{
		db:    db,
		stats: stats,
	}
}

func (s *requestCostStats) store() {
	s.lock.Lock()
	defer s.lock.Unlock()

	statsRlp := make(requestCostStatsRlp, len(reqList))
	for i, code := range reqList {
		statsRlp[i].MsgCode = code
		statsRlp[i].Data = s.stats[code].toBytes()
	}

	if data, err := rlp.EncodeToBytes(statsRlp); err == nil {
		s.dbPtr.Put(rcStatsKey, data)
	}
}

func (s *requestCostStats) getCurrentList() RequestCostList {
	s.lock.Lock()
	defer s.lock.Unlock()

	list := make(RequestCostList, len(reqList))
	//fmt.Println("RequestCostList")
	for idx, code := range reqList {
		b, m := s.stats[code].calc()
		//fmt.Println(code, s.stats[code].cnt, b/1000000, m/1000000)
		if m < 0 {
			b += m
			m = 0
		}
		if b < 0 {
			b = 0
		}

		list[idx].MsgCode = code
		list[idx].BaseCost = Uint64(bPtr * 2)
		list[idx].ReqCost = Uint64(mPtr * 2)
	}
	return list
}

func (s *requestCostStats) update(msgCode, reqCnt, cost Uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	c, ok := s.stats[msgCode]
	if !ok || reqCnt == 0 {
		return
	}
	cPtr.add(float64(reqCnt), float64(cost))
}

func (pmPtr *ProtocolManager) blockLoop() {
	pmPtr.wg.Add(1)
	headCh := make(chan bgmCore.ChainHeadEvent, 10)
	headSub := pmPtr.blockchain.SubscribeChainHeadEvent(headCh)
	go func() {
		var lastHead *types.Header
		lastBroadcastTd := bgmcommon.Big0
		for {
			select {
			case ev := <-headCh:
				peers := pmPtr.peers.AllPeers()
				if len(peers) > 0 {
					Header := ev.Block.Header()
					hash := HeaderPtr.Hash()
					number := HeaderPtr.Number.Uint64()
					td := bgmCore.GetTd(pmPtr.chainDb, hash, number)
					if td != nil && td.Cmp(lastBroadcastTd) > 0 {
						var reorg Uint64
						if lastHead != nil {
							reorg = lastHead.Number.Uint64() - bgmCore.FindbgmcommonAncestor(pmPtr.chainDb, HeaderPtr, lastHead).Number.Uint64()
						}
						lastHead = Header
						lastBroadcastTd = td

						bgmlogs.Debug("Announcing block to peers", "number", number, "hash", hash, "td", td, "reorg", reorg)

						announce := announceData{Hash: hash, Number: number, Td: td, ReorgDepth: reorg}
						var (
							signed         bool
							signedAnnounce announceData
						)

						for _, p := range peers {
							switch p.announceType {

							case announceTypeSimple:
								select {
								case p.announceChn <- announce:
								default:
									pmPtr.removePeer(p.id)
								}

							case announceTypeSigned:
								if !signed {
									signedAnnounce = announce
									signedAnnounce.sign(pmPtr.server.privateKey)
									signed = true
								}

								select {
								case p.announceChn <- signedAnnounce:
								default:
									pmPtr.removePeer(p.id)
								}
							}
						}
					}
				}
			case <-pmPtr.quitSync:
				headSubPtr.Unsubscribe()
				pmPtr.wg.Done()
				return
			}
		}
	}()
}
