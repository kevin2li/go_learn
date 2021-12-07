package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Stu struct {
	Name   string  `json:"name"`
	Gender bool    `json:"gender"`
	Score  float64 `json:"score"`
}

func ErrorWrap(err error, message string) error {
	if err != nil {
		err = errors.Wrap(err, message)
		// fmt.Printf("%+v\n", err)
		return err
	}
	return nil
}

func mkdir() error {
	err := os.MkdirAll("a/b/c", 0666)
	err = ErrorWrap(err, "error creating")
	if err != nil {
		return err
	}
	return nil
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

func main() {
	log.Println("Started!")
	filepath := "/home/likai/code/go_program/go_learn/heights.txt"

	// heights := strings.Split(string(content), "")
	// fmt.Printf("%+v\n", heights)
	// fmt.Println(len(heights))
	// for i, v := range heights {
	// 	fmt.Println(i, v)
	// }

	// read file
	// results := make([]string, 0)
	heights := make([]int, 0)
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", heights)
}
