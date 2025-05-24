package main

import (
	"errors"
	"math"
	"math/rand"
)

const (
	INIT_SIZE    int64 = 8
	FORCE_RATIO  int64 = 2
	GROW_RATIO   int64 = 2
	DEFAULT_STEP int   = 1
)

var (
	EP_ERR = errors.New("expand error")
	EX_ERR = errors.New("key exists error")
	NK_ERR = errors.New("key doesnt exist error")
)

// 数据库元素类型
type Entry struct {
	Key  *Gobj
	Val  *Gobj
	next *Entry
}

type htable struct {
	table []*Entry // 链表入口
	size  int64
	mask  int64 // 掩码，始终等于 size - 1 用于快速计算哈希槽位：index = hash & mask
	used  int64 // 槽位的个数(数组+链表)
}

type DictType struct {
	HashFunc  func(key *Gobj) int64
	EqualFunc func(k1, k2 *Gobj) bool
}

/*
匿名内置

	数据库
*/
type Dict struct {
	DictType
	hts       [2]*htable
	rehashidx int64
}

func (dict *Dict) isRehashing() bool {
	return dict.rehashidx != -1
}

// 一次执行多少个 step
func (dict *Dict) rehashStep() {
	//TODO: check iterators
	dict.rehash(DEFAULT_STEP)
}

// rehash 过程
func (dict *Dict) rehash(step int) {
	for step > 0 {
		if dict.hts[0].used == 0 {
			dict.hts[0] = dict.hts[1]
			dict.hts[1] = nil
			dict.rehashidx = -1
			return
		}
		for dict.hts[0].table[dict.rehashidx] == nil {
			dict.rehashidx += 1
		}
		entry := dict.hts[0].table[dict.rehashidx]
		for entry != nil {
			ne := entry.next
			var idx int64 = dict.HashFunc(entry.Key) & dict.hts[1].mask
			entry.next = dict.hts[1].table[idx] // 头插法
			dict.hts[1].table[idx] = entry
			dict.hts[0].used -= 1
			dict.hts[1].used += 1
			entry = ne
		}
		dict.hts[0].table[dict.rehashidx] = nil
		dict.rehashidx += 1
		step -= 1
	}
}

func nextPower(size int64) int64 {
	for i := INIT_SIZE; i < math.MaxInt64; i *= 2 {
		if i >= size {
			return i
		}
	}
	return -1
}

func (dict *Dict) expand(size int64) error {
	// 保证 2 的幂次方
	sz := nextPower(size)
	// 达到阈值 size 就可以开始 rehash
	if dict.isRehashing() || (dict.hts[0] != nil && dict.hts[0].size >= sz) {
		return EP_ERR
	}
	var ht htable
	ht.size = sz
	ht.mask = sz - 1
	ht.table = make([]*Entry, sz)
	ht.used = 0
	// check for init
	if dict.hts[0] == nil {
		dict.hts[0] = &ht
		return nil
	}
	// start rehashing
	dict.hts[1] = &ht
	dict.rehashidx = 0
	return nil
}

func (dict *Dict) expandIfNeeded() error {
	if dict.isRehashing() {
		return nil
	}
	if dict.hts[0] == nil {
		return dict.expand(INIT_SIZE)
	}
	if (dict.hts[0].used > dict.hts[0].size) && (dict.hts[0].used/dict.hts[0].size > FORCE_RATIO) {
		return dict.expand(dict.hts[0].size * GROW_RATIO)
	}
	return nil
}

// 寻找用于插入的 index
func (dict *Dict) keyIndex(key *Gobj) int64 {
	err := dict.expandIfNeeded()
	if err != nil {
		return -1
	}
	var hash int64 = dict.HashFunc(key)
	var idx int64
	for i := 0; i <= 1; i++ {
		idx = hash & dict.hts[i].mask
		entry := dict.hts[i].table[idx]
		for entry != nil {
			if dict.EqualFunc(entry.Key, key) {
				return -1
			}
			entry = entry.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	return idx
}

// 分摊 rehash 到每一步
func (dict *Dict) AddRaw(key *Gobj) *Entry {
	if dict.isRehashing() {
		dict.rehashStep()
	}
	idx := dict.keyIndex(key)
	if idx == -1 {
		return nil
	}
	var ht *htable
	if dict.isRehashing() {
		ht = dict.hts[1]
	} else {
		ht = dict.hts[0]
	}
	var e Entry
	e.Key = key
	key.IncrRefCount()
	e.next = ht.table[idx]
	ht.table[idx] = &e
	ht.used += 1
	return &e
}

func (dict *Dict) Add(key, val *Gobj) error {
	entry := dict.AddRaw(key)
	if entry == nil {
		return EX_ERR
	}
	entry.Val = val
	val.IncrRefCount()
	return nil
}

func (dict *Dict) Set(key, val *Gobj) {
	err := dict.Add(key, val)
	if err == nil {
		return
	}
	entry := dict.Find(key)
	entry.Val.DecrRefCount()
	entry.Val = val
	val.IncrRefCount()
}

// 配合 delete 使用
func freeEntry(e *Entry) {
	e.Key.DecrRefCount()
	e.Val.DecrRefCount()
}

func (dict *Dict) Find(key *Gobj) *Entry {
	if dict.hts[0] == nil {
		return nil
	}
	if dict.isRehashing() {
		dict.rehashStep()
	}
	for i := 0; i <= 1; i++ {
		idx := dict.HashFunc(key)
		entry := dict.hts[0].table[idx]
		for entry != nil {
			if dict.EqualFunc(entry.Key, key) {
				return entry
			}
			entry = entry.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	return nil
}

func (dict *Dict) Get(key *Gobj) *Gobj {
	entry := dict.Find(key)
	if entry == nil {
		return nil
	}
	return entry.Val
}

// delete
func (dict *Dict) Delete(key *Gobj) error {
	if dict.hts[0] == nil {
		return NK_ERR
	}
	if dict.isRehashing() {
		dict.rehashStep()
	}
	// find key & delete & decr refcount
	h := dict.HashFunc(key)
	for i := 0; i <= 1; i++ {
		idx := h & dict.hts[i].mask
		e := dict.hts[i].table[idx]
		var prev *Entry
		for e != nil {
			if dict.EqualFunc(e.Key, key) {
				if prev == nil {
					dict.hts[i].table[idx] = e.next
				} else {
					prev.next = e.next
				}
				freeEntry(e)
				return nil
			}
			prev = e
			e = e.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	// key doesnt exist
	return NK_ERR
}

// random get
func (dict *Dict) RandomGet() *Entry {
	if dict.hts[0] == nil {
		return nil
	}
	t := 0
	if dict.isRehashing() {
		dict.rehashStep()
		if dict.hts[1] != nil && dict.hts[1].used > dict.hts[0].used {
			// simplify the logic, random get in the bigger table
			t = 1
		}
	}
	// random slot
	idx := rand.Int63n(dict.hts[t].size)
	cnt := 0
	for dict.hts[t].table[idx] == nil && cnt < 1000 {
		idx = rand.Int63n(dict.hts[t].size)
		cnt += 1
	}
	if dict.hts[t].table[idx] == nil {
		return nil
	}
	// random entry
	var listLen int64
	p := dict.hts[t].table[idx]
	for p != nil {
		listLen += 1
		p = p.next
	}
	listIdx := rand.Int63n(listLen)
	p = dict.hts[t].table[idx]
	for i := int64(0); i < listIdx; i++ {
		p = p.next
	}
	return p
}
