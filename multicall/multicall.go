package multicall

import (
	"context"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var multicallContractAddress = common.HexToAddress("0x5e227AD1969Ea493B43F840cfF78d08a6fc17796")

type ETHProviderInterface interface {
    GetClient() ethclient.Client;
}
func GetBalances(addresses []string, ETHProvider ETHProviderInterface) ([]big.Int, error) {
	var balances []big.Int

	ethProvider := ETHProvider.GetClient()

	abiReader, err := os.Open("/workspaces/keyswarm/multicall/multicallContract.abi")
	if err != nil {
		panic(err)
	}

	multicallContractABI, err := abi.JSON(abiReader)
	if err != nil {
		panic(err)
	}

	type Call struct {
		Target   common.Address
		CallData []byte
	}

	var calldatas []Call
	for _, address := range addresses {
		individualcalldata, err := multicallContractABI.Pack("getEthBalance", common.HexToAddress(address))
		calldatas = append(calldatas, Call{multicallContractAddress, individualcalldata})
		if err != nil {
			panic(err)
		}
	}

	calldata, err := multicallContractABI.Pack("aggregate", calldatas)
	if err != nil {
		panic(err)
	}

	var callmsg ethereum.CallMsg
	callmsg.To = &multicallContractAddress
	callmsg.Data = calldata

	var rawresult []byte
	for {
		rawresult, err = ethProvider.CallContract(context.Background(), callmsg, nil)
		if err != nil {
			if err.Error() == "execution aborted (timeout = 5s)" {
				log.Default().Println("[!] Timout hit, retrying")
				continue
			}
			panic(err)
		}
		break // no error, continue through the program
	}

	result, err := multicallContractABI.Methods["aggregate"].Outputs.UnpackValues(rawresult)
	if err != nil {
		panic(err)
	}

	for _, rawBalance := range result[1].([][]byte) {
		intBalance := new(big.Int)
		intBalance.SetBytes(rawBalance)
		balances = append(balances, *intBalance)
	}

	return balances, nil
}
