package main

import (
	"fmt"
)

type Stu struct {
	Name   string  `json:"name"`
	Gender bool    `json:"gender"`
	Score  float64 `json:"score"`
}

func main() {
	fmt.Println("started!")
	// file.ReadFile("/home/likai/code/go_program/go_learn/main.go")
	// dirs, err := file.ListDir(".")
	// if err != nil {
	// 	return
	// }
	// fmt.Printf("%v\n", dirs)

	// container_learn.ListLearn()
	stu := Stu{"Kevin", true, 100.0}
	fmt.Printf("\n---stu---\nvalue:\t%v,\ntype:\t%T\n------\n", stu, stu)
	stu.Name = "Lucy"
	fmt.Printf("\n---stu---\nvalue:\t%v,\ntype:\t%T\n------\n", stu, stu)
	
}
