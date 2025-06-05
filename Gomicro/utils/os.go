package utils

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// GetCurrentProcessID 获取当前进程的 PID，并以字符串形式返回。
func GetCurrentProcessID() string {
	return strconv.Itoa(os.Getegid())
}

func GetCurrentGoroutineID() string {
	buf := make([]byte, 128)              // 分配一个足够大的缓冲区以存储调用栈信息
	buf = buf[:runtime.Stack(buf, false)] // 获取当前 Goroutine 的栈信息；false 表示只获取当前 Goroutine
	stackInfo := string(buf)              // 转换为字符串以便处理
	// 样例 stackInfo： "goroutine 10 [running]:\nmain.main()\n..."
	// 提取 goroutine ID
	return strings.TrimSpace( // 去除前后空白字符
		strings.Split( // 分割字符串，取出 goroutine ID
			strings.Split(stackInfo, "[running]")[0], // 截取 [running] 之前的部分，如 "goroutine 10 "
			"goroutine")[1],                          // 再从 "goroutine 10 " 中提取 " 10 "
	)
}

// GetProcessAndGoroutineIDStr 返回当前进程 ID 和当前 Goroutine ID 的拼接字符串，格式为 "PID_GID"。
func GetProcessAndGoroutineIDStr() string {
	return fmt.Sprintf("%s_%s", GetCurrentProcessID(), GetCurrentGoroutineID())
}
