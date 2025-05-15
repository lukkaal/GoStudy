package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup // 创建一个 WaitGroup

	// 启动三个 goroutine
	for i := 1; i <= 3; i++ {
		wg.Add(1) // 增加等待的 goroutine 数量
		go func(i int) {
			defer wg.Done() // 完成后通知 WaitGroup
			fmt.Printf("Goroutine %d is done\n", i)
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	fmt.Println("All goroutines are done.")
}
