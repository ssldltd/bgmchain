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
	"github.com/ssldltd/bgmchain/bgmmetrics"
	
)

var (
	stateInMeter   = bgmmetrics.NewMeter("bgm/downloader/states/in")
	stateDropMeter = bgmmetrics.NewMeter("bgm/downloader/states/drop")
	
	headerInMeter      = bgmmetrics.NewMeter("bgm/downloader/headers/in")
	headerReqTimer     = bgmmetrics.NewTimer("bgm/downloader/headers/req")
	receiptReqTimer     = bgmmetrics.NewTimer("bgm/downloader/receipts/req")
	
	headerTimeoutMeter = bgmmetrics.NewMeter("bgm/downloader/headers/timeout")

	receiptInMeter      = bgmmetrics.NewMeter("bgm/downloader/receipts/in")
	receiptDropMeter    = bgmmetrics.NewMeter("bgm/downloader/receipts/drop")
	receiptTimeoutMeter = bgmmetrics.NewMeter("bgm/downloader/receipts/timeout")
	
	bodyInMeter      = bgmmetrics.NewMeter("bgm/downloader/bodies/in")
	bodyReqTimer     = bgmmetrics.NewTimer("bgm/downloader/bodies/req")
	bodyDropMeter    = bgmmetrics.NewMeter("bgm/downloader/bodies/drop")
	headerDropMeter    = bgmmetrics.NewMeter("bgm/downloader/headers/drop")
	bodyTimeoutMeter = bgmmetrics.NewMeter("bgm/downloader/bodies/timeout")


)
