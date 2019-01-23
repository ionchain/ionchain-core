package ipos

import (
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/consensus"
	"github.com/ionchain/ionchain-core/crypto"
	"github.com/ionchain/ionchain-core/common"
	"math/big"
	"errors"
	"bytes"
	"fmt"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errInvalidDifficulty = errors.New("non-positive difficulty")

	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")

	errUnknownBlock = errors.New("unknown block")

	errInvalidBlockSignature      = errors.New("invalid block signature")
	errInvalidGenerationSignature = errors.New("invalid generation signature")
	errInvalidHit                 = errors.New("invalid hit")
	errUnableMine                 = errors.New("unable mine block")
)

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
// TODO (karalabe): Move the chain maker into this package and make this private!
func calcDifficulty(chain consensus.ChainReader, header *types.Header) error {
	//currentDiff := new(big.Int).Div(math.MaxBig64, header.BaseTarget)
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)

	elapsedTime := new(big.Int).Set(header.Time).Sub(header.Time, parent.Time)
	preBase_elapsedTime := new(big.Int).Mul(parent.BaseTarget, elapsedTime)
	currentDiff := new(big.Int).Div(DIFFICULTY_MULTIPLIER, preBase_elapsedTime)

	if currentDiff.Cmp(big.NewInt(0)) == 0 {
		currentDiff = big.NewInt(1)
	}

	headerDiff := header.Difficulty
	if currentDiff.Cmp(headerDiff) != 0 {
		return fmt.Errorf("invalid difficulty have %d ,want %d ", header.Difficulty, currentDiff)
	}
	return nil
}

func (c *IPos) verifyBaseTarget(chain consensus.ChainReader, header *types.Header) error {
	baseTarget := c.calcBaseTargetNew(chain, header)
	header_baseTarget := header.BaseTarget
	if baseTarget.Cmp(header_baseTarget) != 0 {
		return fmt.Errorf("invalid baseTarget have %d ,want %d ", header_baseTarget, baseTarget)
	}
	if err := calcDifficulty(chain, header); err != nil {
		return err
	}
	return nil;
}

func (c *IPos) verifyGenerationSignature(chain consensus.ChainReader, header *types.Header) error {
	//header.GenerationSignature
	sig := c.generationSignature(chain, header)
	if bytes.Equal(header.GenerationSignature, sig) == false {
		return fmt.Errorf("invalid generationSignature have %x ,want %x ", header.GenerationSignature, sig)
	}
	return nil
}

func (c *IPos) verifyBlockSignature(chain consensus.ChainReader, header *types.Header) error {
	//从区块中还原出公钥
	signer, err := ecrecover(header)
	if err != nil {
		return err
	}
	// 对比
	if bytes.Equal(header.Coinbase[:], signer[:]) == false {
		return fmt.Errorf("invalid blockSignature have %x, want %x ", header.Coinbase, signer)
	}

	return nil
}

func ecrecover(header *types.Header) (common.Address, error) {
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), header.BlockSignature) // 从签名信息中恢复出公钥

	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	return signer, nil
}
