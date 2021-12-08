package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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

type HashSet map[string]bool

func (h HashSet) Len() int {
	return len(h)
}

func (h HashSet) Add(s string) {
	if _, ok := h[s]; !ok {
		h[s] = true
	}
}

func (h HashSet) Remove(s string) error {
	if _, ok := h[s]; !ok {
		return errors.New(fmt.Sprintf("KeyError: Key `%s` does not exist!", s))
	}
	delete(h, s)
	return nil
}

func (h HashSet) GetData() []string {
	var result []string
	for k := range h {
		result = append(result, k)
	}
	return result
}

// remove duplicate addrs
func Unique(addrs []string) []string {
	set := make(HashSet)
	for _, addr := range addrs {
		set.Add(addr)
	}
	result := set.GetData()
	return result
}

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

func IsCoinbaseTx(tx Transaction) bool {
	return tx.Inputs[0].Coinbase
}

func MultiInputHeuristic(addr string, tx Transaction) []string {
	if IsInTxInputs(addr, tx) {
		in_addrs := GetTxInAddrs(tx)
		return in_addrs
	}
	return nil
}

func CoinbaseHeuristic(addr string, tx Transaction) []string {
	if IsCoinbaseTx(tx) && IsInTxOutputs(addr, tx) {
		out_addrs := GetTxOutAddrs(tx)
		return out_addrs
	}
	return nil
}

// TODO
func ChangeHeuristic(addr string, tx Transaction) []string {

	return nil
}

func ClusterByAddr(addr string, txs []Transaction, addrList chan []string) {
	var result []string
	result = append(result, addr)
	for _, tx := range txs {
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
	result = Unique(result)
	addrList <- result
}

func Cluster(addr string, txs []Transaction) []string {
	var finalAddrList = make(HashSet)
	finalAddrList.Add(addr)
	var queue = make([]string, 0)
	queue = append(queue, addr)
	var iterations = 1
iter:
	fmt.Printf("================================Iteration %d started!================================\n", iterations)
	fmt.Printf("INFO: total: %d addresses.\n", len(queue))
	var n = len(queue)
	addrList := make(chan []string, n)
	for i := 0; i < n; i++ {
		addr := queue[i]
		fmt.Printf("[%d/%d] Start cluster from address: %s...\n", i+1, n, addr)
		go ClusterByAddr(addr, txs, addrList)
	}
	queue = make([]string, 0)
	for i := 0; i < n; i++ {
		addrs := <-addrList
		for _, addr := range addrs {
			if _, ok := finalAddrList[addr]; !ok {
				queue = append(queue, addr)
				finalAddrList.Add(addr)
			}
		}
	}
	// whether have new address
	if len(queue) > 0 {
		fmt.Printf("new addresses added: %+v\n", queue)
		iterations++
		goto iter
	}
	result := finalAddrList.GetData()
	return result
}

func main() {
	txs, err := ReadTransaction("/home/likai/code/go_program/go_learn/result/block_height=712039.json")
	if err != nil {
		log.Fatal(err)
	}
	start_addr := "209140e6850ff1bda7ca9e49492d5e9741333be5ae9adc6886e45ba46fdd6800"
	fmt.Printf("INFO: Start cluster from address: %s!\n", start_addr)
	result := Cluster(start_addr, txs)
	fmt.Println("--------------------------------Cluster Finished!--------------------------------")
	fmt.Printf("INFO: Final cluster result: %v\n", result)

	// for _, tx := range txs {
	// 	fmt.Println(tx.Txid, len(GetTxInAddrs(tx)))
	// }
}
