package main

import "fmt"

func selectSort(slice []int) {
	for i := 0; i < len(slice); i++ {
		for j := i + 1; j < len(slice); j++ {
			if slice[j] < slice[j-1] {
				slice[j], slice[i] = slice[i], slice[j]
			}
		}
	}
	fmt.Println(slice)
}

func main() {
	arr := []int{64, 34, 25, 12, 22, 11, 90}
	selectSort(arr)
}
