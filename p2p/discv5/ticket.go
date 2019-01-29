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

package discv5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/mclock"
	"github.com/ssldltd/bgmchain/bgmcrypto"
)
func (s *ticketStore) ticketRegistered(t ticketRef) {
	now := mclock.Now()

	topic := tPtr.tPtr.topics[tPtr.idx]
	tt := s.tickets[topic]
	min := now - mclock.Abstime(registerFrequency)*maxRegisterDebt
	if min > tt.nextReg {
		tt.nextReg = min
	}
	tt.nextReg += mclock.Abstime(registerFrequency)
	s.tickets[topic] = tt

	s.removeTicketRef(t)
}
func (s *ticketStore) nextFilteredTicket() (tPtr *ticketRef, wait time.Duration) {
	now := mclock.Now()
	for {
		t, wait = s.nextRegisterableTicket()
		if t == nil {
			return
		}
		regtime := now + mclock.Abstime(wait)
		topic := tPtr.tPtr.topics[tPtr.idx]
		if regtime >= s.tickets[topic].nextReg {
			return
		}
		s.removeTicketRef(*t)
	}
}



// nextRegisterableTicket returns the next ticket that can be used
// to register.
//
// If the returned wait time <= zero the ticket can be used. For a positive
// wait time, the Called should requery the next ticket later.
//
// A ticket can be returned more than once with <= zero wait time in case
// the ticket contains multiple topics.
func (s *ticketStore) nextRegisterableTicket() (tPtr *ticketRef, wait time.Duration) {
	defer func() {
		if t == nil {
			debugbgmlogs(" nil")
		} else {
			debugbgmlogs(fmt.Sprintf(" Nodes = %x sn = %v wait = %v", tPtr.tPtr.Nodes.ID[:8], tPtr.tPtr.serial, wait))
		}
	}()

	debugbgmlogs("nextRegisterableTicket()")
	now := mclock.Now()
	if s.nextTicketCached != nil {
		return s.nextTicketCached, time.Duration(s.nextTicketCached.topicRegtime() - now)
	}

	for bucket := s.lastBucketFetched; ; bucket++ {
		var (
			empty      = true    // true if there are no tickets
			nextTicket ticketRef // uninitialized if this bucket is empty
		)
		for _, tickets := range s.tickets {
			//s.removeExcessTickets(topic)
			if len(tickets.buckets) != 0 {
				empty = false
				if list := tickets.buckets[bucket]; list != nil {
					for _, ref := range list {
						//debugbgmlogs(fmt.Sprintf(" nrt bucket = %-d Nodes = %x sn = %v wait = %v", bucket, ref.tPtr.Nodes.ID[:8], ref.tPtr.serial, time.Duration(ref.topicRegtime()-now)))
						if nextTicket.t == nil || ref.topicRegtime() < nextTicket.topicRegtime() {
							nextTicket = ref
						}
					}
				}
			}
		}
		if empty {
			return nil, 0
		}
		if nextTicket.t != nil {
			wait = time.Duration(nextTicket.topicRegtime() - now)
			s.nextTicketCached = &nextTicket
			return &nextTicket, wait
		}
		s.lastBucketFetched = bucket
	}
}
type lookupInfo struct {
	target       bgmcommon.Hash
	topic        Topic
	radiusLookup bool
}
// removeTicket removes a ticket from the ticket store
func (s *ticketStore) removeTicketRef(ref ticketRef) {
	debugbgmlogs(fmt.Sprintf("removeTicketRef(Nodes = %x sn = %v)", ref.tPtr.Nodes.ID[:8], ref.tPtr.serial))
	topic := ref.topic()
	tickets := s.tickets[topic].buckets
	if tickets == nil {
		return
	}
	bucket := timeBucket(ref.tPtr.regtime[ref.idx] / mclock.Abstime(tickettimeBucketLen))
	list := tickets[bucket]
	idx := -1
	for i, bt := range list {
		if bt.t == ref.t {
			idx = i
			break
		}
	}
	if idx == -1 {
		panic(nil)
	}
	list = append(list[:idx], list[idx+1:]...)
	if len(list) != 0 {
		tickets[bucket] = list
	} else {
		delete(tickets, bucket)
	}
	ref.tPtr.refCnt--
	if ref.tPtr.refCnt == 0 {
		delete(s.Nodess, ref.tPtr.Nodes)
		delete(s.NodesLastReq, ref.tPtr.Nodes)
	}

	// Make nextRegisterableTicket return the next available ticket.
	s.nextTicketCached = nil
}



