package ipos

import (
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/consensus"
	"github.com/ionchain/ionchain-core/rpc"
	"github.com/ionchain/ionchain-core/core/state"
	"github.com/ionchain/ionchain-core/ioncdb"
	"math/big"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/crypto/sha3"
	"sync"
	"github.com/ionchain/ionchain-core/rlp"
	"fmt"
	"runtime"
	"github.com/ionchain/ionchain-core/common/math"
	"time"
	"crypto/sha256"
	"github.com/ionchain/ionchain-core/accounts"
	"gopkg.in/fatih/set.v0"
)

/*const (
	//INITIAL_BASE_TARGET int64 = 153722867
	INITIAL_BASE_TARGET int64 = 180143985
	MAX_BALANCE_NXT     int64 = 800000000 // IONC 8亿
	MAX_BASE_TARGET     int64 = MAX_BALANCE_NXT * INITIAL_BASE_TARGET
)*/
const (
	BLOCK_TIME          int64 = 15;
	MAX_BLOCKTIME_LIMIT int64 = BLOCK_TIME + 7
	MIN_BLOCKTIME_LIMIT int64 = BLOCK_TIME - 7
)

const MAX_BALANCE_IONC int64 = 800000000 // IONC 8亿

const TARGET = BLOCK_TIME * MAX_BALANCE_IONC

var (
	INITIAL_BASE_TARGET int64 = new(big.Int).Div(math.MaxBig63, big.NewInt(TARGET)).Int64()
	MAX_BASE_TARGET     int64 = INITIAL_BASE_TARGET * 50 // main 50  ,test MAX_BALANCE_IONC
	MIN_BASE_TARGET     int64 = INITIAL_BASE_TARGET * 9 / 10
	BASE_TARGET_GAMMA   int64 = 64

	DIFFICULTY_MULTIPLIER = new(big.Int).Mul(math.MaxBig64, big.NewInt(60))
)

