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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mteam88/keyswarm/multicall"
)

// config
const HQ = "http://localhost:8000/"
const initialGeneratorCount int = 4
const initialFiltererCount int = 75

var minimumBalanceWei *big.Int = big.NewInt(1)

const reportSpeed int = 2 // seconds
const MULTICALL_SIZE int = 8000

// definitions
var scannedKeys int = 0
var totalKeys int = 0
var ETHProviders []ETHProvider
var generators uint32
var filterers uint32

type ETHProvider struct {
	RawURL string
	isMax  bool
	client ethclient.Client
}

func (E ETHProvider) GetClient() ethclient.Client { return E.client }

func main() {
	ETHProviders = loadETHProviders()

	genkeys := make(chan []string, 100000)
	keyswithbalance := make(chan []string)

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
			time.Sleep(time.Second * time.Duration(reportSpeed))
			log.Default().Println("[$] Scanned Keys Per Second: ", (scannedKeys / reportSpeed))
			log.Default().Println("[$] Overflow: ", len(genkeys))
			log.Default().Println("[i] generators running", generators)
			log.Default().Println("[i] filterers running", filterers)
			totalKeys += scannedKeys
			scannedKeys = 0
		}
	}()
	go filterForBalance(genkeys, keyswithbalance)

	for i := 0; i < initialGeneratorCount; i++ {
		go generateKeys(genkeys, keyswithbalance)
	}

	go callhome(keyswithbalance)

	select {} // do not stop main thread
}
func generateKeys(generatedkeys chan []string, keyswithbalance chan []string) {
	atomic.AddUint32(&generators, 1)
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
		sendKey:
		for {
			select {
			case generatedkeys <- []string{privateKey, address}:
				break sendKey
			default:
				go filterForBalance(generatedkeys, keyswithbalance)
			}
		}
	}
}

func fakeFilter(generatedkeys chan []string, keyswithbalance chan []string) {
	atomic.AddUint32(&filterers, 1)
	for{
		<-generatedkeys
		scannedKeys++
	}
}

func filterForBalance(generatedkeys chan []string, keyswithbalance chan []string) {
	atomic.AddUint32(&filterers, 1)
	buf := make(chan []string, MULTICALL_SIZE)
	go func() {
		for {
			select {
			case buf <- <-generatedkeys: // Buffer not full
			default: // Buffer must be full, making multicall request
				var keysInBatch [MULTICALL_SIZE][]string
				for i := 0; i < MULTICALL_SIZE; i++ {
					keysInBatch[i] = <-buf
				}
				for keyIndex, hasBalance := range hasbalance(keysInBatch[:]) {
					if hasBalance == 0 || hasBalance == 1 {
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
	return multicall.GetBalances(publicKeyPairs, ethprovider) // Maybe should use client initialized earlier.
}

func beacon(keypair []string) {
	_, err := http.Get(HQ + keypair[0])
	if err != nil {
		panic(err.Error())
	}
	log.Default().Println("[!] BEACON CALL: " + "\n[-] Private: " + keypair[0] + "\n[-] Public: " + keypair[1])
}

func loadETHProviders() []ETHProvider {
	RawInfuraKeys := strings.Split(os.Getenv("INFURA_KEYS"), ",")
	InfuraKeys := []ETHProvider{}
	for _, key := range RawInfuraKeys {
		if key == "" {
			break
		}
		RawUrl := "https://mainnet.infura.io/v3/" + key
		client, err := ethclient.Dial(RawUrl)
		if err != nil {
			panic(err.Error())
		}
		InfuraKeys = append(InfuraKeys, ETHProvider{RawUrl, false, *client})
	}
	ETHProviders = append(ETHProviders, InfuraKeys...)
	if len(ETHProviders) == 0 {
		panic("Please provide some api keys. You may need a .env file. See the docs.")
	}
	return ETHProviders
}
