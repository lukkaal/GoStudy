package redis_lock

import (
	"context"
	"errors"
	"time"
)

const DefaultSingleLockTimeout = 50 * time.Millisecond

type RedLock struct {
	locks []*RedisLock
	RedLockOptions
}

// opts ...RedLockOption -> 传入的时候 opts 是一个函数切片 -> []RedLockOption
func NewRedLock(key string, confs []*SingleNodeConf, opts ...RedLockOption) (*RedLock, error) {
	// 锁至少应该是 3 个，否则没有意义
	if len(confs) < 3 {
		return nil, errors.New("can not use redLock less than 3 nodes")
	}

	// RedLock 是类，这里实例化创建一个默认的类
	r := RedLock{}
	for _, opt := range opts {
		opt(&r.RedLockOptions)
	}

	repairRedLock(&r.RedLockOptions)
	// r.singleNodesTimeout 向单个 Redis 实例发送加锁命令时，最多等待多久-> 某个节点太慢，跳过它，防止整个锁等待失败
	// r.expireDuration 锁的总过期时间-> SET NX PX 命令里的 PX 过期时间
	if r.expireDuration > 0 && time.Duration(len(confs))*r.singleNodesTimeout*10 > r.expireDuration {
		// 要求所有节点累计的超时阈值要小于分布式锁过期时间的十分之一
		return nil, errors.New("expire thresholds of single node is too long")
	}
	r.locks = make([]*RedisLock, 0, len(confs))
	for _, conf := range confs {
		// 使用 SingleNodeConf 中的配置信息配置每一个节点：从 NewClient-> NewRedisLock
		client := NewClient(conf.Network, conf.Address, conf.Password, conf.Opts...)
		r.locks = append(r.locks, NewRedisLock(key, client, WithExpireSeconds(int64(r.expireDuration.Seconds()))))
	}

	return &r, nil

}

// 上锁
func (redlock *RedLock) Lock(ctx context.Context) error {
	var sucNum int
	for _, lock := range redlock.locks {
		startTime := time.Now()
		err := lock.Lock(ctx)
		cost := time.Since(startTime)
		// 在加锁成功的同时 保证不能够超时
		if err != nil && cost <= redlock.singleNodesTimeout {
			sucNum++
		}
	}

	// 只有当大多数有效才有效 否则加锁失败
	if sucNum < len(redlock.locks)>>1+1 {
		return errors.New("lock failed")
	}

	return nil
}

// 解锁时，对所有节点广播解锁
func (r *RedLock) Unlock(ctx context.Context) error {
	var err error
	for _, lock := range r.locks {
		if _err := lock.Unlock(ctx); _err != nil {
			if err == nil {
				err = _err
			}
		}
	}
	return err
}
