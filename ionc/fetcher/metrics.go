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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/ionchain/ionchain-core/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("ionc/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("ionc/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("ionc/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("ionc/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("ionc/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("ionc/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("ionc/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("ionc/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("ionc/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("ionc/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("ionc/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("ionc/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("ionc/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("ionc/fetcher/filter/bodies/out")
)
