package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Stu struct {
	Name   string  `json:"name"`
	Gender bool    `json:"gender"`
	Score  float64 `json:"score"`
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
	var result []string
	result_dir := "/home/likai/code/go_program/go_learn/result"
	for i := 712000; i < 712800; i++ {
		path := filepath.Join(result_dir, fmt.Sprintf("block_height=%d.json", i))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			result = append(result, strconv.Itoa(i))
		}
	}
	fmt.Println("total: ", len(result))
	content := strings.Join(result, " ")
	Save("failed_block.txt", []byte(content), os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
}
