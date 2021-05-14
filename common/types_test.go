package common

import (
	"fmt"
	"github.com/ionchain/ionchain-core/common/base58"
	"math/big"
	"testing"
)

func TestBase58ToAddress(t *testing.T) {
	addr := "IONCNWtZkzqPEVM4wxX9j1MqY7ra9P2quNPF1"
	res, err := Base58ToAddress(addr)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)
}

func TestGetAddress(t *testing.T) {
	baseAddress := BigToAddress(big.NewInt(10))//0x000000000000000000000000000000000000000A
	fmt.Println(baseAddress)
	fmt.Println(baseAddress.String())
}

func TestNewAddressToBytes(t *testing.T) {
	addr := "IONC11111111111111111112DHgZTF"
	res, _ := Base58ToAddress(addr)
	fmt.Println(res)

	r := []byte{'1','2','3','4'}
	fmt.Println(r)
}

func TestBase58(t *testing.T){
	str := "123456"

	res := base58.Encode([]byte(str))
	fmt.Println(res) //RVu1HWU5

	res1,_ := base58.Decode("RVu1HWU5")
	fmt.Println(string(res1))//123456

	if str != string(res1){
		t.Error("base58 error")
	}

}
