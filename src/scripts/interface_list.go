package main

import (
	"fmt"
	"math/rand"
)

type LinkList struct {
	Head *LinkEle
	Tail *LinkEle
}

type LinkEle struct {
	Data interface{}
	Pre  *LinkEle
	Next *LinkEle
}

func (le *LinkEle) GetData() interface{} {
	return le.Data
}

func (ll *LinkList) InsertHead(le *LinkEle) { // 从头部插入

	if ll.Tail == nil && ll.Head == nil {
		ll.Tail = le
		ll.Head = ll.Tail
		return
	}

	ll.Head.Pre = le
	le.Pre = nil
	le.Next = ll.Head
	ll.Head = le
}

func (ll *LinkList) InsertTail(le *LinkEle) { // 从尾部插入
	if ll.Tail == nil && ll.Head == nil {
		ll.Tail = le
		ll.Head = ll.Tail
		return
	}
	ll.Tail.Next = le
	le.Pre = ll.Tail
	le.Next = nil
	ll.Tail = le
}

func (ll *LinkList) InsertIndex(le *LinkEle, index int) {
	if index < 0 {
		return
	}

	if ll.Head == nil {
		ll.Head = le
		ll.Tail = ll.Head
		return
	}

	node := ll.Head
	indexfind := 0
	for ; indexfind < index; indexfind++ {
		if node.Next == nil {
			break
		}
		node = node.Next
	}

	if indexfind != index {
		fmt.Println("index is out of range")
		return
	}
	//node 后边的节点缓存起来
	nextnode := node.Next

	//node 和le连接起来
	node.Next = le
	le.Pre = node

	if node == ll.Tail {
		ll.Tail = le
		return
	}

	//le和next node 连接起来
	if nextnode != nil {
		nextnode.Pre = le
		le.Next = nextnode
	}
}

// func (ll *LinkList) Insertindex(le *LinkEle, index int) {
// 	if index < 0 {
// 		return
// 	}
// 	if ll.Head == nil {
// 		ll.Head = le
// 		ll.Tail = ll.Head
// 		return
// 	}
// 	node := ll.Head
// 	indexfind := 0
// 	for ; indexfind < index; indexfind++ {
// 		if node.Next == nil {
// 			break
// 		}
// 		node = node.Next
// 	}

// 	if indexfind != index {
// 		fmt.Println("out ", "of ", "range")
// 		return
// 	}

// 	nextnode := node.Next

// 	node.Next = le
// 	le.Pre = node

// 	if node == ll.Tail {
// 		ll.Tail = le
// 		return
// 	}

// 	if nextnode != nil {
// 		le.Next = node.Next
// 		node.Next.Pre = le
// 	}

// }

func main() {
	ll := &LinkList{nil, nil}
	fmt.Println("insert head .....................")
	for i := 0; i < 2; i++ {
		num := rand.Intn(100)
		node1 := &LinkEle{Data: num, Next: nil, Pre: nil}
		ll.InsertHead(node1)
		fmt.Println(num)
	}
	fmt.Println("after insert head .................")
	for node := ll.Head; node != nil; node = node.Next {
		val, ok := node.GetData().(int)
		if !ok {
			fmt.Println("interface transfer error")
			break
		}
		fmt.Println(val)
	}

	fmt.Println("insert tail .....................")
	for i := 0; i < 2; i++ {
		num := rand.Intn(100)
		node1 := &LinkEle{Data: num, Next: nil, Pre: nil}
		ll.InsertTail(node1)
		fmt.Println(num)
	}

	fmt.Println("after insert tail .................")
	for node := ll.Head; node != nil; node = node.Next {
		val, ok := node.GetData().(int)
		if !ok {
			fmt.Println("interface transfer error")
			break
		}
		fmt.Println(val)
	}

	fmt.Println("insert after third element........")
	{
		num := rand.Intn(100)
		node1 := &LinkEle{Data: num, Next: nil, Pre: nil}
		ll.InsertIndex(node1, 2)
		fmt.Println(num)
	}

	fmt.Println("after insert index .................")
	for node := ll.Head; node != nil; node = node.Next {
		val, ok := node.GetData().(int)
		if !ok {
			fmt.Println("interface transfer error")
			break
		}
		fmt.Println(val)
	}
}
