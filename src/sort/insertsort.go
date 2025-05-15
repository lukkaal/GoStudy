package main

import "fmt"

func insertSort(slice []int) {
	for i := 0; i < len(slice); i++ {
		for j := i; j > 0; j-- {
			if slice[j] < slice[j-1] {
				slice[j], slice[j-1] = slice[j-1], slice[j]
			}
		}
	}
	fmt.Println(slice)
}

func main() {
	arr := []int{64, 34, 25, 12, 22, 11, 90}
	insertSort(arr)
}
