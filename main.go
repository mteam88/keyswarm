package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
)

//config
const HQ="http://localhost:8000/"
const producerCount int = 8
const minimumBalanceWei int = 1
var InfuraKeys []string = strings.Split(os.Getenv("INFURA_KEYS"), ",")
var scannedkeys int = 0

func main() {
	genkeys := make(chan []string)
	keyswithbalance := make(chan []string)
	done := make(chan bool)
	wg := sync.WaitGroup{}

	go func() {
		lastkeys := make([]int, 5) 
		for {
			fmt.Println("[$] Keys Per Second: ", (sum(lastkeys)/len(lastkeys)))
			lastkeys = append(lastkeys, scannedkeys)
			time.Sleep(time.Second)
		}
	}()

	for i := 0; i < producerCount; i++ {
		wg.Add(1)
		go generatekeys(genkeys, keyswithbalance, i, &wg)
	}

	go callhome(keyswithbalance, done)

	wg.Wait()
	close(keyswithbalance) // should never happen
	<-done
}
func generatekeys(generatedkeys chan []string, keyswithbalance chan []string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()
	go filterforbalance(generatedkeys, keyswithbalance)
	for {
		page := new(big.Int)
		page.SetBytes(generateRandomBytes(249/8))
		// fmt.Println("[:] Scanning page: ", page)

		out, err := exec.Command("./xkeygen", "eth", page.String()).Output()
		if err != nil {
			panic(err)
		}
		go parsexkeygen(out, generatedkeys)
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

func callhome(keyswithbalance <-chan []string, done chan<- bool) {
	for keypair := range keyswithbalance {
		beacon(keypair)
	}
	done <- true
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
	infuraKey := InfuraKeys[mathrand.Intn(len(InfuraKeys))]
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/" + infuraKey)
    if err != nil {
        return -1, err
    }

    account := common.HexToAddress(keypair[1])
    balance, err := client.BalanceAt(context.Background(), account, nil)
    if err != nil {
        return -1, err
    }
	return int(balance.Int64()), nil
}

func generateRandomBytes(n int) ([]byte) {

    b := make([]byte, n)
    _, err := rand.Read(b)
    
    if err != nil {
        return nil
    }

    return b
}

func beacon(keypair []string) {
	_, err := http.Get(HQ+keypair[0])
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("[!] BEACON CALL: " +"\n[-] Private: "+ keypair[0] +"\n[-] Public: "+ keypair[1])
}

func parsexkeygen(out []byte, outch chan []string) {
	entries := strings.Split(string(out), "\n")
	for _, entry := range entries {
		entry = strings.ReplaceAll(entry, "{", "")
		entry = strings.ReplaceAll(entry, "}", "")
		outch <-strings.Split(entry, " ")
	}
}

func sum(array []int) int {  
	result := 0  
	for _, v := range array {  
	 result += v  
	}  
	return result  
   }