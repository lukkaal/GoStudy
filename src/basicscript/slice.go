package main

import "fmt"

func main() {
	// 数组声明方法
	var bytearray [8]byte // 长度为8的字节数组
	fmt.Println("bytearray:", bytearray)

	var pointarray [4]*float64 // 指针数组，元素是 float64 的指针
	fmt.Println("pointarray:", pointarray)

	var mularray [3][5]int // 多维数组：3 行 5 列
	fmt.Println("mularray:", mularray)

	// 打印数组长度
	fmt.Printf("pointarray len is %v\n", len(pointarray))

	// 数组遍历（传统 for）
	for i := 0; i < len(pointarray); i++ {
		fmt.Println("Element", i, "of array is", pointarray[i])
	}

	// 采用 range 遍历
	for i, v := range pointarray {
		fmt.Println("Array element [", i, "] =", v)
	}

	array := [5]int{1, 2, 3, 4, 5}
	//根据数组生成切片
	//切片
	var mySlice []int = array[:3]
	fmt.Println("Elements of array")
	for _, v := range array {
		fmt.Print(v, " ")
	}
	fmt.Println("\nElements of mySlice: ")
	for _, v := range mySlice {
		fmt.Print(v, " ")
	}

	//直接创建元素个数为5的数组切片
	mkslice := make([]int, 5)
	fmt.Println("\n", mkslice)
	fmt.Println("len(mkslice):", len(mkslice))
	fmt.Println("cap(mkslice):", cap(mkslice))
	//创建初始元素个数为5的切片，元素都为0，且预留10个元素存储空间
	mkslice2 := make([]int, 5, 10)
	fmt.Println("\n", mkslice2)
	fmt.Println("len(mkslice2):", len(mkslice2))
	fmt.Println("cap(mkslice2):", cap(mkslice2))
	mkslice3 := []int{1, 2, 3, 4, 5}
	fmt.Println("\n", mkslice3)

	//元素遍历
	for i := 0; i < len(mkslice3); i++ {
		fmt.Println("mkslice3[", i, "] =", mkslice3[i])
	}

	//range 遍历
	for i, v := range mkslice3 {
		fmt.Println("mkslice3[", i, "] =", v)
	}
}
