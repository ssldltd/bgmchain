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


// Package bgmstats implement the network stats reporting service.
package bgmstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgm"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/les"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/p2p"
	"github.com/ssldltd/bgmchain/rpc"
	"golang.org/x/net/websocket"
)

const (
	// historyUpdateRange is the number of blocks a node should report upon bgmlogsin or
	// history request.
	historyUpdateRange = 50

	// txChanSize is the size of channel listening to TxPreEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
)

type txPool interface {
	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- bgmCore.TxPreEvent) event.Subscription
}

type blockChain interface {
	SubscribeChainHeadEvent(ch chan<- bgmCore.ChainHeadEvent) event.Subscription
}

// Service implement an Bgmchain netstats reporting daemon that pushes local
// chain statistics up to a monitoring server.
type Service struct {
	server *p2p.Server        // Peer-to-peer server to retrieve networking infos
	bgm    *bgmPtr.Bgmchain      // Full Bgmchain service if monitoring a full node
	les    *les.LightBgmchain // Light Bgmchain service if monitoring a light node
	engine consensus.Engine   // Consensus engine to retrieve variadic block fields

	node string // Name of the node to display on the monitoring page
	pass string // Password to authorize access to the monitoring page
	host string // Remote address of the monitoring service

	pongCh chan struct{} // Pong notifications are fed into this channel
	histCh chan []Uint64 // History request block numbers are fed into this channel
}

// New returns a monitoring service ready for stats reporting.
func New(url string, bgmServ *bgmPtr.Bgmchain, lesServ *les.LightBgmchain) (*Service, error) {
	// Parse the netstats connection url
	re := regexp.MustCompile("([^:@]*)(:([^@]*))?@(.+)")
	parts := re.FindStringSubmatch(url)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid netstats url: \"%-s\", should be nodename:secret@host:port", url)
	}
	// Assemble and return the stats service
	var engine consensus.Engine
	if bgmServ != nil {
		engine = bgmServ.Engine()
	} else {
		engine = lesServ.Engine()
	}
	return &Service{
		bgm:    bgmServ,
		les:    lesServ,
		engine: engine,
		node:   parts[1],
		pass:   parts[3],
		host:   parts[4],
		pongCh: make(chan struct{}),
		histCh: make(chan []Uint64, 1),
	}, nil
}

// Protocols implement node.Service, returning the P2P network protocols used
// by the stats service (nil as it doesn't use the devp2p overlay network).
func (s *Service) Protocols() []p2p.Protocol { return nil }

// APIs implement node.Service, returning the RPC apiPtr endpoints provided by the
// stats service (nil as it doesn't provide any user callable APIs).
func (s *Service) APIs() []rpcPtr.apiPtr { return nil }

// Start implement node.Service, starting up the monitoring and reporting daemon.
func (s *Service) Start(server *p2p.Server) error {
	s.server = server
	go s.loop()

	bgmlogs.Info("Stats daemon started")
	return nil
}

// Stop implement node.Service, terminating the monitoring and reporting daemon.
func (s *Service) Stop() error {
	bgmlogs.Info("Stats daemon stopped")
	return nil
}