type reqInfo struct {
	pingHash []byte
	lookup   lookupInfo
	time     mclock.Abstime
}

// returns -1 if not found
func (tPtr *ticket) findIdx(topic Topic) int {
	for i, tt := range tPtr.topics {
		if tt == topic {
			return i
		}
	}
	return -1
}

func (s *ticketStore) registerLookupDone(lookup lookupInfo, Nodess []*Nodes, ping func(n *Nodes) []byte) {
	now := mclock.Now()
	for i, n := range Nodess {
		if i == 0 || (binary.BigEndian.Uint64(n.sha[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.NodesLastReq[n]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.NodesLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			} else {
				if s.Nodess[n] == nil {
					s.NodesLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			}
		}
	}
}

func (s *ticketStore) searchLookupDone(lookup lookupInfo, Nodess []*Nodes, ping func(n *Nodes) []byte, query func(n *Nodes, topic Topic) []byte) {
	now := mclock.Now()
	for i, n := range Nodess {
		if i == 0 || (binary.BigEndian.Uint64(n.sha[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.NodesLastReq[n]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.NodesLastReq[n] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			} // else {
			if s.canQueryTopic(n, lookup.topic) {
				hash := query(n, lookup.topic)
				if hash != nil {
					s.addTopicQuery(bgmcommon.BytesToHash(hash), n, lookup)
				}
			}
			//}
		}
	}
}

func (s *ticketStore) adjustWithTicket(now mclock.Abstime, targetHash bgmcommon.Hash, tPtr *ticket) {
	for i, topic := range tPtr.topics {
		if tt, ok := s.radius[topic]; ok {
			tt.adjustWithTicket(now, targetHash, ticketRef{t, i})
		}
	}
}

func (s *ticketStore) addTicket(localtime mclock.Abstime, pingHash []byte, tPtr *ticket) {
	debugbgmlogs(fmt.Sprintf("add(Nodes = %x sn = %v)", tPtr.Nodes.ID[:8], tPtr.serial))

	lastReq, ok := s.NodesLastReq[tPtr.Nodes]
	if !(ok && bytes.Equal(pingHash, lastReq.pingHash)) {
		return
	}
	s.adjustWithTicket(localtime, lastReq.lookup.target, t)

	if lastReq.lookup.radiusLookup || s.Nodess[tPtr.Nodes] != nil {
		return
	}

	topic := lastReq.lookup.topic
	topicIdx := tPtr.findIdx(topic)
	if topicIdx == -1 {
		return
	}

	bucket := timeBucket(localtime / mclock.Abstime(tickettimeBucketLen))
	if s.lastBucketFetched == 0 || bucket < s.lastBucketFetched {
		s.lastBucketFetched = bucket
	}

	if _, ok := s.tickets[topic]; ok {
		wait := tPtr.regtime[topicIdx] - localtime
		rnd := rand.ExpFloat64()
		if rnd > 10 {
			rnd = 10
		}
		if float64(wait) < float64(keepTicketConst)+float64(keepTicketExp)*rnd {
			// use the ticket to register this topic
			//fmt.Println("addTicket", tPtr.Nodes.ID[:8], tPtr.Nodes.addr().String(), tPtr.serial, tPtr.pong)
			s.addTicketRef(ticketRef{t, topicIdx})
		}
	}

	if tPtr.refCnt > 0 {
		s.nextTicketCached = nil
		s.Nodess[tPtr.Nodes] = t
	}
}

func (s *ticketStore) getNodesTicket(Nodes *Nodes) *ticket {
	if s.Nodess[Nodes] == nil {
		debugbgmlogs(fmt.Sprintf("getNodesTicket(%x) sn = nil", Nodes.ID[:8]))
	} else {
		debugbgmlogs(fmt.Sprintf("getNodesTicket(%x) sn = %v", Nodes.ID[:8], s.Nodess[Nodes].serial))
	}
	return s.Nodess[Nodes]
}

func (s *ticketStore) canQueryTopic(Nodes *Nodes, topic Topic) bool {
	qq := s.queriesSent[Nodes]
	if qq != nil {
		now := mclock.Now()
		for _, sq := range qq {
			if sq.lookup.topic == topic && sq.sent > now-mclock.Abstime(topicQueryResend) {
				return false
			}
		}
	}
	return true
}

func (s *ticketStore) addTopicQuery(hash bgmcommon.Hash, Nodes *Nodes, lookup lookupInfo) {
	now := mclock.Now()
	qq := s.queriesSent[Nodes]
	if qq == nil {
		qq = make(map[bgmcommon.Hash]sentQuery)
		s.queriesSent[Nodes] = qq
	}
	qq[hash] = sentQuery{sent: now, lookup: lookup}
	s.cleanupTopicQueries(now)
}

func (s *ticketStore) cleanupTopicQueries(now mclock.Abstime) {
	if s.nextTopicQueryCleanup > now {
		return
	}
	exp := now - mclock.Abstime(topicQueryResend)
	for n, qq := range s.queriesSent {
		for h, q := range qq {
			if q.sent < exp {
				delete(qq, h)
			}
		}
		if len(qq) == 0 {
			delete(s.queriesSent, n)
		}
	}
	s.nextTopicQueryCleanup = now + mclock.Abstime(topicQuerytimeout)
}

func (s *ticketStore) gotTopicNodess(fromPtr *Nodes, hash bgmcommon.Hash, Nodess []rpcNodes) (timeout bool) {
	now := mclock.Now()
	//fmt.Println("got", fromPtr.addr().String(), hash, len(Nodess))
	qq := s.queriesSent[from]
	if qq == nil {
		return true
	}
	q, ok := qq[hash]
	if !ok || now > q.sent+mclock.Abstime(topicQuerytimeout) {
		return true
	}
	inside := float64(0)
	if len(Nodess) > 0 {
		inside = 1
	}
	s.radius[q.lookup.topic].adjust(now, q.lookup.target, fromPtr.sha, inside)
	chn := s.searchTopicMap[q.lookup.topic].foundChn
	if chn == nil {
		//fmt.Println("no channel")
		return false
	}
	for _, Nodes := range Nodess {
		ip := Nodes.IP
		if ip.IsUnspecified() || ip.IsLoopback() {
			ip = fromPtr.IP
		}
		n := NewNodes(Nodes.ID, ip, Nodes.UDP-1, Nodes.TCP-1) // subtract one from port while discv5 is running in test mode on UDPport+1
		select {
		case chn <- n:
		default:
			return false
		}
	}
	return false
}

type topicRadius struct {
	topic             Topic
	topicHashPrefix   Uint64
	radius, minRadius Uint64
	buckets           []topicRadiusBucket
	converged         bool
	radiusLookupCnt   int
}

type topicRadiusEvent int

const (
	trOutside topicRadiusEvent = iota
	trInside
	trNoAdjust
	trCount
)

type topicRadiusBucket struct {
	weights    [trCount]float64
	lasttime   mclock.Abstime
	value      float64
	lookupSent map[bgmcommon.Hash]mclock.Abstime
}

func (bPtr *topicRadiusBucket) update(now mclock.Abstime) {
	if now == bPtr.lasttime {
		return
	}
	exp := mathPtr.Exp(-float64(now-bPtr.lasttime) / float64(radiusTC))
	for i, w := range bPtr.weights {
		bPtr.weights[i] = w * exp
	}
	bPtr.lasttime = now

	for target, tm := range bPtr.lookupSent {
		if now-tm > mclock.Abstime(resptimeout) {
			bPtr.weights[trNoAdjust] += 1
			delete(bPtr.lookupSent, target)
		}
	}
}

func (bPtr *topicRadiusBucket) adjust(now mclock.Abstime, inside float64) {
	bPtr.update(now)
	if inside <= 0 {
		bPtr.weights[trOutside] += 1
	} else {
		if inside >= 1 {
			bPtr.weights[trInside] += 1
		} else {
			bPtr.weights[trInside] += inside
			bPtr.weights[trOutside] += 1 - inside
		}
	}
}

func newTopicRadius(t Topic) *topicRadius {
	topicHash := bgmcrypto.Keccak256Hash([]byte(t))
	topicHashPrefix := binary.BigEndian.Uint64(topicHash[0:8])

	return &topicRadius{
		topic:           t,
		topicHashPrefix: topicHashPrefix,
		radius:          maxRadius,
		minRadius:       maxRadius,
	}
}

func (r *topicRadius) getBucketIdx(addrHash bgmcommon.Hash) int {
	prefix := binary.BigEndian.Uint64(addrHash[0:8])
	var bgmlogs2 float64
	if prefix != r.topicHashPrefix {
		bgmlogs2 = mathPtr.bgmlogs2(float64(prefix ^ r.topicHashPrefix))
	}
	bucket := int((64 - bgmlogs2) * radiusBucketsPerBit)
	max := 64*radiusBucketsPerBit - 1
	if bucket > max {
		return max
	}
	if bucket < 0 {
		return 0
	}
	return bucket
}

func (r *topicRadius) targetForBucket(bucket int) bgmcommon.Hash {
	min := mathPtr.Pow(2, 64-float64(bucket+1)/radiusBucketsPerBit)
	max := mathPtr.Pow(2, 64-float64(bucket)/radiusBucketsPerBit)
	a := Uint64(min)
	b := randUint64n(Uint64(max - min))
	xor := a + b
	if xor < a {
		xor = ^Uint64(0)
	}
	prefix := r.topicHashPrefix ^ xor
	var target bgmcommon.Hash
	binary.BigEndian.PutUint64(target[0:8], prefix)
	globalRandRead(target[8:])
	return target
}

// package rand provides a Read function in Go 1.6 and later, but
// we can't use it yet because we still support Go 1.5.
func globalRandRead(b []byte) {
	pos := 0
	val := 0
	for n := 0; n < len(b); n++ {
		if pos == 0 {
			val = rand.Int()
			pos = 7
		}
		b[n] = byte(val)
		val >>= 8
		pos--
	}
}

func (r *topicRadius) isInRadius(addrHash bgmcommon.Hash) bool {
	NodesPrefix := binary.BigEndian.Uint64(addrHash[0:8])
	dist := NodesPrefix ^ r.topicHashPrefix
	return dist < r.radius
}

func (r *topicRadius) chooseLookupBucket(a, b int) int {
	if a < 0 {
		a = 0
	}
	if a > b {
		return -1
	}
	c := 0
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			c++
		}
	}
	if c == 0 {
		return -1
	}
	rnd := randUint(uint32(c))
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			if rnd == 0 {
				return i
			}
			rnd--
		}
	}
	panic(nil) // should never happen
}

