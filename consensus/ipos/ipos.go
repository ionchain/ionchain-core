package ipos

import (
	"crypto/sha256"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/ionchain/ionchain-core/accounts"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/common/math"
	"github.com/ionchain/ionchain-core/consensus"
	"github.com/ionchain/ionchain-core/core/state"
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/ioncdb"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/rlp"
	"github.com/ionchain/ionchain-core/rpc"
	"github.com/ionchain/ionchain-core/trie"
	"golang.org/x/crypto/sha3"
	"math/big"
	"runtime"
	"sync"
	"time"
)

/*const (
	//INITIAL_BASE_TARGET int64 = 153722867
	INITIAL_BASE_TARGET int64 = 180143985
	MAX_BALANCE_NXT     int64 = 800000000 // IONC 8亿
	MAX_BASE_TARGET     int64 = MAX_BALANCE_NXT * INITIAL_BASE_TARGET
)*/
const (
	BlockTime         uint64 = 15
	MaxBlockTimeLimit        = BlockTime + 2
	MinBlockTimeLimit        = BlockTime - 2
)

const MAX_BALANCE_IONC uint64 = 800000000 // IONC 8亿

const TARGET = BlockTime * MAX_BALANCE_IONC

var (
	BaseTargetGamma   uint64 = 64
	InitialBaseTarget        = math.MaxBig63.Uint64() / TARGET
	MaxBaseTarget            = InitialBaseTarget * MAX_BALANCE_IONC // main 50  ,test MAX_BALANCE_IONC
	MinBaseTarget            = InitialBaseTarget * 9 / 10

	DifficultyMultiplier = new(big.Int).Mul(math.MaxBig64, big.NewInt(60))

	allowedFutureBlockTime = 15 * time.Second // Max time from current time allowed for blocks, before they're considered future blocks
)

// IONC proof-of-stake protocol constants.
var (
	maxUncles = 2 // Maximum number of uncles allowed in a single block 最多可引用叔父区块数目

)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// SealHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func (c *IPos) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,

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
	db          ioncdb.Database
	IpcEndpoint string

	signer common.Address // ionchain address of the signing key 签名的地址
	signFn SignerFn       // Signer function to authorize hashes with

	lock      sync.RWMutex // Protects the signer fields
	closeOnce sync.Once    // Ensures exit channel will not be closed twice.
}

func New(db ioncdb.Database, IpcEndpoint string) *IPos {

	return &IPos{
		db:          db,
		IpcEndpoint: IpcEndpoint,
	}
}

