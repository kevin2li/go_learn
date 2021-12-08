package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
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

func InsertTransaction(driver neo4j.Driver, tx Transaction) error {
	session := driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()
	in_addrs, out_addrs := GetTxInAddrs(tx), GetTxOutAddrs(tx)
	// var cql = "MERGE (addr1:Addr { address: $address1, name: $address1 })-[:Transfer]->(tx:Transaction { id: $txid, name: $txid })-[:Transfer]->(addr2:Addr { address: $address2, name: $address2 }) RETURN addr1, addr2, tx"
	var input_cql = "MERGE (addr1:Addr { address: $address1, name: $address1 })-[:Transfer]->(tx:Transaction { id: $txid, name: $txid }) RETURN addr1, tx;"
	var output_cql = "MERGE (tx:Transaction { id: $txid, name: $txid })-[:Transfer]->(addr2:Addr { address: $address2, name: $address2 }) RETURN addr2, tx"
	for _, addr := range in_addrs {
		params := map[string]interface{}{
			"address1": addr,
			"txid":     tx.Txid,
		}
		var insertInputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run(input_cql, params)
			if err != nil {
				return nil, err
			}
			return records, nil
		}
		_, err := session.WriteTransaction(insertInputFn)
		if err != nil {
			err = errors.Wrap(err, "insert transaction failed!")
			return err
		}
	}
	for _, addr := range out_addrs {
		params := map[string]interface{}{
			"address2": addr,
			"txid":     tx.Txid,
		}
		var insertInputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run(output_cql, params)
			if err != nil {
				return nil, err
			}
			return records, nil
		}
		_, err := session.WriteTransaction(insertInputFn)
		if err != nil {
			err = errors.Wrap(err, "insert transaction failed!")
			return err
		}
	}
	return nil
}

func main() {
	// Neo4j 4.0, defaults to no TLS therefore use bolt:// or neo4j://
	dbUri := "neo4j://localhost:7687"
	driver, err := neo4j.NewDriver(dbUri, neo4j.BasicAuth("neo4j", "test", ""))
	if err != nil {
		err = errors.Wrap(err, "")
		log.Fatal(err)
	}
	defer driver.Close()
	txs, err := ReadTransaction("/home/likai/code/go_program/go_learn/result/block_height=711900.json")
	if err != nil {
		log.Fatal(err)
	}

	err = InsertTransaction(driver, txs[25])
	if err != nil {
		err = errors.Wrap(err, "")
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
