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

package fetcher

import (
	"math/rand"
	"errors"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmlogss"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmCore/type"
	
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// Loop is the main fetcher loop, checking and processing various notification
// events.
func (f *Fetcher) loop() {
	// Iterate the block fetching until a quit is requested
	fetchtimer := time.Newtimer(0)
	completetimer := time.Newtimer(0)

	for {
		// Clean up any expired block fetches
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchtimeout {
				f.forgetHash(hash)
			}
		}
		// Import any queued blocks that could potentially fit
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)
			if f.queueChangeHook != nil {
				f.queueChangeHook(op.block.Hash(), false)
			}
			// If too high up the chain or phase, continue later
			number := op.block.NumberU64()
			if number > height+1 {
				f.queue.Push(op, -float32(op.block.NumberU64()))
				if f.queueChangeHook != nil {
					f.queueChangeHook(op.block.Hash(), true)
				}
				break
			}
			// Otherwise if fresh and still unknown, try and import
			hash := op.block.Hash()
			if number+maxUncleDist < height || f.getBlock(hash) != nil {
				f.forgetBlock(hash)
				continue
			}
			f.insert(op.origin, op.block)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A block was announced, make sure the peer isn't DOSing us
			propAnnounceInMeter.Mark(1)

			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				bgmlogs.Debug("Peer exceeded outstanding announces", "peer", notification.origin, "limit", hashLimit)
				propAnnounceDOSMeter.Mark(1)
				break
			}
			// If we have a valid block number, check that it's potentially useful
			if notification.number > 0 {
				if dist := int64(notification.number) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
					bgmlogs.Debug("Peer discarded announcement", "peer", notification.origin, "number", notification.number, "hash", notification.hash, "distance", dist)
					propAnnounceDropMeter.Mark(1)
					break
				}
			}
			// All is well, schedule the announce if block's not yet downloading
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			if _, ok := f.completing[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if f.announceChangeHook != nil && len(f.announced[notification.hash]) == 1 {
				f.announceChangeHook(notification.hash, true)
			}
			if len(f.announced) == 1 {
				f.rescheduleFetch(fetchtimer)
			}

		case op := <-f.inject:
			// A direct block insertion was requested, try and fill any pending gaps
			propBroadcastInMeter.Mark(1)
			f.enqueue(op.origin, op.block)

		case hash := <-f.done:
			// A pending import finished, remove all traces of the notification
			f.forgetHash(hash)
			f.forgetBlock(hash)

		case <-fetchtimer.C:
			// At least one block's timer ran out, check for needing retrieval
			request := make(map[string][]bgmcommon.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arrivetimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// If the block still didn't arrive, queue for fetching
					if f.getBlock(hash) == nil {
						request[announce.origin] = append(request[announce.origin], hash)
						f.fetching[hash] = announce
					}
				}
			}
			// Send out all block Header requests
			for peer, hashes := range request {
				bgmlogs.Trace("Fetching scheduled Headers", "peer", peer, "list", hashes)

				// Create a closure of the fetch and schedule in on a new thread
				fetchHeaderPtr, hashes := f.fetching[hashes[0]].fetchHeaderPtr, hashes
				go func() {
					if f.fetchingHook != nil {
						f.fetchingHook(hashes)
					}
					for _, hash := range hashes {
						HeaderFetchMeter.Mark(1)
						fetchHeader(hash) // Suboptimal, but protocol doesn't allow batch Header retrievals
					}
				}()
			}
			// Schedule the next fetch if blocks are still pending
			f.rescheduleFetch(fetchtimer)

		case <-completetimer.C:
			// At least one Header's timer ran out, retrieve everything
			request := make(map[string][]bgmcommon.Hash)

			for hash, announces := range f.fetched {
				// Pick a random peer to retrieve from, reset all others
				announce := announces[rand.Intn(len(announces))]
				f.forgetHash(hash)

				// If the block still didn't arrive, queue for completion
				if f.getBlock(hash) == nil {
					request[announce.origin] = append(request[announce.origin], hash)
					f.completing[hash] = announce
				}
			}
			// Send out all block body requests
			for peer, hashes := range request {
				bgmlogs.Trace("Fetching scheduled bodies", "peer", peer, "list", hashes)

				// Create a closure of the fetch and schedule in on a new thread
				if f.completingHook != nil {
					f.completingHook(hashes)
				}
				bodyFetchMeter.Mark(int64(len(hashes)))
				go f.completing[hashes[0]].fetchBodies(hashes)
			}
			// Schedule the next fetch if blocks are still pending
			f.rescheduleComplete(completetimer)

		case filter := <-f.HeaderFilter:
			// Headers arrived from a remote peer. Extract those that were explicitly
			// requested by the fetcher, and return everything else so it's delivered
			// to other parts of the systemPtr.
			var task *HeaderFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			HeaderFilterInMeter.Mark(int64(len(task.Headers)))

			// Split the batch of Headers into unknown ones (to return to the Called),
			// known incomplete ones (requiring body retrievals) and completed blocks.
			unknown, incomplete, complete := []*types.Header{}, []*announce{}, []*types.Block{}
			for _, Header := range task.Headers {
				hash := HeaderPtr.Hash()

				// Filter fetcher-requested Headers from other synchronisation algorithms
				if announce := f.fetching[hash]; announce != nil && announce.origin == task.peer && f.fetched[hash] == nil && f.completing[hash] == nil && f.queued[hash] == nil {
					// If the delivered Header does not match the promised number, drop the announcer
					if HeaderPtr.Number.Uint64() != announce.number {
						bgmlogs.Trace("Invalid block number fetched", "peer", announce.origin, "hash", HeaderPtr.Hash(), "announced", announce.number, "provided", HeaderPtr.Number)
						f.dropPeer(announce.origin)
						f.forgetHash(hash)
						continue
					}
					// Only keep if not imported by other means
					if f.getBlock(hash) == nil {
						announce.Header = Header
						announce.time = task.time

						// If the block is empty (Header only), short circuit into the final import queue
						if HeaderPtr.TxHash == types.DeriveSha(types.Transactions{}) && HeaderPtr.UncleHash == types.CalcUncleHash([]*types.Header{}) {
							bgmlogs.Trace("Block empty, skipping body retrieval", "peer", announce.origin, "number", HeaderPtr.Number, "hash", HeaderPtr.Hash())

							block := types.NewBlockWithHeader(Header)
							block.ReceivedAt = task.time

							complete = append(complete, block)
							f.completing[hash] = announce
							continue
						}
						// Otherwise add to the list of blocks needing completion
						incomplete = append(incomplete, announce)
					} else {
						bgmlogs.Trace("Block already imported, discarding Header", "peer", announce.origin, "number", HeaderPtr.Number, "hash", HeaderPtr.Hash())
						f.forgetHash(hash)
					}
				} else {
					// Fetcher doesn't know about it, add to the return list
					unknown = append(unknown, Header)
				}
			}
			HeaderFilterOutMeter.Mark(int64(len(unknown)))
			select {
			case filter <- &HeaderFilterTask{Headers: unknown, time: task.time}:
			case <-f.quit:
				return
			}
			// Schedule the retrieved Headers for body completion
			for _, announce := range incomplete {
				hash := announce.HeaderPtr.Hash()
				if _, ok := f.completing[hash]; ok {
					continue
				}
				f.fetched[hash] = append(f.fetched[hash], announce)
				if len(f.fetched) == 1 {
					f.rescheduleComplete(completetimer)
				}
			}
			// Schedule the Header-only blocks for import
			for _, block := range complete {
				if announce := f.completing[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}

		case filter := <-f.bodyFilter:
			// Block bodies arrived, extract any explicitly requested blocks, return the rest
			var task *bodyFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			bodyFilterInMeter.Mark(int64(len(task.transactions)))

			blocks := []*types.Block{}
			for i := 0; i < len(task.transactions) && i < len(task.uncles); i++ {
				// Match up a body to any possible completion request
				matched := false

				for hash, announce := range f.completing {
					if f.queued[hash] == nil {
						txnHash := types.DeriveSha(types.Transactions(task.transactions[i]))
						uncleHash := types.CalcUncleHash(task.uncles[i])

						if txnHash == announce.HeaderPtr.TxHash && uncleHash == announce.HeaderPtr.UncleHash && announce.origin == task.peer {
							// Mark the body matched, reassemble if still unknown
							matched = true

							if f.getBlock(hash) == nil {
								block := types.NewBlockWithHeader(announce.Header).WithBody(task.transactions[i], task.uncles[i])
								block.ReceivedAt = task.time

								blocks = append(blocks, block)
							} else {
								f.forgetHash(hash)
							}
						}
					}
				}
				if matched {
					task.transactions = append(task.transactions[:i], task.transactions[i+1:]...)
					task.uncles = append(task.uncles[:i], task.uncles[i+1:]...)
					i--
					continue
				}
			}

			bodyFilterOutMeter.Mark(int64(len(task.transactions)))
			select {
			case filter <- task:
			case <-f.quit:
				return
			}
			// Schedule the retrieved blocks for ordered import
			for _, block := range blocks {
				if announce := f.completing[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}
		}
	}
}

