package ipos

import (
	"context"
	"fmt"
	"github.com/ionchain/ionchain-core"
	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/ioncclient"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/rpc"
	"golang.org/x/crypto/sha3"
	"math/big"
	"strings"
	"sync"
)

const (
	IPOSContractAddress = "IONC11111111111111111112DHgZTF"
)

var ecClient *ioncclient.Client

var rpcLocker = new(sync.Mutex)

func mintPower(addr common.Address, ipcEndpoint string) (*big.Int, error) {
	ec, err := dialRPC(ipcEndpoint) //ipc:~/.ionc/db/ionc.ipc 成功后放入一个缓存
	if err != nil {
		fmt.Errorf("error %v \n", err)
		return nil, err
	}

	hw := sha3.NewLegacyKeccak256()
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
func dialRPC(endpoint string) (*ioncclient.Client, error) {
	rpcLocker.Lock()
	defer rpcLocker.Unlock()
	if ecClient != nil {
		return ecClient, nil
	}

	if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		endpoint = endpoint[4:]
	}
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	ec := ioncclient.NewClient(client)
	if ecClient == nil {
		ecClient = ec
	}

	return ec, nil
}
