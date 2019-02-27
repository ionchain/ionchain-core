// Copyright 2015 The go-ionchain Authors
// This file is part of the go-ionchain library.
//
// The go-ionchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ionchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ionchain library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ionchain/ionchain-core/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("ionc/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("ionc/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("ionc/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("ionc/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("ionc/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("ionc/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("ionc/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("ionc/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("ionc/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("ionc/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("ionc/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("ionc/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("ionc/downloader/states/in")
	stateDropMeter = metrics.NewMeter("ionc/downloader/states/drop")
)