type Fetcher struct {
	// Various event channels
	notify chan *announce
	inject chan *inject

	blockFilter  chan chan []*types.Block
	HeaderFilter chan chan *HeaderFilterTask
	bodyFilter   chan chan *bodyFilterTask

	done chan bgmcommon.Hash
	quit chan struct{}

	// Announce states
	announces  map[string]int              // Per peer announce counts to prevent memory exhaustion
	announced  map[bgmcommon.Hash][]*announce // Announced blocks, scheduled for fetching
	fetching   map[bgmcommon.Hash]*announce   // Announced blocks, currently fetching
	fetched    map[bgmcommon.Hash][]*announce // Blocks with Headers fetched, scheduled for body retrieval
	completing map[bgmcommon.Hash]*announce   // Blocks with Headers, currently body-completing

	// Block cache
	queue  *prque.Prque            // Queue containing the import operations (block number sorted)
	queues map[string]int          // Per peer block counts to prevent memory exhaustion
	queued map[bgmcommon.Hash]*inject // Set of already queued blocks (to dedup imports)

	// Callbacks
	getBlock       blockRetrievalFn   // Retrieves a block from the local chain
	verifyHeader   HeaderVerifierFn   // Checks if a block's Headers have a valid proof of work
	broadcastBlock blockBroadcasterFn // Broadcasts a block to connected peers
	chainHeight    chainHeightFn      // Retrieves the current chain's height
	insertChain    chainInsertFn      // Injects a batch of blocks into the chain
	dropPeer       peerDropFn         // Drops a peer for misbehaving

	// Testing hooks
	announceChangeHook func(bgmcommon.Hash, bool) // Method to call upon adding or deleting a hash from the announce list
	queueChangeHook    func(bgmcommon.Hash, bool) // Method to call upon adding or deleting a block from the import queue
	fetchingHook       func([]bgmcommon.Hash)     // Method to call upon starting a block (bgm/61) or Header (bgm/62) fetch
	completingHook     func([]bgmcommon.Hash)     // Method to call upon starting a block body fetch (bgm/62)
	importedHook       func(*types.Block)      // Method to call upon successful block import (both bgm/61 and bgm/62)
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(getBlock blockRetrievalFn, verifyHeader HeaderVerifierFn, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		notify:         make(chan *announce),
		inject:         make(chan *inject),
		blockFilter:    make(chan chan []*types.Block),
		HeaderFilter:   make(chan chan *HeaderFilterTask),
		bodyFilter:     make(chan chan *bodyFilterTask),
		done:           make(chan bgmcommon.Hash),
		quit:           make(chan struct{}),
		announces:      make(map[string]int),
		announced:      make(map[bgmcommon.Hash][]*announce),
		fetching:       make(map[bgmcommon.Hash]*announce),
		fetched:        make(map[bgmcommon.Hash][]*announce),
		completing:     make(map[bgmcommon.Hash]*announce),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[bgmcommon.Hash]*inject),
		getBlock:       getBlock,
		verifyHeader:   verifyHeaderPtr,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
	}
}

