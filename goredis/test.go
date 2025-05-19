package test

import (
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

func testownpc() {
	pid := unix.Getpid()
	fmt.Println("当前进程 PID:", pid)

	var uts unix.Utsname
	err := unix.Uname(&uts)
	if err != nil {
		log.Fatalf("Uname 获取失败: %v", err)
	}

	// 根据类型选择不同转换函数
	fmt.Printf("操作系统名称: %s\n", charsToStringFromBytes(uts.Sysname[:]))
	fmt.Printf("主机名称: %s\n", charsToStringFromBytes(uts.Nodename[:]))

	// Release 如果是 []byte 类型，用下面的转换
	fmt.Printf("内核版本: %s\n", charsToStringFromBytes(uts.Release[:]))
}

// []byte 转 string
func charsToStringFromBytes(ca []byte) string {
	n := 0
	for ; n < len(ca); n++ {
		if ca[n] == 0 {
			break
		}
	}
	return string(ca[:n])
}
