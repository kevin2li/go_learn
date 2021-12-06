package main

import (
	"fmt"
	"time"
)

func resolve(i int, mychan chan int){
	time.Sleep(1 * time.Second)
	mychan <- i
}

// func main() {
// 	mychan := make(chan int, 4)
// 	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
// 	cnt := 0
// 	for _, v := range a {
// 		go resolve(v, mychan)
// 		cnt++
// 	}
// 	for i := 0; i < cnt; i++ {
// 		fmt.Printf("read mychan: %d\n", <-mychan)
// 	}
// }
