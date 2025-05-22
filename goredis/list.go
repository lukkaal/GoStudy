package main

type Node struct {
	Val  *Gobj
	next *Node
	prev *Node
}

type ListType struct {
	EqualFunc func(a, b *Gobj) bool
}

type List struct {
	ListType
	head   *Node
	tail   *Node
	length int
}

func ListCreate(listType ListType) *List {
	var list List
	list.ListType = listType
	return &list
}

func (list *List) Length() int {
	return list.length
}

func (list *List) First() *Node {
	return list.head
}

func (list *List) Last() *Node {
	return list.tail
}

func (list *List) Find(val *Gobj) *Node {
	node := list.head
	for node != nil {
		if list.EqualFunc(node.Val, val) {
			break
		}
		node = node.next
	}
	return node
}

func (list *List) Append(val *Gobj) {
	var node Node
	node.Val = val
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.prev = list.tail
		list.tail.next = &node
		list.tail = list.tail.next
	}
	list.length += 1
}

func (list *List) LPush(val *Gobj) {
	var n Node
	n.Val = val
	if list.head == nil {
		list.head = &n
		list.tail = &n
	} else {
		n.next = list.head
		list.head.prev = &n
		list.head = &n
	}
	list.length += 1
}

func (list *List) DelNode(n *Node) {
	if n == nil {
		return
	}
	if list.head == n {
		if n.next != nil {
			n.next.prev = nil
		}
		list.head = n.next
		n.next = nil
	} else if list.tail == n {
		if n.prev != nil {
			n.prev.next = nil
		}
		list.tail = n.prev
		n.prev = nil
	} else {
		if n.prev != nil {
			n.prev.next = n.next
		}
		if n.next != nil {
			n.next.prev = n.prev
		}
		n.prev = nil
		n.next = nil
	}
	list.length -= 1
}

func (list *List) Delete(val *Gobj) {
	list.DelNode(list.Find(val))
}
