package main

import (
	"fmt"
	"sort"
)

func modify(data map[string]int, key string, value int) {
	v, res := data[key]
	//不存在,则res是false
	if !res {
		fmt.Println("key not find")
		return
	}
	fmt.Println("key is ", key, "value is ", v)
	data[key] = value
}
func sortprintmap(data map[string]int) {
	slice := make([]string, 0)
	for k, _ := range data {
		slice = append(slice, k)
	}
	sort.Strings(slice)
	for _, s := range slice {
		d, e := data[s]
		// e 是 s 这个 key 是否真的在 map 中的标志
		if !e {
			continue // 如果 key 不存在，就跳过
		}
		fmt.Println("key is ", s, "value is ", d)

	}
}

func myfuncv(args ...int) {
	for _, arg := range args {
		fmt.Println(arg)
	}
}

func main() {
	var data map[string]int = map[string]int{"bob": 18, "luce": 28}
	modify(data, "lilei", 28)
	modify(data, "luce", 22)
	fmt.Println(data)
	fmt.Println("len(data):", len(data))
	//map 使用前一定要初始化，可以显示初始化，也可以用make
	var data2 map[string]int = make(map[string]int, 3)
	fmt.Println(data2)
	//当key不存在时，则会插入
	data2["sven"] = 19
	fmt.Println(data2)
	//当key存在时，则修改
	data2["sven"] = 299
	fmt.Println(data2)
	data2["bob"] = 178
	data2["Arean"] = 33
	//map是无序的,遍历输出
	for key, value := range data2 {
		fmt.Println("key: ", key, "value: ", value)
	}
	sortprintmap(data2)
	// var data3 map[string]int
	// data3 = make(map[string]int, 3)
	myfuncv(1, 2, 4, 7, 8)
}
