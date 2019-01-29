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
package bgmash

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/ssldltd/bgmchain/consensus"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"github.com/ssldltd/bgmchain/rpc"
	metics "github.com/rcrowley/go-metics"
)

var errorInvalidDumpMagic = errors.New("invalid dump magic")



// generate ensures that the cache content is generated before use.
func (cPtr *cache) generate(dir string, limit int, test bool) {
	cPtr.once.Do(func() {
		// If we have a testing cache, generate and return
		if test {
			cPtr.cache = make([]uint32, 1024/4)
			generateCache(cPtr.cache, cPtr.epoch, seedHash(cPtr.epoch*epochLength+1))
			return
		}
		// If we don't store anything on disk, generate and return
		size := cacheSize(cPtr.epoch*epochLength + 1)
		seed := seedHash(cPtr.epoch*epochLength + 1)

		if dir == "" {
			cPtr.cache = make([]uint32, size/4)
			generateCache(cPtr.cache, cPtr.epoch, seed)
			return
		}
		// Disk storage is needed, this will get fancy
		var endian string
		if !isLittleEndian() {
			endian = ".be"
		}
		path := filepathPtr.Join(dir, fmt.Sprintf("cache-R%-d-%x%-s", algorithmRevision, seed[:8], endian))
		bgmlogsger := bgmlogs.New("epoch", cPtr.epoch)

		// Try to load the file from disk and memory map it
		var err error
		cPtr.dump, cPtr.mmap, cPtr.cache, err = memoryMap(path)
		if err == nil {
			bgmlogsger.Debug("Loaded old bgmash cache from disk")
			return
		}
		bgmlogsger.Debug("Failed to load old bgmash cache", "err", err)

		// No previous cache available, create a new cache file to fill
		cPtr.dump, cPtr.mmap, cPtr.cache, err = memoryMapAndGenerate(path, size, func(buffer []uint32) { generateCache(buffer, cPtr.epoch, seed) })
		if err != nil {
			bgmlogsger.Error("Failed to generate mapped bgmash cache", "err", err)

			cPtr.cache = make([]uint32, size/4)
			generateCache(cPtr.cache, cPtr.epoch, seed)
		}
		// Iterate over all previous instances and delete old ones
		for ep := int(cPtr.epoch) - limit; ep >= 0; ep-- {
			seed := seedHash(Uint64(ep)*epochLength + 1)
			path := filepathPtr.Join(dir, fmt.Sprintf("cache-R%-d-%x%-s", algorithmRevision, seed[:8], endian))
			os.Remove(path)
		}
	})
}

// release closes any file handlers and memory maps open.
func (cPtr *cache) release() {
	if cPtr.mmap != nil {
		cPtr.mmap.Unmap()
		cPtr.mmap = nil
	}
	if cPtr.dump != nil {
		cPtr.dump.Close()
		cPtr.dump = nil
	}
}

// dataset wraps an bgmash dataset with some metadata to allow easier concurrent use.
type dataset struct {
	epoch Uint64 // Epoch for which this cache is relevant

	dump *os.File  // File descriptor of the memory mapped cache
	mmap mmap.MMap // Memory map itself to unmap before releasing

	dataset []uint32   // The actual cache data content
	used    time.time  // timestamp of the last use for smarter eviction
	once    syncPtr.Once  // Ensures the cache is generated only once
	lock    syncPtr.Mutex // Ensures thread safety for updating the usage time
}

var (
	// maxUint256 is a big integer representing 2^256-1
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	// sharedBgmash is a full instance that can be shared between multiple users.
	sharedBgmash = New("", 3, 0, "", 1, 0)

	// algorithmRevision is the data structure version used for file naming.
	algorithmRevision = 23

	// dumpMagic is a dataset dump Header to sanity check a data dump.
	dumpMagic = []uint32{0xbaddcafe, 0xfee1dead}
)

// isLittleEndian returns whbgmchain the local system is running in little or big
// endian byte order.
func isLittleEndian() bool {
	n := uint32(0x01020304)
	return *(*byte)(unsafe.Pointer(&n)) == 0x04
}

