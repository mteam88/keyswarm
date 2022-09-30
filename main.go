package main

import (
	"encoding/hex"
	"log"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mteam88/keyswarm/multicall"
)

// config
const HQ = "http://localhost:8000/"
const producerCount int = 8

var minimumBalanceWei *big.Int = big.NewInt(0)

const reportSpeed int = 5
const MULTICALL_SIZE int = 8000

// definitions
var scannedKeys int = 0
var totalKeys int = 0
var ETHProviders []ETHProvider

func main() {
	ETHProviders = loadETHProviders()

	genkeys := make(chan []string)
	keyswithbalance := make(chan []string)
	wg := sync.WaitGroup{}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Default().Println("[X] Total Keys Scanned: ", totalKeys+scannedKeys)
			panic("Keyboard Interrupt")
		}
	}()

	go func() {
		for {
			log.Default().Println("[$] Keys Per Second: ", (scannedKeys / reportSpeed))
			log.Default().Println("[$] Generatedkeys size: ", len(genkeys))
			totalKeys += scannedKeys
			scannedKeys = 0
			time.Sleep(time.Second * time.Duration(reportSpeed))
		}
	}()
	for i := 0; i < producerCount; i++ {
		wg.Add(1)
		go generatekeys(genkeys, keyswithbalance, i, &wg)
		// Another consumer that makes requests to check that accounts in genkeys have balance, then sends them to keyswithbalance
		go filterforbalance(genkeys, keyswithbalance)
	}

	go callhome(keyswithbalance)

	wg.Wait()
	close(keyswithbalance) // should never happen
}
func generatekeys(generatedkeys chan []string, keyswithbalance chan []string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()
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
	buf := make(chan []string, MULTICALL_SIZE)
	go func() {
		for kpair := range generatedkeys {
			buf <- kpair
		}
	}()
	go func() {
		for {
			if len(buf) >= MULTICALL_SIZE {
				var keysInBatch [MULTICALL_SIZE][]string
				for i := 0; i < MULTICALL_SIZE; i++ {
					keysInBatch[i] = <-buf
				}
				for keyIndex,hasBalance := range(hasbalance(keysInBatch[:])) {
					if hasBalance == 1 {
						keyswithbalance <- keysInBatch[keyIndex]
					}
				}
			}
		}
	}()
}

func callhome(keyswithbalance <-chan []string) {
	for keypair := range keyswithbalance {
		beacon(keypair)
	}
}

func hasbalance(keypairs [][]string) []int {
	var retVal []int
	bals, err := getbalance(keypairs)
	for _, bal := range bals {
		if err == nil {
			scannedKeys++
			retVal = append(retVal, bal.Cmp(minimumBalanceWei))
		} else {
			panic(err)
		}
	}
	return retVal
}

func getbalance(keypairs [][]string) ([]big.Int, error) { //returns slice of wei balances for keypairs
	var ethprovider ETHProvider
	for {
		ethprovider = ETHProviders[mathrand.Intn(len(ETHProviders))]
		if !ethprovider.isMax {
			break
		}
	}
	var publicKeyPairs []string
	for _, keyPair := range keypairs {
		publicKeyPairs = append(publicKeyPairs, keyPair[1])
	}
	return multicall.GetBalances(publicKeyPairs, ethprovider.RawURL) // Maybe should use client initialized earlier.
}

func beacon(keypair []string) {
	_, err := http.Get(HQ + keypair[0])
	if err != nil {
		panic(err.Error())
	}
	log.Default().Println("[!] BEACON CALL: " + "\n[-] Private: " + keypair[0] + "\n[-] Public: " + keypair[1])
}

type ETHProvider struct {
	RawURL string
	isMax  bool
}

func loadETHProviders() []ETHProvider {
	RawInfuraKeys := strings.Split(os.Getenv("INFURA_KEYS"), ",")
	InfuraKeys := []ETHProvider{}
	for _, key := range RawInfuraKeys {
		if key == "" {
			break
		}
		InfuraKeys = append(InfuraKeys, ETHProvider{"https://mainnet.infura.io/v3/" + key, false})
	}
	ETHProviders = append(ETHProviders, InfuraKeys...)
	if len(ETHProviders) == 0 {
		panic("Please provide some api keys. You may need a .env file. See the docs.")
	}
	return ETHProviders
}
