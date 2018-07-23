package ipos

import (
	"strings"
	"github.com/ionchain/ionchain-core/rpc"
	ionchain "github.com/ionchain/ionchain-core"
	"context"
	"github.com/ionchain/ionchain-core/ethclient"
	"github.com/ionchain/ionchain-core/common"
	"math/big"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/crypto/sha3"
)

const (
	IPOSContractAddress = "0x0000000000000000000000000000000000000100"
)

func mintPower(addr common.Address, ipcEndpoint string) (*big.Int, error) {
	client, err := dialRPC(ipcEndpoint) //ipc:~/.ionc/db/geth.ipc
	if err != nil {
		return nil, err
	}

	ec := ethclient.NewClient(client)

	hw := sha3.NewKeccak256()
	hw.Write([]byte("mintPower(address)"))
	funcName := hw.Sum(nil)[0:4]

	data := append(funcName[:], common.LeftPadBytes(addr[:], 32)...)

	contractAddr := common.HexToAddress(IPOSContractAddress)
	call := ionchain.CallMsg{To: &contractAddr, Data: data}
	ctx := context.Background()
	a, _ := ec.CallContract(ctx, call, nil)
	ether := big.NewInt(params.Ether)
	b := new(big.Int).SetBytes(a)

	return b.Div(b, ether), nil

}
func dialRPC(endpoint string) (*rpc.Client, error) {
	if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		// Backwards compatibility with geth < 1.5 which required
		// these prefixes.
		endpoint = endpoint[4:]
	}
	return rpc.Dial(endpoint)
}
