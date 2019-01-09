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
	"github.com/ssldltd/bgmchain/metrics"
)

var (
	propBroadcastInMeter   = metrics.NewMeter("bgm/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("bgm/fetcher/prop/broadcasts/out")
	headerFilterOutMeter = metrics.NewMeter("bgm/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("bgm/fetcher/filter/bodies/in")
	propBroadcastDropMeter = metrics.NewMeter("bgm/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("bgm/fetcher/prop/broadcasts/dos")
	
	propAnnounceInMeter   = metrics.NewMeter("bgm/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("bgm/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("bgm/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("bgm/fetcher/prop/announces/dos")

	headerFetchMeter = metrics.NewMeter("bgm/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("bgm/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("bgm/fetcher/filter/headers/in")
	
	bodyFilterOutMeter   = metrics.NewMeter("bgm/fetcher/filter/bodies/out")
)