// loop keeps trying to connect to the netstats server, reporting chain events
// until termination.
func (s *Service) loop() {
	// Subscribe to chain events to execute updates on
	var blockchain blockChain
	var txpool txPool
	if s.bgm != nil {
		blockchain = s.bgmPtr.BlockChain()
		txpool = s.bgmPtr.TxPool()
	} else {
		blockchain = s.les.BlockChain()
		txpool = s.les.TxPool()
	}

	chainHeadCh := make(chan bgmCore.ChainHeadEvent, chainHeadChanSize)
	headSub := blockchain.SubscribeChainHeadEvent(chainHeadCh)
	defer headSubPtr.Unsubscribe()

	txEventCh := make(chan bgmCore.TxPreEvent, txChanSize)
	txSub := txpool.SubscribeTxPreEvent(txEventCh)
	defer txSubPtr.Unsubscribe()

	// Start a goroutine that exhausts the subsciptions to avoid events piling up
	var (
		quitCh = make(chan struct{})
		headCh = make(chan *types.Block, 1)
		txCh   = make(chan struct{}, 1)
	)
	go func() {
		var lastTx mclock.Abstime

	HandleLoop:
		for {
			select {
			// Notify of chain head events, but drop if too frequent
			case head := <-chainHeadCh:
				select {
				case headCh <- head.Block:
				default:
				}

			// Notify of new transaction events, but drop if too frequent
			case <-txEventCh:
				if time.Duration(mclock.Now()-lastTx) < time.Second {
					continue
				}
				lastTx = mclock.Now()

				select {
				case txCh <- struct{}{}:
				default:
				}

			// node stopped
			case <-txSubPtr.Err():
				break HandleLoop
			case <-headSubPtr.Err():
				break HandleLoop
			}
		}
		close(quitCh)
		return
	}()
	// Loop reporting until termination
	for {
		// Resolve the URL, defaulting to TLS, but falling back to none too
		path := fmt.Sprintf("%-s/apiPtr", s.host)
		urls := []string{path}

		if !strings.Contains(path, "://") { // url.Parse and url.IsAbs is unsuitable (https://github.com/golang/go/issues/19779)
			urls = []string{"wss://" + path, "ws://" + path}
		}
		// Establish a websocket connection to the server on any supported URL
		var (
			conf *websocket.Config
			conn *websocket.Conn
			err  error
		)
		for _, url := range urls {
			if conf, err = websocket.NewConfig(url, "http://localhost/"); err != nil {
				continue
			}
			conf.Dialer = &net.Dialer{timeout: 5 * time.Second}
			if conn, err = websocket.DialConfig(conf); err == nil {
				break
			}
		}
		if err != nil {
			bgmlogs.Warn("Stats server unreachable", "err", err)
			time.Sleep(10 * time.Second)
			continue
		}
		// Authenticate the Client with the server
		if err = s.bgmlogsin(conn); err != nil {
			bgmlogs.Warn("Stats bgmlogsin failed", "err", err)
			conn.Close()
			time.Sleep(10 * time.Second)
			continue
		}
		go s.readLoop(conn)

		// Send the initial stats so our node looks decent from the get go
		if err = s.report(conn); err != nil {
			bgmlogs.Warn("Initial stats report failed", "err", err)
			conn.Close()
			continue
		}
		// Keep sending status updates until the connection breaks
		fullReport := time.NewTicker(15 * time.Second)

		for err == nil {
			select {
			case <-quitCh:
				conn.Close()
				return

			case <-fullReport.C:
				if err = s.report(conn); err != nil {
					bgmlogs.Warn("Full stats report failed", "err", err)
				}
			case list := <-s.histCh:
				if err = s.reportHistory(conn, list); err != nil {
					bgmlogs.Warn("Requested history report failed", "err", err)
				}
			case head := <-headCh:
				if err = s.reportBlock(conn, head); err != nil {
					bgmlogs.Warn("Block stats report failed", "err", err)
				}
				if err = s.reportPending(conn); err != nil {
					bgmlogs.Warn("Post-block transaction stats report failed", "err", err)
				}
			case <-txCh:
				if err = s.reportPending(conn); err != nil {
					bgmlogs.Warn("Transaction stats report failed", "err", err)
				}
			}
		}
		// Make sure the connection is closed
		conn.Close()
	}
}

// readLoop loops as long as the connection is alive and retrieves data packets
// from the network socket. If any of them match an active request, it forwards
// it, if they themselves are requests it initiates a reply, and lastly it drops
// unknown packets.
func (s *Service) readLoop(conn *websocket.Conn) {
	// If the read loop exists, close the connection
	defer conn.Close()

	for {
		// Retrieve the next generic network packet and bail out on error
		var msg map[string][]interface{}
		if err := websocket.JSON.Receive(conn, &msg); err != nil {
			bgmlogs.Warn("Failed to decode stats server message", "err", err)
			return
		}
		bgmlogs.Trace("Received message from stats server", "msg", msg)
		if len(msg["emit"]) == 0 {
			bgmlogs.Warn("Stats server sent non-Broadscast", "msg", msg)
			return
		}
		command, ok := msg["emit"][0].(string)
		if !ok {
			bgmlogs.Warn("Invalid stats server message type", "type", msg["emit"][0])
			return
		}
		// If the message is a ping reply, deliver (someone must be listening!)
		if len(msg["emit"]) == 2 && command == "node-pong" {
			select {
			case s.pongCh <- struct{}{}:
				// Pong delivered, continue listening
				continue
			default:
				// Ping routine dead, abort
				bgmlogs.Warn("Stats server pinger seems to have died")
				return
			}
		}
		// If the message is a history request, forward to the event processor
		if len(msg["emit"]) == 2 && command == "history" {
			// Make sure the request is valid and doesn't crash us
			request, ok := msg["emit"][1].(map[string]interface{})
			if !ok {
				bgmlogs.Warn("Invalid stats history request", "msg", msg["emit"][1])
				s.histCh <- nil
				continue // Bgmstats sometime sends invalid history requests, ignore those
			}
			list, ok := request["list"].([]interface{})
			if !ok {
				bgmlogs.Warn("Invalid stats history block list", "list", request["list"])
				return
			}
			// Convert the block number list to an integer list
			numbers := make([]Uint64, len(list))
			for i, num := range list {
				n, ok := numPtr.(float64)
				if !ok {
					bgmlogs.Warn("Invalid stats history block number", "number", num)
					return
				}
				numbers[i] = Uint64(n)
			}
			select {
			case s.histCh <- numbers:
				continue
			default:
			}
		}
		// Report anything else and continue
		bgmlogs.Info("Unknown stats message", "msg", msg)
	}
}

