package main

import "fmt"

func GetName() (string, string, string) {
	return "Tom", "Jerry", "Luka"
}

func main() {
	// 整形
	var v1 int
	// 字符串
	var v2 string

	// 变量赋值
	v1 = 10
	v2 = "hello"
	fmt.Println("v2:", v2)

	// 变量初始化
	v11 := "day02"
	fmt.Println("v11:", v11)

	// 第二种初始化方式
	var v12 int = 13

	// 变量交换
	v1, v12 = v12, v1
	fmt.Printf("v1: %v, v12: %v\n", v1, v12)

	// 函数返回值赋值
	_, _, nickName := GetName()
	fmt.Printf("nickName : %v\n", nickName)

	// 常量
	const Pi float64 = 3.141592653
	const zero = 0.0
	const (
		size int64 = 1024
		eof        = -1
	)
	const u, v float32 = 0, 3
	const a, b, c = 3, 4, "foo"
}
