package main

import (
	"fmt"
	"time"
)

type LoaderInter interface {
	Load()
}

type Loader struct {
}

// 值接收者，接口可以是指针和值
func (ld Loader) Load() {
	fmt.Println("Loader begin load data....")
	<-time.After(time.Second * 10)
	fmt.Println("Loader load data success...")
}

type ProducerInter interface {
	Produce() LoaderInter
}

type Producer struct {
	max int
}

/*
指针接收者
只有 *Producer 类型（指针）拥有这个 Produce() 方法。
所以 只有指针类型的 Producer 才实现了 ProducerInter 接口。
*/
func (p *Producer) Produce() LoaderInter {
	if p.max <= 0 {
		return nil
	}
	ld := &Loader{}
	p.max--
	return ld
}

func Create(producer ProducerInter) {
	//...

}