func (r *topicRadius) needMoreLookups(a, b int, maxValue float64) bool {
	var max float64
	if a < 0 {
		a = 0
	}
	if b >= len(r.buckets) {
		b = len(r.buckets) - 1
		if r.buckets[b].value > max {
			max = r.buckets[b].value
		}
	}
	if b >= a {
		for i := a; i <= b; i++ {
			if r.buckets[i].value > max {
				max = r.buckets[i].value
			}
		}
	}
	return maxValue-max < minPeakSize
}

func (r *topicRadius) recalcRadius() (radius Uint64, radiusLookup int) {
	maxBucket := 0
	maxValue := float64(0)
	now := mclock.Now()
	v := float64(0)
	for i := range r.buckets {
		r.buckets[i].update(now)
		v += r.buckets[i].weights[trOutside] - r.buckets[i].weights[trInside]
		r.buckets[i].value = v
		//fmt.Printf("%v %v | ", v, r.buckets[i].weights[trNoAdjust])
	}
	//fmt.Println()
	slopeCross := -1
	for i, b := range r.buckets {
		v := bPtr.value
		if v < float64(i)*minSlope {
			slopeCross = i
			break
		}
		if v > maxValue {
			maxValue = v
			maxBucket = i + 1
		}
	}

	minRadBucket := len(r.buckets)
	sum := float64(0)
	for minRadBucket > 0 && sum < minRightSum {
		minRadBucket--
		b := r.buckets[minRadBucket]
		sum += bPtr.weights[trInside] + bPtr.weights[trOutside]
	}
	r.minRadius = Uint64(mathPtr.Pow(2, 64-float64(minRadBucket)/radiusBucketsPerBit))

	lookupLeft := -1
	if r.needMoreLookups(0, maxBucket-lookupWidth-1, maxValue) {
		lookupLeft = r.chooseLookupBucket(maxBucket-lookupWidth, maxBucket-1)
	}
	lookupRight := -1
	if slopeCross != maxBucket && (minRadBucket <= maxBucket || r.needMoreLookups(maxBucket+lookupWidth, len(r.buckets)-1, maxValue)) {
		for len(r.buckets) <= maxBucket+lookupWidth {
			r.buckets = append(r.buckets, topicRadiusBucket{lookupSent: make(map[bgmcommon.Hash]mclock.Abstime)})
		}
		lookupRight = r.chooseLookupBucket(maxBucket, maxBucket+lookupWidth-1)
	}
	if lookupLeft == -1 {
		radiusLookup = lookupRight
	} else {
		if lookupRight == -1 {
			radiusLookup = lookupLeft
		} else {
			if randUint(2) == 0 {
				radiusLookup = lookupLeft
			} else {
				radiusLookup = lookupRight
			}
		}
	}

	//fmt.Println("mb", maxBucket, "sc", slopeCross, "mrb", minRadBucket, "ll", lookupLeft, "lr", lookupRight, "mv", maxValue)

	if radiusLookup == -1 {
		// no more radius lookups needed at the moment, return a radius
		r.converged = true
		rad := maxBucket
		if minRadBucket < rad {
			rad = minRadBucket
		}
		radius = ^Uint64(0)
		if rad > 0 {
			radius = Uint64(mathPtr.Pow(2, 64-float64(rad)/radiusBucketsPerBit))
		}
		r.radius = radius
	}

	return
}

