package main

import (
	"fmt"
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

func fibonacci(n int, c chan int) {
	x, y := 0, 1
	for i := 0; i < n; i++ {
		c <- x
		x, y = y, x+y
	}
	close(c)
}

func main() {
	c := make(chan int, 10)
	go fibonacci(cap(c), c)
	// range 函数遍历每个从通道接收到的数据，因为 c 在发送完 10 个
	// 数据之后就关闭了通道，所以这里我们 range 函数在接收到 10 个数据
	// 之后就结束了。如果上面的 c 通道不关闭，那么 range 函数就不
	// 会结束，从而在接收第 11 个数据的时候就阻塞了。
	for v := range c {
		fmt.Println(v)
	}
}
