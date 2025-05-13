package main

import (
	"fmt"
	"reflect"
)

type Hero struct {
	name string
	Id   int
}

func (h Hero) PrintData() {
	fmt.Println("Hero name is ", h.name, " id is ", h.Id)
}

func (h Hero) SetName(name string) {
	h.name = name
}

func (h *Hero) SetName2(name string) {
	h.name = name
}

func (h *Hero) PrintData2() {
	fmt.Println("Hero name is ", h.name, " id is ", h.Id)
}

func ReflectTypeValue(itf interface{}) {

	rtype := reflect.TypeOf(itf)
	fmt.Println("reflect type is ", rtype)
	rvalue := reflect.ValueOf(itf)
	fmt.Println("reflect value is ", rvalue)
	fmt.Println("reflect  value kind is", rvalue.Kind())
	fmt.Println("reflect type kind is", rtype.Kind())
	fmt.Println("reflect  value type is", rvalue.Type())
}

func ReflectStructPtrElem(itf interface{}) {
	rvalue := reflect.ValueOf(itf)
	for i := 0; i < rvalue.Elem().NumField(); i++ {
		elevalue := rvalue.Elem().Field(i)
		fmt.Println("element ", i, " its type is ", elevalue.Type())
		fmt.Println("element ", i, " its kind is ", elevalue.Kind())
		fmt.Println("element ", i, " its value is ", elevalue)
		field := rvalue.Elem().Field(i)
		if (field.Kind() == reflect.Float32 || field.Kind() == reflect.Float64) && elevalue.CanSet() {
			elevalue.SetFloat(100.0)
		}

	}

	if rvalue.Elem().Field(1).CanSet() {
		// rvalue.Elem().Field(1).SetInt(100)
		rvalue.Elem().Field(1).Set(reflect.ValueOf(100))
		fmt.Println("struct element 1 changed to ", rvalue.Elem().Field(1))
	} else {
		fmt.Println("struct element 1 can't be changed")
	}

}

func main() {
	ReflectTypeValue(Hero{name: "zack", Id: 1})
	ReflectStructPtrElem(&Hero{name: "Rolin", Id: 20})
}