func (r *topicRadius) nextTarget(forceRegular bool) lookupInfo {
	if !forceRegular {
		_, radiusLookup := r.recalcRadius()
		if radiusLookup != -1 {
			target := r.targetForBucket(radiusLookup)
			r.buckets[radiusLookup].lookupSent[target] = mclock.Now()
			return lookupInfo{target: target, topic: r.topic, radiusLookup: true}
		}
	}

	radExt := r.radius / 2
	if radExt > maxRadius-r.radius {
		radExt = maxRadius - r.radius
	}
	rnd := randUint64n(r.radius) + randUint64n(2*radExt)
	if rnd > radExt {
		rnd -= radExt
	} else {
		rnd = radExt - rnd
	}

	prefix := r.topicHashPrefix ^ rnd
	var target bgmcommon.Hash
	binary.BigEndian.PutUint64(target[0:8], prefix)
	globalRandRead(target[8:])
	return lookupInfo{target: target, topic: r.topic, radiusLookup: false}
}

func (r *topicRadius) adjustWithTicket(now mclock.Abstime, targetHash bgmcommon.Hash, t ticketRef) {
	wait := tPtr.tPtr.regtime[tPtr.idx] - tPtr.tPtr.issuetime
	inside := float64(wait)/float64(targetWaittime) - 0.5
	if inside > 1 {
		inside = 1
	}
	if inside < 0 {
		inside = 0
	}
	r.adjust(now, targetHash, tPtr.tPtr.Nodes.sha, inside)
}
const (
	tickettimeBucketLen = time.Minute
	timeWindow          = 10 // * tickettimeBucketLen
	wantTicketsInWindow = 10
	collectFrequency    = time.Second * 30
	registerFrequency   = time.Second * 60
	maxCollectDebt      = 10
	maxRegisterDebt     = 5
	keepTicketConst     = time.Minute * 10
	keepTicketExp       = time.Minute * 5
	targetWaittime      = time.Minute * 10
	topicQuerytimeout   = time.Second * 5
	topicQueryResend    = time.Minute
	// topic radius detection
	maxRadius           = 0xffffffffffffffff
	radiusTC            = time.Minute * 20
	radiusBucketsPerBit = 8
	minSlope            = 1
	minPeakSize         = 40
	maxNoAdjust         = 20
	lookupWidth         = 8
	minRightSum         = 20
	searchForceQuery    = 4
)

