package main

import (
	"fmt"
	"reflect"
)

func main() {
	var num float64 = 13.14
	rtype := reflect.TypeOf(num)
	fmt.Println("reflect type is ", rtype)
	rvalue := reflect.ValueOf(num)
	fmt.Println("reflect value is ", rvalue)
	fmt.Println("reflect  value kind is", rvalue.Kind())
	fmt.Println("reflect type kind is", rtype.Kind())
	fmt.Println("reflect  value type is", rvalue.Type())

	rptrvalue := reflect.ValueOf(&num)
	fmt.Println("reflect value is ", rptrvalue)
	fmt.Println("reflect  value kind is", rptrvalue.Kind())
	fmt.Println("reflect type kind is", rptrvalue.Kind())
	fmt.Println("reflect  value type is", rptrvalue.Type())
	if rptrvalue.Elem().CanSet() {
		rptrvalue.Elem().SetFloat(131)
	}

	fmt.Println(num)
}
