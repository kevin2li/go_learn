package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

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

func f(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func main() {
	log.Println("Started!")
	fmt.Printf(f("hello, %s\n", "kevin!"))
}
