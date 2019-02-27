// Copyright 2014 The go-ionchain Authors
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

// Package ionc implements the ionchain protocol.
package ionc

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ionchain/ionchain-core/accounts"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/common/hexutil"
	"github.com/ionchain/ionchain-core/consensus"
	"github.com/ionchain/ionchain-core/consensus/ipos"
	"github.com/ionchain/ionchain-core/core"
	"github.com/ionchain/ionchain-core/core/bloombits"
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/core/vm"
	"github.com/ionchain/ionchain-core/ionc/downloader"
	"github.com/ionchain/ionchain-core/ionc/filters"
	"github.com/ionchain/ionchain-core/ionc/gasprice"
	"github.com/ionchain/ionchain-core/ioncdb"
	"github.com/ionchain/ionchain-core/event"
	"github.com/ionchain/ionchain-core/internal/ethapi"
	"github.com/ionchain/ionchain-core/log"
	"github.com/ionchain/ionchain-core/miner"
	"github.com/ionchain/ionchain-core/node"
	"github.com/ionchain/ionchain-core/p2p"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/rlp"
	"github.com/ionchain/ionchain-core/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// ionchain implements the ionchain full node service.
type IONChain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the ionchain
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool	//交易池
	blockchain      *core.BlockChain	//区块链
	protocolManager *ProtocolManager	// 协议管理
	lesServer       LesServer	// 轻节点客户端

	// DB interfaces
	// leveldb数据库
	chainDb ioncdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine	//共识引擎
	accountManager *accounts.Manager	//账户管理

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *EthApiBackend

	miner     *miner.Miner	//挖矿
	gasPrice  *big.Int
	etherbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

func (s *IONChain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new ionchain object (including the
// initialisation of the common ionchain object)
func New(ctx *node.ServiceContext, config *Config) (*IONChain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run ionc.Ethereum in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
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

	eth := &IONChain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb), //共识引擎
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		etherbase:      config.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Ethereum protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	// 检查数据库里面存储的BlockChainVersion和客户端的BlockChainVersion的版本是否一致
	if !config.SkipBcVersionCheck {
		//数据库中的BlockChainVersion
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run ionc upgradedb.\n", bcVersion, core.BlockChainVersion)
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
	eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	//交易池
	eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)

	// 创建协议管理器
	if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	eth.ApiBackend = &EthApiBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"ionc",
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

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ioncdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ioncdb.LDBDatabase); ok {
		db.Meter("ionc/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an ionchain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db ioncdb.Database) consensus.Engine {
	return ipos.New(db, ctx.IpcEndpoint)
}

// APIs returns the collection of RPC services the ionchain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *IONChain) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "ionc",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, /*{
			Namespace: "ionc",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		},*/ {
			Namespace: "ionc",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "ionc",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *IONChain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *IONChain) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("etherbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *IONChain) SetEtherbase(etherbase common.Address) {
	self.lock.Lock()
	self.etherbase = etherbase
	self.lock.Unlock()

	self.miner.SetEtherbase(etherbase)
}

//启动挖矿程序
func (s *IONChain) StartMining(local bool) error {

	// 从命令行参数中获取矿工账号
	eb, err := s.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	// ipos
	if ipos, ok := s.engine.(*ipos.IPos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		ipos.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *IONChain) StopMining()         { s.miner.Stop() }
func (s *IONChain) IsMining() bool      { return s.miner.Mining() }
func (s *IONChain) Miner() *miner.Miner { return s.miner }

func (s *IONChain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *IONChain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *IONChain) TxPool() *core.TxPool               { return s.txPool }
func (s *IONChain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *IONChain) Engine() consensus.Engine           { return s.engine }
func (s *IONChain) ChainDb() ioncdb.Database           { return s.chainDb }
func (s *IONChain) IsListening() bool                  { return true } // Always listening
func (s *IONChain) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *IONChain) NetVersion() uint64                 { return s.networkId }
func (s *IONChain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *IONChain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// ionchain protocol implementation.
func (s *IONChain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// IONChain protocol.
func (s *IONChain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
