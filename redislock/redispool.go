package redis_lock

import (
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 定义一个全局的 Redis 连接池
var redisPool *redis.Pool

func init() {
	redisAddr := "localhost:6379" // 例如: "127.0.0.1:6379" 或 "your-aliyun-redis-instance.aliyuncs.com:6379"
	redisPassword := ""           // 如果有密码，请填写
	redisDB := 0                  // 数据库索引

	// 连接池
	redisPool = &redis.Pool{
		MaxIdle:     10,                // 最大空闲连接数
		MaxActive:   100,               // 最大激活连接数，0 表示无限制
		IdleTimeout: 240 * time.Second, // 客户端空闲连接超时时间
		Dial: func() (redis.Conn, error) {
			opts := []redis.DialOption{
				redis.DialConnectTimeout(5 * time.Second),
				redis.DialReadTimeout(3 * time.Second),
				redis.DialWriteTimeout(3 * time.Second),
				redis.DialDatabase(redisDB),
			}
			if redisPassword != "" {
				opts = append(opts, redis.DialPassword(redisPassword))
			}
			c, err := redis.Dial("tcp", redisAddr, opts...)
			if err != nil {
				return nil, fmt.Errorf("Redis 连接池 Dial 错误: %v", err)
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			// 如果连接闲置超过1分钟，发送 PING 命令测试连接活性
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			if err != nil {
				log.Printf("连接池 PING 命令执行失败: %v", err)
			}
			return err
		},
	}
}

func test2() {
	fmt.Println("--- 开始 Redigo 功能测试范例 ---")
	conn := redisPool.Get()
	defer conn.Close()

	fmt.Println("\n--- 1. 字符串操作 ---")
	_, err := conn.Do("SET", "USER:1:NAME", "John Doe")
	if err != nil {
		log.Fatalf("SET user:1:name 失败: %v", err)
	}
	name, err := redis.String(conn.Do("GET", "USER:1:NAME"))
	if err != nil {
		log.Fatalf("GET user:1:name 失败: %v", err)
	}
	fmt.Printf("GET user:1:name -> %s\n", name)

	// 设置带过期时间的键
	_, err = conn.Do("SETEX", "temp_key", 5, "This key will expire in 5 seconds") // 5秒过期
	if err != nil {
		log.Fatalf("SETEX temp_key 失败: %v", err)
	}
	fmt.Println("SETEX temp_key 成功，5秒后过期。")

	expiredValue, err := redis.String(conn.Do("GET", "temp_key"))
	if err != nil && err != redis.ErrNil { // ErrNil 表示键不存在
		log.Fatalf("GET expired_key 失败: %v", err)
	}
	if expiredValue == "" {
		fmt.Println("temp_key 已过期或不存在，符合预期。")
	}

	// 2. 列表 (List) 操作
	fmt.Println("\n--- 2. 列表操作 ---")
	_, err = conn.Do("RPUSH", "my_list", "item1", "item2", "item3") // 从右侧插入
	if err != nil {
		log.Fatalf("RPUSH my_list 失败: %v", err)
	}
	fmt.Println("RPUSH my_list 'item1', 'item2', 'item3' 成功。")

	// 获取列表所有元素
	listItems, err := redis.Strings(conn.Do("LRANGE", "my_list", 0, -1))
	if err != nil {
		log.Fatalf("LRANGE my_list 失败: %v", err)
	}
	fmt.Printf("LRANGE my_list -> %v\n", listItems)

	// 从左侧弹出
	poppedItem, err := redis.String(conn.Do("LPOP", "my_list"))
	if err != nil {
		log.Fatalf("LPOP my_list 失败: %v", err)
	}
	fmt.Printf("LPOP my_list -> %s\n", poppedItem)
	// 3. 哈希 (Hash) 操作
	fmt.Println("\n--- 3. 哈希操作 ---")
	_, err = conn.Do("HMSET", "user:2", "name", "Bob", "age", 25, "email", "bob@example.com")
	if err != nil {
		log.Fatalf("HMSET user:2 失败: %v", err)
	}
	fmt.Println("HMSET user:2 成功。")

	// 获取单个字段
	userAge, err := redis.Int(conn.Do("HGET", "user:2", "age"))
	if err != nil {
		log.Fatalf("HGET user:2 age 失败: %v", err)
	}
	fmt.Printf("HGET user:2 age -> %d\n", userAge)

	// 获取多个字段 (使用 redis.Values 转换为 []interface{})
	userDetails, err := redis.Values(conn.Do("HMGET", "user:2", "name", "email"))
	if err != nil {
		log.Fatalf("HMGET user:2 name email 失败: %v", err)
	}
	fmt.Printf("HMGET user:2 name email -> %v\n", userDetails) // 注意这里会是 []interface{}

	// 4. 集合 (Set) 操作
	fmt.Println("\n--- 4. 集合操作 ---")
	_, err = conn.Do("SADD", "tags", "go", "redis", "database")
	if err != nil {
		log.Fatalf("SADD tags 失败: %v", err)
	}
	fmt.Println("SADD tags 成功。")

	isMember, err := redis.Bool(conn.Do("SISMEMBER", "tags", "go"))
	if err != nil {
		log.Fatalf("SISMEMBER tags go 失败: %v", err)
	}
	fmt.Printf("SISMEMBER tags go -> %t\n", isMember)

	// 获取所有成员
	allTags, err := redis.Strings(conn.Do("SMEMBERS", "tags"))
	if err != nil {
		log.Fatalf("SMEMBERS tags 失败: %v", err)
	}
	fmt.Printf("SMEMBERS tags -> %v\n", allTags)

	// 5. 管道 (Pipelining)
	fmt.Println("\n--- 5. 管道 (Pipelining) 操作 ---")
	conn.Send("SET", "pipe_key_1", "value_pipe_1")
	conn.Send("GET", "pipe_key_1")
	conn.Send("INCR", "pipe_counter")
	conn.Send("GET", "pipe_counter")
	conn.Flush() // 将缓冲区中的命令一次性发送

	// 按照发送顺序接收结果
	_, err = conn.Receive() // SET 返回 OK
	if err != nil {
		log.Fatalf("管道接收 1 失败: %v", err)
	}
	fmt.Println("管道: SET 命令结果已接收。")

	valPipe1, err := redis.String(conn.Receive()) // GET 返回 value_pipe_1
	if err != nil {
		log.Fatalf("管道接收 2 失败: %v", err)
	}
	fmt.Printf("管道: GET pipe_key_1 -> %s\n", valPipe1)

	_, err = conn.Receive() // INCR 返回计数
	if err != nil {
		log.Fatalf("管道接收 3 失败: %v", err)
	}
	fmt.Println("管道: INCR 命令结果已接收。")

	counterPipe, err := redis.Int(conn.Receive()) // GET 返回最终计数
	if err != nil {
		log.Fatalf("管道接收 4 失败: %v", err)
	}
	fmt.Printf("管道: GET pipe_counter -> %d\n", counterPipe)

	// 清理测试数据 (可选)
	fmt.Println("\n--- 清理测试数据 ---")
	_, err = conn.Do("DEL", "user:1:name", "my_list", "user:2", "tags", "pipe_key_1", "pipe_counter")
	if err != nil {
		log.Printf("清理测试数据失败: %v", err)
	} else {
		fmt.Println("测试数据已清理。")
	}

	fmt.Println("\n--- Redigo 功能测试范例结束 ---")
}