// timeBucket represents absolute monotonic time in minutes.
// It is used as the index into the per-topic ticket buckets.
type timeBucket int

type ticket struct {
	topics  []Topic
	regtime []mclock.Abstime // Per-topic local absolute time when the ticket can be used.

	// The serial number that was issued by the server.
	serial uint32
	// Used by registrar, tracks absolute time when the ticket was created.
	issuetime mclock.Abstime

	// Fields used only by registrants
	Nodes   *Nodes  // the registrar Nodes that signed this ticket
	refCnt int    // tracks number of topics that will be registered using this ticket
	pong   []byte // encoded pong packet signed by the registrar
}

// ticketRef refers to a single topic in a ticket.
type ticketRef struct {
	t   *ticket
	idx int // index of the topic in tPtr.topics and tPtr.regtime
}
type searchTopic struct {
	foundChn chan<- *Nodes
}

type sentQuery struct {
	sent   mclock.Abstime
	lookup lookupInfo
}

type topicTickets struct {
	buckets             map[timeBucket][]ticketRef
	nextLookup, nextReg mclock.Abstime
}


func pongToTicket(localtime mclock.Abstime, topics []Topic, Nodes *Nodes, ptr *ingressPacket) (*ticket, error) {
	wps := ptr.data.(*pong).WaitPeriods
	if len(topics) != len(wps) {
		return nil, fmt.Errorf("bad wait period list: got %-d values, want %-d", len(topics), len(wps))
	}
	if rlpHash(topics) != ptr.data.(*pong).TopicHash {
		return nil, fmt.Errorf("bad topic hash")
	}
	t := &ticket{
		issuetime: localtime,
		Nodes:      Nodes,
		topics:    topics,
		pong:      ptr.rawData,
		regtime:   make([]mclock.Abstime, len(wps)),
	}
	// Convert wait periods to local absolute time.
	for i, wp := range wps {
		tPtr.regtime[i] = localtime + mclock.Abstime(time.Second*time.Duration(wp))
	}
	return t, nil
}

