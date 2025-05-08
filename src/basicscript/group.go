package main

import (
	"fmt"
)

// 匿名组合和派生
type Base struct {
	Name string
}

func (base *Base) Foo() {
	fmt.Println("this is Base Foo")
}

func (base *Base) Bar() {
	fmt.Println("this is Base Bar")
}

type Foo struct {
	//匿名组合
	Base
}

func (foo *Foo) Foo() {
	foo.Base.Foo()
	fmt.Println("this is Foo Foo")
}

func main() {
	foo := &Foo{}
	//Foo继承Base，所以拥有Name属性
	foo.Name = "foobase"
	//Foo 重写(覆盖)了Base的Foo
	foo.Foo()
	//Foo继承了Base的Bar函数
	foo.Bar()
	//显示调用基类Base的Foo
	foo.Base.Foo()
	// 由于Foo继承Base后重写了Foo方法，所以想要调用Base的Foo方法，需要显示调用。
}
