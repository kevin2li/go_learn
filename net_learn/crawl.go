package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	pb "github.com/schollz/progressbar/v3"
	cli "github.com/spf13/cobra"
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
	getBlockUrl string   // https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false
	getTxUrl    string   // https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s
	savedir     string   // result save directory
	page        int      // number of tx each request get
	ua_pool     []string // browser user agent pool
}

func (c *Crawler) getUaPool(ua_path string) error {
	var ua_pool []string
	content, err := os.ReadFile(ua_path)
	if err != nil {
		err = errors.Wrap(err, "read ua_path failed")
		return err
	}
	err = json.Unmarshal(content, &ua_pool)
	if err != nil {
		err = errors.Wrap(err, "unmarshall ua_pool failed")
		return err
	}
	c.ua_pool = ua_pool
	return nil
}

func (c *Crawler) GetBlocksByHeights(heights string) ([]Block, error) {
	/* construct request */
	url := fmt.Sprintf(c.getBlockUrl, heights)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", c.ua_pool[rand.Intn(len(c.ua_pool))])
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
		err = errors.Wrap(err, fmt.Sprintf("unmarshall error, response is:\n %s", string(body)[:200]))
		return nil, err
	}
	return blocks, nil
}

func (c *Crawler) GetTxsByHashs(txHashs string) ([]Transaction, error) {
	/* construct request */
	url := fmt.Sprintf(c.getTxUrl, txHashs)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("request for %s error!", url))
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua_pool[rand.Intn(len(c.ua_pool))])
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
		err = errors.Wrap(err, fmt.Sprintf("read response failed, request url is: %s", url))
		return nil, err
	}

	/* save response */
	var txs []Transaction
	err = json.Unmarshal(body, &txs)
	if err != nil {
		err = errors.Wrap(err, "unmarshall failed\n")
		Save("error_page.html", body, os.O_CREATE|os.O_WRONLY)
		return nil, err
	}
	return txs, nil
}

func (c *Crawler) GetBlocks(heights []int) ([]Block, error) {
	fmt.Printf("INFO: get txids in blocks with heights = %v...\n", heights)
	var all_blocks []Block
	var n = len(heights)
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, heights[0])
	bar := GetProgressBar(n, desc)
	for i, h := range heights {
		blocks, err := c.GetBlocksByHeights(strconv.Itoa(h))
		if err != nil {
			return nil, err
		}
		all_blocks = append(all_blocks, blocks...)
		bar.Add(1)
		bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i+1, n, h))
	}
	bar.Close()
	fmt.Println()
	return all_blocks, nil
}

func (c *Crawler) GetBlocksInRange(low, high int) ([]Block, error) {
	fmt.Printf("INFO: get txids in blocks with heights in range [%d, %d)...\n", low, high)
	var all_blocks []Block
	var n = high - low
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, low)
	bar := GetProgressBar(n, desc)
	for i := low; i < high; i++ {
		blocks, err := c.GetBlocksByHeights(strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		all_blocks = append(all_blocks, blocks...)
		bar.Add(1)
		bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i-low+1, n, i))
	}
	bar.Close()
	fmt.Println()
	return all_blocks, nil
}

func (c *Crawler) DownloadOneBlock(block *Block, done chan int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("%+v\n", err)
			done <- int(block.Height)
		} else {
			done <- 0
		}
	}()
	fmt.Printf("INFO: Download block at height %d...\n", block.Height)
	p, n, tx_hashs := 0, len(block.Tx), ""
	desc := fmt.Sprintf("[cyan][1/%d][reset] Block %d:", n, block.Height)
	bar := GetProgressBar(n, desc)
	var all_txs []Transaction
	for i, tx_hash := range block.Tx {
		tx_hashs = fmt.Sprintf("%s,%s", tx_hashs, tx_hash)
		p++
		// every `page` hash issue a request
		if (p+1)%c.page == 0 || i == n-1 {
			retry_times := 0
		request:
			txs, err := c.GetTxsByHashs(tx_hashs[1:])
			if err != nil {
				if retry_times < 5 {
					retry_times++
					time.Sleep(time.Duration(retry_times*10) * time.Second)
					fmt.Printf("Retry download block %d...\n", block.Height)
					goto request
				}
				panic(err)
			}
			all_txs = append(all_txs, txs...)
			bar.Add(p)
			p, tx_hashs = 0, ""
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Block %d:", i, n, block.Height))
		}
	}
	bar.Close()
	fmt.Println()
	obj, err := json.Marshal(all_txs)
	if err != nil {
		err = errors.Wrap(err, "Marshal Error")
		panic(err)
	}
	savepath := filepath.Join(c.savedir, fmt.Sprintf("block_height=%d.json", block.Height))
	Save(savepath, obj, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
}

