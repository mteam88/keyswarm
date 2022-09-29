package multicall

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var multicallContractAddress = common.HexToAddress("0x5e227AD1969Ea493B43F840cfF78d08a6fc17796")

func GetBalances(addresses []string, ETHProviderURL string) ([]big.Int, error) {
	var balances []big.Int

	ethProvider, err := ethclient.Dial(ETHProviderURL)
	if err != nil {
		panic(err)
	}

	abiReader, err := os.Open("/workspaces/keyswarm/multicall/multicallContract.abi")
	if err != nil {
		panic(err)
	}

	multicallContractABI, err := abi.JSON(abiReader)
	if err != nil {
		panic(err)
	}

	type Call struct {
        target string;
        callData []byte;
    }

	var calldatas []Call;
	for _, address := range addresses {
		individualcalldata, err := multicallContractABI.Pack("getEthBalance", common.HexToAddress(address))
		calldatas = append(calldatas, Call{multicallContractAddress.String(), individualcalldata})
		if err != nil {
			panic(err)
		}
	}

	m, exists := multicallContractABI.Methods["aggregate"]
	if exists {
		fmt.Println(m.Inputs.Pack(calldatas))
	}

	var callmsg ethereum.CallMsg
	callmsg.To = &multicallContractAddress
//	callmsg.Data = calldata

	result, err := ethProvider.CallContract(context.Background(), callmsg, nil)
	if err != nil {
		panic(err)
	}
	intBalance := new(big.Int)
	intBalance.SetBytes(result)
	balances = append(balances, *intBalance)
	return balances, nil
}