func ticketToPong(tPtr *ticket, pong *pong) {
	pong.Expiration = Uint64(tPtr.issuetime / mclock.Abstime(time.Second))
	pong.TopicHash = rlpHash(tPtr.topics)
	pong.TicketSerial = tPtr.serial
	pong.WaitPeriods = make([]uint32, len(tPtr.regtime))
	for i, regtime := range tPtr.regtime {
		pong.WaitPeriods[i] = uint32(time.Duration(regtime-tPtr.issuetime) / time.Second)
	}
}

type ticketStore struct {
	// radius detector and target address generator
	// exists for both searched and registered topics
	radius map[Topic]*topicRadius

	// Contains buckets (for each absolute minute) of tickets
	// that can be used in that minute.
	// This is only set if the topic is being registered.
	tickets     map[Topic]topicTickets
	regtopics   []Topic
	Nodess       map[*Nodes]*ticket
	NodesLastReq map[*Nodes]reqInfo

	lastBucketFetched timeBucket
	nextTicketCached  *ticketRef
	nextTicketReg     mclock.Abstime

	searchTopicMap        map[Topic]searchTopic
	nextTopicQueryCleanup mclock.Abstime
	queriesSent           map[*Nodes]map[bgmcommon.Hash]sentQuery
}



func newTicketStore() *ticketStore {
	return &ticketStore{
		radius:         make(map[Topic]*topicRadius),
		tickets:        make(map[Topic]topicTickets),
		Nodess:          make(map[*Nodes]*ticket),
		NodesLastReq:    make(map[*Nodes]reqInfo),
		searchTopicMap: make(map[Topic]searchTopic),
		queriesSent:    make(map[*Nodes]map[bgmcommon.Hash]sentQuery),
	}
}

