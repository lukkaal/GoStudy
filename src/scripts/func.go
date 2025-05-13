package main

import (
	"fmt"
)

func myfunc(args ...interface{}) {
	for _, arg := range args {
		// fmt.Println(arg)
		switch v := arg.(type) { // 使用类型断言来判断类型 只能在switch中使用
		case int:
			fmt.Println("arg is an int:", v)
		case string:
			fmt.Println("arg is a string:", v)
		default:
			fmt.Println("unknown type")
		}
	}
}

func myfuncv(args ...int) {
	for _, arg := range args {
		fmt.Println(arg)
	}
}

func main() {
	myfuncv(1, 2, 4, 7, 8)
	myfunc(42, "hello", 3.14, true)
}
