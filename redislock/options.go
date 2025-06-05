package redis_lock

import "time"

const (
	// 默认连接池超过 10 s 释放连接
	DefaultIdleTimeoutSeconds = 10
	// 默认最大激活连接数
	DefaultMaxActive = 100
	// 默认最大空闲连接数
	DefaultMaxIdle = 20

	// 默认的分布式锁过期时间
	DefaultLockExpireSeconds = 30
	// 看门狗工作时间间隙
	WatchDogWorkStepSeconds = 10
)

type ClientOptions struct {
	maxIdle            int
	idleTimeoutSeconds int
	maxActive          int
	wait               bool
	// 以下是连接 Redis 服务器的必填参数
	network  string // 网络类型，通常是 "tcp"
	address  string // Redis 服务器的地址和端口，例如 "localhost:6379" 或 "192.168.1.100:6379"
	password string // 连接 Redis 服务器所需的密码（如果 Redis 启用了认证）
}

type ClientOption func(c *ClientOptions)

func WithMaxIdle(maxIdle int) ClientOption {
	return func(c *ClientOptions) {
		c.maxIdle = maxIdle
	}
}

func WithIdleTimeoutSeconds(idleTimeoutSeconds int) ClientOption {
	return func(c *ClientOptions) {
		c.idleTimeoutSeconds = idleTimeoutSeconds
	}
}

// WithMaxActive 返回一个 ClientOption，用于设置连接池的最大活跃连接数。
func WithMaxActive(maxActive int) ClientOption {
	return func(c *ClientOptions) {
		c.maxActive = maxActive
	}
}

// 在等待模式下，如果连接池已满，客户端会阻塞直到有可用连接。
func WithWaitMode() ClientOption {
	return func(c *ClientOptions) {
		c.wait = true
	}
}

// repairClient 函数用于校验和修正 ClientOptions 中的值，确保它们是合理的。
// 如果用户没有指定或者指定了不合理的值（例如负数），则会使用默认值。
func repairClient(c *ClientOptions) {
	if c.maxIdle < 0 { // 如果最大空闲连接数小于 0，则使用默认值
		c.maxIdle = DefaultMaxIdle
	}

	if c.idleTimeoutSeconds < 0 { // 如果空闲超时时间小于 0，则使用默认值
		c.idleTimeoutSeconds = DefaultIdleTimeoutSeconds
	}

	if c.maxActive < 0 { // 如果最大活跃连接数小于 0，则使用默认值
		c.maxActive = DefaultMaxActive
	}
}

// LockOptions 结构体定义了分布式锁的各种行为参数。
type LockOptions struct {
	isBlock             bool  // 是否在获取锁失败时阻塞等待(是不是阻塞锁)
	blockWaitingSeconds int64 // 阻塞模式下，最长等待时间（秒）
	expireSeconds       int64 // 锁的过期时间（秒），防止死锁
	watchDogMode        bool  // 是否启用看门狗模式，自动延长锁的有效期
}

// LockOption 是一个函数类型，用于配置分布式锁的获取行为。
type LockOption func(r *LockOptions)

// WithBlock 返回一个 LockOption，用于将锁设置为阻塞模式。
// 在阻塞模式下，如果锁已被占用，客户端会等待直到锁可用。
func WithBlock() LockOption {
	return func(o *LockOptions) {
		o.isBlock = true
	}
}

// WithBlockWaitingSeconds 返回一个 LockOption，用于设置阻塞模式下的最长等待时间（秒）。
// 只有当 WithBlock 被启用时，此选项才有效。
func WithBlockWaitingSeconds(waitingSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.blockWaitingSeconds = waitingSeconds
	}
}

// WithExpireSeconds 返回一个 LockOption，用于设置分布式锁的过期时间（秒）。
// 这是一个重要的安全机制，防止锁永远不被释放。
func WithExpireSeconds(expireSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.expireSeconds = expireSeconds
	}
}

// repairLock 函数用于校验和修正 LockOptions 中的值，并根据需要启动看门狗模式。
func repairLock(o *LockOptions) {
	// 如果锁是阻塞模式，但未设置或设置了不合理的等待时间，则使用默认值 5 秒。
	if o.isBlock && o.blockWaitingSeconds <= 0 {
		o.blockWaitingSeconds = 5 // 默认阻塞等待时间上限为 5 秒
	}

	// 如果用户显式设置了锁的过期时间（大于 0），则直接返回，不启动看门狗。
	if o.expireSeconds > 0 {
		return
	}

	// 如果用户未显式指定锁的过期时间，则此时会启用看门狗模式。
	// 锁的过期时间将设置为默认值，并且 watchDogMode 会被设置为 true，
	// 这意味着系统会尝试自动续期此锁。
	o.expireSeconds = DefaultLockExpireSeconds // 使用默认过期时间
	o.watchDogMode = true                      // 启用看门狗模式
}

// RedLockOption 是一个函数类型，用于配置 Redlock 算法相关的参数。
type RedLockOption func(*RedLockOptions)

// RedLockOptions 结构体包含了 Redlock 算法的配置参数。
// Redlock 是一种更复杂的分布式锁算法，用于提高高可用性环境下的锁的鲁棒性。
type RedLockOptions struct {
	singleNodesTimeout time.Duration // 在 Redlock 算法中，操作单个 Redis 节点（如获取锁）的超时时间
	expireDuration     time.Duration // 整个 Redlock 分布式锁的有效总时长
}

// WithSingleNodesTimeout 返回一个 RedLockOption，用于设置 Redlock 操作单个节点的超时时间。
func WithSingleNodesTimeout(singleNodesTimeout time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.singleNodesTimeout = singleNodesTimeout
	}
}

// WithRedLockExpireDuration 返回一个 RedLockOption，用于设置 Redlock 的总过期时间。
func WithRedLockExpireDuration(expireDuration time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.expireDuration = expireDuration
	}
}

// SingleNodeConf 结构体用于定义 Redlock 算法中每个独立的 Redis 节点的连接信息。
// Redlock 需要在多个独立的 Redis 实例上操作。
type SingleNodeConf struct {
	Network  string         // Redis 节点的网络类型（如 "tcp"）
	Address  string         // Redis 节点的地址和端口
	Password string         // Redis 节点的密码
	Opts     []ClientOption // 针对该节点特定的客户端连接选项，可以覆盖默认 ClientOptions
}

// repairRedLock 函数用于校验和修正 RedLockOptions 中的值。
func repairRedLock(o *RedLockOptions) {
	// 如果单个节点操作超时时间小于等于 0，则使用默认值。
	// DefaultSingleLockTimeout 这个常量在当前代码片段中未定义，
	// 预期在实际项目中会在其他地方定义。
	if o.singleNodesTimeout <= 0 {
		// 假设 DefaultSingleLockTimeout 已经被定义，例如：
		// const DefaultSingleLockTimeout = 100 * time.Millisecond
		o.singleNodesTimeout = DefaultSingleLockTimeout
	}
}

// 另一个文件中
// ********
// 红锁中每个节点默认的处理超时时间为 50 ms
// const DefaultSingleLockTimeout = 50 * time.Millisecond
