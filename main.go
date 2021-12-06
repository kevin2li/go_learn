package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type Stu struct {
	Name   string  `json:"name"`
	Gender bool    `json:"gender"`
	Score  float64 `json:"score"`
}

func main() {
	path := "/home/likai/code/go_program/go_learn/main.go"
	s := filepath.Base(path)
	s2 := filepath.Dir(path)
	os.Stat(path)
	fmt.Println(s, s2)
}
