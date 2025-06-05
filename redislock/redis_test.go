package redis_lock

import (
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

func test() {
	redisAddr := "localhost:6379"
	redisPassword := ""
	redisDB := 0

	/*
		type DialOption struct {
		f func(*dialOptions)}
	*/
	// 准备连接选项
	opts := []redis.DialOption{
		redis.DialConnectTimeout(5 * time.Second), // 连接超时5秒
		redis.DialReadTimeout(3 * time.Second),    // 读取超时3秒
		redis.DialWriteTimeout(3 * time.Second),   // 写入超时3秒
		redis.DialDatabase(redisDB),               // 设置数据库索引
	}
	// 设置了密码则要记录密码
	if redisPassword != "" {
		opts = append(opts, redis.DialPassword(redisPassword))
	}
	// 尝试建立 Redis 连接
	fmt.Printf("尝试连接 Redis 服务器: %s...\n", redisAddr)
	conn, err := redis.Dial("tcp", redisAddr, opts...)
	if err != nil {
		log.Fatalf("连接 Redis 服务器失败: %v", err)
	}
	/*
		如果中途出现错误或 return 提前退出，
		defer 也会被执行，保证资源总是被正确释放。
	*/
	defer conn.Close()
	pong, err := redis.String(conn.Do("PING"))
	if err != nil {
		log.Fatalf("PING 命令执行失败: %v\n这可能意味着连接不稳定或Redis服务出现问题。", err)
	}
	if pong == "PONG" {
		fmt.Println("Redis PING 命令返回 'PONG'。Redis 服务器运行正常。")
	} else {
		fmt.Printf("Redis PING 命令返回意外结果: %s\n", pong)
	}

	fmt.Println("\n------------------------------------")
	fmt.Println("Redis 连接和基本功能测试通过！")
	fmt.Println("现在可以运行更复杂的 Redigo 示例了。")
	fmt.Println("------------------------------------")
}
