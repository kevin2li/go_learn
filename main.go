package main

import (
	"fmt"
	"log"
	"os"
	"sort"

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
func main() {
	log.Println("Started!")
	arr := []int{190, 23, 45, 42, 58}
	fmt.Printf("%+v\n", arr)
	arr = sort.IntSlice(arr)
	// sort.Ints(arr)
	sort.Sort(arr)
	fmt.Printf("%+v\n", arr)
	fmt.Printf("%+v\n", arr)

}
