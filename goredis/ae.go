package main

import (
	"log"
	"time"

	"golang.org/x/sys/unix"
)

type FeType int

const (
	AE_READABLE FeType = 1
	AE_WRITABLE FeType = 2
)

type TeType int

// 普通事件和一次性事件
const (
	AE_ONCE   TeType = 2
	AE_NORMAL TeType = 1
)

type FileProc func(loop *AeLoop, fd int, extra interface{})
type TimeProc func(loop *AeLoop, fd int, extra interface{})

type AeFileEvent struct {
	fd    int
	mask  FeType
	when  int64
	proc  FileProc
	extra interface{} // 通用字段 用于给处理函数 proc 传递额外上下文数据
	next  *AeFileEvent
}

type AeTimeEvent struct {
	id       int
	mask     TeType // 事件次数类型
	when     int64
	interval int64 // 重复触发的时间间隔
	proc     TimeProc
	extra    interface{}
	next     *AeTimeEvent
}

type AeLoop struct {
	FileEvents      map[int]*AeFileEvent
	TimeEvents      *AeTimeEvent
	fileEventFd     int
	timeEventNextId int
	stop            bool
}

// fe到epoll的映射关系
var fe2ep [3]uint32 = [3]uint32{0, unix.EPOLLIN, unix.EPOLLOUT}

func getFeKey(fd int, mask FeType) int {
	if mask == AE_READABLE {
		return fd
	} else {
		return fd * -1
	}
}

// 从注册的 AeFileEvent 中获取事件类型 r/w
func (loop *AeLoop) getEpollMask(fd int) uint32 {
	var ev uint32
	if loop.FileEvents[getFeKey(fd, AE_READABLE)] != nil {
		ev |= unix.EPOLLIN
	} else if loop.FileEvents[getFeKey(fd, AE_WRITABLE)] != nil {
		ev |= unix.EPOLLOUT
	}
	return ev
}

func (loop *AeLoop) addFileEvent(fd int, mask FeType, proc FileProc, extra interface{}) {
	// epoll_ctl 添加事件
	ev := loop.getEpollMask(fd)
	// 如果已经注册了该事件且事件类型相同
	if ev&fe2ep[mask] != 0 {
		return
	}
	op := unix.EPOLL_CTL_ADD
	// 如果已注册但是事件类型不同
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	ev |= fe2ep[mask] // 获取事件类型
	err := unix.EpollCtl(loop.fileEventFd, op, fd, &unix.EpollEvent{Events: ev, Fd: int32(fd)})
	if err != nil {
		log.Println("epoll_ctl error:", err)
		return
	}

	// 注册aefileevent (用户态维护)
	var fe AeFileEvent
	fe.fd = fd
	fe.mask = mask
	fe.proc = proc
	fe.extra = extra
	loop.FileEvents[getFeKey(fd, mask)] = &fe
	log.Printf("ae add file event fd:%v, mask:%v\n", fd, mask)

}

func (loop *AeLoop) RemoveFileEvent(fd int, mask FeType) {
	// epoll_ctl 删除事件
	op := unix.EPOLL_CTL_DEL
	ev := loop.getEpollMask(fd)
	ev &= ^fe2ep[mask] // 根据 mask 清除事件类型
	// 如果还有事件 说明只需要修改事件类型
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	err := unix.EpollCtl(loop.fileEventFd, op, fd, &unix.EpollEvent{Events: ev, Fd: int32(fd)})
	if err != nil {
		log.Println("epoll_ctl error:", err)
		return
	}
	// 删除用户态的对应事件，如果关心多个事件则分多次注册和删除
	loop.FileEvents[getFeKey(fd, mask)] = nil // 一次只会删除一个关心的事件
	log.Printf("ae remove file event fd:%v, mask:%v\n", fd, mask)
}

func GetMsTime() int64 {
	return time.Now().UnixNano() / 1e6
}

