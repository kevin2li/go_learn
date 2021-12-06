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
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	pb "github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
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
	page        int    // number of tx each request get
	heights     []int  // a sequence of block `heights` to download
}

func (this *Crawler) GetBlocksByHeights(heights string) ([]Block, error) {
	/* construct request */
	url := fmt.Sprintf("https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false", heights)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("request for %s failed!", url))
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		err = errors.Wrap(err, "read response failed")
		return nil, err
	}

	/* save response */
	var blocks []Block
	err = json.Unmarshal(body, &blocks)
	if err != nil {
		err = errors.Wrap(err, "unmarshall failed\n request url is: "+url+"\n response is: "+string(body)[:200])
		return nil, err
	}
	return blocks, nil
}

func (this *Crawler) GetTxsByHashs(txHashs string) ([]Transaction, error) {
	/* construct request */
	url := fmt.Sprintf(this.getTxUrl, txHashs)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("request for %s error!", url))
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.55 Safari/537.36 Edg/96.0.1054.41")
	req.Header.Set("Content-Type", "application/json")

	/* issue request and wait response*/
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("request for %s failed!", url))
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		err = errors.Wrap(err, "read response failed, request url is: "+url)
		return nil, err
	}

	/* save response */
	var txs []Transaction
	err = json.Unmarshal(body, &txs)
	if err != nil {
		err = errors.Wrap(err, "unmarshall failed\n request url is: "+url+"\n response is: "+string(body)[:200])
		return nil, err
	}
	return txs, nil
}

func (this *Crawler) GetBlocksInRange() ([]Block, error) {
	var all_blocks []Block
	for i := this.low; i < this.high; i++ {
		blocks, err := this.GetBlocksByHeights(strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		all_blocks = append(all_blocks, blocks...)
	}
	return all_blocks, nil
}

func (this *Crawler) DownloadOneBlock(block *Block, done chan int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("%+v\n", err)
			done <- int(block.Height)
		} else {
			done <- 0
		}
	}()
	fmt.Printf("INFO: Download block with height %d...\n", block.Height)
	p, n, tx_hashs := 0, len(block.Tx), ""
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, block.Height)
	bar := GetProgressBar(n, desc)
	var all_txs []Transaction
	for i, tx_hash := range block.Tx {
		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
		p++
		// every `page` hash issue a request
		if (p+1)%this.page == 0 || i == n-1 {
			txs, err := this.GetTxsByHashs(tx_hashs[1:])
			if err != nil {
				panic(err)
			}
			all_txs = append(all_txs, txs...)
			bar.Add(p)
			p, tx_hashs = 0, ""
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i, n, block.Height))
		}
	}
	bar.Close()
	obj, err := json.Marshal(all_txs)
	if err != nil {
		err = errors.Wrap(err, "Marshal Error")
		panic(err)
	}
	savepath := filepath.Join(this.savedir, fmt.Sprintf("block_height=%d.json", block.Height))
	Save(savepath, obj)
}

func (this *Crawler) DownloadAllBlocks(blocks []Block) {
	n := len(blocks)
	failedBlocks := make([]int, 0)
	fmt.Printf("INFO: Download block with height between %d and %d...\n", this.low, this.high)
	done := make(chan int, n)
	for i := 0; i < n; i++ {
		go this.DownloadOneBlock(&blocks[i], done)
	}
	for i := 0; i < n; i++ {
		if h := <-done; h != 0 {
			failedBlocks = append(failedBlocks, h)
		}
	}
	close(done)
	log.Printf("Total : %d, Success: %d, Failure: %d\n", n, n-len(failedBlocks), len(failedBlocks))
	if len(failedBlocks) > 0 {
		sort.Ints(failedBlocks)
		log.Printf("Failed blocks are: %v\n", failedBlocks)
	}
}

func GetProgressBar(max int, desc string) *pb.ProgressBar {
	bar := pb.NewOptions(max,
		// pb.OptionSetWriter(ansi.NewAnsiStdout()),
		pb.OptionEnableColorCodes(true),
		pb.OptionShowBytes(true),
		pb.OptionSetWidth(100),
		pb.OptionSetDescription(desc),
		pb.OptionSetTheme(pb.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
}

func Save(path string, content []byte) error {
	// check path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0766)
	}
	// open or create file for writing
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	defer file.Close()
	if err != nil {
		err = errors.Wrap(err, "open file "+path+" failed")
		return err
	}
	// write content
	writer := bufio.NewWriter(file)
	_, err = writer.Write(content)
	writer.Flush()
	if err != nil {
		err = errors.Wrap(err, "save file "+path+" failed")
		return err
	}
	return nil
}

func Start(crawler *Crawler) {
	t1 := time.Now()
	log.Println("Started!")
	fmt.Printf("INFO: Get blocks in range (%d, %d)...\n", crawler.low, crawler.high)
	blocks, err := crawler.GetBlocksInRange()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	crawler.DownloadAllBlocks(blocks)
	t2 := time.Now()
	log.Println("Finished!")
	fmt.Printf("Time elapsed: %.2f minutes\n", t2.Sub(t1).Minutes())
}

func main() {
	var crawler = &Crawler{
		getBlockUrl: "https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false",
		getTxUrl:    "https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s",
	}
	app := &cli.App{
		Name:  "Crawl",
		Usage: "crawl [-l low] [-h high] [-s savedir]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "savedir",
				Aliases:     []string{"s"},
				Usage:       "Save result at `savedir`",
				Destination: &crawler.savedir,
				Value:       "result",
			},
			&cli.IntFlag{
				Name:        "low",
				Aliases:     []string{"l"},
				Usage:       "Start at block height `low`",
				Destination: &crawler.low,
			},
			&cli.IntFlag{
				Name:        "high",
				Aliases:     []string{"e"},
				Usage:       "End at block height `high`(not included)",
				Destination: &crawler.high,
			},
			&cli.IntSliceFlag{
				Name:        "download",
				Aliases:     []string{"d"},
				Usage:       "Specify a sequence of block `heights` to download",
			},
		},
		Action: func(c *cli.Context) error {
			// Start(crawler)
			fmt.Println(c.Int("d"))
			for _, v := range c.FlagNames() {
				fmt.Printf("%s %#v\n", v, c.Value(v))
			}
			return nil
		},
	}
	app.Run(os.Args)
}
