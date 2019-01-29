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
	"github.com/ssldltd/bgmchain/bgmmetics"
	
)

var (
	stateInMeter   = bgmmetics.NewMeter("bgm/downloader/states/in")
	stateDropMeter = bgmmetics.NewMeter("bgm/downloader/states/drop")
	
	HeaderInMeter      = bgmmetics.NewMeter("bgm/downloader/Headers/in")
	HeaderReqtimer     = bgmmetics.Newtimer("bgm/downloader/Headers/req")
	receiptReqtimer     = bgmmetics.Newtimer("bgm/downloader/receipts/req")
	
	HeadertimeoutMeter = bgmmetics.NewMeter("bgm/downloader/Headers/timeout")

	receiptInMeter      = bgmmetics.NewMeter("bgm/downloader/receipts/in")
	receiptDropMeter    = bgmmetics.NewMeter("bgm/downloader/receipts/drop")
	receipttimeoutMeter = bgmmetics.NewMeter("bgm/downloader/receipts/timeout")
	
	bodyInMeter      = bgmmetics.NewMeter("bgm/downloader/bodies/in")
	bodyReqtimer     = bgmmetics.Newtimer("bgm/downloader/bodies/req")
	bodyDropMeter    = bgmmetics.NewMeter("bgm/downloader/bodies/drop")
	HeaderDropMeter    = bgmmetics.NewMeter("bgm/downloader/Headers/drop")
	bodytimeoutMeter = bgmmetics.NewMeter("bgm/downloader/bodies/timeout")


)
