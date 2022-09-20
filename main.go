package main

import (
	"os/exec"
	"fmt"
	"strings"
)

var HQ="https://www.google.com"
//var successCh := make(string, 5)

func main() {
	out, err := exec.Command("./xkeygen", "eth", "1").Output()
	if err != nil {
		panic(err)
	}
	fmt.Println(parsexkeygen(out))
}

func checkbalance(keypair []string) int {
	
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