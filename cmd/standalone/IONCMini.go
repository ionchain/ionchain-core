package main

import (
	"runtime"

	"github.com/ionchain/ionchain-core/common/hexutil"
	"github.com/ionchain/ionchain-core/accounts"
	core "github.com/ionchain/ionchain-core/core_ionc"
	consensus "github.com/ionchain/ionchain-core/consensus_ionc"
	"github.com/ionchain/ionchain-core/consensus_ionc/ipos"
	miner "github.com/ionchain/ionchain-core/miner_ionc"
	"github.com/ionchain/ionchain-core/ethdb"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/log"
	"github.com/ionchain/ionchain-core/core_ionc/vm"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/rlp"
	"github.com/ionchain/ionchain-core/event"
	"github.com/ionchain/ionchain-core/node"

	"fmt"
	"math/big"
)

type IONCMini struct {
	config      *Config
	chainConfig *params.ChainConfig

	stopDbUpgrade func() error // stop chain db sequential key upgrade

	accountManager *accounts.Manager	//账户管理
	txPool          *core.TxPool	//交易池
	blockchain      *core.BlockChain	//区块链

	// DB interfaces
	// leveldb数据库
	chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine	//共识引擎
	miner     *miner.Miner	//挖矿
	gasPrice  *big.Int
	etherbase common.Address
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext,config *Config) (*IONCMini, error) {

	chainDb, err := CreateDB(ctx, config, "chaindata")		// 创建leveldb数据库
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)	// 数据库格式升级
	// 设置创世区块。 如果数据库里面已经有创世区块那么从数据库里面取出(私链)。或者是从代码里面获取默认值。
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	//构建以太坊对象
	eth := &IONCMini{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb), //共识引擎
		stopDbUpgrade:  stopDbUpgrade,
		gasPrice:       config.GasPrice,
		etherbase:      config.Etherbase,
	}

	// 检查数据库里面存储的BlockChainVersion和客户端的BlockChainVersion的版本是否一致
	if !config.SkipBcVersionCheck {
		//数据库中的BlockChainVersion
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	// vm虚拟机配置
	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	//创建区块链 主链
	eth.blockchain, err = core.NewBlockChain(chainDb, eth.chainConfig, eth.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	//交易池
	eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)

	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	return eth, nil
}

func (s *IONCMini) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *IONCMini) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *IONCMini) TxPool() *core.TxPool               { return s.txPool }
func (s *IONCMini) ChainDb() ethdb.Database            { return s.chainDb }
func (s *IONCMini) EventMux() *event.TypeMux           { return s.eventMux }

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}


func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geth",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Ethereum service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db ethdb.Database) consensus.Engine {

	return ipos.New(db)

}