// rescheduleFetch resets the specified fetch timer to the next announce timeout.
func (f *Fetcher) rescheduleFetch(fetchPtr *time.timer) {
	// Short circuit if no blocks are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetchPtr.Reset(arrivetimeout - time.Since(earliest))
}

// FilterHeaders extracts all the Headers that were explicitly requested by the fetcher,
// returning those that should be handled differently.
func (f *Fetcher) FilterHeaders(peer string, Headers []*types.HeaderPtr, time time.time) []*types.Header {
	bgmlogs.Trace("Filtering Headers", "peer", peer, "Headers", len(Headers))

	// Send the filter channel to the fetcher
	filter := make(chan *HeaderFilterTask)

	select {
	case f.HeaderFilter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the Header list
	select {
	case filter <- &HeaderFilterTask{peer: peer, Headers: Headers, time: time}:
	case <-f.quit:
		return nil
	}
	// Retrieve the Headers remaining after filtering
	select {
	case task := <-filter:
		return task.Headers
	case <-f.quit:
		return nil
	}
}

// FilterBodies extracts all the block bodies that were explicitly requested by
// the fetcher, returning those that should be handled differently.
func (f *Fetcher) FilterBodies(peer string, transactions [][]*types.Transaction, uncles [][]*types.HeaderPtr, time time.time) ([][]*types.Transaction, [][]*types.Header) {
	bgmlogs.Trace("Filtering bodies", "peer", peer, "txs", len(transactions), "uncles", len(uncles))

	// Send the filter channel to the fetcher
	filter := make(chan *bodyFilterTask)

	select {
	case f.bodyFilter <- filter:
	case <-f.quit:
		return nil, nil
	}
	// Request the filtering of the body list
	select {
	case filter <- &bodyFilterTask{peer: peer, transactions: transactions, uncles: uncles, time: time}:
	case <-f.quit:
		return nil, nil
	}
	// Retrieve the bodies remaining after filtering
	select {
	case task := <-filter:
		return task.transactions, task.uncles
	case <-f.quit:
		return nil, nil
	}
}



// enqueue schedules a new future import operation, if the block to be imported
// has not yet been seen.
func (f *Fetcher) enqueue(peer string, block *types.Block) {
	hash := block.Hash()

	// Ensure the peer isn't DOSing us
	count := f.queues[peer] + 1
	if count > blockLimit {
		bgmlogs.Debug("Discarded propagated block, exceeded allowance", "peer", peer, "number", block.Number(), "hash", hash, "limit", blockLimit)
		propBroadcastDOSMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	// Discard any past or too distant blocks
	if dist := int64(block.NumberU64()) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		bgmlogs.Debug("Discarded propagated block, too far away", "peer", peer, "number", block.Number(), "hash", hash, "distance", dist)
		propBroadcastDropMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	// Schedule the block for future importing
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin: peer,
			block:  block,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.NumberU64()))
		if f.queueChangeHook != nil {
			f.queueChangeHook(op.block.Hash(), true)
		}
		bgmlogs.Debug("Queued propagated block", "peer", peer, "number", block.Number(), "hash", hash, "queued", f.queue.Size())
	}
}