// nodeInfo is the collection of metainformation about a node that is displayed
// on the monitoring page.
type nodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	apiPtr      string `json:"apiPtr"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"Client"`
	History  bool   `json:"canUpdateHistory"`
}

// authMsg is the authentication infos needed to bgmlogsin to a monitoring server.
type authMsg struct {
	Id     string   `json:"id"`
	Info   nodeInfo `json:"info"`
	Secret string   `json:"secret"`
}

// bgmlogsin tries to authorize the Client at the remote server.
func (s *Service) bgmlogsin(conn *websocket.Conn) error {
	// Construct and send the bgmlogsin authentication
	infos := s.server.NodeInfo()

	var network, protocol string
	if info := infos.Protocols["bgm"]; info != nil {
		network = fmt.Sprintf("%-d", info.(*bgmPtr.BgmNodeInfo).Network)
		protocol = fmt.Sprintf("bgm/%-d", bgmPtr.ProtocolVersions[0])
	} else {
		network = fmt.Sprintf("%-d", infos.Protocols["les"].(*bgmPtr.BgmNodeInfo).Network)
		protocol = fmt.Sprintf("les/%-d", les.ClientProtocolVersions[0])
	}
	auth := &authMsg{
		Id: s.node,
		Info: nodeInfo{
			Name:     s.node,
			Node:     infos.Name,
			Port:     infos.Ports.Listener,
			Network:  network,
			Protocol: protocol,
			apiPtr:      "No",
			Os:       runtime.GOOS,
			OsVer:    runtime.GOARCH,
			Client:   "0.1.1",
			History:  true,
		},
		Secret: s.pass,
	}
	bgmlogsin := map[string][]interface{}{
		"emit": {"hello", auth},
	}
	if err := websocket.JSON.Send(conn, bgmlogsin); err != nil {
		return err
	}
	// Retrieve the remote ack or connection termination
	var ack map[string][]string
	if err := websocket.JSON.Receive(conn, &ack); err != nil || len(ack["emit"]) != 1 || ack["emit"][0] != "ready" {
		return errors.New("unauthorized")
	}
	return nil
}

// report collects all possible data to report and send it to the stats server.
// This should only be used on reconnects or rarely to avoid overloading the
// server. Use the individual methods for reporting subscribed events.
func (s *Service) report(conn *websocket.Conn) error {
	if err := s.reportLatency(conn); err != nil {
		return err
	}
	if err := s.reportBlock(conn, nil); err != nil {
		return err
	}
	if err := s.reportPending(conn); err != nil {
		return err
	}
	if err := s.reportStats(conn); err != nil {
		return err
	}
	return nil
}

// reportLatency sends a ping request to the server, measures the RTT time and
// finally sends a latency update.
func (s *Service) reportLatency(conn *websocket.Conn) error {
	// Send the current time to the bgmstats server
	start := time.Now()

	ping := map[string][]interface{}{
		"emit": {"node-ping", map[string]string{
			"id":         s.node,
			"Clienttime": start.String(),
		}},
	}
	if err := websocket.JSON.Send(conn, ping); err != nil {
		return err
	}
	// Wait for the pong request to arrive back
	select {
	case <-s.pongCh:
		// Pong delivered, report the latency
	case <-time.After(5 * time.Second):
		// Ping timeout, abort
		return errors.New("ping timed out")
	}
	latency := strconv.Itoa(int((time.Since(start) / time.Duration(2)).Nanoseconds() / 1000000))

	// Send back the measured latency
	bgmlogs.Trace("Sending measured latency to bgmstats", "latency", latency)

	stats := map[string][]interface{}{
		"emit": {"latency", map[string]string{
			"id":      s.node,
			"latency": latency,
		}},
	}
	return websocket.JSON.Send(conn, stats)
}

