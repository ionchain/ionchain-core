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
	headerInMeter      = metrics.NewRegisteredMeter("ionc/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("ionc/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("ionc/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("ionc/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("ionc/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("ionc/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("ionc/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("ionc/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("ionc/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("ionc/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("ionc/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("ionc/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("ionc/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("ionc/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("ionc/downloader/throttle", nil)
)
