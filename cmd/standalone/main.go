package main

import (
	"github.com/tendermint/tendermint/config"
	miner "github.com/ionchain/ionchain-core/miner_ionc"
)

func main(){
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))
}