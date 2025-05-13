package stru

type Integer int

func (a Integer) Less(b Integer) bool {
	return a < b
}

// 构造函数和初始化
type Rect struct {
	X, y          float64
	width, height float64
}

func (r *Rect) Area() float64 {
	return r.width * r.height
}

// 作为跨包结构体成员变量的测试

// func main() {
// 	var varint1 Integer = 100
// 	var varint2 Integer = 200
// 	fmt.Println(varint1.Less(varint2))
// 	var rect1 Rect = Rect{0, 0, 10, 20}
// 	fmt.Println(rect1.Area())
// 	fmt.Println(rect1.x, rect1.y, rect1.width, rect1.height)
// }
