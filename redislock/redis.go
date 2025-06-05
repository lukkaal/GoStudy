package redis_lock

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 接口 interface{}
// *Client LockClient
type LockClient interface {
	SetNEX(ctx context.Context, key, value string, expireSeconds int64) (int64, error)
	Eval(ctx context.Context, src string, keyCount int, keysAndArgs []interface{}) (interface{}, error)
}

// Client Redis 客户端.
type Client struct {
	ClientOptions
	pool *redis.Pool
}

// 直接新建
func (c *Client) getRedisConn() (redis.Conn, error) {
	if c.address == "" {
		panic("Cannot get redis address from config")
	}
	var opts []redis.DialOption
	opts = append(opts,
		redis.DialConnectTimeout(5*time.Second), // 连接建立超时，5秒
		redis.DialReadTimeout(3*time.Second),    // 读操作超时，3秒
		redis.DialWriteTimeout(3*time.Second),   // 写操作超时，3秒
	)

	if len(c.password) > 0 {
		opts = append(opts, redis.DialPassword(c.password))
	}

	conn, err := redis.Dial(c.network, c.address, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// 走连接池
/*
func (p *Pool) GetContext(ctx context.Context) (Conn, error) {
	pc, err := p.get(ctx) // wait = true 连接池满时会阻塞等待连接可用（和 context 联动）
	if err != nil {
		return errorConn{err}, err
	}
	return &activeConn{p: p, pc: pc}, nil
}
*/
func (c *Client) GetConn(ctx context.Context) (redis.Conn, error) {
	return c.pool.GetContext(ctx) // ctx 防止超时
}

// 获取连接池
func (c *Client) getRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     c.maxIdle,
		IdleTimeout: time.Duration(c.idleTimeoutSeconds) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := c.getRedisConn()
			if err != nil {
				return nil, err
			}
			return c, nil
		},
		MaxActive: c.maxActive,
		Wait:      c.wait, // 连接池满了，如果还有 goroutine 请求连接，作为客户端愿不愿意阻塞等待
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewClient(network, address, password string, opts ...ClientOption) *Client {
	// opts 是一个函数切片 -> []ClientOption
	c := Client{
		ClientOptions: ClientOptions{
			network:  network,
			address:  address,
			password: password,
		},
	}
	// type ClientOption func(c *ClientOptions)
	for _, opt := range opts {
		opt(&c.ClientOptions) // ClientOption(c *ClientOptions) 执行函数
	}

	repairClient(&c.ClientOptions)

	pool := c.getRedisPool()
	return &Client{
		pool: pool,
	}
}

// context.Context 是 一个接口类型（interface）
// 只有 key 不存在时，能够 set 成功. set 时携带上超时时间，单位秒.
func (c *Client) SetNEX(ctx context.Context, key, value string, expireSeconds int64) (int64, error) {
	if key == "" || value == "" {
		return -1, errors.New("redis SET keyNX or value can't be empty")
	}

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close() // 将连接归还到连接池中，不是关闭 TCP 长连接本身

	reply, err := conn.Do("SET", key, value, "EX", expireSeconds, "NX")
	if err != nil {
		return -1, nil
	}

	r, _ := reply.(int64)
	return r, nil

}

// Eval 支持使用 lua 脚本. Lua 脚本在 Redis 服务器端作为一个事务执行
func (c *Client) Eval(ctx context.Context, src string, keyCount int, keysAndArgs []interface{}) (interface{}, error) {
	args := make([]interface{}, 2+len(keysAndArgs))
	args[0] = src
	args[1] = keyCount
	copy(args[2:], keysAndArgs)

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	return conn.Do("EVAL", args...)
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("redis GET key can't be empty")
	}
	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return redis.String(conn.Do("GET", key))
}

func (c *Client) SET(ctx context.Context, key string, value string) (int64, error) {
	if key == "" || value == "" {
		return -1, errors.New("redis SET key or value can't be empty")
	}

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	reply, err := conn.Do("SET", key, value)
	if err != nil {
		return -1, err
	}

	if respStr, ok := reply.(string); ok && strings.ToLower(respStr) == "ok" {
		return 1, nil
	}

	return redis.Int64(reply, err)
}

// 没有 expire
func (c *Client) SetNX(ctx context.Context, key, value string) (int64, error) {
	if key == "" || value == "" {
		return -1, errors.New("redis SET key NX or value can't be empty")
	}

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	reply, err := conn.Do("SET", key, value, "NX")
	if err != nil {
		return -1, err
	}

	respStr, ok := reply.(string)
	if ok {
		if strings.ToLower(respStr) == "ok" {
			return 1, nil
		}
	}

	return redis.Int64(reply, err)
}

func (c *Client) Del(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("redis DEL key can't be empty")
	}

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Do("DEL", key)
	return err
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	if key == "" {
		return -1, errors.New("redis INCR key can't be empty")
	}

	conn, err := c.pool.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	return redis.Int64(conn.Do("INCR", key))
}
