package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	pb "github.com/schollz/progressbar/v3"
	cli "github.com/spf13/cobra"
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

func ReadTransactionDir(blockDir string) ([]Transaction, error) {
	var all_txs []Transaction
	files, err := os.ReadDir(blockDir)
	if err != nil {
		log.Fatal(err)
	}
	n := len(files)
	bar := GetProgressBar(n)
	defer bar.Close()
	for _, file := range files {
		block_height_path := filepath.Join(blockDir, file.Name())
		bar.Describe(fmt.Sprintf("loading tx in %s:", file.Name()))
		txs, err := ReadTransaction(block_height_path)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("read %s error", block_height_path))
			return nil, err
		}
		all_txs = append(all_txs, txs...)
		bar.Add(1)
	}
	return all_txs, nil
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

// TODO: ChangeHeuristic
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
		fmt.Printf("[%d/%d] Starting cluster from address: %s\n", i+1, n, addr)
		go ClusterByAddr(addr, txs, addrList)
	}
	queue = make([]string, 0) // clear queue
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
		fmt.Printf("INFO: new addresses added: %+v\n\n", queue)
		iterations++
		goto iter
	}
	result := finalAddrList.GetData()
	return result
}

func GetProgressBar(max int) *pb.ProgressBar {
	bar := pb.NewOptions(max,
		// pb.OptionSetWriter(ansi.NewAnsiStdout()),
		pb.OptionEnableColorCodes(true),
		pb.OptionShowBytes(true),
		pb.OptionSetWidth(15),
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

func StartCluster(dataset_path string, start_addr string){
	fmt.Println("INFO: Loading transactions....")
	all_txs, err := ReadTransaction(dataset_path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("INFO: Load all transactions done!")
	fmt.Printf("INFO: Start cluster from address: %s!\n", start_addr)
	result := Cluster(start_addr, all_txs)
	fmt.Println("\n--------------------------------Cluster Finished!--------------------------------")
	fmt.Printf("INFO: cluster total %d addresses, final cluster result:\n %v\n", len(result), result)
}

func main() {
	var (
		dataset_path string // all_txs.json
		start_addr   string // 1KFHE7w8BhaENAswwryaoccDb6qcT6DbYY
	)
	var rootCmd = &cli.Command{Use: "analyzer"}

	var clusterCmd = &cli.Command{
		Use:   "cluster -f [dataset_path] [address]",
		Short: "cluster address in given transcation dataset",
		Long:  `cluster address in given transcation dataset.`,
		Args: func(cmd *cli.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("you should only give one argument!")
			}
			return nil
		},
		Run: func(cmd *cli.Command, args []string) {
			t1 := time.Now()
			log.Println("Started!")
			start_addr = args[0]
			StartCluster(dataset_path, start_addr)
			t2 := time.Now()
			log.Println("Finished!")
			fmt.Printf("Time elapsed: %.2f minutes\n", t2.Sub(t1).Minutes())
		},
	}
	clusterCmd.Flags().StringVarP(&dataset_path, "dataset_path", "f", "", "path to load transcation dataset")
	clusterCmd.MarkFlagRequired("dataset_path")
	// Add subcommand
	rootCmd.AddCommand(clusterCmd)
	rootCmd.Execute()
}
