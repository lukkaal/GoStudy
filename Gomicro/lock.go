package redis_lock

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/xiaoxuxiansheng/redis_lock/utils"
)

// RedisLockKeyPrefix 是 Redis 分布式锁使用的键名前缀
const RedisLockKeyPrefix = "REDIS_LOCK_PREFIX_"

// 锁被其他客户端持有的错误
var ErrLockAcquiredByOthers = errors.New("Lock has been occupied")

// Redis 返回的空值错误（键不存在时）
var ErrNil = redis.ErrNil

// IsRetryableErr 判断错误是否是可重试错误
func IsRetryableErr(err error) bool {
	return errors.Is(err, ErrLockAcquiredByOthers)
}

type RedisLock struct {
	LockOptions
	key    string     // 锁的标识键
	token  string     // 使用方的身份标识
	client LockClient // 接口类型 不可使用指针

	// runningDog 原子共享 更像是一个 flag 用于防止如网络波动带来的开启多个看门狗
	runningDog int32              // 初始化为 0  是 RedisLock 实例的一个共享状态
	stopDog    context.CancelFunc // 初始化为 1
}

func NewRedisLock(key string, client LockClient, opts ...LockOption) *RedisLock {
	redislock := &RedisLock{
		key:    key,
		token:  utils.GetProcessAndGoroutineIDStr(),
		client: client,
	}
	for _, opt := range opts {
		opt(&redislock.LockOptions) // 带参调用
	}
	repairLock(&redislock.LockOptions)
	return redislock
}

func (lockclient *RedisLock) getLockKey() string {
	return RedisLockKeyPrefix + lockclient.key
}

func (lockclient *RedisLock) TryLock(ctx context.Context) error {
	reply, err := lockclient.client.SetNEX(ctx, lockclient.getLockKey(), lockclient.token, lockclient.expireSeconds)
	if err != nil {
		return err
	}
	if reply != 1 {
		return fmt.Errorf("reply: %d, err: %w", reply, ErrLockAcquiredByOthers)
	}
	return nil
}

// blockingLock 阻塞模式下使用时钟触发 持续尝试加锁
func (lockclient *RedisLock) BlockingLock(ctx context.Context) error {
	// func time.After(d time.Duration) <-chan time.Time
	timeoutchannel := time.After(time.Duration(lockclient.blockWaitingSeconds) * time.Second)
	// containing a channel that will send the current time on the channel after each tick
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// 不断从 ticker.C 通道中读取值但忽略读取的内容
	/*
			for {
		    <-ticker.C  // 阻塞等待 ticker 触发（每 50ms）
		    // 执行循环体代码
			}
	*/
	for range ticker.C {
		select {
		case <-ctx.Done():
			// 如果外部 Lock 的 context 被取消（例如超时或手动取消），退出并返回对应错误
			return fmt.Errorf("lock failed, ctx timeout, err: %w", ctx.Err())
			// 如果 blockWaitingSeconds 到期还没加锁成功，返回“锁被其他人占用”的超时错误
		case <-timeoutchannel:
			return fmt.Errorf("block waiting time out, err: %w", ErrLockAcquiredByOthers)

		default:
		}

		err := lockclient.TryLock(ctx)
		if err == nil {
			return nil
		}
		if !IsRetryableErr(err) {
			return err
		}
	}

	// 语法连贯但无法触发
	return nil
}

// 使用 EVAL 来设置 Lua 脚本达到延长锁的目的(看门狗的作用)
func (redislock *RedisLock) DelayExpire(ctx context.Context, expireSeconds int64) error {
	keysAndArgs := []interface{}{redislock.getLockKey(), redislock.token, expireSeconds}
	reply, err := redislock.client.Eval(ctx, LuaCheckAndExpireDistributionLock, 1, keysAndArgs)
	if err != nil {
		return err
	}
	ret, _ := reply.(int64)
	if ret != 1 {
		return errors.New("can not expire lock without ownership of lock")
	}
	return nil
}

// 看门狗
func (redislock *RedisLock) RunWatchDog(ctx context.Context) {
	ticker := time.NewTicker(WatchDogWorkStepSeconds * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
			//
		}
		// 延长时间
		_ = redislock.DelayExpire(ctx, WatchDogWorkStepSeconds+5)
	}
}

func (r *RedisLock) WatchDog(ctx context.Context) {
	// 1. 非看门狗模式，不处理
	if !r.watchDogMode {
		return
	}
	// *** 共享状态的原子性与可见性问题 ***
	// 2. 确保之前启动的看门狗已经正常回收 并设置为 1
	for !atomic.CompareAndSwapInt32(&r.runningDog, 0, 1) {
	}

	// 3. 启动看门狗
	ctx, r.stopDog = context.WithCancel(ctx)
	// r.stopDog 显示调用或者自行调用释放删除
	go func() {
		defer func() {
			atomic.StoreInt32(&r.runningDog, 0)
		}()
		r.RunWatchDog(ctx)
	}()
}

func (redislock *RedisLock) Lock(ctx context.Context) (err error) {
	// defer 最后连接成功后加锁
	// 加锁成功，启动看门狗
	defer func() {
		if err != nil {
			return
		}
		redislock.WatchDog(ctx)
	}()

	// 尝试加锁和阻塞加锁/非阻塞加锁 逻辑解耦
	err = redislock.TryLock(ctx) // 不使用 :=
	if err == nil {
		return nil
	}

	// 非阻塞模式下 直接返回
	if !redislock.isBlock {
		return err
	}

	// 错误类型不可重试，直接返回
	if !IsRetryableErr(err) {
		return err
	}

	// 阻塞模式，尝试轮询加锁
	err = redislock.BlockingLock(ctx)
	return err
}

// Unlock 解锁. 基于 lua 脚本实现操作原子性.
func (r *RedisLock) Unlock(ctx context.Context) error {
	defer func() {
		// 停止看门狗
		if r.stopDog != nil {
			r.stopDog()
		}
	}()

	keysAndArgs := []interface{}{r.getLockKey(), r.token}
	reply, err := r.client.Eval(ctx, LuaCheckAndDeleteDistributionLock, 1, keysAndArgs)
	if err != nil {
		return err
	}

	if ret, _ := reply.(int64); ret != 1 {
		return errors.New("can not unlock without ownership of lock")
	}

	return nil
}
