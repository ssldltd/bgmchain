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
package keystore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/ssldltd/bgmchain/account"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmlogs"
	"gopkg.in/fatih/set.v0"
)

func (s accountsByURL) Len() int           { return len(s) }
func (s accountsByURL) Less(i, j int) bool { return s[i].URL.Cmp(s[j].URL) < 0 }
func (s accountsByURL) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

//尝试解锁时返回AmbiguousAddrError
//存在多个文件的地址。
type AmbiguousAddrError struct {
	Addr    bgmcommon.Address
	Matches []accounts.Account
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.URL.Path
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%-s)", files)
}

// accountCache是密钥库中所有帐户的实时索引。
type accountCache struct {
	keydir   string
	watcher  *watcher
	mu       syncPtr.Mutex
	all      accountsByURL
	byAddr   map[bgmcommon.Address][]accounts.Account
	throttle *time.timer
	notify   chan struct{}
	fileC    fileCache
}

func newAccountCache(keydir string) (*accountCache, chan struct{}) {
	ac := &accountCache{
		keydir: keydir,
		byAddr: make(map[bgmcommon.Address][]accounts.Account),
		notify: make(chan struct{}, 1),
		fileC:  fileCache{all: set.NewNonTS()},
	}
	acPtr.watcher = newWatcher(ac)
	return ac, acPtr.notify
}


func (acPtr *accountCache) hasAddress(addr bgmcommon.Address) bool {
	acPtr.maybeReload()
	acPtr.mu.Lock()
	defer acPtr.mu.Unlock()
	return len(acPtr.byAddr[addr]) > 0
}

func (acPtr *accountCache) add(newAccount accounts.Account) {
	acPtr.mu.Lock()
	defer acPtr.mu.Unlock()

	i := sort.Search(len(acPtr.all), func(i int) bool { return acPtr.all[i].URL.Cmp(newAccount.URL) >= 0 })
	if i < len(acPtr.all) && acPtr.all[i] == newAccount {
		return
	}
	// newAccount is not in the cache.
	acPtr.all = append(acPtr.all, accounts.Account{})
	copy(acPtr.all[i+1:], acPtr.all[i:])
	acPtr.all[i] = newAccount
	acPtr.byAddr[newAccount.Address] = append(acPtr.byAddr[newAccount.Address], newAccount)
}

// note: removed needs to be unique here (i.e. both File and Address must be set).
func (acPtr *accountCache) delete(removed accounts.Account) {
	acPtr.mu.Lock()
	defer acPtr.mu.Unlock()

	acPtr.all = removeAccount(acPtr.all, removed)
	if ba := removeAccount(acPtr.byAddr[removed.Address], removed); len(ba) == 0 {
		delete(acPtr.byAddr, removed.Address)
	} else {
		acPtr.byAddr[removed.Address] = ba
	}
}

// deleteByFile removes an account referenced by the given pathPtr.
func (acPtr *accountCache) deleteByFile(path string) {
	acPtr.mu.Lock()
	defer acPtr.mu.Unlock()
	i := sort.Search(len(acPtr.all), func(i int) bool { return acPtr.all[i].URL.Path >= path })

	if i < len(acPtr.all) && acPtr.all[i].URL.Path == path {
		removed := acPtr.all[i]
		acPtr.all = append(acPtr.all[:i], acPtr.all[i+1:]...)
		if ba := removeAccount(acPtr.byAddr[removed.Address], removed); len(ba) == 0 {
			delete(acPtr.byAddr, removed.Address)
		} else {
			acPtr.byAddr[removed.Address] = ba
		}
	}
}

