package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/pkg/errors"
	pb "github.com/schollz/progressbar/v3"
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

type Params = map[string]interface{}

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
		if utxo.Address != "" {
			in_addrs = append(in_addrs, utxo.Address)
		}
	}
	return in_addrs
}

func GetTxOutAddrs(tx Transaction) []string {
	var out_addrs []string
	for _, utxo := range tx.Outputs {
		if utxo.Address != "" {
			out_addrs = append(out_addrs, utxo.Address)
		}
	}
	return out_addrs
}

func GetTxTime(tx Transaction) string {
	timeLayout := "2006-01-02 15:04:05"
	return time.Unix(int64(tx.Time), 0).Format(timeLayout)
}

func InsertTransaction(driver neo4j.Driver, tx Transaction) error {
	session := driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()
	in_addrs, out_addrs := GetTxInAddrs(tx), GetTxOutAddrs(tx)
	var createTx_cql = "MERGE (tx:Transaction {id: $txid, name: $txid}, in_degree: $in_degree, out_degree: $out_degree, time: $time, height: $height)"
	// 1. create tx node
	params := Params{
		"txid": tx.Txid,
		"in_degree": len(in_addrs),
		"out_degree": len(out_addrs),
		"time": GetTxTime(tx),
		"height": tx.Block.Height,
	}
	var insertInputFn = func(tx neo4j.Transaction) (interface{}, error) {
		records, err := tx.Run(createTx_cql, params)
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
	// 2. create tx input addrs node
	for _, addr := range in_addrs {
		params := Params{
			"address1": addr,
			"txid":     tx.Txid,
		}
		// create addr node
		var insertInputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run("MERGE (addr1:Addr { address: $address1, name: $address1 }) RETURN addr1", params)
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
		// create relationship
		insertInputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run("MATCH (addr1:Addr { address: $address1, name: $address1 }), (tx:Transaction {id: $txid, name: $txid}) CREATE (addr1)-[:In]->(tx)  RETURN addr1, tx", params)
			if err != nil {
				return nil, err
			}
			return records, nil
		}
		_, err = session.WriteTransaction(insertInputFn)
		if err != nil {
			err = errors.Wrap(err, "insert transaction failed!")
			return err
		}
	}
	// 3. create tx output addrs node
	for _, addr := range out_addrs {
		params := Params{
			"address2": addr,
			"txid":     tx.Txid,
		}
		// create addr node
		var insertOutputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run("MERGE (addr2:Addr { address: $address2, name: $address2 }) RETURN addr2", params)
			if err != nil {
				return nil, err
			}
			return records, nil
		}
		_, err := session.WriteTransaction(insertOutputFn)
		if err != nil {
			err = errors.Wrap(err, "insert transaction failed!")
			return err
		}
		// create relationship
		insertOutputFn = func(tx neo4j.Transaction) (interface{}, error) {
			records, err := tx.Run("MATCH (addr2:Addr { address: $address2, name: $address2 }), (tx:Transaction {id: $txid, name: $txid}) CREATE (tx)-[:Out]->(addr2) RETURN addr2, tx", params)
			if err != nil {
				return nil, err
			}
			return records, nil
		}
		_, err = session.WriteTransaction(insertOutputFn)
		if err != nil {
			err = errors.Wrap(err, "insert transaction failed!")
			return err
		}
	}
	return nil
}

func GetProgressBar(max int) *pb.ProgressBar {
	bar := pb.NewOptions(max,
		// pb.OptionSetWriter(ansi.NewAnsiStdout()),
		pb.OptionEnableColorCodes(true),
		pb.OptionShowBytes(true),
		pb.OptionSetWidth(40),
		pb.OptionShowCount(),
		pb.OptionThrottle(65*time.Millisecond),
		pb.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		pb.OptionSetTheme(pb.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
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
	var n = len(txs)
	bar := GetProgressBar(n)
	defer bar.Close()
	for _, tx := range txs {
		err = InsertTransaction(driver, tx)
		if err != nil {
			err = errors.Wrap(err, "")
			log.Fatal(err)
		}
		bar.Add(1)
	}
	fmt.Println("Done!")
}
