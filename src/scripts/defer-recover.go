package main

import (
	"log"
)

func DeferParam() {
	i := 0
	defer func(m int) {
		log.Println(m)
	}(i)
	i++
}

func DeferNoParam() {
	i := 0
	defer func() {
		log.Println(i)
	}()
	i++
}

func DeferOrder() {
	for i := 0; i < 5; i++ {
		defer func(m int) {
			log.Println(m)
		}(i)
	}
}

// 链式调用
type Slice []int

func NewSlice() *Slice {
	slice := make(Slice, 0)
	return &slice
}

func (s *Slice) AddSlice(val int) *Slice {
	*s = append(*s, val)
	log.Println(val)
	return s
}

func main() {
	DeferParam()
	DeferNoParam()
	DeferOrder()

	s := NewSlice()
	defer s.AddSlice(1).AddSlice(3)
	s.AddSlice(2)
}
