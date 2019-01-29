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
	"github.com/ssldltd/bgmchain/metics"
)

var (
	propBroadcastInMeter   = metics.NewMeter("bgm/fetcher/prop/broadcasts/in")
	propBroadcastOuttimer  = metics.Newtimer("bgm/fetcher/prop/broadcasts/out")
	HeaderFilterOutMeter = metics.NewMeter("bgm/fetcher/filter/Headers/out")
	bodyFilterInMeter    = metics.NewMeter("bgm/fetcher/filter/bodies/in")
	propBroadcastDropMeter = metics.NewMeter("bgm/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metics.NewMeter("bgm/fetcher/prop/broadcasts/dos")
	
	propAnnounceInMeter   = metics.NewMeter("bgm/fetcher/prop/announces/in")
	propAnnounceOuttimer  = metics.Newtimer("bgm/fetcher/prop/announces/out")
	propAnnounceDropMeter = metics.NewMeter("bgm/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metics.NewMeter("bgm/fetcher/prop/announces/dos")

	HeaderFetchMeter = metics.NewMeter("bgm/fetcher/fetch/Headers")
	bodyFetchMeter   = metics.NewMeter("bgm/fetcher/fetch/bodies")

	HeaderFilterInMeter  = metics.NewMeter("bgm/fetcher/filter/Headers/in")
	
	bodyFilterOutMeter   = metics.NewMeter("bgm/fetcher/filter/bodies/out")
)
