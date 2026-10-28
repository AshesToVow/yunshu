package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CasbinPolicyChannel 策略变更广播频道。多副本部署时，写库副本发布、其余副本收到后重载。
const CasbinPolicyChannel = "yunshu:casbin:policy"

const (
	// policyReloadDebounce 合并短时间内的多次变更，避免批量授权触发多次全量重载。
	policyReloadDebounce = 500 * time.Millisecond
	// policySubscribeRetry 订阅断开后的重连间隔。
	policySubscribeRetry = 3 * time.Second
)

// PublishCasbinPolicyChange 广播一次策略变更。source 为发布副本标识，订阅端据此忽略自身消息。
func PublishCasbinPolicyChange(ctx context.Context, rdb *redis.Client, source string) error {
	if rdb == nil {
		return ErrRedisRequired
	}
	if err := rdb.Publish(ctx, CasbinPolicyChannel, strings.TrimSpace(source)).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	return nil
}

// WatchCasbinPolicyChanges 订阅策略变更并调用 reload 重载策略，返回停止函数。
// Redis 未配置或 reload 为空时退化为空操作（调用方仍可依赖定时轮询兜底）。
func WatchCasbinPolicyChanges(ctx context.Context, rdb *redis.Client, log *slog.Logger, source string, reload func() error) func() {
	if rdb == nil || reload == nil {
		return func() {}
	}
	if log == nil {
		log = slog.Default()
	}
	source = strings.TrimSpace(source)

	watchCtx, cancel := context.WithCancel(ctx)
	go func() {
		for watchCtx.Err() == nil {
			sub := rdb.Subscribe(watchCtx, CasbinPolicyChannel)
			ch := sub.Channel()
			timer := time.NewTimer(policyReloadDebounce)
			timer.Stop()
			pending := false

		drain:
			for {
				select {
				case <-watchCtx.Done():
					_ = sub.Close()
					_ = timer.Stop()
					return
				case msg, ok := <-ch:
					if !ok {
						_ = sub.Close()
						_ = timer.Stop()
						break drain
					}
					if source != "" && strings.TrimSpace(msg.Payload) == source {
						continue
					}
					if !pending {
						pending = true
						timer.Reset(policyReloadDebounce)
					}
				case <-timer.C:
					pending = false
					if err := reload(); err != nil {
						log.Error("casbin policy reload from push failed", "error", err)
						continue
					}
					log.Info("casbin policy reloaded from push")
				}
			}

			select {
			case <-watchCtx.Done():
				return
			case <-time.After(policySubscribeRetry):
			}
		}
	}()
	return cancel
}
