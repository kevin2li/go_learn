package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	progressbar "github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var (
	page    = 5                // number of tx each request get
	bufsize = runtime.NumCPU() // buffer size of chan
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

type Crawler struct {
	getBlockUrl string // https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false
	getTxUrl    string // https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s
	low         int    // crawl start from this block height
	high        int    // crawl end at this block height
	savedir     string // result save directory
}

func (this *Crawler) GetBlocksByHeights(heights string) []Block {
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

func (this *Crawler) GetTxsByHashs(txHashs string) []Transaction {
	/* construct request */
	url := fmt.Sprintf(this.getTxUrl, txHashs)
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

func (this *Crawler) GetBlocksInRange() []Block {
	var all_blocks []Block
	for i := this.low; i < this.high; i++ {
		blocks := this.GetBlocksByHeights(strconv.Itoa(i))
		all_blocks = append(all_blocks, blocks...)
	}
	return all_blocks
}

func (this *Crawler) ExtractOneBlock(block *Block, page int, tx_chan chan []Transaction) {
	fmt.Printf("INFO: Extract block with height %d...\n", block.Height)
	p, n, tx_hashs := 0, len(block.Tx), ""
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, block.Height)
	bar := GetProgressBar(n, desc)
	var all_txs []Transaction
	for i, tx_hash := range block.Tx {
		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
		p++
		// every <page> hash issue a request
		if (p+1)%page == 0 || i == n-1 {
			var txs []Transaction = this.GetTxsByHashs(tx_hashs[1:])
			all_txs = append(all_txs, txs...)
			bar.Add(p)
			p, tx_hashs = 0, ""
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i, n, block.Height))
		}
	}
	bar.Close()
	tx_chan <- all_txs
}

func (this *Crawler) ExtractAllBlocks(blocks []Block, page int, bufsize int) []Transaction {
	n := len(blocks)
	fmt.Printf("INFO: Extract block with height between %d and %d...\n", blocks[n-1].Height, blocks[0].Height)
	var all_txs []Transaction
	tx_chan := make(chan []Transaction, bufsize)
	for i := 0; i < n; i++ {
		go this.ExtractOneBlock(&blocks[i], page, tx_chan)
	}
	for i := 0; i < n; i++ {
		txs := <-tx_chan
		obj, err := json.Marshal(txs)
		check(err, "Marshal Error")
		Save(fmt.Sprintf("./tmp/%d.json", i), obj)
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

func check(err error, message string) {
	if err != nil {
		if message != "" {
			log.Println(message)
		}
		log.Fatal(err)
	}
}

func Save(path string, content []byte) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0666)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	check(err, "open file failed")
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.Write(content)
	writer.Flush()
	check(err, "save file failed")
}

func Start(crawler *Crawler) {
	t1 := time.Now()
	fmt.Println("INFO: Started at:", t1)
	fmt.Printf("INFO: Get blocks in range (%d, %d)...\n", crawler.low, crawler.high)
	var blocks []Block = crawler.GetBlocksInRange()
	all_txs := crawler.ExtractAllBlocks(blocks, page, bufsize)
	obj, err := json.Marshal(all_txs)
	check(err, "Marshal Error")
	Save(crawler.savedir, obj)
	t2 := time.Now()
	fmt.Println("\nINFO: Finished at:", time.Now())
	fmt.Printf("Time elapsed: %.2f minutes\n", t2.Sub(t1).Minutes())
}

func main() {
	var crawler = &Crawler{
		getBlockUrl: "https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false",
		getTxUrl:    "https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s",
	}
	app := &cli.App{
		Name:  "Crawl",
		Usage: "crawl [-l low] [-h high] [-s savepath]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "savepath",
				Aliases:     []string{"s"},
				Usage:       "Save result at `SAVEPATH`",
				Destination: &crawler.savedir,
				Value:       "result.json",
			},
			&cli.IntFlag{
				Name:        "low",
				Aliases:     []string{"l"},
				Usage:       "Start at block height `LOW`",
				Destination: &crawler.low,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "high",
				Aliases:     []string{"e"},
				Usage:       "End at block height `HIGH`",
				Destination: &crawler.high,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			Start(crawler)
			return nil
		},
	}
	err := app.Run(os.Args)
	check(err, "")
}
