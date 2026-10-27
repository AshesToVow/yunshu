package logplatform

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const logCronLockMinRenewInterval = 5 * time.Second

var logCronWorkerID = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

var luaRenewLogCronLock = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('EXPIRE', KEYS[1], ARGV[2])
end
return 0
`)

var luaReleaseLogCronLock = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

// acquireLogCronLease 获取跨副本任务租约并在持有期间自动续期，release 仅删除自己持有的锁。
// Redis 不可用时退化为单实例语义（恒持有），重叠由调用方进程内标记拦截。
func acquireLogCronLease(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) (func(), bool) {
	if rdb == nil {
		return func() {}, true
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	token := logCronWorkerID + ":" + key
	ok, err := rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		slog.Default().With("component", "logcron").Warn("acquire cron lease failed", "key", key, "error", err)
		return nil, false
	}
	if !ok {
		return nil, false
	}
	stop := make(chan struct{})
	go func() {
		interval := ttl / 3
		if interval < logCronLockMinRenewInterval {
			interval = logCronLockMinRenewInterval
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_ = luaRenewLogCronLock.Run(renewCtx, rdb, []string{key}, token, int(ttl.Seconds())).Err()
				cancel()
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = luaReleaseLogCronLock.Run(releaseCtx, rdb, []string{key}, token).Err()
		})
	}, true
}

// runLogCronTask 同步执行任务：同一实例内跳过重叠轮次，多副本通过 Redis 租约只跑一份。
func runLogCronTask(ctx context.Context, rdb *redis.Client, running *atomic.Bool, key string, ttl time.Duration, job func()) bool {
	if !running.CompareAndSwap(false, true) {
		slog.Default().With("component", "logcron").Info("skip overlapping run", "task", key)
		return false
	}
	defer running.Store(false)
	release, ok := acquireLogCronLease(ctx, rdb, key, ttl)
	if !ok {
		slog.Default().With("component", "logcron").Info("skip run: lease held by another replica", "task", key)
		return false
	}
	defer release()
	job()
	return true
}