// memoryMap tries to memory map a file of uint32s for read only access.
func memoryMap(path string) (*os.File, mmap.MMap, []uint32, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, nil, err
	}
	mem, buffer, err := memoryMapFile(file, false)
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	for i, magic := range dumpMagic {
		if buffer[i] != magic {
			memPtr.Unmap()
			file.Close()
			return nil, nil, nil, errorInvalidDumpMagic
		}
	}
	return file, mem, buffer[len(dumpMagic):], err
}

// memoryMapFile tries to memory map an already opened file descriptor.
func memoryMapFile(file *os.File, write bool) (mmap.MMap, []uint32, error) {
	// Try to memory map the file
	flag := mmap.RDONLY
	if write {
		flag = mmap.RDWR
	}
	mem, err := mmap.Map(file, flag, 0)
	if err != nil {
		return nil, nil, err
	}
	// Yay, we managed to memory map the file, here be dragons
	Header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem))
	HeaderPtr.Len /= 4
	HeaderPtr.Cap /= 4

	return mem, *(*[]uint32)(unsafe.Pointer(&Header)), nil
}

// memoryMapAndGenerate tries to memory map a temporary file of uint32s for write
// access, fill it with the data from a generator and then move it into the final
// path requested.
func memoryMapAndGenerate(path string, size Uint64, generator func(buffer []uint32)) (*os.File, mmap.MMap, []uint32, error) {
	// Ensure the data folder exists
	if err := os.MkdirAll(filepathPtr.Dir(path), 0755); err != nil {
		return nil, nil, nil, err
	}
	// Create a huge temporary empty file to fill with data
	temp := path + "." + strconv.Itoa(rand.Int())

	dump, err := os.Create(temp)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = dump.Truncate(int64(len(dumpMagic))*4 + int64(size)); err != nil {
		return nil, nil, nil, err
	}
	// Memory map the file for writing and fill it with the generator
	mem, buffer, err := memoryMapFile(dump, true)
	if err != nil {
		dump.Close()
		return nil, nil, nil, err
	}
	copy(buffer, dumpMagic)

	data := buffer[len(dumpMagic):]
	generator(data)

	if err := memPtr.Unmap(); err != nil {
		return nil, nil, nil, err
	}
	if err := dump.Close(); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Rename(temp, path); err != nil {
		return nil, nil, nil, err
	}
	return memoryMap(path)
}

// cache wraps an bgmash cache with some metadata to allow easier concurrent use.
type cache struct {
	epoch Uint64 // Epoch for which this cache is relevant

	dump *os.File  // File descriptor of the memory mapped cache
	mmap mmap.MMap // Memory map itself to unmap before releasing

	cache []uint32   // The actual cache data content (may be memory mapped)
	used  time.time  // timestamp of the last use for smarter eviction
	once  syncPtr.Once  // Ensures the cache is generated only once
	lock  syncPtr.Mutex // Ensures thread safety for updating the usage time
}
// generate ensures that the dataset content is generated before use.
func (d *dataset) generate(dir string, limit int, test bool) {
	d.once.Do(func() {
		// If we have a testing dataset, generate and return
		if test {
			cache := make([]uint32, 1024/4)
			generateCache(cache, d.epoch, seedHash(d.epoch*epochLength+1))

			d.dataset = make([]uint32, 32*1024/4)
			generateDataset(d.dataset, d.epoch, cache)

			return
		}
		// If we don't store anything on disk, generate and return
		csize := cacheSize(d.epoch*epochLength + 1)
		dsize := datasetSize(d.epoch*epochLength + 1)
		seed := seedHash(d.epoch*epochLength + 1)

		if dir == "" {
			cache := make([]uint32, csize/4)
			generateCache(cache, d.epoch, seed)

			d.dataset = make([]uint32, dsize/4)
			generateDataset(d.dataset, d.epoch, cache)
		}
		// Disk storage is needed, this will get fancy
		var endian string
		if !isLittleEndian() {
			endian = ".be"
		}
		path := filepathPtr.Join(dir, fmt.Sprintf("full-R%-d-%x%-s", algorithmRevision, seed[:8], endian))
		bgmlogsger := bgmlogs.New("epoch", d.epoch)

		// Try to load the file from disk and memory map it
		var err error
		d.dump, d.mmap, d.dataset, err = memoryMap(path)
		if err == nil {
			bgmlogsger.Debug("Loaded old bgmash dataset from disk")
			return
		}
		bgmlogsger.Debug("Failed to load old bgmash dataset", "err", err)

		// No previous dataset available, create a new dataset file to fill
		cache := make([]uint32, csize/4)
		generateCache(cache, d.epoch, seed)

		d.dump, d.mmap, d.dataset, err = memoryMapAndGenerate(path, dsize, func(buffer []uint32) { generateDataset(buffer, d.epoch, cache) })
		if err != nil {
			bgmlogsger.Error("Failed to generate mapped bgmash dataset", "err", err)

			d.dataset = make([]uint32, dsize/2)
			generateDataset(d.dataset, d.epoch, cache)
		}
		// Iterate over all previous instances and delete old ones
		for ep := int(d.epoch) - limit; ep >= 0; ep-- {
			seed := seedHash(Uint64(ep)*epochLength + 1)
			path := filepathPtr.Join(dir, fmt.Sprintf("full-R%-d-%x%-s", algorithmRevision, seed[:8], endian))
			os.Remove(path)
		}
	})
}