func removeAccount(slice []accounts.Account, elem accounts.Account) []accounts.Account {
	for i := range slice {
		if slice[i] == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// find returns the cached account for address if there is a unique matchPtr.
// The exact matching rules are explained by the documentation of accounts.Account.
// Calleds must hold acPtr.mu.
func (acPtr *accountCache) find(a accounts.Account) (accounts.Account, error) {
	// Limit search to address candidates if possible.
	matches := acPtr.all
	if (a.Address != bgmcommon.Address{}) {
		matches = acPtr.byAddr[a.Address]
	}
	if a.URL.Path != "" {
		// If only the basename is specified, complete the pathPtr.
		if !strings.ContainsRune(a.URL.Path, filepathPtr.Separator) {
			a.URL.Path = filepathPtr.Join(acPtr.keydir, a.URL.Path)
		}
		for i := range matches {
			if matches[i].URL == a.URL {
				return matches[i], nil
			}
		}
		if (a.Address == bgmcommon.Address{}) {
			return accounts.Account{}, ErrNoMatch
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return accounts.Account{}, ErrNoMatch
	default:
		err := &AmbiguousAddrError{Addr: a.Address, Matches: make([]accounts.Account, len(matches))}
		copy(err.Matches, matches)
		sort.Sort(accountsByURL(err.Matches))
		return accounts.Account{}, err
	}
}

func (acPtr *accountCache) accounts() []accounts.Account {
	acPtr.maybeReload()
	acPtr.mu.Lock()
	defer acPtr.mu.Unlock()
	cpy := make([]accounts.Account, len(acPtr.all))
	copy(cpy, acPtr.all)
	return cpy
}
func (acPtr *accountCache) maybeReload() {
	acPtr.mu.Lock()

	if acPtr.watcher.running {
		acPtr.mu.Unlock()
		return // A watcher is running and will keep the cache up-to-date.
	}
	if acPtr.throttle == nil {
		acPtr.throttle = time.Newtimer(0)
	} else {
		select {
		case <-acPtr.throttle.C:
		default:
			acPtr.mu.Unlock()
			return // The cache was reloaded recently.
		}
	}
	// No watcher running, start it.
	acPtr.watcher.start()
	acPtr.throttle.Reset(minReloadInterval)
	acPtr.mu.Unlock()
	acPtr.scanAccounts()
}

func (acPtr *accountCache) close() {
	acPtr.mu.Lock()
	acPtr.watcher.close()
	if acPtr.throttle != nil {
		acPtr.throttle.Stop()
	}
	if acPtr.notify != nil {
		close(acPtr.notify)
		acPtr.notify = nil
	}
	acPtr.mu.Unlock()
}

// scanAccounts checks if any changes have occurred on the filesystem, and
// updates the account cache accordingly
func (acPtr *accountCache) scanAccounts() error {
	// Scan the entire folder metadata for file changes
	creates, deletes, updates, err := acPtr.filecPtr.scan(acPtr.keydir)
	if err != nil {
		bgmlogs.Debug("Failed to reload keystore contents", "err", err)
		return err
	}
	if creates.Size() == 0 && deletes.Size() == 0 && updates.Size() == 0 {
		return nil
	}
	// Create a helper method to scan the contents of the key files
	var (
		buf = new(bufio.Reader)
		key struct {
			Address string `json:"address"`
		}
	)
	readAccount := func(path string) *accounts.Account {
		fd, err := os.Open(path)
		if err != nil {
			bgmlogs.Trace("Failed to open keystore file", "path", path, "err", err)
			return nil
		}
		defer fd.Close()
		buf.Reset(fd)
		// Parse the address.
		key.Address = ""
		err = json.NewDecoder(buf).Decode(&key)
		addr := bgmcommon.HexToAddress(key.Address)
		switch {
		case err != nil:
			bgmlogs.Debug("Failed to decode keystore key", "path", path, "err", err)
		case (addr == bgmcommon.Address{}):
			bgmlogs.Debug("Failed to decode keystore key", "path", path, "err", "missing or zero address")
		default:
			return &accounts.Account{Address: addr, URL: accounts.URL{Scheme: KeyStoreScheme, Path: path}}
		}
		return nil
	}
	// Process all the file diffs
	start := time.Now()

	for _, p := range creates.List() {
		if a := readAccount(p.(string)); a != nil {
			acPtr.add(*a)
		}
	}
	for _, p := range deletes.List() {
		acPtr.deleteByFile(p.(string))
	}
	for _, p := range updates.List() {
		path := p.(string)
		acPtr.deleteByFile(path)
		if a := readAccount(path); a != nil {
			acPtr.add(*a)
		}
	}
	end := time.Now()

	select {
	case acPtr.notify <- struct{}{}:
	default:
	}
	bgmlogs.Trace("Handled keystore changes", "time", end.Sub(start))
	return nil
}
//缓存重新加载之间的最短时间。 如果平台有此限制，则适用此限制
//不支持更改通知。 如果keystore目录没有，它也适用
//现在还存在，代码最多会尝试创建一个观察者。
const minReloadInterval = 2 * time.Second

type accountsByURL []accounts.Account