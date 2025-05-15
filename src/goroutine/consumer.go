package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	PRODUCER_MAX = 5
	CONSUMER_MAX = 2
	PRODUCT_MAX  = 20
)

var productcount = 0
var lock sync.Mutex // 协程锁
var wgrp sync.WaitGroup

var produce_wait chan struct{}
var consume_wait chan struct{} // chan struct{} 不能传递数据，只是用来通知 是生产者和消费者的信号量

var stopProduce = false
var stopConsume = false // 是否停止生产的变量，一个被挂起阻塞，其余根据这个变量实现睡眠

// 生产者
// 生产者
func Produce(index int, wgrp *sync.WaitGroup) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Producer ", index, " panic")
		}
		wgrp.Done()
	}()

	for {
		time.Sleep(time.Second)
		lock.Lock()
		if stopProduce {
			fmt.Println("Producer ", index, " stop produce, sleep 5 seconds")
			lock.Unlock()
			time.Sleep(time.Second * 5)
			continue
		}
		fmt.Println("Producer ", index, " begin produce")
		if productcount >= PRODUCT_MAX {
			fmt.Println("Products are full")
			stopProduce = true
			lock.Unlock()
			//产品满了，生产者wait
			<-produce_wait // 谁被阻塞了 谁就负责唤醒
			fmt.Println("Producer ", index, " wake up")
			lock.Lock()
			stopProduce = false
			lock.Unlock()
			continue
		}
		productcount++
		fmt.Println("Products count is ", productcount)
		if stopConsume {
			var consumActive struct{}
			consume_wait <- consumActive
		}
		lock.Unlock()
	}
}

// 消费者
func Consume(index int, wgrp *sync.WaitGroup) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Consumer ", index, " panic")
		}
		wgrp.Done()
	}()

	for {
		time.Sleep(time.Second)
		lock.Lock()
		fmt.Println("Consumer ", index, " begin consume")
		if productcount <= 0 {
			fmt.Println("Products are empty")
			lock.Unlock()
			//产品空了，消费者等待
			<-consume_wait
			continue
		}
		lastcount := productcount
		productcount--
		fmt.Println("Products count is ", productcount)
		lock.Unlock()
		//产品数由PRODUCT_MAX变少，激活生产者
		if lastcount == PRODUCT_MAX {
			var productActive struct{}
			produce_wait <- productActive
		}

	}
}

func main() {
	wgrp.Add(PRODUCER_MAX + CONSUMER_MAX)
	produce_wait = make(chan struct{})
	consume_wait = make(chan struct{})
	for i := 0; i < CONSUMER_MAX; i++ {
		go Consume(i, &wgrp)
	}
	for i := 0; i < PRODUCER_MAX; i++ {
		go Produce(i, &wgrp)
	}

	wgrp.Wait()
}
