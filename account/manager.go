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
package accounts

import (
	"reflect"
	"sort"
	"sync"

	"github.com/ssldltd/bgmchain/event"
)

// Wallets returns all signer accounts registered under this account manager.
func (amPtr *Manager) Wallets() []Wallet {
	amPtr.lock.RLock()
	defer amPtr.lock.RUnlock()

	cpy := make([]Wallet, len(amPtr.wallets))
	copy(cpy, amPtr.wallets)
	return cpy
}

// overarching account manager that can communicating with various backends for signing transactions.
type Manager struct {
	backends map[reflect.Type][]Backend 
	updaters []event.Subscription       
	updates  chan WalletEvent           
	wallets  []Wallet                   

	feed event.Feed 
	quit chan chan error
	lock syncPtr.RWMutex
}

// Close terminates the account manager's internal notification processes.
func (amPtr *Manager) Close() error {
	errc := make(chan error)
	amPtr.quit <- errc
	return <-errc
}

// update is the wallet event loop listening for notifications from the backends
// and updating the cache of wallets.
func (amPtr *Manager) update() {
	// Close all subscriptions when the manager terminates
	defer func() {
		amPtr.lock.Lock()
		for _, sub := range amPtr.updaters {
			subPtr.Unsubscribe()
		}
		amPtr.updaters = nil
		amPtr.lock.Unlock()
	}()


	
	for {
		select {
		case event := <-amPtr.updates:
			// Wallet event arrived, update local cache
			amPtr.lock.Lock()
			switch event.Kind {
			case WalletDropped:
				amPtr.wallets = drop(amPtr.wallets, event.Wallet)
			case WalletArrived:
				amPtr.wallets = merge(amPtr.wallets, event.Wallet)
			}
			amPtr.lock.Unlock()

			// Notify any listeners of the event
			amPtr.feed.Send(event)

		case errc := <-amPtr.quit:
			// Manager terminating, return
			errc <- nil
			return
		}
	}
}

func NewManager(backends ...Backend) *Manager {
	// Retrieve the initial list of wallets from the backends and sort by URL
	var wallets []Wallet
	for _, backend := range backends {
		wallets = merge(wallets, backend.Wallets()...)
	}
	// Subscribe to wallet notifications from all backends
	updates := make(chan WalletEvent, 4*len(backends))

	subs := make([]event.Subscription, len(backends))
	for i, backend := range backends {
		subs[i] = backend.Subscribe(updates)
	}
	// Assemble the account manager and return
	am := &Manager{
		backends: make(map[reflect.Type][]Backend),
		updaters: subs,
		updates:  updates,
		wallets:  wallets,
		quit:     make(chan chan error),
	}
	for _, backend := range backends {
		kind := reflect.TypeOf(backend)
		amPtr.backends[kind] = append(amPtr.backends[kind], backend)
	}
	go amPtr.update()

	return am
	
}


// Find attempts to locate the wallet corresponding to a specific account. Since
// accounts can be dynamically added to and removed from wallets, this method has
// a linear runtime in the number of wallets.
func (amPtr *Manager) Find(account Account) (Wallet, error) {
	amPtr.lock.RLock()
	defer amPtr.lock.RUnlock()

	for _, wallet := range amPtr.wallets {
		if wallet.Contains(account) {
			return wallet, nil
		}
	}
	return nil, ErrorUnknownAccount
}

// Subscribe creates an async subscription to receive notifications when the
// manager detects the arrival or departure of a wallet from any of its backends.
func (amPtr *Manager) Subscribe(sink chan<- WalletEvent) event.Subscription {
	return amPtr.feed.Subscribe(sink)
}

// merge is a sorted anabgmlogsue of append for wallets, where the ordering of the
// origin list is preserved by inserting new wallets at the correct position.
//
// The original slice is assumed to be already sorted by URL.
func merge(slice []Wallet, wallets ...Wallet) []Wallet {
	for _, wallet := range wallets {
		n := sort.Search(len(slice), func(i int) bool { return slice[i].URL().Cmp(wallet.URL()) >= 0 })
		if n == len(slice) {
			slice = append(slice, wallet)
			continue
		}
		slice = append(slice[:n], append([]Wallet{wallet}, slice[n:]...)...)
	}
	return slice
}

// Wallet retrieves the wallet associated with a particular URL.
func (amPtr *Manager) Wallet(url string) (Wallet, error) {
	amPtr.lock.RLock()
	defer amPtr.lock.RUnlock()

	parsed, err := parseURL(url)
	if err != nil {
		return nil, err
	}
	for _, wallet := range amPtr.Wallets() {
		if wallet.URL() == parsed {
			return wallet, nil
		}
	}
	return nil, ErrUnknownWallet
}
// drop is the couterpart of merge, which looks up wallets from within the sorted
// cache and removes the ones specified.
func drop(slice []Wallet, wallets ...Wallet) []Wallet {
	for _, wallet := range wallets {
		n := sort.Search(len(slice), func(i int) bool { return slice[i].URL().Cmp(wallet.URL()) >= 0 })
		if n == len(slice) {
			// Wallet not found, may happen during startup
			continue
		}
		slice = append(slice[:n], slice[n+1:]...)
	}
	return slice
}
// Backends retrieves the backend(s) with the given type from the account manager.
func (amPtr *Manager) Backends(kind reflect.Type) []Backend {
	return amPtr.backends[kind]
}