// Copyright 2019 The go-ionchain Authors
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

package ionc

import (
	"github.com/ionchain/ionchain-core/core"
	"github.com/ionchain/ionchain-core/core/forkid"
	"github.com/ionchain/ionchain-core/p2p/dnsdisc"
	"github.com/ionchain/ionchain-core/p2p/enode"
	"github.com/ionchain/ionchain-core/rlp"
)

// ethEntry is the "ionc" ENR entry which advertises ionc protocol
// on the discovery network.
type ethEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e ethEntry) ENRKey() string {
	return "ionc"
}

// startEthEntryUpdate starts the ENR updater loop.
func (eth *IonChain) startEthEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := eth.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(eth.currentEthEntry())
			case <-sub.Err():
				// Would be nice to sync with ionc.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (eth *IonChain) currentEthEntry() *ethEntry {
	return &ethEntry{ForkID: forkid.NewID(eth.blockchain.Config(), eth.blockchain.Genesis().Hash(),
		eth.blockchain.CurrentHeader().Number.Uint64())}
}

// setupDiscovery creates the node discovery source for the ionc protocol.
func (eth *IonChain) setupDiscovery() (enode.Iterator, error) {
	if len(eth.config.DiscoveryURLs) == 0 {
		return nil, nil
	}
	client := dnsdisc.NewClient(dnsdisc.Config{})
	return client.NewIterator(eth.config.DiscoveryURLs...)
}