// Author retrieves the ionchain address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
// 返回挖出区块的矿工地址
func (c *IPos) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock ionchain ethash engine.
func (c *IPos) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {

	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := mapset.NewSet(), make(map[common.Hash]*types.Header) //TODO

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Contains(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := c.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
// 校验区块头 检查是否符合共识
func (c *IPos) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {

	// Short circuit if the header is known, or it's parent not
	// 验证header是否已存在，parent是否不存在
	number := header.Number.Uint64()
	//fmt.Printf("header:%+v \n", chain.GetHeader(header.Hash(), number))
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	// 开始校验
	return c.verifyHeader(chain, header, parent, false, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
// 与VerifyHeader类似，批量校验区块头，返回 quit channel 用来取消操作，results channel 异步取出结果
func (c *IPos) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	//fmt.Printf("VerifyHeaders,headerLength: %v \nseals:%+v \nchain:%+v\n", len(headers), seals, chain)

	if len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = c.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {

			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (c *IPos) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool, index int) error {
	//fmt.Printf("verifyHeaderWorker: len(headers): %v ,index: %v \n", len(headers), index)
	var parent *types.Header
	//if index != 0 {
	//fmt.Printf("区块号： %v\n计算出来的父块hash： %v\n, header.ParentHash : %v \n\n", headers[index-1].Number.Uint64(),
	//headers[index-1].Hash().String(), headers[index].ParentHash.String())
	//}
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		//fmt.Printf("报错了\n ")
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return c.verifyHeader(chain, headers[index], parent, false, seals[index])
}

func (c *IPos) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	// extra 最大32字节
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if !uncle {
		if header.Time > uint64(time.Now().Add(allowedFutureBlockTime).Unix()) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time { // 区块时间错误
		return errZeroBlockTime
	}

	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit > math.MaxBig63.Uint64() {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, math.MaxBig63)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %v, gasLimit %v", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	//两个区块的差 > 父块gaslimit/1024  并且  当前区块gaslimit < MinGasLimit(5000)
	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := c.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
// 校验是否符合共识规则（nonce，签名）
func (c *IPos) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	//fmt.Printf("VerifySeal, number: %v \n", header.Number)

	//fmt.Printf("%v,", header.Number)
	//fmt.Printf("header: %+v \nheader.ParentHash: %v\n", header, header.ParentHash)

	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	// 校验 baseTarget 与 hit

	// 区块签名
	if err := c.verifyBlockSignature(chain, header); err != nil {
		return err
	}

	// 区块GenerationSignature
	if err := c.verifyGenerationSignature(chain, header); err != nil {
		return err
	}

	//baseTarget
	if err := c.verifyBaseTarget(chain, header); err != nil {
		return err
	}
	// hit
	if c.verifyHit(chain, header) == false {
		fmt.Errorf("invalid hit")
	}

	// difficult

	return nil
}

func Min(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

func Max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

//计算新的baseTarget难度
func (c *IPos) calcBaseTargetNew(chain consensus.ChainHeaderReader, header *types.Header) *big.Int {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	//new
	prevBaseTarget := parent.BaseTarget
	var baseTarget *big.Int
	var min uint64
	var max uint64
	parentHeight := parent.Number.Uint64()
	if parentHeight > 2 && parentHeight%2 == 0 {
		//fmt.Printf("calcBaseTargetNew -- blockNumber: %v ", header.Number.Uint64())
		prev1 := chain.GetHeader(parent.ParentHash, parent.Number.Uint64()-1)
		prev2 := chain.GetHeader(prev1.ParentHash, prev1.Number.Uint64()-1)
		blockTimeAverage := (header.Time - prev2.Time) / 3
		//fmt.Printf("blockTimeAverage = %v ", blockTimeAverage)
		if blockTimeAverage > BlockTime { // 出块速度变慢 ，将baseTarget调大使保证金小的人也可以出块
			// 出块时间最大 MAX_BLOCKTIME_LIMIT
			//if parent.UncleHash == types.EmptyUncleHash {
			//	min = blockTimeAverage
			//
			//} else {
			min = Min(blockTimeAverage, MaxBlockTimeLimit)
			//}
			baseTarget = new(big.Int).Mul(prevBaseTarget, new(big.Int).SetUint64(min))
			baseTarget = baseTarget.Div(baseTarget, new(big.Int).SetUint64(BlockTime))
			//baseTarget = (prevBaseTarget * Min(blockTimeAverage, MAX_BLOCKTIME_LIMIT)) / BLOCK_TIME;
		} else { // 出块速度变快 将baseTarget 调小使保证金大的人可以出块
			// 出块时间最小 MIN_BLOCKTIME_LIMIT
			// 时间间隔/Block_time * GAMMA/100
			//if parent.UncleHash == types.EmptyUncleHash {
			//	max = BLOCK_TIME - blockTimeAverage
			//} else {
			max = BlockTime - Max(blockTimeAverage, MinBlockTimeLimit)
			//}

			//fmt.Printf("max......... %d \n",max)
			baseTarget = new(big.Int).Mul(prevBaseTarget, new(big.Int).SetUint64(max))
			baseTarget = baseTarget.Mul(baseTarget, new(big.Int).SetUint64(BaseTargetGamma))
			baseTarget = baseTarget.Div(baseTarget, new(big.Int).SetUint64(100*BlockTime))
			baseTarget = new(big.Int).Sub(prevBaseTarget, baseTarget)
			//baseTarget = prevBaseTarget - prevBaseTarget*BASE_TARGET_GAMMA*(BLOCK_TIME-Max(blockTimeAverage, MIN_BLOCKTIME_LIMIT))/(100*BLOCK_TIME);
			//fmt.Printf("blockTimeAverage:%v ,prevBaseTarget:%v,newBaseTarget:%v \n", blockTimeAverage, prevBaseTarget, baseTarget)
			//fmt.Printf(",maxBaseTarget:%v ,minBaseTarget=%v", MaxBaseTarget, MinBaseTarget)

		}
		// 暂时注释
		if baseTarget.Cmp(big.NewInt(0)) < 0 || baseTarget.Cmp(new(big.Int).SetUint64(MaxBaseTarget)) > 0 {
			baseTarget = new(big.Int).SetUint64(MaxBaseTarget)
		}
		// 暂时注释
		if baseTarget.Cmp(new(big.Int).SetUint64(MinBaseTarget)) < 0 {
			baseTarget = new(big.Int).SetUint64(MinBaseTarget)
		}
	} else {
		baseTarget = prevBaseTarget
	}
	//fmt.Printf("returnNewBaseTarget: %v \n", baseTarget.Uint64())

	return baseTarget
	//new
}

func (c *IPos) calcBaseTargetOld(chain consensus.ChainReader, header *types.Header) uint64 {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	// old
	//计算baseTarget
	parentBaseTarget := parent.BaseTarget
	df, _ := math.SafeSub(header.Time, parent.Time)
	diff := new(big.Int).SetUint64(df)
	newBaseTarget := new(big.Int).Set(parentBaseTarget).Mul(parentBaseTarget, diff)
	newBaseTarget = newBaseTarget.Div(newBaseTarget, big.NewInt(60)) // 时间越长得到的BaseTarget越大，当时间到达3600时 100%出块 3600/60=60

	// 最大余额8亿，MAX_BASE_TARGET =  8亿 * 初始baseTarget   2 ** 57
	// 如果baseTarget 超过最大值 则 设置为最大值
	newBaseTargetUint64 := newBaseTarget.Uint64()
	if newBaseTargetUint64 < 0 || newBaseTargetUint64 > MaxBaseTarget {
		newBaseTargetUint64 = MaxBaseTarget
	}

	// 小于父区块的baseTarget一半
	if newBaseTargetUint64 < parentBaseTarget.Uint64()/2 {
		newBaseTargetUint64 = parentBaseTarget.Uint64() / 2
	}
	// 等于0 ，最小是1
	if newBaseTargetUint64 == 0 {
		newBaseTargetUint64 = 1
	}
	// 父区块baseTarget 两倍
	twofoldCurBaseTarget := parentBaseTarget.Uint64() * 2
	if twofoldCurBaseTarget < 0 { // 溢出 最大 64位
		twofoldCurBaseTarget = MaxBaseTarget
	}
	// 大于 父区块baseTarget两倍
	if newBaseTargetUint64 > twofoldCurBaseTarget {
		newBaseTargetUint64 = twofoldCurBaseTarget
	}

	return newBaseTargetUint64
	//old
}

/*func (c *IPos) calcBaseTarget(chain consensus.ChainReader, header *types.Header) (int64) {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	//计算baseTarget
	parentBaseTarget := parent.BaseTarget
	diff := new(big.Int).Set(header.Time).Sub(header.Time, parent.Time)
	newBaseTarget := new(big.Int).Set(parentBaseTarget).Mul(parentBaseTarget, diff)
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

	return newBaseTargetInt64

}*/

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
// 返回共识所需要的区块头，baseTarget
func (c *IPos) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	res := c.calcBaseTargetNew(chain, header)
	//fmt.Printf(" Prepare---blockNumber: %v ,baseTarget: %v\n", header.Number.Uint64(), res.Uint64())
	header.BaseTarget.Set(res)

	// 更新难度
	//cumulativeDifficulty
	//currentDiff := new(big.Int).Div(math.MaxBig64, header.BaseTarget)
	//currentDiff = new(big.Int).Add(currentDiff, parent.Difficulty) // 不做累计难度

	et, _ := math.SafeSub(header.Time, parent.Time)
	elapsedTime := new(big.Int).SetUint64(et)

	preBaseElapsedTime := new(big.Int).Mul(parent.BaseTarget, elapsedTime)
	currentDiff := new(big.Int).Div(DifficultyMultiplier, preBaseElapsedTime)

	if currentDiff.Cmp(big.NewInt(0)) == 0 {
		header.Difficulty = big.NewInt(1)
	} else {
		header.Difficulty = currentDiff
	}

	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
// 返回最终的区块
func (c *IPos) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// AccumulateRewards(chain.Config(), state, header) // 计算区块奖励 ，奖励放入state中

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number)) // 计算世界状态的根，EIP158 是否删除空的对象

	//return types.NewBlock(header, txs, uncles, receipts, new(trie.Trie)), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *IPos) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

//获取当前节点地址和父块签名的总hash
func (c *IPos) getHit(chain consensus.ChainHeaderReader, header *types.Header) *big.Int {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	/*hw := sha3.NewKeccak256()
	hw.Write(parentHeader.generationSignature)
	hw.Write(header.Coinbase[:])
	hit := hw.Sum(nil)[0:8]*/

	hw := sha256.New()
	hw.Write(parentHeader.GenerationSignature)
	hw.Write(header.Coinbase[:])
	hit := hw.Sum(nil)[0:8]

	return new(big.Int).SetBytes([]byte{hit[7], hit[6], hit[5], hit[4], hit[3], hit[2], hit[1], hit[0]})
}

//获取当前节点最快的出块时间
func (c *IPos) getHitTime(chain consensus.ChainHeaderReader, header *types.Header) *big.Int {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	effectiveBalance, err := c.effectiveBalance(chain, header) //当前持币人的有效抵押ionc金额
	if effectiveBalance.Cmp(big.NewInt(0)) == 0 || err != nil {
		return math.MaxBig63
	}
	hit := c.getHit(chain, header) //返回一个随机数，是用上一个区块的签名和当前的coinBase一起hash得到

	effectiveBaseTarget := new(big.Int).Mul(parentHeader.BaseTarget, effectiveBalance)
	// elapseTime = 一个hash值 / (父块baseTarget * 保证金)
	elapseTime := new(big.Int).Div(hit, effectiveBaseTarget)
	//fmt.Printf("number:%v ,elapseTime: %v , baseTarget=%v ,parentHeader.BaseTarget=%v ,effectiveBalance=%v , ", header.Number.Uint64(), elapseTime, header.BaseTarget.Uint64(), parentHeader.BaseTarget, effectiveBalance)
	//fmt.Printf("getHitTime- hit= %v ,effectiveBaseTarget= %v ,elapseTime= %v \n", hit.Uint64(), effectiveBaseTarget.Uint64(), elapseTime)
	// hitTime = 父块时间戳 + elapseTime
	hitTime := new(big.Int).Add(new(big.Int).SetUint64(parentHeader.Time), elapseTime)
	//fmt.Printf("getHitTime: hit=%v , effectiveBaseTarget=%v , elapseTime=%v , hitTime=%v \n", hit, effectiveBaseTarget, elapseTime, hitTime.Uint64())

	//target := new(big.Int).Mul(effectiveBaseTarget, elapseTime)
	//b := hit.Cmp(target) < 0
	//fmt.Printf("verify: %v \n", b)

	return hitTime //                  hit/(parentBaseTarget*抵押额) + parentTimeStamp
}

//校验难度及时间
func (c *IPos) verifyHit(chain consensus.ChainHeaderReader, header *types.Header) bool {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	effectiveBalance, err := c.effectiveBalance(chain, header) //获取保证金数量
	if effectiveBalance.Cmp(big.NewInt(0)) == 0 || err != nil {
		return false
	}

	hit := c.getHit(chain, header) //得到一个hash

	// 需要重新计算
	effectiveBaseTarget := new(big.Int).Mul(parentHeader.BaseTarget, effectiveBalance)

	elapsedTime := new(big.Int).SetUint64(header.Time - parentHeader.Time)

	//target = 父块baseTarger * 保证金数量 * （当前区块时间 - 父块时间）
	target := new(big.Int).Mul(effectiveBaseTarget, elapsedTime)
	//target := new(big.Int).Set(prevTarget).Add(prevTarget, effectiveBaseTarget)
	//fmt.Printf("verifyHit- hit= %v ,effectiveBaseTarget= %v ,target= %v \n", hit.Uint64(), effectiveBaseTarget.Uint64(), target.Uint64())
	// 暂时注释
	//return hit.Cmp(target) < 0 && (hit.Cmp(prevTarget) >= 0 || elapsedTime.Cmp(timeOut) > 0)
	//fmt.Printf("verifyHit: hit=%v , effectiveBaseTarget=%v , elapsedTime=%v , target=%v \n",hit,effectiveBaseTarget,elapsedTime,target)

	//fmt.Printf("%v ,verifyHit- hit= %v ,effectiveBaseTarget= %v ,target= %v ,elapsedTime= %v \n", header.Number.Uint64(), hit.Uint64(), effectiveBaseTarget.Uint64(), target.Uint64(), elapsedTime)

	return hit.Cmp(target) < 0
}

//调用合约查询生效的保证金是多少
func (c *IPos) effectiveBalance(chain consensus.ChainHeaderReader, header *types.Header) (*big.Int, error) {

	balance, err := mintPower(header.Coinbase, c.IpcEndpoint)
	if err != nil {
		return nil, err
	}
	return balance, nil
	//return new(big.Int).SetInt64(1000000), nil
}

//生成签名
func (c *IPos) generationSignature(chain consensus.ChainHeaderReader, header *types.Header) []byte {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	//sha256(newTransactions || previousBlock.getGenerationSignature() || publickey)
	/*hw := sha3.NewKeccak256()
	hw.Write(header.TxHash[:])                    // tx hash
	hw.Write(parentHeader.generationSignature[:]) //previousBlock.getGenerationSignature()
	hw.Write(header.Coinbase[:])                  // publickey
	return hw.Sum(nil)*/

	hw := sha256.New()
	//hw.Write(header.TxHash[:])                    // tx hash
	hw.Write(parentHeader.GenerationSignature[:]) //previousBlock.getGenerationSignature()
	hw.Write(header.Coinbase[:])                  // publickey
	return hw.Sum(nil)
}

//区块签名
func (c *IPos) blockSignature(chain consensus.ChainHeaderReader, header *types.Header) ([]byte, error) {
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	sighash, err := signFn(accounts.Account{Address: signer}, "", c.SealHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	return sighash, nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
// 尝试补全区块（nonce，签名）
// 判断是否有出块权
func (c *IPos) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}, errmsg chan error) {
	//func (c *IPos) Seal(chain consensus.ChainHeaderReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// 判断出块权
	var timeOut <-chan time.Time

	headerTime := time.Unix(int64(header.Time), 0)
	hitTime := c.getHitTime(chain, header) //hitTime是当前节点可以出块的最短时间

	hitTime = hitTime.Add(hitTime, big.NewInt(1))

	hTime := time.Unix(hitTime.Int64(), 0)

	timeOut = time.After(hTime.Sub(headerTime))
	//fmt.Printf("in Seal func hitTime = %v , timeOut = %v \n", hitTime.Int64(), hTime.Sub(headerTime).Seconds())

	//fmt.Printf("%v ,parentTime: %v,hTime: %v,headerTime: %v \n",
	//	header.Number.Uint64(),
	//	chain.GetHeader(header.ParentHash, header.Number.Uint64()-1).Time,
	//	hTime.Unix(), headerTime.Unix())

Loop:
	for {
		select {
		case <-stop:
			//fmt.Printf("1111111111111111111111111111111111111111111111 \n")
			return
		case <-timeOut:
			//fmt.Printf("超时 \n")
			break Loop
		}
	}

	// 判断出块权
	if ok := c.verifyHit(chain, header); !ok {
		//fmt.Printf("verifyHit failed \n")
		errmsg <- errUnableMineTime
	}

	number := header.Number.Uint64()
	if number == 0 {
		errmsg <- errUnknownBlock
	}
	// 计算baseTarget是否符合要求,校验hit

	// 给区块添加签名,baseTarget,BlockGenerationSignature,cumulativeDifficulty

	//1. generationSignature

	header.GenerationSignature = c.generationSignature(chain, header)

	//2. blockSignature 添加签名
	//header.blockSignature

	sighash, err := c.blockSignature(chain, header)
	if err != nil {
		errmsg <- err
	}
	header.BlockSignature = sighash
	//fmt.Printf("在Seal中签名,head: %+v \n", header)
	//fmt.Printf("给resultCh发送消息：header: %+v \n", header)
	results <- block.WithSeal(header)


}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *IPos) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return nil
}

//new Interfaces

//used for test
func (c *IPos) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

func (c *IPos) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	//if c.signFn != nil {
	//	//fmt.Printf("...signFn...\n")
	//	header.GenerationSignature = c.generationSignature(chain, header)
	//	sighash, err := c.blockSignature(chain, header)
	//	if err != nil {
	//		return nil, err
	//	}
	//	header.BlockSignature = sighash
	//}
	//fmt.Printf("在finalize中签名： header: %+v \n", header)
	// Header seems complete, assemble into a block and return
	b := types.NewBlock(header, txs, uncles, receipts, new(trie.Trie))
	//fmt.Printf("after signFn b.header:%+v\n", b.Header())
	return b, nil
}

func (c *IPos) Close() error {
	return nil
}
