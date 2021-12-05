package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Block struct {
	Hash      string   `json:"hash"`
	Height    uint     `json:"height"`
	Mainchain bool     `json:"mainchain"`
	Previous  string   `json:"previous"`
	Time      uint     `json:"time"`
	Version   uint     `json:"version"`
	Bits      uint     `json:"bits"`
	Nonce     uint64   `json:"nonce"`
	Size      uint     `json:"size"`
	Tx        []string `json:"tx"`
	Merkle    string   `json:"merkle"`
	Subsidy   uint     `json:"subsidy"`
	Fees      uint     `json:"fees"`
	Outputs   uint64   `json:"outputs"`
	// Work      uint64   `json:"work"`
	Weight    uint     `json:"weight"`
}

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

// get tx input & output
// url := "https://api.blockchain.info/haskoin-store/btc/transactions?txids=42ace0b46416a974df571b43eb9472a0e45d7436e51602bd7eded5459def3222,811981424b3e2c946b136f9781aeb8dab46c767e26043b440b57febeacdbcd34,13b4916e84da996b5c46953df97a1a0b369cac18cfd23aa3dd965817feff3629,6e6db525382675d18ca64dd651ebb8b20edf84a6302cf582d2e573707f372c52,e0f6c1ea18fe613e28500bbf3fefd569b6bc0d13e3e0cf2f4da5136cec7a2a31"

// get tx_hash
// url := "https://api.blockchain.info/haskoin-store/btc/block/0000000000000000000730d0e713fd5f9bdd385216c544ce50765cd29ee23b1c?notx=false"

// get block_hash
// url := "https://api.blockchain.info/haskoin-store/btc/block/heights?heights=712538,712539&notx=false"

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func GetBlocksByHeights(heights string, blocks *[]Block) {
	/* construct url */
	url := fmt.Sprintf("https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false", heights)
	fmt.Printf("Get: %s\n", url)

	/* construct request */
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	check(err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	check(err)

	/* save response */
	err = json.Unmarshal(body, blocks)
	check(err)
	fmt.Printf("%s\n", "Hit success!")
}

func GetTxsByHashs(txHashs string, tx *[]Transaction) {
	/* construct url */
	url := fmt.Sprintf("https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s", txHashs)
	fmt.Printf("Get: %s\n", url)

	/* construct request */
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	check(err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	check(err)

	/* save response */
	// fmt.Printf("%s\n", string(body))
	err = json.Unmarshal(body, tx)
	check(err)
	fmt.Printf("%s\n\n", "Hit success")
}

func Crawl(url string, done chan bool) {
	fmt.Printf("\nGet: %s\n", url)
	resp, err := http.Get(url)
	check(err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	check(err)
	fmt.Printf("%s\n", string(body)[:50])
	done <- true
}

func Save(path string, content []byte){
    file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	check(err)
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.Write(content)
	writer.Flush()
	check(err)
}

func ExtractOneBlock(block *Block, path string){
	/* 1. get all tx_hash in this block and construct tx_hashs*/
	var txs []Transaction
	tx_hashs := block.Tx[0]
	for _, tx_hash := range block.Tx[1:]{
		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
	}
	/* 2. issue request and save response(tx json) */
	GetTxsByHashs(tx_hashs, &txs)
	obj, err := json.Marshal(txs)
	check(err)
	Save(path, obj)
}

func ExtractAllBlocks(blocks []Block){
	n := len(blocks)
	for i, block := range blocks {
		fmt.Printf("%d/%d: extract block with height=%d and hash=%s\n", i,n, block.Height, block.Hash)
		path := fmt.Sprintf("height=%d.json", block.Height)
		ExtractOneBlock(&block, path)
	}
}

func main() {
	fmt.Println("---Started!---")
	/* GET TOP N BLOCKS */
	/* 1. construct heights*/
	latest_block := 712603
	heights := fmt.Sprintf("%d", latest_block)
	// for i := latest_block-1; i > latest_block-3; i-- {
	// 	heights = fmt.Sprintf("%s,%d", heights, i)
	// }
	/* 2. issue request */
	var blocks []Block
	GetBlocksByHeights(heights, &blocks)

	/* EXTRACT TRANSACTIONS IN ALL BLOCKS */
	ExtractAllBlocks(blocks)
	// for i, block := range blocks {
	// 	fmt.Printf("%d: height: %d, blockhash: %s\n", i, block.Height, block.Hash)
	// 	/* 1. get all tx_hash in this block and construct tx_hashs*/
	// 	var txs []Transaction
	// 	tx_hashs := block.Tx[0]
	// 	for _, tx_hash := range block.Tx[1:]{
	// 		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
	// 	}
	// 	/* 2. issue request and save response(tx json) */
	// 	GetTxsByHashs(tx_hashs, &txs)
	// 	obj, err := json.Marshal(txs)
	// 	check(err)
	// 	Save("result.json", obj)
	// }
	
	// Save("a.txt", []byte("sdjhshfk"))
	// done := make(chan bool, 5)
	// go CrawlTx("0000000000000000000730d0e713fd5f9bdd385216c544ce50765cd29ee23b1c", done)
	// go Crawl("https://api.blockchain.info/haskoin-store/btc/block/0000000000000000000730d0e713fd5f9bdd385216c544ce50765cd29ee23b1c?notx=false", done)
	// <-done
	// <-done
	// close(done)
	fmt.Println("---Finished!---")
}
