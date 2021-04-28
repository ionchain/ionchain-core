package common

import (
	"fmt"
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