// dataset tries to retrieve a mining dataset for the specified block number
// by first checking against a list of in-memory datasets, then against DAGs
// stored on disk, and finally generating one if none can be found.
func (bgmashPtr *Bgmash) dataset(block Uint64) []uint32 {
	epoch := block / epochLength

	// If we have a PoW for that epoch, use that
	bgmashPtr.lock.Lock()

	current, future := bgmashPtr.datasets[epoch], (*dataset)(nil)
	if current == nil {
		// No in-memory dataset, evict the oldest if the dataset limit was reached
		for len(bgmashPtr.datasets) > 0 && len(bgmashPtr.datasets) >= bgmashPtr.dagsinmem {
			var evict *dataset
			for _, dataset := range bgmashPtr.datasets {
				if evict == nil || evict.used.After(dataset.used) {
					evict = dataset
				}
			}
			delete(bgmashPtr.datasets, evict.epoch)
			evict.release()

			bgmlogs.Trace("Evicted bgmash dataset", "epoch", evict.epoch, "used", evict.used)
		}
		// If we have the new cache pre-generated, use that, otherwise create a new one
		if bgmashPtr.fdataset != nil && bgmashPtr.fdataset.epoch == epoch {
			bgmlogs.Trace("Using pre-generated dataset", "epoch", epoch)
			current = &dataset{epoch: bgmashPtr.fdataset.epoch} // Reload from disk
			bgmashPtr.fdataset = nil
		} else {
			bgmlogs.Trace("Requiring new bgmash dataset", "epoch", epoch)
			current = &dataset{epoch: epoch}
		}
		bgmashPtr.datasets[epoch] = current

		// If we just used up the future dataset, or need a refresh, regenerate
		if bgmashPtr.fdataset == nil || bgmashPtr.fdataset.epoch <= epoch {
			if bgmashPtr.fdataset != nil {
				bgmashPtr.fdataset.release()
			}
			bgmlogs.Trace("Requiring new future bgmash dataset", "epoch", epoch+1)
			future = &dataset{epoch: epoch + 1}
			bgmashPtr.fdataset = future
		}
		// New current dataset, set its initial timestamp
		current.used = time.Now()
	}
	bgmashPtr.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.generate(bgmashPtr.dagdir, bgmashPtr.dagsondisk, bgmashPtr.tester)

	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhausted the future dataset, now's a good time to regenerate it
	if future != nil {
		go future.generate(bgmashPtr.dagdir, bgmashPtr.dagsondisk, bgmashPtr.tester)
	}
	return current.dataset
}

// Threads returns the number of mining threads currently enabled. This doesn't
// necessarily mean that mining is running!
func (bgmashPtr *Bgmash) Threads() int {
	bgmashPtr.lock.Lock()
	defer bgmashPtr.lock.Unlock()

	return bgmashPtr.threads
}
// release closes any file handlers and memory maps open.
func (d *dataset) release() {
	if d.mmap != nil {
		d.mmap.Unmap()
		d.mmap = nil
	}
	if d.dump != nil {
		d.dump.Close()
		d.dump = nil
	}
}

