package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/pkg/errors"
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

func ReadTransaction(path string) ([]Transaction, error) {
	var txs []Transaction
	obj, err := os.ReadFile(path)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("read file: %s error", path))
		return nil, err
	}
	err = json.Unmarshal(obj, &txs)
	if err != nil {
		err = errors.Wrap(err, "unmarshall error")
		return nil, err
	}
	return txs, nil
}

func GetTxInAddrs(tx Transaction) []string {
	var in_addrs []string
	for _, utxo := range tx.Inputs {
		in_addrs = append(in_addrs, utxo.Address)
	}
	return in_addrs
}

func GetTxOutAddrs(tx Transaction) []string {
	var out_addrs []string
	for _, utxo := range tx.Outputs {
		out_addrs = append(out_addrs, utxo.Address)
	}
	return out_addrs
}

// if given addr in tx inputs
func IsInTxInputs(addr string, tx Transaction) bool {
	in_addrs := GetTxInAddrs(tx)
	for _, cur_addr := range in_addrs {
		if cur_addr == addr {
			return true
		}
	}
	return false
}

// if given addr in tx outputs
func IsInTxOutputs(addr string, tx Transaction) bool {
	out_addrs := GetTxOutAddrs(tx)
	for _, cur_addr := range out_addrs {
		if cur_addr == addr {
			return true
		}
	}
	return false
}

// remove duplicate addrs
func Unique(addrs []string) []string {
	result := make([]string, 0)
	m := make(map[string]bool)
	for _, addr := range addrs {
		if _, ok := m[addr]; !ok {
			result = append(result, addr)
			m[addr] = true
		}
	}
	return result
}

func MultiInputHeuristic(addr string, tx Transaction) []string {
	if IsInTxInputs(addr, tx) {
		in_addrs := GetTxInAddrs(tx)
		return in_addrs
	}
	return nil
}

func CoinbaseHeuristic(addr string, tx Transaction) []string {
	if IsInTxOutputs(addr, tx) {
		out_addrs := GetTxOutAddrs(tx)
		return out_addrs
	}
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
	txs, err := ReadTransaction("/home/likai/code/go_program/go_learn/result/block_height=712039.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(txs))
	// fmt.Printf("%+v\n", txs[0])
	fmt.Println("Inputs addr:")
	tx := txs[100]
	fmt.Printf("%+v\n", GetTxInAddrs(tx))
	fmt.Println("Outputs addr:")
	fmt.Printf("%+v\n", GetTxOutAddrs(tx))
	fmt.Printf("%+v\n", Unique(GetTxInAddrs(tx)))
}
