package main

import (
	"os/exec"
	"fmt"
	"strings"
	"sync"
	"crypto/rand"
	"math/big"
	"net/http"
)

//config
const HQ="http://localhost:8000/"
const producerCount int = 8
const minimumBalanceWei int = 1


func main() {
	jobs := make(chan []string)
	done := make(chan bool)
	wg := sync.WaitGroup{}

	for i := 0; i < producerCount; i++ {
		wg.Add(1)
		go generatekeys(jobs, i, &wg)
	}

	go callhome(jobs, done)

	wg.Wait()
	close(jobs) // should never happen
	<-done
}
func generatekeys(jobs chan<- []string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		page := new(big.Int)
		page.SetBytes(generateRandomBytes(249/8))
		fmt.Println(page)

		out, err := exec.Command("./xkeygen", "eth", page.String()).Output()
		if err != nil {
			panic(err)
		}
		keypairs := parsexkeygen(out)
		for _, kpair := range keypairs {
			go func(kpair []string) {
				if hasbalance(kpair) {
					jobs <- kpair
				}
			}(kpair)
		}
	}
}

func callhome(jobs <-chan []string, done chan<- bool) {
	for keypair := range jobs {
		beacon(keypair)
	}
	done <- true
}

func hasbalance(keypair []string) bool {
	return getbalance(keypair) > minimumBalanceWei
}

func getbalance(keypair []string) int { //returns wei balance of keypair
	return 6000
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

func parsexkeygen(out []byte) []([]string) {
	entries := strings.Split(string(out), "\n")
	output := make([][]string, 128)
	for i, entry := range entries {
		entry = strings.ReplaceAll(entry, "{", "")
		entry = strings.ReplaceAll(entry, "}", "")
		output[i] = strings.Split(entry, " ")
	}
	return output
}