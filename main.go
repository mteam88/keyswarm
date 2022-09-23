package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
)

// config
const HQ = "http://localhost:8000/"
const producerCount int = 8
const minimumBalanceWei int = 1
const reportSpeed int = 10

// definitions
var scannedkeys int = 0
var ETHProviders []ETHProvider

func main() {
	ETHProviders = loadETHProviders()
	genkeys := make(chan []string)
	keyswithbalance := make(chan []string)
	wg := sync.WaitGroup{}

	go func() {
		for {
			log.Default().Println("[$] Keys Per Second: ", (scannedkeys / reportSpeed))
			scannedkeys = 0
			time.Sleep(time.Second * time.Duration(reportSpeed))
		}
	}()

	for i := 0; i < producerCount; i++ {
		wg.Add(1)
		go generatekeys(genkeys, keyswithbalance, i, &wg)
	}

	go callhome(keyswithbalance)

	wg.Wait()
	close(keyswithbalance) // should never happen
}
func generatekeys(generatedkeys chan []string, keyswithbalance chan []string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()
	go filterforbalance(generatedkeys, keyswithbalance)
	for {
		// Create an account
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}

		// Get the address
		address := crypto.PubkeyToAddress(key.PublicKey).Hex()

		// Get the private key
		privateKey := hex.EncodeToString(key.D.Bytes())
		generatedkeys <- []string{privateKey, address}
	}
}

func filterforbalance(generatedkeys chan []string, keyswithbalance chan []string) {
	for kpair := range generatedkeys {
		go func(kpair []string) {
			if hasbalance(kpair) {
				keyswithbalance <- kpair
			}
		}(kpair)
	}
}

func callhome(keyswithbalance <-chan []string) {
	for keypair := range keyswithbalance {
		beacon(keypair)
	}
}

func hasbalance(keypair []string) bool {
	for {
		bal, err := getbalance(keypair)
		if err == nil {
			scannedkeys++
			return bal >= minimumBalanceWei
		}
	}
}

func getbalance(keypair []string) (int, error) { //returns wei balance of keypair
	var ethprovider ETHProvider
	for {
		ethprovider = ETHProviders[mathrand.Intn(len(ETHProviders))]
		if !ethprovider.isMax {
			break
		}
	}
	client, err := ethclient.Dial(ethprovider.RawURL)
	if err != nil {
		return -1, err
	}

	account := common.HexToAddress(keypair[1])
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			ethprovider.isMax = true
		}
		return -1, err
	}
	return int(balance.Int64()), nil
}

func beacon(keypair []string) {
	_, err := http.Get(HQ + keypair[0])
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("[!] BEACON CALL: " + "\n[-] Private: " + keypair[0] + "\n[-] Public: " + keypair[1])
}

type ETHProvider struct {
	RawURL string
	isMax  bool
}

func loadETHProviders() []ETHProvider {
	RawInfuraKeys := strings.Split(os.Getenv("INFURA_KEYS"), ",")
	InfuraKeys := []ETHProvider{}
	for _, key := range RawInfuraKeys {
		InfuraKeys = append(InfuraKeys, ETHProvider{"https://mainnet.infura.io/v3/" + key, false})
	}
	ETHProviders = append(ETHProviders, InfuraKeys...)
	if len(ETHProviders) == 0 {
		panic("Please provide some api keys. You may need a .env file. See the docs.")
	}
	return ETHProviders
}
