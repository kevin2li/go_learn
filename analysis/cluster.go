package main

import (
	"fmt"
	"sync"
)

type Transaction struct {
	Txid     string `json:"txid"`
	Size     uint   `json:"size"`
	Version  uint   `json:"version"`
	Locktime uint   `json:"locktime"`
	Fee      uint   `json:"fee"`
	Inputs   []struct {
		Coinbase  bool     `json:"coinbase"`
		Txid      string   `json:"txid"`
		Output    uint     `json:"output"`
		Sigscript string   `json:"sigscript"`
		Sequence  uint64   `json:"sequence"`
		Pkscript  string   `json:"pkscript"`
		Value     uint     `json:"value"`
		Address   string   `json:"address"`
		Witness   []string `json:"witness"`
	} `json:"inputs"`
	Outputs []struct {
		Address  string `json:"address"`
		Pkscript string `json:"pkscript"`
		Value    uint   `json:"value"`
		Spent    bool   `json:"spent"`
		Spender  struct {
			Txid  string `json:"txid"`
			Input uint   `json:"input"`
		} `json:"spender,omitempty"`
		Input uint `json:"input,omitempty"`
	} `json:"outputs"`
	Block struct {
		Height   uint `json:"height"`
		Position uint `json:"position"`
	} `json:"block"`
	Deleted bool `json:"deleted"`
	Time    uint `json:"time"`
	Rbf     bool `json:"rbf"`
	Weight  uint `json:"weight"`
}

var addressList map[string]bool

var usedAddr map[string]bool

// if given addr in tx inputs
func IsInTxInputs(addr string, tx Transaction) bool {

	return false
}

// if given addr in tx outputs
func IsInTxOutputs(addr string, tx Transaction) bool {

	return false
}

func MultiInputHeuristic(addr string, tx Transaction) []string {

	return nil
}

func CoinbaseHeuristic(addr string, tx Transaction) []string {

	return nil
}

func ChangeHeuristic(addr string, tx Transaction) []string {

	return nil
}

func ClusterByAddr(addr string, txs []Transaction, addrList chan []string, wg *sync.WaitGroup) error {
	result := []string{}
	tempTxs := []Transaction{}
	result = append(result, addr)
	defer func() {
		wg.Done()
	}()
	// filter out unrelated transactions
	for _, tx := range txs {
		if IsInTxInputs(addr, tx) || IsInTxOutputs(addr, tx) {
			tempTxs = append(tempTxs, tx)
		}
	}
	// refine related address
	for _, tx := range tempTxs {
		// rule1
		out := MultiInputHeuristic(addr, tx)
		if out != nil {
			result = append(result, out...)
		}
		// rule2
		out = CoinbaseHeuristic(addr, tx)
		if out != nil {
			result = append(result, out...)
		}
		// rule3
		out = ChangeHeuristic(addr, tx)
		if out != nil {
			result = append(result, out...)
		}
	}
	addrList <- result
	return nil
}

func Cluster(addr string, txs []Transaction) []string {
	fmt.Printf("%+v\n", addressList)
	addressList[addr] = true
	var queue = make([]string, 0)
	queue = append(queue, addr)
	var wg sync.WaitGroup

	addrList := make(chan []string, 128)
	for len(queue) > 0 {
		a := queue[0]
		queue = queue[1:]
		usedAddr[a] = true
		wg.Add(1)
		go ClusterByAddr(addr, txs, addrList, &wg)
	}
	// for addrs := range <-addrList {
	// for _, addr := range addrs {
	// 	addressList[addr] = true

	// 	// TODO:把未迭代的地址入队
	// 	if !usedAddr[addr] {
	// 		queue = append(queue, addr)
	// 	}
	// }
	// }

	return nil
}

func main() {
	fmt.Println("Started!")
	// all_txs := []Transaction{}
	// addr := "3GpMzyMNaZkN5Lp7vHx7hpT3bQqc97zPb2"
	// Cluster(addr, all_txs)
	fmt.Println(100 / 20)
	fmt.Printf("\n------\nvalue:\t%v,\ntype:\t%T\n------\n", , )
	
}