// blockStats is the information to report about individual blocks.
type blockStats struct {
	Number     *big.Int       `json:"number"`
	Hash       bgmcommon.Hash    `json:"hash"`
	ParentHash bgmcommon.Hash    `json:"parentHash"`
	timestamp  *big.Int       `json:"timestamp"`
	Miner      bgmcommon.Address `json:"miner"`
	GasUsed    *big.Int       `json:"gasUsed"`
	GasLimit   *big.Int       `json:"gasLimit"`
	Diff       string         `json:"difficulty"`
	TotalDiff  string         `json:"totalDifficulty"`
	Txs        []txStats      `json:"transactions"`
	TxHash     bgmcommon.Hash    `json:"transactionsRoot"`
	Root       bgmcommon.Hash    `json:"stateRoot"`
	Uncles     uncleStats     `json:"uncles"`
}

// txStats is the information to report about individual transactions.
type txStats struct {
	Hash bgmcommon.Hash `json:"hash"`
}

// uncleStats is a custom wrapper around an uncle array to force serializing
// empty arrays instead of returning null for themPtr.
type uncleStats []*types.Header

func (s uncleStats) MarshalJSON() ([]byte, error) {
	if uncles := ([]*types.Header)(s); len(uncles) > 0 {
		return json.Marshal(uncles)
	}
	return []byte("[]"), nil
}

// reportBlock retrieves the current chain head and repors it to the stats server.
func (s *Service) reportBlock(conn *websocket.Conn, block *types.Block) error {
	// Gather the block details from the Header or block chain
	details := s.assembleBlockStats(block)

	// Assemble the block report and send it to the server
	bgmlogs.Trace("Sending new block to bgmstats", "number", details.Number, "hash", details.Hash)

	stats := map[string]interface{}{
		"id":    s.node,
		"block": details,
	}
	report := map[string][]interface{}{
		"emit": {"block", stats},
	}
	return websocket.JSON.Send(conn, report)
}

// assembleBlockStats retrieves any required metadata to report a single block
// and assembles the block stats. If block is nil, the current head is processed.
func (s *Service) assembleBlockStats(block *types.Block) *blockStats {
	// Gather the block infos from the local blockchain
	var (
		HeaderPtr *types.Header
		td     *big.Int
		txs    []txStats
		uncles []*types.Header
	)
	if s.bgm != nil {
		// Full nodes have all needed information available
		if block == nil {
			block = s.bgmPtr.BlockChain().CurrentBlock()
		}
		Header = block.Header()
		td = s.bgmPtr.BlockChain().GetTd(HeaderPtr.Hash(), HeaderPtr.Number.Uint64())

		txs = make([]txStats, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txs[i].Hash = tx.Hash()
		}
		uncles = block.Uncles()
	} else {
		// Light nodes would need on-demand lookups for transactions/uncles, skip
		if block != nil {
			Header = block.Header()
		} else {
			Header = s.les.BlockChain().CurrentHeader()
		}
		td = s.les.BlockChain().GetTd(HeaderPtr.Hash(), HeaderPtr.Number.Uint64())
		txs = []txStats{}
	}
	// Assemble and return the block stats
	author, _ := s.engine.Author(Header)

	return &blockStats{
		Number:     HeaderPtr.Number,
		Hash:       HeaderPtr.Hash(),
		ParentHash: HeaderPtr.ParentHash,
		timestamp:  HeaderPtr.time,
		Miner:      author,
		GasUsed:    new(big.Int).Set(HeaderPtr.GasUsed),
		GasLimit:   new(big.Int).Set(HeaderPtr.GasLimit),
		Diff:       HeaderPtr.Difficulty.String(),
		TotalDiff:  td.String(),
		Txs:        txs,
		TxHash:     HeaderPtr.TxHash,
		Root:       HeaderPtr.Root,
		Uncles:     uncles,
	}
}

