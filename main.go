package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	tm "github.com/buger/goterm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mteam88/keyswarm/multicall"
)

// config
const HQ = "http://localhost:8000/"
const initialGeneratorCount int = 4 // Best observed performance, not tested

var minimumBalanceWei *big.Int = big.NewInt(1)

const reportSpeed int = 1 // seconds
const MULTICALL_SIZE int = 8000

// definitions
var ETHProviders []ETHProvider
var ScannerState State
var HitKeysBox tm.Box

type ETHProvider struct {
	RawURL string
	isMax  bool
	client ethclient.Client
}

type State struct {
	totalKeys                int
	scannedKeys              int
	generators               uint32
	filterers                uint32
	runningMulticallRequests uint32
	completedMulticallRequests uint64
}

func (E ETHProvider) GetClient() ethclient.Client { return E.client }

func main() {
	ETHProviders = loadETHProviders()

	genkeys := make(chan []string, 1000)
	keyswithbalance := make(chan []string)

	HitKeysBox = *tm.NewBox(50|tm.PCT, 100|tm.PCT, 0)
	fmt.Fprintln(&HitKeysBox, "Hit Keys:")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			tm.Clear()
			tm.MoveCursor(1, 1)
			tm.Println("KEYSWARM - STOPPED")
			tm.Println("[X] Total Keys Scanned: ", ScannerState.totalKeys+ScannerState.scannedKeys)
			tm.Flush()
			os.Exit(0)
		}
	}()

	go func() {
		for {
			tm.Clear()
			tm.MoveCursor(1, 1)
			tm.Println("KEYSWARM - RUNNING")
			time.Sleep(time.Second * time.Duration(reportSpeed))
			tm.Println("[$] Total Scanned Keys:", ScannerState.totalKeys+ScannerState.scannedKeys)
			tm.Println("[$] Scanned Keys Per Second: ", (ScannerState.scannedKeys / reportSpeed))
			tm.Println("[i] generators running", ScannerState.generators)
			tm.Println("[i] requests running", ScannerState.runningMulticallRequests)
			tm.Println("[i] requests completed", ScannerState.completedMulticallRequests)
			tm.Println("[!] Private Keys with balance", len(keyswithbalance))

			// Move Box to approx center of the screen
			tm.Print(tm.MoveTo(HitKeysBox.String(), 50|tm.PCT, 0|tm.PCT))
			tm.Flush()
			ScannerState.totalKeys += ScannerState.scannedKeys
			ScannerState.scannedKeys = 0
		}
	}()
	filterForBalance(genkeys, keyswithbalance)

	for i := 0; i < initialGeneratorCount; i++ {
		go generateKeys(genkeys, keyswithbalance)
	}

	go callhome(keyswithbalance)

	select {} // do not stop main thread
}
func generateKeys(generatedkeys chan []string, keyswithbalance chan []string) {
	atomic.AddUint32(&ScannerState.generators, 1)
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


func filterForBalance(generatedkeys chan []string, keyswithbalance chan []string) chan []string {
	atomic.AddUint32(&ScannerState.filterers, 1)
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
				go func() {
					atomic.AddUint32(&ScannerState.runningMulticallRequests, 1)
					for keyIndex, hasBalance := range hasbalance(keysInBatch[:]) {
						if hasBalance == 0 || hasBalance == 1 {
							keyswithbalance <- keysInBatch[keyIndex]
						}
					}
					atomic.AddUint32(&ScannerState.runningMulticallRequests, ^uint32(0))
					atomic.AddUint64(&ScannerState.completedMulticallRequests, 1)
				}()
			}
		}
	}()
	return buf
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
			ScannerState.scannedKeys++
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

	fmt.Fprint(&HitKeysBox, "[!] BEACON CALL: " + "\n[-] Private: " + keypair[0] + "\n[-] Public: " + keypair[1]+"\n")
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