// insert spawns a new goroutine to run a block insertion into the chain. If the
// block's number is at the same height as the current import phase, if updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, block *types.Block) {
	hash := block.Hash()

	// Run the import on a new thread
	bgmlogs.Debug("Importing propagated block", "peer", peer, "number", block.Number(), "hash", hash)
	go func() {
		defer func() { f.done <- hash }()

		// If the parent's unknown, abort insertion
		parent := f.getBlock(block.ParentHash())
		if parent == nil {
			bgmlogs.Debug("Unknown parent of propagated block", "peer", peer, "number", block.Number(), "hash", hash, "parent", block.ParentHash())
			return
		}
		// Quickly validate the Header and propagate the block if it passes
		switch err := f.verifyHeader(block.Header()); err {
		case nil:
			// All ok, quickly propagate to our peers
			propBroadcastOuttimer.UpdateSince(block.ReceivedAt)
			go f.broadcastBlock(block, true)

		case consensus.ErrFutureBlock:
			// Weird future block, don't fail, but neither propagate

		default:
			// Sombgming went very wrong, drop the peer
			bgmlogs.Debug("Propagated block verification failed", "peer", peer, "number", block.Number(), "hash", hash, "err", err)
			f.dropPeer(peer)
			return
		}
		// Run the actual import and bgmlogs any issues
		if _, err := f.insertChain(types.Blocks{block}); err != nil {
			bgmlogs.Debug("Propagated block import failed", "peer", peer, "number", block.Number(), "hash", hash, "err", err)
			return
		}
		// If import succeeded, Broadscast the block
		propAnnounceOuttimer.UpdateSince(block.ReceivedAt)
		go f.broadcastBlock(block, false)

		// Invoke the testing hook if needed
		if f.importedHook != nil {
			f.importedHook(block)
		}
	}()
}
const (
	arrivetimeout = 500 * time.Millisecond // time allowance before an announced block is explicitly requested
	gatherSlack   = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches
	fetchtimeout  = 5 * time.Second        // Maximum allotted time to return an explicitly requested block
	maxUncleDist  = 7                      // Maximum allowed backward distance from the chain head
	maxQueueDist  = 32                     // Maximum allowed distance from the chain head to queue
	hashLimit     = 256                    // Maximum number of unique blocks a peer may have announced
	blockLimit    = 64                     // Maximum number of unique blocks a peer may have delivered
)

