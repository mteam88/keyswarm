package multicall

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

var multicallContractAddress = common.HexToAddress("0x5e227AD1969Ea493B43F840cfF78d08a6fc17796")
var multicallContractEthBalanceSelector = "4d2301cc"

func GetBalances(addresses []string, ETHProviderURL string) ([]string, error) {
	ethProvider, err := ethclient.Dial(ETHProviderURL)
	if err != nil {
		panic(err)
	}
	multicallContract, err := NewMulticallCaller(multicallContractAddress, ethProvider)
	if err != nil {
		panic(err)
	}

	fmt.Println("multicallContract: ", multicallContract)

	var calls = []MulticallCall{}
	for _, address := range addresses {
		hashAddress := common.HexToHash(address)
		call := MulticallCall{multicallContractAddress, []byte("0x"+multicallContractEthBalanceSelector+hashAddress.String()[2:])}
		calls = append(calls, call)
	}

	fmt.Println(string(calls[0].CallData))

	var results []byte

	err = multicallContract.contract.Call(&bind.CallOpts{}, nil, "aggregate", common.Hex2Bytes(addresses[0]))
	if err != nil {
		panic(err)
	}
	fmt.Println("Result: ", &results)
	return nil, nil
}