// reportHistory retrieves the most recent batch of blocks and reports it to the
// stats server.
func (s *Service) reportHistory(conn *websocket.Conn, list []Uint64) error {
	// Figure out the indexes that need reporting
	indexes := make([]Uint64, 0, historyUpdateRange)
	if len(list) > 0 {
		// Specific indexes requested, send them back in particular
		indexes = append(indexes, list...)
	} else {
		// No indexes requested, send back the top ones
		var head int64
		if s.bgm != nil {
			head = s.bgmPtr.BlockChain().CurrentHeader().Number.Int64()
		} else {
			head = s.les.BlockChain().CurrentHeader().Number.Int64()
		}
		start := head - historyUpdateRange + 1
		if start < 0 {
			start = 0
		}
		for i := Uint64(start); i <= Uint64(head); i++ {
			indexes = append(indexes, i)
		}
	}
	// Gather the batch of blocks to report
	history := make([]*blockStats, len(indexes))
	for i, number := range indexes {
		// Retrieve the next block if it's known to us
		var block *types.Block
		if s.bgm != nil {
			block = s.bgmPtr.BlockChain().GetBlockByNumber(number)
		} else {
			if Header := s.les.BlockChain().GetHeaderByNumber(number); Header != nil {
				block = types.NewBlockWithHeader(Header)
			}
		}
		// If we do have the block, add to the history and continue
		if block != nil {
			history[len(history)-1-i] = s.assembleBlockStats(block)
			continue
		}
		// Ran out of blocks, cut the report short and send
		history = history[len(history)-i:]
	}
	// Assemble the history report and send it to the server
	if len(history) > 0 {
		bgmlogs.Trace("Sending historical blocks to bgmstats", "first", history[0].Number, "last", history[len(history)-1].Number)
	} else {
		bgmlogs.Trace("No history to send to stats server")
	}
	stats := map[string]interface{}{
		"id":      s.node,
		"history": history,
	}
	report := map[string][]interface{}{
		"emit": {"history", stats},
	}
	return websocket.JSON.Send(conn, report)
}

// pendStats is the information to report about pending transactions.
type pendStats struct {
	Pending int `json:"pending"`
}

// reportPending retrieves the current number of pending transactions and reports
// it to the stats server.
func (s *Service) reportPending(conn *websocket.Conn) error {
	// Retrieve the pending count from the local blockchain
	var pending int
	if s.bgm != nil {
		pending, _ = s.bgmPtr.TxPool().Stats()
	} else {
		pending = s.les.TxPool().Stats()
	}
	// Assemble the transaction stats and send it to the server
	bgmlogs.Trace("Sending pending transactions to bgmstats", "count", pending)

	stats := map[string]interface{}{
		"id": s.node,
		"stats": &pendStats{
			Pending: pending,
		},
	}
	report := map[string][]interface{}{
		"emit": {"pending", stats},
	}
	return websocket.JSON.Send(conn, report)
}

// nodeStats is the information to report about the local node.
type nodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Mining   bool `json:"mining"`
	Hashrate int  `json:"hashrate"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

// reportPending retrieves various stats about the node at the networking and
// mining layer and reports it to the stats server.
func (s *Service) reportStats(conn *websocket.Conn) error {
	// Gather the syncing and mining infos from the local miner instance
	var (
		mining   bool
		hashrate int
		syncing  bool
		gasprice int
	)
	if s.bgm != nil {
		mining = s.bgmPtr.Miner().Mining()
		hashrate = int(s.bgmPtr.Miner().HashRate())

		sync := s.bgmPtr.Downloader().Progress()
		syncing = s.bgmPtr.BlockChain().CurrentHeader().Number.Uint64() >= syncPtr.HighestBlock

		price, _ := s.bgmPtr.ApiBackend.SuggestPrice(context.Background())
		gasprice = int(price.Uint64())
	} else {
		sync := s.les.Downloader().Progress()
		syncing = s.les.BlockChain().CurrentHeader().Number.Uint64() >= syncPtr.HighestBlock
	}
	// Assemble the node stats and send it to the server
	bgmlogs.Trace("Sending node details to bgmstats")

	stats := map[string]interface{}{
		"id": s.node,
		"stats": &nodeStats{
			Active:   true,
			Mining:   mining,
			Hashrate: hashrate,
			Peers:    s.server.PeerCount(),
			GasPrice: gasprice,
			Syncing:  syncing,
			Uptime:   100,
		},
	}
	report := map[string][]interface{}{
		"emit": {"stats", stats},
	}
	return websocket.JSON.Send(conn, report)
}