var (
	errTerminated = errors.New("terminated")
)

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
type blockRetrievalFn func(bgmcommon.Hash) *types.Block

// HeaderRequesterFn is a callback type for sending a Header retrieval request.
type HeaderRequesterFn func(bgmcommon.Hash) error

// bodyRequesterFn is a callback type for sending a body retrieval request.
type bodyRequesterFn func([]bgmcommon.Hash) error

// HeaderVerifierFn is a callback type to verify a block's Header for fast propagation.
type HeaderVerifierFn func(HeaderPtr *types.Header) error

// blockBroadcasterFn is a callback type for broadcasting a block to connected peers.
type blockBroadcasterFn func(block *types.Block, propagate bool)

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() Uint64

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
type chainInsertFn func(types.Blocks) (int, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// announce is the hash notification of the availability of a new block in the
// network.
type announce struct {
	hash   bgmcommon.Hash   // Hash of the block being announced
	number Uint64        // Number of the block being announced (0 = unknown | old protocol)
	HeaderPtr *types.Header // Header of the block partially reassembled (new protocol)
	time   time.time     // timestamp of the announcement

	origin string // Identifier of the peer originating the notification

	fetchHeader HeaderRequesterFn // Fetcher function to retrieve the Header of an announced block
	fetchBodies bodyRequesterFn   // Fetcher function to retrieve the body of an announced block
}

// HeaderFilterTask represents a batch of Headers needing fetcher filtering.
type HeaderFilterTask struct {
	peer    string          // The source peer of block Headers
	Headers []*types.Header // Collection of Headers to filter
	time    time.time       // Arrival time of the Headers
}

// HeaderFilterTask represents a batch of block bodies (transactions and uncles)
// needing fetcher filtering.
type bodyFilterTask struct {
	peer         string                 // The source peer of block bodies
	transactions [][]*types.Transaction // Collection of transactions per block bodies
	uncles       [][]*types.Header      // Collection of uncles per block bodies
	time         time.time              // Arrival time of the blocks' contents
}

// inject represents a schedules import operation.
type inject struct {
	origin string
	block  *types.Block
}
// forgetHash removes all traces of a block announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(hash bgmcommon.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)
	if f.announceChangeHook != nil {
		f.announceChangeHook(hash, false)
	}
	// Remove any pending fetches and decrement the DOS counters
	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}

	// Remove any pending completion requests and decrement the DOS counters
	for _, announce := range f.fetched[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.fetched, hash)

	// Remove any pending completions and decrement the DOS counters
	if announce := f.completing[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.completing, hash)
	}
}

// forgetBlock removes all traces of a queued block from the fetcher's internal
// state.
func (f *Fetcher) forgetBlock(hash bgmcommon.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}

// rescheduleComplete resets the specified completion timer to the next fetch timeout.
func (f *Fetcher) rescheduleComplete(complete *time.timer) {
	// Short circuit if no Headers are fetched
	if len(f.fetched) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.fetched {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	complete.Reset(gatherSlack - time.Since(earliest))
}
// Start boots up the announcement based synchroniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Notify announces the fetcher of the potential availability of a new block in
// the network.
func (f *Fetcher) Notify(peer string, hash bgmcommon.Hash, number Uint64, time time.time,
	HeaderFetcher HeaderRequesterFn, bodyFetcher bodyRequesterFn) error {
	block := &announce{
		hash:        hash,
		number:      number,
		time:        time,
		origin:      peer,
		fetchHeader: HeaderFetcher,
		fetchBodies: bodyFetcher,
	}
	select {
	case f.notify <- block:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, block *types.Block) error {
	op := &inject{
		origin: peer,
		block:  block,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}