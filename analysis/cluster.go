package main

import (
	"fmt"
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

// if given addr in tx inputs
func IsInTxInputs(addr string, tx Transaction) bool{

	return false
}

// if given addr in tx outputs
func IsInTxOutputs(addr string, tx Transaction) bool{

	return false
}


func ClusterByAddr(addr string, txs []Transaction) ([]string, error) {
	result := []string{}
	tempTxs := []Transaction{}
	result = append(result, addr)

	for _, tx := range txs {
		if IsInTxInputs(addr, tx) || IsInTxOutputs(addr, tx){
			tempTxs = append(tempTxs, tx)
		}else{
			// delete tx in txs

		}
	}
	
	return result, nil
}

func main() {
	fmt.Println("Started!")
}
