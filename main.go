package main

import (
	"log"
	"os"

	"github.com/pkg/errors"
)

type Stu struct {
	Name   string  `json:"name"`
	Gender bool    `json:"gender"`
	Score  float64 `json:"score"`
}

func check(err error, message string) {
	if err != nil {
		err = errors.Wrap(err, message)
		log.Fatalf("%+v\n", err)
	}
}
func mkdir() {
	err := os.MkdirAll("a/b/c", 0666)
	check(err, "error creating")
}

func main() {
	log.Println("Started!")
	mkdir()
	// check(err, "mkdir failed")
}
