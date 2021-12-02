package container_learn

import (
	"container/list"
	"fmt"
)

func PrintList(list *list.List) {
	for e := list.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value)
	}
}
func ListLearn() {
	list := list.New()
	// push 3 items from back
	list.PushBack(3)
	list.PushBack([]int{2, 5, 8})
	list.PushBack("hello")
	// push 2 items from front
	PrintList(list)
}
