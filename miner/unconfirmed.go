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

package miner

import (
	"container/ring"
	"sync"

	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/log"
)

//unconfirmed是一个数据结构，用来跟踪用户本地的挖矿信息的，比如挖出了一个块，那么等待足够的后续区块确认之后(5个)，
// 再查看本地挖矿的区块是否包含在规范的区块链内部。

// headerRetriever is used by the unconfirmed block set to verify whether a previously
// mined block is part of the canonical chain or not.
// headerRetriever由未确认的块组使用，以验证先前挖掘的块是否是规范链的一部分。
type headerRetriever interface {
	// GetHeaderByNumber retrieves the canonical header associated with a block number.
	// 根据区块编号从主链上取出区块头
	GetHeaderByNumber(number uint64) *types.Header
}

// unconfirmedBlock is a small collection of metadata about a locally mined block
// that is placed into a unconfirmed set for canonical chain inclusion tracking.
// unconfirmedBlock 是本地挖掘区块的一个小的元数据的集合，用来放入未确认的集合用来追踪本地挖掘的区块是否被包含进入规范的区块链
type unconfirmedBlock struct {
	index uint64
	hash  common.Hash
}

// unconfirmedBlocks implements a data structure to maintain locally mined blocks
// have have not yet reached enough maturity to guarantee chain inclusion. It is
// used by the miner to provide logs to the user when a previously mined block
// has a high enough guarantee to not be reorged out of te canonical chain.

// unconfirmedBlocks 实现了一个数据结构，用来管理本地挖掘的区块，这些区块还没有达到足够的信任度来证明他们已经被规范的区块链接受。
// 它用来给矿工提供信息，以便他们了解他们之前挖到的区块是否被包含进入了规范的区块链。
type unconfirmedBlocks struct {
	chain  headerRetriever // Blockchain to verify canonical status through需要验证的区块链 用这个接口来获取当前的规范的区块头信息
	depth  uint            // Depth after which to discard previous blocks经过多少个区块之后丢弃之前的区块
	blocks *ring.Ring      // Block infos to allow canonical chain cross checks区块信息，以允许规范链交叉检查
	lock   sync.RWMutex    // Protects the fields from concurrent access
}

// newUnconfirmedBlocks returns new data structure to track currently unconfirmed blocks.
func newUnconfirmedBlocks(chain headerRetriever, depth uint) *unconfirmedBlocks {
	return &unconfirmedBlocks{
		chain: chain,
		depth: depth,
	}
}

// Insert adds a new block to the set of unconfirmed ones.
//插入跟踪区块, 当矿工挖到一个区块的时候调用， index是区块的高度， hash是区块的hash值。
func (set *unconfirmedBlocks) Insert(index uint64, hash common.Hash) {
	// If a new block was mined locally, shift out any old enough blocks
	// 如果一个本地的区块挖到了，那么移出已经超过depth的区块
	set.Shift(index)

	// Create the new item as its own ring
	// 循环队列的操作。
	item := ring.New(1)
	item.Value = &unconfirmedBlock{
		index: index,
		hash:  hash,
	}
	// Set as the initial ring or append to the end
	set.lock.Lock()
	defer set.lock.Unlock()

	if set.blocks == nil {
		set.blocks = item
	} else {
		// 移动到循环队列的最后一个元素插入item
		set.blocks.Move(-1).Link(item)
	}
	// Display a log for the user to notify of a new mined block unconfirmed
	log.Info("🔨 mined potential block", "number", index, "hash", hash)
}

// Shift drops all unconfirmed blocks from the set which exceed the unconfirmed sets depth
// allowance, checking them against the canonical chain for inclusion or staleness
// report.
//Shift方法会删除那些index超过传入的index-depth的区块，并检查他们是否在规范的区块链中。
func (set *unconfirmedBlocks) Shift(height uint64) {
	set.lock.Lock()
	defer set.lock.Unlock()

	for set.blocks != nil {
		// Retrieve the next unconfirmed block and abort if too fresh
		// 因为blocks中的区块都是按顺序排列的。排在最开始的肯定是最老的区块。
		// 所以每次只需要检查最开始的那个区块，如果处理完了，就从循环队列里面摘除。
		next := set.blocks.Value.(*unconfirmedBlock)
		if next.index+uint64(set.depth) > height { // 未超过set.depth个区块的确认
			break
		}
		// Block seems to exceed depth allowance, check for canonical status
		// 查询 那个区块高度的区块头
		header := set.chain.GetHeaderByNumber(next.index)
		switch {
		case header == nil:
			log.Warn("Failed to retrieve header of mined block", "number", next.index, "hash", next.hash)
		case header.Hash() == next.hash: // 如果区块头就等于我们自己 ，说明已经在主链上了
			log.Info("🔗 block reached canonical chain", "number", next.index, "hash", next.hash)
		default:// 否则说明我们在侧链上面。
			log.Info("⑂ block  became a side fork", "number", next.index, "hash", next.hash)
		}
		// Drop the block out of the ring
		// 从循环队列删除
		if set.blocks.Value == set.blocks.Next().Value {
			// 如果当前的值就等于我们自己，说明只有循环队列只有一个元素，那么设置未nil
			set.blocks = nil
		} else {
			// 否则移动到最后，然后删除一个，再移动到最前。
			set.blocks = set.blocks.Move(-1)
			set.blocks.Unlink(1)
			set.blocks = set.blocks.Move(1)
		}
	}
}