func (c *Crawler) DownloadAllBlocks(blocks []Block) {
	n := len(blocks)
	failedBlocks := make([]int, 0)
	done := make(chan int, n)

	for i := 0; i < n; i++ {
		go c.DownloadOneBlock(&blocks[i], done)
		// decrease access frequency
		r := 60 + rand.Intn(5)
		time.Sleep(time.Duration(r) * time.Second)
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
		content := fmt.Sprintf("%v\n", failedBlocks)
		content = content[1:len(content)-2] + "\n"
		Save("failed_block_heights.txt", []byte(content), os.O_CREATE|os.O_WRONLY|os.O_APPEND)
		log.Printf("Save failed block heights at: %s", "failed_block_heights.txt")
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

func Strings2Ints(strs []string) ([]int, error) {
	var result []int
	for _, s := range strs {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

func ReadHeights(path string) ([]int, error) {
	heights := make([]int, 0)
	f, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("read %s failed", path))
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		heights_str := strings.Split(line, " ")
		temp_heights, _ := Strings2Ints(heights_str)
		heights = append(heights, temp_heights...)
	}
	if err := scanner.Err(); err != nil {
		err = errors.Wrap(err, "scanner error")
		return nil, err
	}
	return heights, nil
}
func Save(path string, content []byte, flag int) error {
	// check path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0766)
	}
	// open or create file for writing
	file, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Open file `%s` error", path))
		return err
	}
	defer file.Close()
	// write content
	writer := bufio.NewWriter(file)
	_, err = writer.Write(content)
	writer.Flush()
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("save file `%s` error", path))
		return err
	}
	return nil
}

func main() {
	var crawler = &Crawler{
		getBlockUrl: "https://api.blockchain.info/haskoin-store/btc/block/heights?heights=%s&notx=false",
		getTxUrl:    "https://api.blockchain.info/haskoin-store/btc/transactions?txids=%s",
		page:        5,
	}
	crawler.getUaPool("/home/likai/code/go_program/go_learn/net_learn/ua.json")

	var (
		isInterval bool
		filepath   string
	)

	var rootCmd = &cli.Command{Use: "crawler"}

	var downloadCmd = &cli.Command{
		Use:   "download [heights to download]",
		Short: "download transactions in given block heights",
		Long: `download transactions in given block heights.
	Please give reasonable block heights.`,
		Args: func(cmd *cli.Command, args []string) error {
			if isInterval && len(args) != 2 {
				return errors.New("you should only given 2 args with `-r` flag")
			}
			return nil
		},
		Run: func(cmd *cli.Command, args []string) {
			t1 := time.Now()
			log.Println("Started!")
			// download txs in given block heights range
			if isInterval {
				low, _ := strconv.Atoi(args[0])
				high, _ := strconv.Atoi(args[1])
				blocks, err := crawler.GetBlocksInRange(low, high)
				if err != nil {
					log.Fatalf("%+v\n", err)
				}
				crawler.DownloadAllBlocks(blocks)
				// read heights from file
			} else if filepath != "" {
				heights, _ := ReadHeights(filepath)
				blocks, err := crawler.GetBlocks(heights)
				if err != nil {
					log.Fatalf("%+v\n", err)
				}
				crawler.DownloadAllBlocks(blocks)
				// download txs in given heights
			} else {
				heights, _ := Strings2Ints(args)
				blocks, err := crawler.GetBlocks(heights)
				if err != nil {
					log.Fatalf("%+v\n", err)
				}
				crawler.DownloadAllBlocks(blocks)
			}
			t2 := time.Now()
			log.Println("Finished!")
			fmt.Printf("Time elapsed: %.2f minutes\n", t2.Sub(t1).Minutes())
		},
	}
	downloadCmd.Flags().BoolVarP(&isInterval, "interval", "r", false, "")
	downloadCmd.Flags().StringVarP(&filepath, "filepath", "f", "", "file store heights to download")
	downloadCmd.Flags().StringVarP(&crawler.savedir, "savedir", "s", "result", "result save directory")
	// Add subcommand
	rootCmd.AddCommand(downloadCmd)
	rootCmd.Execute()
}