// *TimeEvents 是用户态队列 (*AeTimeEvent)
func (loop *AeLoop) AddTimeEvent(mask TeType, interval int64, proc TimeProc, extra interface{}) int {
	id := loop.timeEventNextId
	loop.timeEventNextId++
	var te AeTimeEvent
	te.id = id
	te.mask = mask
	te.interval = interval
	te.when = GetMsTime() + interval
	te.proc = proc
	te.extra = extra
	te.next = loop.TimeEvents
	loop.TimeEvents = &te
	return id
}

// 链表操作
func (loop *AeLoop) RemoveTimeEvent(id int) {
	p := loop.TimeEvents
	var pre *AeTimeEvent
	for p != nil {
		if p.id == id {
			if pre == nil {
				loop.TimeEvents = p.next
			} else {
				pre.next = p.next
			}
			break
		}
		pre = p
		p = p.next
	}
}

// epoll_create1 创建 epoll fd，并初始化 AeLoop
func AeLoopCreate() (*AeLoop, error) {
	epollFd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &AeLoop{
		fileEventFd:     epollFd,
		FileEvents:      make(map[int]*AeFileEvent),
		TimeEvents:      nil,
		stop:            false,
		timeEventNextId: 1,
	}, nil
}

// 在当前 AeLoop 的所有定时事件中，找出最早触发的时间点（时间戳，毫秒）并返回
func (loop *AeLoop) nearestTime() int64 {
	var nearest int64 = GetMsTime() + 1000
	p := loop.TimeEvents
	for p != nil {
		if p.when < nearest {
			nearest = p.when
		}
		p = p.next
	}
	return nearest
}

func (loop *AeLoop) AeWait() (tes []*AeTimeEvent, fes []*AeFileEvent) {
	/*
		计算下一次事件等待的超时时间 timeout，确保事件循环不会无限等待
		而是根据最近的定时事件动态调整等待时长。
	*/
	timeout := loop.nearestTime() - GetMsTime()
	if timeout < 0 {
		timeout = 10
	}
	// file
	var events [128]unix.EpollEvent
	n, err := unix.EpollWait(loop.fileEventFd, events[:], int(timeout))
	if err != nil {
		log.Println("epoll_wait error:", err)
	}
	if n > 0 {
		log.Println("ae get ", n, " events")
	}
	for i := 0; i < n; i++ {
		if events[i].Events&unix.EPOLLIN != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].Fd), AE_READABLE)]
			if fe != nil {
				fes = append(fes, fe) // 添加到返回的文件事件列表 *AeFileEvent
			}
		}
		if events[i].Events&unix.EPOLLOUT != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].Fd), AE_WRITABLE)]
			if fe != nil {
				fes = append(fes, fe) // 添加到返回的文件事件列表 *AeFileEvent
			}
		}
	}
	// time
	now := GetMsTime()
	p := loop.TimeEvents
	for p != nil {
		if p.when <= now {
			tes = append(tes, p)
		}
		p = p.next
	}
	return
}

func (loop *AeLoop) AeProcess(tes []*AeTimeEvent, fes []*AeFileEvent) {
	// index + value(*AeTimeEvent/ *AeFileEvent)
	for _, te := range tes {
		te.proc(loop, te.id, te.extra)
		// 如果是一次性事件，删除该事件(只针对timeevent)
		if te.mask == AE_ONCE {
			loop.RemoveTimeEvent(te.id)
		} else {
			te.when = GetMsTime() + te.interval
		}
	}
	if len(fes) > 0 {
		log.Println("ae is processing file events")
		for _, fe := range fes {
			fe.proc(loop, fe.fd, fe.extra)
		}
	}
}

func (loop *AeLoop) AeMain() {
	for loop.stop != true {
		tes, fes := loop.AeWait()
		loop.AeProcess(tes, fes)
	}
}

/*
type EpollEvent struct {
    Events uint32
    Fd     int32
    Pad    int32
}
*/