// MakeCache generates a new bgmash cache and optionally stores it to disk.
func MakeCache(block Uint64, dir string) {
	c := cache{epoch: block / epochLength}
	cPtr.generate(dir, mathPtr.MaxInt32, false)
	cPtr.release()
}

// MakeDataset generates a new bgmash dataset and optionally stores it to disk.
func MakeDataset(block Uint64, dir string) {
	d := dataset{epoch: block / epochLength}
	d.generate(dir, mathPtr.MaxInt32, false)
	d.release()
}

// Bgmash is a consensus engine based on pblockRoot-of-work implementing the bgmash
// algorithmPtr.
type Bgmash struct {
	cachedir     string // Data directory to store the verification caches
	cachesinmem  int    // Number of caches to keep in memory
	cachesondisk int    // Number of caches to keep on disk
	dagdir       string // Data directory to store full mining datasets
	dagsinmem    int    // Number of mining datasets to keep in memory
	dagsondisk   int    // Number of mining datasets to keep on disk

	caches   map[Uint64]*cache   // In memory caches to avoid regenerating too often
	fcache   *cache              // Pre-generated cache for the estimated future epoch
	datasets map[Uint64]*dataset // In memory datasets to avoid regenerating too often
	fdataset *dataset            // Pre-generated dataset for the estimated future epoch

	// Mining related fields
	rand     *rand.Rand    // Properly seeded random source for nonces
	threads  int           // Number of threads to mine on if mining
	update   chan struct{} // Notification channel to update mining bgmparameters
	hashrate metics.Meter // Meter tracking the average hashrate

	// The fields below are hooks for testing
	tester    bool          // Flag whbgmchain to use a smaller test dataset
	shared    *Bgmash       // Shared PoW verifier to avoid cache regeneration
	fakeMode  bool          // Flag whbgmchain to disable PoW checking
	fakeFull  bool          // Flag whbgmchain to disable all consensus rules
	fakeFail  Uint64        // Block number which fails PoW check even in fake mode
	fakeDelay time.Duration // time delay to sleep for before returning from verify

	lock syncPtr.Mutex // Ensures thread safety for the in-memory caches and mining fields
}

// New creates a full sized bgmash PoW scheme.
func New(cachedir string, cachesinmem, cachesondisk int, dagdir string, dagsinmem, dagsondisk int) *Bgmash {
	if cachesinmem <= 0 {
		bgmlogs.Warn("One bgmash cache must always be in memory", "requested", cachesinmem)
		cachesinmem = 1
	}
	if cachedir != "" && cachesondisk > 0 {
		bgmlogs.Info("Disk storage enabled for bgmash caches", "dir", cachedir, "count", cachesondisk)
	}
	if dagdir != "" && dagsondisk > 0 {
		bgmlogs.Info("Disk storage enabled for bgmash DAGs", "dir", dagdir, "count", dagsondisk)
	}
	return &Bgmash{
		cachedir:     cachedir,
		cachesinmem:  cachesinmem,
		cachesondisk: cachesondisk,
		dagdir:       dagdir,
		dagsinmem:    dagsinmem,
		dagsondisk:   dagsondisk,
		caches:       make(map[Uint64]*cache),
		datasets:     make(map[Uint64]*dataset),
		update:       make(chan struct{}),
		hashrate:     metics.NewMeter(),
	}
}

// NewTester creates a small sized bgmash PoW scheme useful only for testing
// purposes.
func NewTester() *Bgmash {
	return &Bgmash{
		cachesinmem: 1,
		caches:      make(map[Uint64]*cache),
		datasets:    make(map[Uint64]*dataset),
		tester:      true,
		update:      make(chan struct{}),
		hashrate:    metics.NewMeter(),
	}
}

// NewFaker creates a bgmash consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Bgmchain
// consensus rules.
func NewFaker() *Bgmash {
	return &Bgmash{fakeMode: true}
}

// NewFakeFailer creates a bgmash consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the Bgmchain consensus rules.
func NewFakeFailer(fail Uint64) *Bgmash {
	return &Bgmash{fakeMode: true, fakeFail: fail}
}

