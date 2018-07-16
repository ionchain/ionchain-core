package ipos

import (
	"errors"
	"github.com/ionchain/ionchain-core/core_ionc/types"
	"github.com/ionchain/ionchain-core/common"
	consensus "github.com/ionchain/ionchain-core/consensus_ionc"
	"github.com/ionchain/ionchain-core/rpc"
	"github.com/ionchain/ionchain-core/core_ionc/state"
	"github.com/ionchain/ionchain-core/ethdb"
	"math/big"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/crypto/sha3"
	"sync"
	"github.com/ionchain/ionchain-core/accounts_ionc"
	"github.com/ionchain/ionchain-core/rlp"
)

var (
	errUnknownBlock = errors.New("unknown block")
)

const (
	INITIAL_BASE_TARGET int64 = 153722867
	MAX_BALANCE_NXT     int64 = 800000000 // IONC 8亿
	MAX_BASE_TARGET     int64 = MAX_BALANCE_NXT * INITIAL_BASE_TARGET
)

// IONC proof-of-stake protocol constants.
var (
	frontierBlockReward  *big.Int = big.NewInt(5e+18) // Block reward in wei for successfully mining a block
	byzantiumBlockReward *big.Int = big.NewInt(3e+18) // Block reward in wei for successfully mining a block upward from Byzantium

)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,

		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra, // Yes, this will panic if extra is too short 至少65个字节

		header.BaseTarget,
		header.Coinbase,
		header.GenerationSignature,
	})
	hasher.Sum(hash[:0])
	return hash
}

type IPos struct {
	db ethdb.Database

	signer common.Address // Ethereum address of the signing key 签名的地址
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields
}

func New(db ethdb.Database) *IPos {

	return &IPos{
		db: db,
	}
}

// Author retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
// 返回挖出区块的矿工地址
func (c *IPos) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
// 校验区块头 检查是否符合共识
func (c *IPos) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return nil
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
// 与VerifyHeader类似，批量校验区块头，返回 quit channel 用来取消操作，results channel 异步取出结果
func (c *IPos) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {

	return nil, nil
}

func (c *IPos) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock // 未知区块错误
	}

	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
// 校验是否符合共识规则（nonce，签名）
func (c *IPos) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// 校验 baseTarget 与 hit
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
// 返回共识所需要的区块头，baseTarget
func (c *IPos) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	//计算baseTarget
	parentBaseTarget := parent.BaseTarget
	diff := header.Time.Sub(header.Time, parent.Time)
	newBaseTarget := parentBaseTarget.Mul(parentBaseTarget, diff)
	newBaseTarget = newBaseTarget.Div(newBaseTarget, big.NewInt(60))

	// 最大余额8亿，MAX_BASE_TARGET =  8亿 * 初始baseTarget   2 ** 57
	// 如果baseTarget 超过最大值 则 设置为最大值
	newBaseTargetInt64 := newBaseTarget.Int64()
	if newBaseTargetInt64 < 0 || newBaseTargetInt64 > MAX_BASE_TARGET {
		newBaseTargetInt64 = MAX_BASE_TARGET
	}

	// 小于父区块的baseTarget一半
	if newBaseTargetInt64 < parentBaseTarget.Int64()/2 {
		newBaseTargetInt64 = parentBaseTarget.Int64() / 2
	}
	// 等于0 ，最小是1
	if newBaseTargetInt64 == 0 {
		newBaseTargetInt64 = 1
	}
	// 父区块baseTarget 两倍
	twofoldCurBaseTarget := parentBaseTarget.Int64() * 2;
	if twofoldCurBaseTarget < 0 { // 溢出 最大 64位
		twofoldCurBaseTarget = MAX_BASE_TARGET
	}
	// 大于 父区块baseTarget两倍
	if newBaseTargetInt64 > twofoldCurBaseTarget {
		newBaseTargetInt64 = twofoldCurBaseTarget
	}

	header.BaseTarget.SetInt64(newBaseTargetInt64)

	// 更新难度
	//cumulativeDifficulty
	t, _ := new(big.Int).SetString("18446744073709551616", 10)
	currentDiff := new(big.Int).Div(t, header.BaseTarget)
	//currentDiff = new(big.Int).Add(currentDiff, parent.Difficulty) // 不做累计难度
	header.Difficulty = currentDiff

	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
// 返回最终的区块
func (c *IPos) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, receipts []*types.Receipt) (*types.Block, error) {
	AccumulateRewards(chain.Config(), state, header) // 计算区块奖励 ，奖励放入state中

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number)) // 计算世界状态的根，EIP158 是否删除空的对象

	return types.NewBlock(header, txs, receipts), nil
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
//
// TODO (karalabe): Move the chain maker into this package and make this private!
func AccumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header) {
	// Select the correct block reward based on chain progression
	blockReward := frontierBlockReward
	//if config.IsByzantium(header.Number) {
	//	blockReward = byzantiumBlockReward
	//}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(big.Int).Set(blockReward)
	state.AddBalance(header.Coinbase, reward)
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *IPos) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
// 尝试补全区块（nonce，签名）
func (c *IPos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	// 计算baseTarget是否符合要求,校验hit

	// 给区块添加签名,baseTarget,BlockGenerationSignature,cumulativeDifficulty

	//1. generationSignature
	//sha256(newTransactions || previousBlock.getGenerationSignature() || publickey)
	hw := sha3.NewKeccak256()
	hw.Write(header.TxHash[:])                    // tx hash
	hw.Write(parentHeader.GenerationSignature[:]) //previousBlock.getGenerationSignature()
	hw.Write(header.Coinbase[:])                  // publickey
	header.GenerationSignature = hw.Sum(nil)

	//2. blockSignature 添加签名
	//header.BlockSignature

	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	header.BlockSignature = sighash

	return block.WithSeal(header), nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *IPos) APIs(chain consensus.ChainReader) []rpc.API {
	return nil
}
