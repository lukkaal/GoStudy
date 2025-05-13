package main

import (
	"fmt"
	"reflect"
)

type Hero struct {
	name string
	id   int
}

func (h Hero) PrintData() {
	fmt.Println("Hero name is ", h.name, " id is ", h.id)
}

func (h Hero) SetName(name string) {
	h.name = name
}

func (h *Hero) SetName2(name string) {
	h.name = name
}

func (h *Hero) PrintData2() {
	fmt.Println("Hero name is ", h.name, " id is ", h.id)
}

func ReflectStructMethodName(itf interface{}) {
	rvalue := reflect.ValueOf(itf)

	// 调用值接收者方法，PrintData 是值接收者方法
	fmt.Println("Calling PrintData (value receiver):")
	rvalue.MethodByName("PrintData").Call(nil)

	// 调用有参数的值接收者方法，SetName 是值接收者方法
	params := []reflect.Value{reflect.ValueOf("Rolin")}
	fmt.Println("Calling SetName (value receiver) with params:")
	rvalue.MethodByName("SetName").Call(params)

	// 调用修改后的值接收者方法，验证是否生效
	fmt.Println("Calling PrintData again after SetName (value receiver):")
	rvalue.MethodByName("PrintData").Call(nil)

	// 调用指针接收者方法，PrintData2 是指针接收者方法
	fmt.Println("Calling PrintData2 (pointer receiver):")
	rvalue.MethodByName("PrintData2").Call(nil)

	// 调用有参数的指针接收者方法，SetName2 是指针接收者方法
	fmt.Println("Calling SetName2 (pointer receiver) with params:")
	rvalue.MethodByName("SetName2").Call(params)
	fmt.Println("method name: ", rvalue.Method(3).Type().String())

	// rvalue.Method(3).Call(params)

	// 再次调用指针接收者方法，PrintData2 来验证是否生效
	fmt.Println("Calling PrintData2 again after SetName2 (pointer receiver):")
	rvalue.MethodByName("PrintData2").Call(nil)
}

func ReflectStructPtrMethodPtr(itf interface{}) {
	rvalue := reflect.ValueOf(itf)
	rtype := reflect.TypeOf(itf)
	fmt.Println("Hero pointer struct method list......................")
	for i := 0; i < rvalue.NumMethod(); i++ {
		methodvalue := rvalue.Method(i)
		fmt.Println("method ", i, " value is ", methodvalue)
		methodtype := rtype.Method(i)
		fmt.Println("method ", i, " type is ", methodtype)
		fmt.Println("method ", i, " name is ", methodtype.Name)
		fmt.Println("method ", i, " method.type is ", methodtype.Type)
	}

	//reflect.ValueOf 方法调用,无参方法调用
	fmt.Println(rvalue.Method(1).Call(nil))
	//有参方法调用
	params := []reflect.Value{reflect.ValueOf("Rolin")}
	fmt.Println(rvalue.Method(3).Call(params))
	//修改了，生效
	fmt.Println(rvalue.Method(0).Call(nil))

	fmt.Println("Hero Struct method list......................")
	for i := 0; i < rvalue.Elem().NumMethod(); i++ {
		methodvalue := rvalue.Elem().Method(i)
		fmt.Println("method ", i, " value is ", methodvalue)
		methodtype := rtype.Elem().Method(i)
		fmt.Println("method ", i, " type is ", methodtype)
		fmt.Println("method ", i, " name is ", methodtype.Name)
		fmt.Println("method ", i, " method.type is ", methodtype.Type)
	}
}

func ReflectStructPtrMethodValue(itf interface{}) {
	rvalue := reflect.ValueOf(itf)
	rtype := reflect.TypeOf(itf)
	fmt.Println("Hero pointer struct method list......................")
	for i := 0; i < rvalue.NumMethod(); i++ {
		methodvalue := rvalue.Method(i)
		fmt.Println("method ", i, " value is ", methodvalue)
		methodtype := rtype.Method(i)
		fmt.Println("method ", i, " type is ", methodtype)
		fmt.Println("method ", i, " name is ", methodtype.Name)
		fmt.Println("method ", i, " method.type is ", methodtype.Type)
	}

	//reflect.ValueOf 方法调用,无参方法调用
	fmt.Println(rvalue.Method(0).Call(nil))
	//有参方法调用
	params := []reflect.Value{reflect.ValueOf("Rolin")}
	fmt.Println(rvalue.Method(1).Call(params))

	fmt.Println("Hero Struct method list......................")
	fmt.Println(rvalue.NumMethod())
	for i := 0; i < rvalue.NumMethod(); i++ {
		methodvalue := rvalue.Method(i)
		fmt.Println("method ", i, " value is ", methodvalue)
		methodtype := rtype.Method(i)
		fmt.Println("method ", i, " type is ", methodtype)
		fmt.Println("method ", i, " name is ", methodtype.Name)
		fmt.Println("method ", i, " method.type is ", methodtype.Type)
	}
}

func main() {
	// 反射调用结构体方法
	ReflectStructMethodName(&Hero{name: "Elli", id: 20})
	ReflectStructPtrMethodPtr(&Hero{name: "Elli", id: 20})
	ReflectStructPtrMethodValue(Hero{name: "Elli", id: 20})

}