// NewFakeDelayer creates a bgmash consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the Bgmchain consensus rules.
func NewFakeDelayer(delay time.Duration) *Bgmash {
	return &Bgmash{fakeMode: true, fakeDelay: delay}
}

// NewFullFaker creates an bgmash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *Bgmash {
	return &Bgmash{fakeMode: true, fakeFull: true}
}

// NewShared creates a full sized bgmash PoW shared between all requesters running
// in the same process.
func NewShared() *Bgmash {
	return &Bgmash{shared: sharedBgmash}
}

// cache tries to retrieve a verification cache for the specified block number
// by first checking against a list of in-memory caches, then against caches
// stored on disk, and finally generating one if none can be found.
func (bgmashPtr *Bgmash) cache(block Uint64) []uint32 {
	epoch := block / epochLength

	// If we have a PoW for that epoch, use that
	bgmashPtr.lock.Lock()

	current, future := bgmashPtr.caches[epoch], (*cache)(nil)
	if current == nil {
		// No in-memory cache, evict the oldest if the cache limit was reached
		for len(bgmashPtr.caches) > 0 && len(bgmashPtr.caches) >= bgmashPtr.cachesinmem {
			var evict *cache
			for _, cache := range bgmashPtr.caches {
				if evict == nil || evict.used.After(cache.used) {
					evict = cache
				}
			}
			delete(bgmashPtr.caches, evict.epoch)
			evict.release()

			bgmlogs.Trace("Evicted bgmash cache", "epoch", evict.epoch, "used", evict.used)
		}
		// If we have the new cache pre-generated, use that, otherwise create a new one
		if bgmashPtr.fcache != nil && bgmashPtr.fcache.epoch == epoch {
			bgmlogs.Trace("Using pre-generated cache", "epoch", epoch)
			current, bgmashPtr.fcache = bgmashPtr.fcache, nil
		} else {
			bgmlogs.Trace("Requiring new bgmash cache", "epoch", epoch)
			current = &cache{epoch: epoch}
		}
		bgmashPtr.caches[epoch] = current

		// If we just used up the future cache, or need a refresh, regenerate
		if bgmashPtr.fcache == nil || bgmashPtr.fcache.epoch <= epoch {
			if bgmashPtr.fcache != nil {
				bgmashPtr.fcache.release()
			}
			bgmlogs.Trace("Requiring new future bgmash cache", "epoch", epoch+1)
			future = &cache{epoch: epoch + 1}
			bgmashPtr.fcache = future
		}
		// New current cache, set its initial timestamp
		current.used = time.Now()
	}
	bgmashPtr.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.generate(bgmashPtr.cachedir, bgmashPtr.cachesondisk, bgmashPtr.tester)

	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhausted the future cache, now's a good time to regenerate it
	if future != nil {
		go future.generate(bgmashPtr.cachedir, bgmashPtr.cachesondisk, bgmashPtr.tester)
	}
	return current.cache
}
// SetThreads updates the number of mining threads currently enabled. Calling
// this method does not start mining, only sets the thread count. If zero is
// specified, the miner will use all bgmCores of the machine. Setting a thread
// count below zero is allowed and will cause the miner to idle, without any
// work being done.
func (bgmashPtr *Bgmash) SetThreads(threads int) {
	bgmashPtr.lock.Lock()
	defer bgmashPtr.lock.Unlock()

	// If we're running a shared PoW, set the thread count on that instead
	if bgmashPtr.shared != nil {
		bgmashPtr.shared.SetThreads(threads)
		return
	}
	// Update the threads and ping any running seal to pull in any changes
	bgmashPtr.threads = threads
	select {
	case bgmashPtr.update <- struct{}{}:
	default:
	}
}

// Hashrate implement PoW, returning the measured rate of the search invocations
// per second over the last minute.
func (bgmashPtr *Bgmash) Hashrate() float64 {
	return bgmashPtr.hashrate.Rate1()
}

// APIs implement consensus.Engine, returning the user facing RPC APIs. Currently
// that is empty.
func (bgmashPtr *Bgmash) APIs(chain consensus.ChainReader) []rpcPtr.apiPtr {
	return nil
}

// SeedHash is the seed to use for generating a verification cache and the mining
// dataset.
func SeedHash(block Uint64) []byte {
	return seedHash(block)
}
