package multicall

import (
	"context"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var multicallContractAddress = common.HexToAddress("0x5e227AD1969Ea493B43F840cfF78d08a6fc17796")

//var multicallContractEthBalanceSelector = "4d2301cc"

func GetBalances(addresses []string, ETHProviderURL string) ([]big.Float, error) {
	var balances []big.Float
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

	for _, address := range addresses {
		calldata, err := multicallContractABI.Pack("getEthBalance", common.HexToAddress(address))
		if err != nil {
			panic(err)
		}
	
		var callmsg ethereum.CallMsg
		callmsg.To = &multicallContractAddress // the destination contract (nil for contract creation)
		callmsg.Data = calldata
	
		result, err := ethProvider.CallContract(context.Background(), callmsg, nil)
		if err != nil {
			panic(err)
		}
		intBalance := new(big.Int)
		intBalance.SetBytes(result)
		floatWeiBalance := new(big.Float).SetInt(intBalance)
		floatWeiBalance.Quo(floatWeiBalance, new(big.Float).SetFloat64(params.Ether))
		balances = append(balances, *floatWeiBalance)
	}
	return balances, nil
}
