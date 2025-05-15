package main

import (
	"errors"
	"fmt"
	"time"
)

type LoaderInter interface {
	Load()
}

func Check(li LoaderInter) error {
	//
	ch := make(chan struct{})
	go func() {
		li.Load()
		ch <- struct{}{}
	}()
	select {
	case <-ch:
		return nil
	case <-time.After(time.Second * 5):
		return errors.New("load timeout")
	}
}

type Loader struct {
}

// Loader 实现了 LoaderInter 接口的 Load 方法，那么 Loader 类型就实现了 LoaderInter 接口
func (ld Loader) Load() {
	fmt.Println("Loader begin load data....")
	<-time.After(time.Second * 10)
	fmt.Println("Loader load data success...")
}

func main() {
	ld := &Loader{}
	err := Check(ld)
	if err != nil {
		fmt.Println("err is ", err)
	}

	time.Sleep(time.Second * 11)

}