// IONC proof-of-stake protocol constants.
var (
	maxUncles                     = 2                 // Maximum number of uncles allowed in a single block 最多可引用叔父区块数目

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
	lock   sync.RWMutex   // Protects the signer fields
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
	uncles, ancestors := set.New(), make(map[common.Hash]*types.Header)

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
		if uncles.Has(hash) {
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
func (c *IPos) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {

	// Short circuit if the header is known, or it's parent not
	// 验证header是否已存在，parent是否不存在
	number := header.Number.Uint64()
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
func (c *IPos) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {

	// If we're running a full engine faking, accept any input as valid
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

func (c *IPos) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return c.verifyHeader(chain, headers[index], parent, false, seals[index])
}

func (c *IPos) verifyHeader(chain consensus.ChainReader, header *types.Header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	// extra 最大32字节
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if uncle {
		if header.Time.Cmp(math.MaxBig256) > 0 {
			return errLargeBlockTime
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 { //未来区块
			return consensus.ErrFutureBlock
		}
	}
	if header.Time.Cmp(parent.Time) <= 0 { // 区块时间错误
		return errZeroBlockTime
	}

	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit.Cmp(math.MaxBig63) > 0 {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, math.MaxBig63)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed.Cmp(header.GasLimit) > 0 {
		return fmt.Errorf("invalid gasUsed: have %v, gasLimit %v", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := new(big.Int).Set(parent.GasLimit)
	diff = diff.Sub(diff, header.GasLimit)
	diff.Abs(diff)

	limit := new(big.Int).Set(parent.GasLimit)
	limit = limit.Div(limit, params.GasLimitBoundDivisor)

	if diff.Cmp(limit) >= 0 || header.GasLimit.Cmp(params.MinGasLimit) < 0 {
		return fmt.Errorf("invalid gas limit: have %v, want %v += %v", header.GasLimit, parent.GasLimit, limit)
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
func (c *IPos) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
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

func Min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func (c *IPos) calcBaseTargetNew(chain consensus.ChainReader, header *types.Header) (*big.Int) {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	//new
	prevBaseTarget := parent.BaseTarget
	var baseTarget *big.Int
	var min int64
	var max int64
	parentHeight := parent.Number.Uint64()
	if parentHeight > 2 && parentHeight%2 == 0 {
		prev_1 := chain.GetHeader(parent.ParentHash, parent.Number.Uint64()-1)
		prev_2 := chain.GetHeader(prev_1.ParentHash, prev_1.Number.Uint64()-1)
		blocktimeAverage := (header.Time.Int64() - prev_2.Time.Int64()) / 3
		if blocktimeAverage > BLOCK_TIME { // 出块速度变慢 ，将baseTarget调大使保证金小的人也可以出块
			// 出块时间最大 MAX_BLOCKTIME_LIMIT
			if parent.UncleHash == types.EmptyUncleHash{
				min = blocktimeAverage

			}else{
				min = Min(blocktimeAverage, MAX_BLOCKTIME_LIMIT)
			}
			baseTarget = new(big.Int).Set(prevBaseTarget).Mul(prevBaseTarget, big.NewInt(min))
			baseTarget = baseTarget.Div(baseTarget, big.NewInt(BLOCK_TIME))
			//baseTarget = (prevBaseTarget * Min(blocktimeAverage, MAX_BLOCKTIME_LIMIT)) / BLOCK_TIME;
		} else { // 出块速度变快 将baseTarget 调小使保证金大的人可以出块
			// 出块时间最小 MIN_BLOCKTIME_LIMIT
			// 时间间隔/Block_time * GAMMA/100
			if parent.UncleHash == types.EmptyUncleHash{
				max = BLOCK_TIME - blocktimeAverage
			}else{
				max = BLOCK_TIME - Max(blocktimeAverage, MIN_BLOCKTIME_LIMIT)
			}

			//fmt.Printf("max......... %d \n",max)
			baseTarget = new(big.Int).Set(prevBaseTarget).Mul(prevBaseTarget,big.NewInt(max))
			baseTarget = baseTarget.Mul(baseTarget,big.NewInt(BASE_TARGET_GAMMA))
			baseTarget = baseTarget.Div(baseTarget,big.NewInt(100 * BLOCK_TIME))
			baseTarget = new(big.Int).Set(prevBaseTarget).Sub(prevBaseTarget, baseTarget)
			//baseTarget = prevBaseTarget - prevBaseTarget*BASE_TARGET_GAMMA*(BLOCK_TIME-Max(blocktimeAverage, MIN_BLOCKTIME_LIMIT))/(100*BLOCK_TIME);
		}
		// 暂时注释
		//if baseTarget.Cmp(big.NewInt(0)) < 0 || baseTarget.Cmp(big.NewInt(MAX_BASE_TARGET)) > 0 {
		//	baseTarget = big.NewInt(MAX_BASE_TARGET);
		//}
		// 暂时注释
		//if baseTarget.Cmp(big.NewInt(MIN_BASE_TARGET)) < 0 {
		//	baseTarget = big.NewInt(MIN_BASE_TARGET);
		//}
	} else {
		baseTarget = prevBaseTarget;
	}
	return baseTarget
	//new
}

func (c *IPos) calcBaseTargetOld(chain consensus.ChainReader, header *types.Header) (int64) {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	// old
	//计算baseTarget
	parentBaseTarget := parent.BaseTarget
	diff := new(big.Int).Set(header.Time).Sub(header.Time, parent.Time)
	newBaseTarget := new(big.Int).Set(parentBaseTarget).Mul(parentBaseTarget, diff)
	newBaseTarget = newBaseTarget.Div(newBaseTarget, big.NewInt(60)) // 时间越长得到的BaseTarget越大，当时间到达3600时 100%出块 3600/60=60

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
func (c *IPos) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	header.BaseTarget.Set(c.calcBaseTargetNew(chain, header))

	// 更新难度
	//cumulativeDifficulty
	//currentDiff := new(big.Int).Div(math.MaxBig64, header.BaseTarget)
	//currentDiff = new(big.Int).Add(currentDiff, parent.Difficulty) // 不做累计难度

	elapsedTime := new(big.Int).Set(header.Time).Sub(header.Time, parent.Time)
	preBase_elapsedTime := new(big.Int).Mul(parent.BaseTarget, elapsedTime)
	currentDiff := new(big.Int).Div(DIFFICULTY_MULTIPLIER, preBase_elapsedTime)

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
func (c *IPos) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// AccumulateRewards(chain.Config(), state, header) // 计算区块奖励 ，奖励放入state中

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number)) // 计算世界状态的根，EIP158 是否删除空的对象

	return types.NewBlock(header, txs, uncles, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *IPos) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

func (c *IPos) getHit(chain consensus.ChainReader, header *types.Header) *big.Int {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	/*hw := sha3.NewKeccak256()
	hw.Write(parentHeader.GenerationSignature)
	hw.Write(header.Coinbase[:])
	hit := hw.Sum(nil)[0:8]*/

	hw := sha256.New()
	hw.Write(parentHeader.GenerationSignature)
	hw.Write(header.Coinbase[:])
	hit := hw.Sum(nil)[0:8]

	return new(big.Int).SetBytes([]byte{hit[7], hit[6], hit[5], hit[4], hit[3], hit[2], hit[1], hit[0]})
}

func (c *IPos) getHitTime(chain consensus.ChainReader, header *types.Header) *big.Int {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	effectiveBalance, err := c.effectiveBalance(chain, header)
	if effectiveBalance.Cmp(big.NewInt(0)) == 0 || err != nil {
		return math.MaxBig63
	}
	hit := c.getHit(chain, header)

	effective := new(big.Int).Set(parentHeader.BaseTarget).Mul(parentHeader.BaseTarget, effectiveBalance)
	effective = new(big.Int).Set(hit).Div(hit, effective)
	hitTime := new(big.Int).Set(parentHeader.Time).Add(parentHeader.Time, effective)
	return hitTime
}

func (c *IPos) verifyHit(chain consensus.ChainReader, header *types.Header) bool {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	effectiveBalance, err := c.effectiveBalance(chain, header)
	if effectiveBalance.Cmp(big.NewInt(0)) == 0 || err != nil {
		return false
	}

	hit := c.getHit(chain, header)

	// 需要重新计算
	effectiveBaseTarget := new(big.Int).Set(parentHeader.BaseTarget).Mul(parentHeader.BaseTarget, effectiveBalance)

	elapsedTime := new(big.Int).Set(header.Time).Sub(header.Time, parentHeader.Time)

	prevTarget := new(big.Int).Set(effectiveBaseTarget).Mul(effectiveBaseTarget, elapsedTime)
	target := new(big.Int).Set(prevTarget).Add(prevTarget, effectiveBaseTarget)

	timeOut := new(big.Int).SetInt64(3600) // 1h

	//fmt.Printf("hit %d,prevTarget %d,target %d,elapsedTime %d,hit.Cmp(target) %d,hit.Cmp(prevTarget) %d ", hit, prevTarget, target, elapsedTime, hit.Cmp(target), hit.Cmp(prevTarget))
	fmt.Printf("prevTarget %v,hit.Cmp(target) %d,hit.Cmp(prevTarget) %d ,diff %d \n", prevTarget, hit.Cmp(target), hit.Cmp(prevTarget), new(big.Int).Set(hit).Sub(hit, prevTarget))
	fmt.Printf("hit.Cmp(target) < 0 && (hit.Cmp(prevTarget) >= 0 || elapsedTime.Cmp(timeOut) > 0) %b \n", hit.Cmp(target) < 0 && (hit.Cmp(prevTarget) >= 0 || elapsedTime.Cmp(timeOut) > 0))

	// 暂时注释
	//return hit.Cmp(target) < 0 && (hit.Cmp(prevTarget) >= 0 || elapsedTime.Cmp(timeOut) > 0)
	return hit.Cmp(target) < 0
}

func (c *IPos) effectiveBalance(chain consensus.ChainReader, header *types.Header) (*big.Int, error) {

	balance, err := mintPower(header.Coinbase, c.IpcEndpoint)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (c *IPos) generationSignature(chain consensus.ChainReader, header *types.Header) ([]byte) {
	parentHeader := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	//sha256(newTransactions || previousBlock.getGenerationSignature() || publickey)
	/*hw := sha3.NewKeccak256()
	hw.Write(header.TxHash[:])                    // tx hash
	hw.Write(parentHeader.GenerationSignature[:]) //previousBlock.getGenerationSignature()
	hw.Write(header.Coinbase[:])                  // publickey
	return hw.Sum(nil)*/

	hw := sha256.New()
	//hw.Write(header.TxHash[:])                    // tx hash
	hw.Write(parentHeader.GenerationSignature[:]) //previousBlock.getGenerationSignature()
	hw.Write(header.Coinbase[:])                  // publickey
	return hw.Sum(nil)
}

func (c *IPos) blockSignature(chain consensus.ChainReader, header *types.Header) ([]byte, error) {
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	return sighash, nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
// 尝试补全区块（nonce，签名）
// 判断是否有出块权
func (c *IPos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// 判断出块权
	var timeOut <-chan time.Time
	hitTime := c.getHitTime(chain, header)
	time_diff := new(big.Int).Set(header.Time).Sub(header.Time, hitTime)


	if time_diff.Cmp(big.NewInt(-15)) < 0 /*|| !c.verifyHit(chain, header)*/ {
		//sleepTime := new(big.Int).Set(header.Time).Sub(header.Time, hitTime).Int64()
		//if sleepTime > 15 { // 最大延迟
		//	sleepTime = 15
		//}
		//time.Sleep(time.Duration(sleepTime))

		timeOut = time.After(14 * time.Second)
	} else {
		timeOut = time.After(0)
	}
Loop:
	for {
		select {
		case <-stop:
			return nil, nil
		case <-timeOut:
			break Loop
		}
	}

	// 判断出块权
	if ok := c.verifyHit(chain, header); !ok {
		return nil, errUnableMine
	}

	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	// 计算baseTarget是否符合要求,校验hit

	// 给区块添加签名,baseTarget,BlockGenerationSignature,cumulativeDifficulty

	//1. generationSignature

	header.GenerationSignature = c.generationSignature(chain, header)

	//2. blockSignature 添加签名
	//header.BlockSignature

	if sighash, err := c.blockSignature(chain, header); err != nil {
		return nil, err
	} else {
		header.BlockSignature = sighash
	}

	return block.WithSeal(header), nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *IPos) APIs(chain consensus.ChainReader) []rpc.API {
	return nil
}