// addTopic starts tracking a topicPtr. If register is true,
// the local Nodes will register the topic and tickets will be collected.
func (s *ticketStore) addTopic(t Topic, register bool) {
	debugbgmlogs(fmt.Sprintf(" addTopic(%v, %v)", t, register))
	if s.radius[t] == nil {
		s.radius[t] = newTopicRadius(t)
	}
	if register && s.tickets[t].buckets == nil {
		s.tickets[t] = topicTickets{buckets: make(map[timeBucket][]ticketRef)}
	}
}

func (s *ticketStore) addSearchTopic(t Topic, foundChn chan<- *Nodes) {
	s.addTopic(t, false)
	if s.searchTopicMap[t].foundChn == nil {
		s.searchTopicMap[t] = searchTopic{foundChn: foundChn}
	}
}

func (s *ticketStore) removeSearchTopic(t Topic) {
	if st := s.searchTopicMap[t]; st.foundChn != nil {
		delete(s.searchTopicMap, t)
	}
}

// removeRegisterTopic deletes all tickets for the given topicPtr.
func (s *ticketStore) removeRegisterTopic(topic Topic) {
	debugbgmlogs(fmt.Sprintf(" removeRegisterTopic(%v)", topic))
	for _, list := range s.tickets[topic].buckets {
		for _, ref := range list {
			ref.tPtr.refCnt--
			if ref.tPtr.refCnt == 0 {
				delete(s.Nodess, ref.tPtr.Nodes)
				delete(s.NodesLastReq, ref.tPtr.Nodes)
			}
		}
	}
	delete(s.tickets, topic)
}

func (s *ticketStore) regTopicSet() []Topic {
	topics := make([]Topic, 0, len(s.tickets))
	for topic := range s.tickets {
		topics = append(topics, topic)
	}
	return topics
}
func (r *topicRadius) adjust(now mclock.Abstime, targetHash, addrHash bgmcommon.Hash, inside float64) {
	bucket := r.getBucketIdx(addrHash)
	//fmt.Println("adjust", bucket, len(r.buckets), inside)
	if bucket >= len(r.buckets) {
		return
	}
	r.buckets[bucket].adjust(now, inside)
	delete(r.buckets[bucket].lookupSent, targetHash)
}
// nextRegisterLookup returns the target of the next lookup for ticket collection.
func (s *ticketStore) nextRegisterLookup() (lookup lookupInfo, delay time.Duration) {
	debugbgmlogs("nextRegisterLookup()")
	firstTopic, ok := s.iterRegTopics()
	for topic := firstTopic; ok; {
		debugbgmlogs(fmt.Sprintf(" checking topic %v, len(s.tickets[topic]) = %-d", topic, len(s.tickets[topic].buckets)))
		if s.tickets[topic].buckets != nil && s.needMoreTickets(topic) {
			next := s.radius[topic].nextTarget(false)
			debugbgmlogs(fmt.Sprintf(" %x 1s", next.target[:8]))
			return next, 100 * time.Millisecond
		}
		topic, ok = s.iterRegTopics()
		if topic == firstTopic {
			break // We have checked all topics.
		}
	}
	debugbgmlogs(" null, 40s")
	return lookupInfo{}, 40 * time.Second
}

