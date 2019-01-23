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
	"fmt"
	"sync"
)

const (
	IPOSContractAddress = "0x0000000000000000000000000000000000000100"
)

var ecClient *ethclient.Client

var rpcLocker = new(sync.Mutex)

func mintPower(addr common.Address, ipcEndpoint string) (*big.Int, error) {
	ec, err := dialRPC(ipcEndpoint) //ipc:~/.ionc/db/geth.ipc 成功后放入一个缓存
	if err != nil {
		fmt.Errorf("error %v \n", err)
		return nil, err
	}

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
func dialRPC(endpoint string) (*ethclient.Client, error) {
	rpcLocker.Lock()
	defer rpcLocker.Unlock()
	if ecClient != nil {
		return ecClient, nil
	}

	if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		endpoint = endpoint[4:]
	}
	client, err := rpc.Dial(endpoint)
	if (err != nil) {
		return nil, err
	}

	ec := ethclient.NewClient(client)
	if ecClient == nil {
		ecClient = ec
	}

	return ec, nil
}
