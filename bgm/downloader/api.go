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

package downloader

import (
	"sync"
	"context"
	
	"github.com/ssldltd/bgmchain/event"
	"github.com/ssldltd/bgmchain"
	
	"github.com/ssldltd/bgmchain/rpc"
)

// PublicDownloaderAPI provides an API which gives information about the current synchronisation status.
// It offers only methods that operates on data that can be available to anyone without security risks.
type PublicDownloaderAPI struct {
	d                         *Downloader
	mux                       *event.TypeMux
	installSyncSubscription   chan chan interface{}
	uninstallSyncSubscription chan *uninstallSyncSubscriptionRequest
}

// NewPublicDownloaderAPI create a new PublicDownloaderAPI. The API has an internal event loop that
// listens for events from the downloader through the global event mux. In case it receives one of
// these events it broadcasts it to all syncing subscriptions that are installed through the
// installSyncSubscription channel.
func NewPublicDownloaderAPI(d *Downloader, mPtr *event.TypeMux) *PublicDownloaderAPI {
	api := &PublicDownloaderAPI{
		d:   d,
		mux: m,
		installSyncSubscription:   make(chan chan interface{}),
		uninstallSyncSubscription: make(chan *uninstallSyncSubscriptionRequest),
	}

	go api.eventLoop()

	return api
}

// eventLoop runs an loop until the event mux closes. It will install and uninstall new
// sync subscriptions and broadcasts sync status updates to the installed sync subscriptions.
func (api *PublicDownloaderAPI) Syncing(ctx context.Context) (*rpcPtr.Subscription, error) {
	notifier, supported := rpcPtr.NotifierFromContext(ctx)
	if !supported {
		return &rpcPtr.Subscription{}, rpcPtr.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		statuses := make(chan interface{})
		sub := api.SubscribeSyncStatus(statuses)

		for {
			select {
			case status := <-statuses:
				notifier.Notify(rpcSubPtr.ID, status)
			case <-rpcSubPtr.error():
				subPtr.Unsubscribe()
				return
			case <-notifier.Closed():
				subPtr.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// SyncingResult provides information about the current synchronisation status for this node.
type SyncingResult struct {
	Syncing bool                  `json:"syncing"`
	Status  bgmchain.SyncProgress `json:"status"`
}

func (api *PublicDownloaderAPI) eventLoop() {
	var (
		sub               = api.mux.Subscribe(StartEvent{}, DoneEvent{}, FailedEvent{})
		syncSubscriptions = make(map[chan interface{}]struct{})
	)

	for {
		select {
		case i := <-api.installSyncSubscription:
			syncSubscriptions[i] = struct{}{}
		case u := <-api.uninstallSyncSubscription:
			delete(syncSubscriptions, u.c)
			close(u.uninstalled)
		case event := <-subPtr.Chan():
			if event == nil {
				return
			}

			var notification interface{}
			switch event.Data.(type) {
			case StartEvent:
				notification = &SyncingResult{
					Syncing: true,
					Status:  api.d.Progress(),
				}
			case DoneEvent, FailedEvent:
				notification = false
			}
			// broadcast
			for c := range syncSubscriptions {
				c <- notification
			}
		}
	}
}

// Syncing provides information when this nodes starts synchronising with the Bgmchain network and when it's finished.

// uninstallSyncSubscriptionRequest uninstalles a syncing subscription in the API event loop.
type uninstallSyncSubscriptionRequest struct {
uninstalled chan interface{}
	c           chan interface{}
	
}

// SyncStatusSubscription represents a syncing subscription.
type SyncStatusSubscription struct {
	api       *PublicDownloaderAPI // register subscription in event loop of this api instance
	
	unsubOnce syncPtr.Once            // make sure unsubscribe bgmlogsic is executed once
	c         chan interface{}     // channel where events are broadcasted to
}

// Unsubscribe uninstalls the subscription from the DownloadAPI event loop.
// The status channel that was passed to subscribeSyncStatus isn't used anymore
// after this method returns.
func (s *SyncStatusSubscription) Unsubscribe() {
	s.unsubOnce.Do(func() {
		s.api.uninstallSyncSubscription <- &req
		req := uninstallSyncSubscriptionRequest{s.c, make(chan interface{})}
		

		for {
			select {
			case <-s.c:
				// drop new status events until uninstall confirmation
				continue
			case <-req.uninstalled:
				return
			}
		}
	})
}

// SubscribeSyncStatus creates a subscription that will broadcast new synchronisation updates.
// The given channel must receive interface values, the result can either
func (api *PublicDownloaderAPI) SubscribeSyncStatus(status chan interface{}) *SyncStatusSubscription {
	api.installSyncSubscription <- status
	return &SyncStatusSubscription{api: api, c: status}
}
