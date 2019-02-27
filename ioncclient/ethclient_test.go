// Copyright 2016 The go-ionchain Authors
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

package ioncclient

import "github.com/ionchain/ionchain-core"

// Verify that Client implements the ionchain interfaces.
var (
	_ = ionchain.ChainReader(&Client{})
	_ = ionchain.TransactionReader(&Client{})
	_ = ionchain.ChainStateReader(&Client{})
	_ = ionchain.ChainSyncReader(&Client{})
	_ = ionchain.ContractCaller(&Client{})
	_ = ionchain.GasEstimator(&Client{})
	_ = ionchain.GasPricer(&Client{})
	_ = ionchain.LogFilterer(&Client{})
	_ = ionchain.PendingStateReader(&Client{})
	// _ = ionchain.PendingStateEventer(&Client{})
	_ = ionchain.PendingContractCaller(&Client{})
)
