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

package miner

import (
	"sync"

	"sync/atomic"

	"github.com/ionchain/ionchain-core/consensus"
	"github.com/ionchain/ionchain-core/log"
)

type ForgeAgent struct {
	mu sync.Mutex

	workCh        chan *Work
	stop          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *Result

	forgeCh chan struct{}

	chain  consensus.ChainReader
	engine consensus.Engine

	isMining int32 // isMining indicates whether the agent is currently mining
}

func NewForgeAgent(chain consensus.ChainReader, engine consensus.Engine) *ForgeAgent {
	miner := &ForgeAgent{
		chain:   chain,
		engine:  engine,
		stop:    make(chan struct{}, 1),
		workCh:  make(chan *Work, 1),
		forgeCh: make(chan struct{}, 1),
	}
	return miner
}

func (self *ForgeAgent) Work() chan<- *Work            { return self.workCh } //通过调用Work() 方法和微机可以和 workCh获取联系
func (self *ForgeAgent) ForgeCh() chan struct{}      { return self.forgeCh }
func (self *ForgeAgent) SetReturnCh(ch chan<- *Result) { self.returnCh = ch }

func (self *ForgeAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&self.isMining, 1, 0) {
		return // agent already stopped
	}
	self.stop <- struct{}{}
done:
// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			break done
		}
	}
}

//采用本地cpu进行挖矿
func (self *ForgeAgent) Start() {
	if !atomic.CompareAndSwapInt32(&self.isMining, 0, 1) {
		return // agent already started
	}
	go self.update()
}

//开始挖矿
func (self *ForgeAgent) update() {
out:
	for {
		select {

		// workCh 队列中存放挖矿信号
		case work := <-self.workCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			//启动挖矿线程
			go self.mine(work, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.stop: // 收到停止挖矿的信号
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
			break out
		}
	}
}

// 开始挖矿
func (self *ForgeAgent) mine(work *Work, stop <-chan struct{}) {

	// 填充区块头：ethash共识寻找nonce，poa共识 签名区块头
	if result, err := self.engine.Seal(self.chain, work.Block, stop); result != nil {
		log.Info("Successfully sealed new block", "number", result.Number(), "hash", result.Hash())
		self.returnCh <- &Result{work, result}
	} else {
		if err != nil {
			self.forgeCh <- struct{}{}
			log.Warn("Block sealing failed", "err", err)
		}
		self.returnCh <- nil
	}
}
