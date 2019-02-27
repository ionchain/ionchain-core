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

package ionc

import (
	"context"
	"math/big"

	"github.com/ionchain/ionchain-core"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/common/hexutil"
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/internal/ioncapi"
	"github.com/ionchain/ionchain-core/rlp"
	"github.com/ionchain/ionchain-core/rpc"
)

// ContractBackend implements bind.ContractBackend with direct calls to ionchain
// internals to support operating on contracts within subprotocols like ionc and
// swarm.
//
// Internally this backend uses the already exposed API endpoints of the ionchain
// object. These should be rewritten to internal Go method calls when the Go API
// is refactored to support a clean library use.
type ContractBackend struct {
	eapi  *ioncapi.PublicIONChainAPI        // Wrapper around the ionchain object to access metadata
	bcapi *ioncapi.PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *ioncapi.PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data
}

// NewContractBackend creates a new native contract backend using an existing
// ionchain object.
func NewContractBackend(apiBackend ioncapi.Backend) *ContractBackend {
	return &ContractBackend{
		eapi:  ioncapi.NewPublicIONChainAPI(apiBackend),
		bcapi: ioncapi.NewPublicBlockChainAPI(apiBackend),
		txapi: ioncapi.NewPublicTransactionPoolAPI(apiBackend, new(ioncapi.AddrLocker)),
	}
}

// CodeAt retrieves any code associated with the contract from the local API.
func (b *ContractBackend) CodeAt(ctx context.Context, contract common.Address, blockNum *big.Int) ([]byte, error) {
	return b.bcapi.GetCode(ctx, contract, toBlockNumber(blockNum))
}

// CodeAt retrieves any code associated with the contract from the local API.
func (b *ContractBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return b.bcapi.GetCode(ctx, contract, rpc.PendingBlockNumber)
}

// ContractCall implements bind.ContractCaller executing an ionchain contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (b *ContractBackend) CallContract(ctx context.Context, msg ionchain.CallMsg, blockNum *big.Int) ([]byte, error) {
	out, err := b.bcapi.Call(ctx, toCallArgs(msg), toBlockNumber(blockNum))
	return out, err
}

// ContractCall implements bind.ContractCaller executing an ionchain contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (b *ContractBackend) PendingCallContract(ctx context.Context, msg ionchain.CallMsg) ([]byte, error) {
	out, err := b.bcapi.Call(ctx, toCallArgs(msg), rpc.PendingBlockNumber)
	return out, err
}

func toCallArgs(msg ionchain.CallMsg) ioncapi.CallArgs {
	args := ioncapi.CallArgs{
		To:   msg.To,
		From: msg.From,
		Data: msg.Data,
	}
	if msg.Gas != nil {
		args.Gas = hexutil.Big(*msg.Gas)
	}
	if msg.GasPrice != nil {
		args.GasPrice = hexutil.Big(*msg.GasPrice)
	}
	if msg.Value != nil {
		args.Value = hexutil.Big(*msg.Value)
	}
	return args
}

func toBlockNumber(num *big.Int) rpc.BlockNumber {
	if num == nil {
		return rpc.LatestBlockNumber
	}
	return rpc.BlockNumber(num.Int64())
}

// PendingAccountNonce implements bind.ContractTransactor retrieving the current
// pending nonce associated with an account.
func (b *ContractBackend) PendingNonceAt(ctx context.Context, account common.Address) (nonce uint64, err error) {
	out, err := b.txapi.GetTransactionCount(ctx, account, rpc.PendingBlockNumber)
	if out != nil {
		nonce = uint64(*out)
	}
	return nonce, err
}

// SuggestGasPrice implements bind.ContractTransactor retrieving the currently
// suggested gas price to allow a timely execution of a transaction.
func (b *ContractBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.eapi.GasPrice(ctx)
}

// EstimateGasLimit implements bind.ContractTransactor triing to estimate the gas
// needed to execute a specific transaction based on the current pending state of
// the backend blockchain. There is no guarantee that this is the true gas limit
// requirement as other transactions may be added or removed by miners, but it
// should provide a basis for setting a reasonable default.
func (b *ContractBackend) EstimateGas(ctx context.Context, msg ionchain.CallMsg) (*big.Int, error) {
	out, err := b.bcapi.EstimateGas(ctx, toCallArgs(msg))
	return out.ToInt(), err
}

// SendTransaction implements bind.ContractTransactor injects the transaction
// into the pending pool for execution.
func (b *ContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	raw, _ := rlp.EncodeToBytes(tx)
	_, err := b.txapi.SendRawTransaction(ctx, raw)
	return err
}
