package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	progressbar "github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

const (
	page    = 5 // number of tx each request get
	bufsize = 4 // buffer size of chan
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
	Weight uint `json:"weight"`
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

func check(err error, message string) {
	if err != nil {
		if message != "" {
			log.Println(message)
		}
		log.Fatal(err)
	}
}

func GetBlocksByHeights(heights string) []Block {
	/* construct request */
	url := fmt.Sprintf("https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false", heights)
	req, err := http.NewRequest("GET", url, nil)
	check(err, fmt.Sprintf("request for %s failed!", url))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	check(err, "request failed")
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	check(err, "read response failed")

	/* save response */
	var blocks []Block
	err = json.Unmarshal(body, &blocks)
	check(err, "Unmarshal failed")
	return blocks
}

func GetBlocksInRange(low int, high int) []Block {
	var all_blocks []Block
	for i := high; i > low; i-- {
		blocks := GetBlocksByHeights(strconv.Itoa(i))
		all_blocks = append(all_blocks, blocks...)
	}
	return all_blocks
}

func GetTxsByHashs(txHashs string) []Transaction {
	/* construct request */
	url := fmt.Sprintf("https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s", txHashs)
	req, err := http.NewRequest("GET", url, nil)
	check(err, fmt.Sprintf("request for %s error!", url))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	check(err, "request failed")
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	check(err, "read response failed")

	/* save response */
	var txs []Transaction
	err = json.Unmarshal(body, &txs)
	check(err, "unmarshall failed")
	return txs
}

func Save(path string, content []byte) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	check(err, "open file failed")
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.Write(content)
	writer.Flush()
	check(err, "save file failed")
}

func ExtractOneBlock(block *Block, page int, tx_chan chan []Transaction) {
	p, n, tx_hashs := 0, len(block.Tx), ""
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, block.Height)
	bar := GetProgressBar(n, desc)
	var all_txs []Transaction
	for i, tx_hash := range block.Tx {
		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
		p++
		// every <page> hash issue a request
		if (p+1)%page == 0 || i == n-1 {
			var txs []Transaction = GetTxsByHashs(tx_hashs[1:])
			all_txs = append(all_txs, txs...)
			p, tx_hashs = 0, ""
			bar.Add(p)
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i, n, block.Height))
		}
	}
	bar.Close()
	tx_chan <- all_txs
}

func ExtractAllBlocks(blocks []Block, page int, bufsize int) []Transaction {
	n := len(blocks)
	fmt.Printf("INFO: Extract block with height between %d and %d...\n", blocks[n-1].Height, blocks[0].Height)
	var all_txs []Transaction
	tx_chan := make(chan []Transaction, bufsize)
	for _, block := range blocks {
		go ExtractOneBlock(&block, page, tx_chan)
		txs := <-tx_chan
		all_txs = append(all_txs, txs...)
	}
	return all_txs
}

func GetProgressBar(max int, desc string) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(max,
		// progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(100),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
}

func Start(low, high int, savepath string) {
	t1 := time.Now()
	fmt.Println("Started at:", t1)
	var blocks []Block = GetBlocksInRange(low, high)
	all_txs := ExtractAllBlocks(blocks, page, bufsize)
	obj, err := json.Marshal(all_txs)
	check(err, "Marshal Error")
	Save(savepath, obj)
	t2 := time.Now()
	fmt.Println("\nFinished at:", time.Now())
	fmt.Printf("Time elapsed: %.2f minutes\n", t2.Sub(t1).Minutes())
}

func main() {
	var (
		low, high int
		savepath  string
	)
	app := &cli.App{
		Name:  "Crawl",
		Usage: "crawl [-l low] [-h high] [-s savepath]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "savepath",
				Aliases:     []string{"s"},
				Usage:       "Save result at `SAVEPATH`",
				Destination: &savepath,
				Value:       "result.json",
			},
			&cli.IntFlag{
				Name:        "low",
				Aliases:     []string{"l"},
				Usage:       "Start at block height `LOW`",
				Destination: &low,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "high",
				Aliases:     []string{"e"},
				Usage:       "End at block height `HIGH`(not included)",
				Destination: &high,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			Start(low, high, savepath)
			return nil
		},
	}
	err := app.Run(os.Args)
	check(err, "")
}
