package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// type Interface interface {
// 	// Len is the number of elements in the collection.
// 	Len() int
// 	// Less reports whether the element with
// 	// index i should sort before the element with index j.
// 	Less(i, j int) bool
// 	// Swap swaps the elements with indexes i and j.
// 	Swap(i, j int)
// }

type Hero struct {
	Name    string
	Attack  int
	Defence int
	GenTime int64
}

type HeroList []*Hero

func (hl HeroList) Len() int {
	return len(hl)
}

func (hl HeroList) Less(i, j int) bool {
	if i < 0 || j < 0 {
		return true
	}

	lenth := len(hl)
	if i >= lenth || j >= lenth {
		return true
	}

	if hl[i].Attack != hl[j].Attack {
		return hl[i].Attack < hl[j].Attack
	}

	if hl[i].Defence != hl[j].Defence {
		return hl[i].Defence < hl[j].Defence
	}

	return hl[i].GenTime < hl[j].GenTime
}

func (hl HeroList) Swap(i, j int) { // Go 中函数如果不写返回值，就相当于 C++ 中的 void
	if i < 0 || j < 0 {
		return
	}

	lenth := len(hl)
	if i >= lenth || j >= lenth {
		return
	}

	hl[i], hl[j] = hl[j], hl[i]

}

func main() {
	var herolists HeroList
	for i := 0; i < 10; i++ {
		generate := time.Now().Unix()
		name := fmt.Sprintf("Hero%d", generate)
		hero := Hero{
			Name:    name,
			Attack:  rand.Intn(100),
			Defence: rand.Intn(200),
			GenTime: generate,
		}
		herolists = append(herolists, &hero)
		time.Sleep(time.Duration(1) * time.Second)
	}

	sort.Sort(herolists)
	for _, value := range herolists {
		fmt.Print(value.Name, " ", value.Attack, " ", value.Defence, " ", value.GenTime, "\n")
	}

	var ife interface{}
	ife = herolists
	val, ok := ife.(HeroList)
	if !ok {
		fmt.Println("ife can't transfer to HeroList!")
		return
	}
	fmt.Println("herolist's len is ", val.Len())
}
