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

package bgmCore

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmCore/types"
	"github.com/ssldltd/bgmchain/bgmdb"
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain/bgmlogs"
)

// the background and write the segment results into the database. These can be
type ChainIndexerBackend interface {
	// any partially completed operations (in case of a reorg).
	Reset(section Uint64, prevHead bgmcommon.Hash) error

	// will ensure a sequential order of Headers.
	Process(HeaderPtr *types.Header)

	// Commit finalizes the section metadata and stores it into the database.
	Commit() error
}

// ChainIndexerChain interface is used for connecting the indexer to a blockchain
type ChainIndexerChain interface {
	// CurrentHeader retrieves the latest locally known HeaderPtr.
	CurrentHeader() *types.Header

	// SubscribeChainEvent subscribes to new head Header notifications.
	SubscribeChainEvent(ch chan<- ChainEvent) event.Subscription
}
// Further child ChainIndexers can be added which use the output of the parent
// section indexer. These child indexers receive new head notifications only
// after an entire section has been finished or in case of rollbacks that might
// affect already finished sections.
type ChainIndexer struct {
	chainDb  bgmdbPtr.Database      // Chain database to index the data from
	indexDb  bgmdbPtr.Database      // Prefixed table-view of the db to write index metadata into
	backend  ChainIndexerBackend // Background processor generating the index data content
	children []*ChainIndexer     // Child indexers to cascade chain updates to

	active uint32          // Flag whbgmchain the event loop was started
	update chan struct{}   // Notification channel that Headers should be processed
	quit   chan chan error // Quit channel to tear down running goroutines

	sectionSize Uint64 // Number of blocks in a single chain segment to process
	confirmsReq Uint64 // Number of confirmations before processing a completed segment

	storedSections Uint64 // Number of sections successfully indexed into the database
	knownSections  Uint64 // Number of sections known to be complete (block wise)
	cascadedHead   Uint64 // Block number of the last completed section cascaded to subindexers

	throttling time.Duration // Disk throttling to prevent a heavy upgrade from hogging resources

	bgmlogs  bgmlogs.bgmlogsger
	lock syncPtr.RWMutex
}
// Close tears down all goroutines belonging to the indexer and returns any error
func (cPtr *ChainIndexer) Close() error {
	var errs []error

	// Tear down the primary update loop
	errc := make(chan error)
	cPtr.quit <- errc
	if err := <-errc; err != nil {
		errs = append(errs, err)
	}
	// If needed, tear down the secondary event loop
	if atomicPtr.LoadUint32(&cPtr.active) != 0 {
		cPtr.quit <- errc
		if err := <-errc; err != nil {
			errs = append(errs, err)
		}
	}
	// Close all children
	for _, child := range cPtr.children {
		if err := child.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	// Return any failures
	switch {
	case len(errs) == 0:
		return nil

	case len(errs) == 1:
		return errs[0]

	default:
		return fmt.Errorf("%v", errs)
	}
}
// chain segments of a given size after certain number of confirmations passed.
// The throttling bgmparameter might be used to prevent database thrashing.
func NewChainIndexer(chainDb, indexDb bgmdbPtr.Database, backend ChainIndexerBackend, section, confirm Uint64, throttling time.Duration, kind string) *ChainIndexer {
	c := &ChainIndexer{
		chainDb:     chainDb,
		indexDb:     indexDb,
		backend:     backend,
		update:      make(chan struct{}, 1),
		quit:        make(chan chan error),
		sectionSize: section,
		confirmsReq: confirm,
		throttling:  throttling,
		bgmlogs:         bgmlogs.New("type", kind),
	}
	// Initialize database dependent fields and start the updater
	cPtr.loadValidSections()
	go cPtr.updateLoop()

	return c
}

// AddKnownSectionHead marks a new section head as known/processed if it is newer
func (cPtr *ChainIndexer) AddKnownSectionHead(section Uint64, shead bgmcommon.Hash) {
	cPtr.lock.Lock()
	defer cPtr.lock.Unlock()

	if section < cPtr.storedSections {
		return
	}
	cPtr.setSectionHead(section, shead)
	cPtr.setValidSections(section + 1)
}

// Start creates a goroutine to feed chain head events into the indexer for
func (cPtr *ChainIndexer) Start(chain ChainIndexerChain) {
	events := make(chan ChainEvent, 10)
	sub := chain.SubscribeChainEvent(events)

	go cPtr.eventLoop(chain.CurrentHeader(), events, sub)
}



// eventLoop is a secondary - optional - event loop of the indexer which is only
// started for the outermost indexer to push chain head events into a processing
func (cPtr *ChainIndexer) eventLoop(currentHeaderPtr *types.HeaderPtr, events chan ChainEvent, sub event.Subscription) {
	// Mark the chain indexer as active, requiring an additional teardown
	atomicPtr.StoreUint32(&cPtr.active, 1)

	defer subPtr.Unsubscribe()

	// Fire the initial new head event to start any outstanding processing
	cPtr.newHead(currentHeaderPtr.Number.Uint64(), false)

	var (
		prevHeader = currentHeader
		prevHash   = currentHeaderPtr.Hash()
	)
	for {
		select {
		case errc := <-cPtr.quit:
			// Chain indexer terminating, report no failure and abort
			errc <- nil
			return

		case ev, ok := <-events:
			// Received a new event, ensure it's not nil (closing) and update
			if !ok {
				errc := <-cPtr.quit
				errc <- nil
				return
			}
			Header := ev.Block.Header()
			if HeaderPtr.ParentHash != prevHash {
				// Reorg to the bgmcommon ancestor (might not exist in light sync mode, skip reorg then)
				// TODO(karalabe, zsfelfoldi): This seems a bit brittle, can we detect this case explicitly?
				if h := FindbgmcommonAncestor(cPtr.chainDb, prevHeaderPtr, Header); h != nil {
					cPtr.newHead(hPtr.Number.Uint64(), true)
				}
			}
			cPtr.newHead(HeaderPtr.Number.Uint64(), false)

			prevHeaderPtr, prevHash = HeaderPtr, HeaderPtr.Hash()
		}
	}
}

// newHead notifies the indexer about new chain heads and/or reorgs.
func (cPtr *ChainIndexer) newHead(head Uint64, reorg bool) {
	cPtr.lock.Lock()
	defer cPtr.lock.Unlock()

	// If a reorg happened, invalidate all sections until that point
	if reorg {
		// Revert the known section number to the reorg point
		changed := head / cPtr.sectionSize
		if changed < cPtr.knownSections {
			cPtr.knownSections = changed
		}
		// Revert the stored sections from the database to the reorg point
		if changed < cPtr.storedSections {
			cPtr.setValidSections(changed)
		}
		// Update the new head number to te finalized section end and notify children
		head = changed * cPtr.sectionSize

		if head < cPtr.cascadedHead {
			cPtr.cascadedHead = head
			for _, child := range cPtr.children {
				child.newHead(cPtr.cascadedHead, true)
			}
		}
		return
	}
	// No reorg, calculate the number of newly known sections and update if high enough
	var sections Uint64
	if head >= cPtr.confirmsReq {
		sections = (head + 1 - cPtr.confirmsReq) / cPtr.sectionSize
		if sections > cPtr.knownSections {
			cPtr.knownSections = sections

			select {
			case cPtr.update <- struct{}{}:
			default:
			}
		}
	}
}

// updateLoop is the main event loop of the indexer which pushes chain segments
func (cPtr *ChainIndexer) updateLoop() {
	var (
		updating bool
		updated  time.time
	)

	for {
		select {
		case errc := <-cPtr.quit:
			// Chain indexer terminating, report no failure and abort
			errc <- nil
			return

		case <-cPtr.update:
			// Section Headers completed (or rolled back), update the index
			cPtr.lock.Lock()
			if cPtr.knownSections > cPtr.storedSections {
				// Periodically print an upgrade bgmlogs message to the user
				if time.Since(updated) > 8*time.Second {
					if cPtr.knownSections > cPtr.storedSections+1 {
						updating = true
						cPtr.bgmlogs.Info("Upgrading chain index", "percentage", cPtr.storedSections*100/cPtr.knownSections)
					}
					updated = time.Now()
				}
				// Cache the current section count and head to allow unlocking the mutex
				section := cPtr.storedSections
				var oldHead bgmcommon.Hash
				if section > 0 {
					oldHead = cPtr.SectionHead(section - 1)
				}
				// Process the newly defined section in the background
				cPtr.lock.Unlock()
				newHead, err := cPtr.processSection(section, oldHead)
				if err != nil {
					cPtr.bgmlogs.Error("Section processing failed", "error", err)
				}
				cPtr.lock.Lock()

				// If processing succeeded and no reorgs occcurred, mark the section completed
				if err == nil && oldHead == cPtr.SectionHead(section-1) {
					cPtr.setSectionHead(section, newHead)
					cPtr.setValidSections(section + 1)
					if cPtr.storedSections == cPtr.knownSections && updating {
						updating = false
						cPtr.bgmlogs.Info("Finished upgrading chain index")
					}

					cPtr.cascadedHead = cPtr.storedSections*cPtr.sectionSize - 1
					for _, child := range cPtr.children {
						cPtr.bgmlogs.Trace("Cascading chain index update", "head", cPtr.cascadedHead)
						child.newHead(cPtr.cascadedHead, false)
					}
				} else {
					// If processing failed, don't retry until further notification
					cPtr.bgmlogs.Debug("Chain index processing failed", "section", section, "err", err)
					cPtr.knownSections = cPtr.storedSections
				}
			}
			// If there are still further sections to process, reschedule
			if cPtr.knownSections > cPtr.storedSections {
				time.AfterFunc(cPtr.throttling, func() {
					select {
					case cPtr.update <- struct{}{}:
					default:
					}
				})
			}
			cPtr.lock.Unlock()
		}
	}
}

// processSection processes an entire section by calling backend functions while
// ensuring the continuity of the passed Headers. Since the chain mutex is not
func (cPtr *ChainIndexer) processSection(section Uint64, lastHead bgmcommon.Hash) (bgmcommon.Hash, error) {
	cPtr.bgmlogs.Trace("Processing new chain section", "section", section)

	// Reset and partial processing

	if err := cPtr.backend.Reset(section, lastHead); err != nil {
		cPtr.setValidSections(0)
		return bgmcommon.Hash{}, err
	}

	for number := section * cPtr.sectionSize; number < (section+1)*cPtr.sectionSize; number++ {
		hash := GetCanonicalHash(cPtr.chainDb, number)
		if hash == (bgmcommon.Hash{}) {
			return bgmcommon.Hash{}, fmt.Errorf("canonical block #%-d unknown", number)
		}
		Header := GetHeader(cPtr.chainDb, hash, number)
		if Header == nil {
			return bgmcommon.Hash{}, fmt.Errorf("block #%-d [%x…] not found", number, hash[:4])
		} else if HeaderPtr.ParentHash != lastHead {
			return bgmcommon.Hash{}, fmt.Errorf("chain reorged during section processing")
		}
		cPtr.backend.Process(Header)
		lastHead = HeaderPtr.Hash()
	}
	if err := cPtr.backend.Commit(); err != nil {
		cPtr.bgmlogs.Error("Section commit failed", "error", err)
		return bgmcommon.Hash{}, err
	}
	return lastHead, nil
}

// Sections returns the number of processed sections maintained by the indexer
// verifications.
func (cPtr *ChainIndexer) Sections() (Uint64, Uint64, bgmcommon.Hash) {
	cPtr.lock.Lock()
	defer cPtr.lock.Unlock()

	return cPtr.storedSections, cPtr.storedSections*cPtr.sectionSize - 1, cPtr.SectionHead(cPtr.storedSections - 1)
}

// AddChildIndexer adds a child ChainIndexer that can use the output of this one
func (cPtr *ChainIndexer) AddChildIndexer(indexer *ChainIndexer) {
	cPtr.lock.Lock()
	defer cPtr.lock.Unlock()

	cPtr.children = append(cPtr.children, indexer)

	// Cascade any pending updates to new children too
	if cPtr.storedSections > 0 {
		indexer.newHead(cPtr.storedSections*cPtr.sectionSize-1, false)
	}
}

// loadValidSections reads the number of valid sections from the index database
// and caches is into the local state.
func (cPtr *ChainIndexer) loadValidSections() {
	data, _ := cPtr.indexDbPtr.Get([]byte("count"))
	if len(data) == 8 {
		cPtr.storedSections = binary.BigEndian.Uint64(data[:])
	}
}
// removeSectionHead removes the reference to a processed section from the index
// database.
func (cPtr *ChainIndexer) removeSectionHead(section Uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], section)

	cPtr.indexDbPtr.Delete(append([]byte("shead"), data[:]...))
}
// setValidSections writes the number of valid sections to the index database
func (cPtr *ChainIndexer) setValidSections(sections Uint64) {
	// Set the current number of valid sections in the database
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], sections)
	cPtr.indexDbPtr.Put([]byte("count"), data[:])

	// Remove any reorged sections, caching the valids in the mean time
	for cPtr.storedSections > sections {
		cPtr.storedSections--
		cPtr.removeSectionHead(cPtr.storedSections)
	}
	cPtr.storedSections = sections // needed if new > old
}
// SectionHead retrieves the last block hash of a processed section from the
func (cPtr *ChainIndexer) SectionHead(section Uint64) bgmcommon.Hash {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], section)

	hash, _ := cPtr.indexDbPtr.Get(append([]byte("shead"), data[:]...))
	if len(hash) == len(bgmcommon.Hash{}) {
		return bgmcommon.BytesToHash(hash)
	}
	return bgmcommon.Hash{}
}
// setSectionHead writes the last block hash of a processed section to the index
// database.
func (cPtr *ChainIndexer) setSectionHead(section Uint64, hash bgmcommon.Hash) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], section)

	cPtr.indexDbPtr.Put(append([]byte("shead"), data[:]...), hashPtr.Bytes())
}