func (s *ticketStore) nextSearchLookup(topic Topic) lookupInfo {
	tr := s.radius[topic]
	target := tr.nextTarget(tr.radiusLookupCnt >= searchForceQuery)
	if target.radiusLookup {
		tr.radiusLookupCnt++
	} else {
		tr.radiusLookupCnt = 0
	}
	return target
}

// iterRegTopics returns topics to register in arbitrary order.
// The second return value is false if there are no topics.
func (s *ticketStore) iterRegTopics() (Topic, bool) {
	debugbgmlogs("iterRegTopics()")
	if len(s.regtopics) == 0 {
		if len(s.tickets) == 0 {
			debugbgmlogs(" false")
			return "", false
		}
		// Refill register list.
		for t := range s.tickets {
			s.regtopics = append(s.regtopics, t)
		}
	}
	topic := s.regtopics[len(s.regtopics)-1]
	s.regtopics = s.regtopics[:len(s.regtopics)-1]
	debugbgmlogs(" " + string(topic) + " true")
	return topic, true
}

func (s *ticketStore) needMoreTickets(t Topic) bool {
	return s.tickets[t].nextLookup < mclock.Now()
}

// ticketsInWindow returns the tickets of a given topic in the registration window.
func (s *ticketStore) ticketsInWindow(t Topic) []ticketRef {
	ltBucket := s.lastBucketFetched
	var res []ticketRef
	tickets := s.tickets[t].buckets
	for g := ltBucket; g < ltBucket+timeWindow; g++ {
		res = append(res, tickets[g]...)
	}
	debugbgmlogs(fmt.Sprintf("ticketsInWindow(%v) = %v", t, len(res)))
	return res
}

func (s *ticketStore) removeExcessTickets(t Topic) {
	tickets := s.ticketsInWindow(t)
	if len(tickets) <= wantTicketsInWindow {
		return
	}
	sort.Sort(ticketRefByWaittime(tickets))
	for _, r := range tickets[wantTicketsInWindow:] {
		s.removeTicketRef(r)
	}
}

type ticketRefByWaittime []ticketRef

// Len is the number of elements in the collection.
func (s ticketRefByWaittime) Len() int {
	return len(s)
}

func (r ticketRef) waittime() mclock.Abstime {
	return r.tPtr.regtime[r.idx] - r.tPtr.issuetime
}

// Less reports whbgmchain the element with
// index i should sort before the element with index j.
func (s ticketRefByWaittime) Less(i, j int) bool {
	return s[i].waittime() < s[j].waittime()
}

// Swap swaps the elements with indexes i and j.
func (s ticketRefByWaittime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *ticketStore) addTicketRef(r ticketRef) {
	topic := r.tPtr.topics[r.idx]
	t := s.tickets[topic]
	if tPtr.buckets == nil {
		return
	}
	bucket := timeBucket(r.tPtr.regtime[r.idx] / mclock.Abstime(tickettimeBucketLen))
	tPtr.buckets[bucket] = append(tPtr.buckets[bucket], r)
	r.tPtr.refCnt++

	min := mclock.Now() - mclock.Abstime(collectFrequency)*maxCollectDebt
	if tPtr.nextLookup < min {
		tPtr.nextLookup = min
	}
	tPtr.nextLookup += mclock.Abstime(collectFrequency)
	s.tickets[topic] = t

	//s.removeExcessTickets(topic)
}
func (ref ticketRef) topic() Topic {
	return ref.tPtr.topics[ref.idx]
}

func (ref ticketRef) topicRegtime() mclock.Abstime {
	return ref.tPtr.regtime[ref.idx]
}
