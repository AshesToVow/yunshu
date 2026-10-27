package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var luaRenewAlertLease = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('EXPIRE', KEYS[1], ARGV[2])
end
return 0
`)

// acquireAlertLease 获取带自动续期的分布式租约；release 幂等且仅删除自己持有的锁。
// 无 Redis（单机）时返回空 token 与空 release，视为恒持有。
func (s *AlertService) acquireAlertLease(ctx context.Context, key string, ttl time.Duration) (string, bool, func()) {
	if s.redis == nil {
		return "", true, func() {}
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	token := fmt.Sprintf("%s-%s", s.webhookWorkerID, key)
	ok, err := s.redis.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !ok {
		return "", false, func() {}
	}
	stop := make(chan struct{})
	go func() {
		interval := ttl / 3
		if interval < 5*time.Second {
			interval = 5 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_ = luaRenewAlertLease.Run(relCtx, s.redis, []string{key}, token, int(ttl.Seconds())).Err()
				cancel()
			}
		}
	}()
	var once sync.Once
	release := func() {
		once.Do(func() {
			close(stop)
			relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = luaReleaseAlertWebhookLease.Run(relCtx, s.redis, []string{key}, token).Err()
		})
	}
	return token, true, release
}

func (s *AlertService) releaseAlertLease(key, token string) {
	if s.redis == nil || token == "" {
		return
	}
	relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = luaReleaseAlertWebhookLease.Run(relCtx, s.redis, []string{key}, token).Err()
}
