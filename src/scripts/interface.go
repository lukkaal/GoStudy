package main

import (
	"fmt"
)

type Bird interface {
	Fly() string
}

type Plane struct {
	name string
}

func (p *Plane) Fly() (ret string) {
	fmt.Println(p.name, " can fly like a bird")
	ret = p.name
	return
}

type Butterfly struct {
	name string
}

func (bf *Butterfly) Fly() string {
	fmt.Println(bf.name, " can fly like a bird")
	return bf.name
}

func GetFlyType(bird Bird) {
	_, ok := bird.(*Butterfly)
	if ok {
		fmt.Println("type is *butterfly")
		return
	}

	_, ok = bird.(*Plane)
	if ok {
		fmt.Println("type is *Plane")
		return
	}

	fmt.Println("unknown type")
}

/*


 */

type Human struct {
}

func (*Human) Walk() {

}

func GetFlyType2(inter interface{}) {
	_, ok := inter.(*Butterfly)
	if ok {
		fmt.Println("type is *butterfly")
		return
	}

	_, ok = inter.(*Plane)
	if ok {
		fmt.Println("type is *Plane")
		return
	}
	_, ok = inter.(*Human)
	if ok {
		fmt.Println("type is *Human")
		return
	}
	fmt.Println("unknown type")
}
func main() {
	var b Bird

	p := &Plane{name: "Boeing"}
	b = p
	b.Fly() // 输出：Boeing  can fly like a bird

	bf := &Butterfly{name: "Monarch"}
	b = bf
	b.Fly() // 输出：Monarch  can fly like a bird
	pl := &Plane{name: "plane"}
	bf = &Butterfly{name: "butterfly"}
	GetFlyType(pl)
	GetFlyType(bf)

	/*
	 */
	hu := &Human{}
	GetFlyType2(pl)
	GetFlyType2(bf)
	GetFlyType2(hu)
}
