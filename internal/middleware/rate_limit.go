package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type memoryRegisterLimiter struct {
	mu        sync.Mutex
	ipHits    map[string][]time.Time
	emailHits map[string][]time.Time
	bans      map[string]time.Time
}

func newMemoryRegisterLimiter() *memoryRegisterLimiter {
	return &memoryRegisterLimiter{
		ipHits:    make(map[string][]time.Time),
		emailHits: make(map[string][]time.Time),
		bans:      make(map[string]time.Time),
	}
}

func (m *memoryRegisterLimiter) pruneLocked(now time.Time) {
	for k, until := range m.bans {
		if now.After(until) {
			delete(m.bans, k)
		}
	}
	trim := func(hits map[string][]time.Time, window time.Duration) {
		for k, ts := range hits {
			kept := ts[:0]
			for _, t := range ts {
				if now.Sub(t) <= window {
					kept = append(kept, t)
				}
			}
			if len(kept) == 0 {
				delete(hits, k)
			} else {
				hits[k] = kept
			}
		}
	}
	trim(m.ipHits, time.Minute)
	trim(m.emailHits, time.Hour)
}

func (m *memoryRegisterLimiter) allowIP(ip string, limit int, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(now)
	if until, ok := m.bans["ip:"+ip]; ok && now.Before(until) {
		return false
	}
	m.ipHits[ip] = append(m.ipHits[ip], now)
	if len(m.ipHits[ip]) > limit {
		m.bans["ip:"+ip] = now.Add(30 * time.Minute)
		delete(m.ipHits, ip)
		return false
	}
	return true
}

func (m *memoryRegisterLimiter) allowEmail(email string, limit int, now time.Time) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(now)
	m.emailHits[email] = append(m.emailHits[email], now)
	return len(m.emailHits[email]) <= limit
}

var defaultMemoryRegisterLimiter = newMemoryRegisterLimiter()

// RegistrationRateLimit applies simple Redis-backed limits on registration attempts.
// - IP level: 20 attempts per minute, then temporary ban 30 minutes
// - Email/username level: 3 attempts per hour
// Redis 不可用时回退到进程内限流（单实例有效）。
func RegistrationRateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			applyMemoryRegistrationRateLimit(c, defaultMemoryRegisterLimiter)
			return
		}
		ctx := context.Background()

		// Check if IP is currently banned
		ip := c.ClientIP()
		banKey := fmt.Sprintf("ban:ip:%s", ip)
		if rdb.Exists(ctx, banKey).Val() > 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "当前 IP 因频繁请求已被临时封禁，请稍后再试"})
			return
		}

		// IP rate limit: 20 per minute
		ipKey := fmt.Sprintf("rl:register:ip:%s", ip)
		ipLimit := int64(20)
		if n, err := rdb.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				rdb.Expire(ctx, ipKey, time.Minute)
			}
			if n > ipLimit {
				// set a temporary ban
				rdb.Set(ctx, banKey, "1", 30*time.Minute)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "当前 IP 注册尝试过于频繁，请稍后再试"})
				return
			}
		}

		payload := readRegisterPayload(c)
		if payload.Email != "" {
			emailKey := fmt.Sprintf("rl:register:email:%s", strings.ToLower(strings.TrimSpace(payload.Email)))
			emailLimit := int64(3)
			if n, err := rdb.Incr(ctx, emailKey).Result(); err == nil {
				if n == 1 {
					rdb.Expire(ctx, emailKey, time.Hour)
				}
				if n > emailLimit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "该邮箱注册尝试次数过多，请稍后再试"})
					return
				}
			}
		}

		c.Next()
	}
}

func applyMemoryRegistrationRateLimit(c *gin.Context, limiter *memoryRegisterLimiter) {
	if limiter == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "注册限流暂不可用，请稍后再试"})
		return
	}
	now := time.Now()
	ip := c.ClientIP()
	if !limiter.allowIP(ip, 20, now) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "当前 IP 注册尝试过于频繁，请稍后再试"})
		return
	}
	payload := readRegisterPayload(c)
	if payload.Email != "" && !limiter.allowEmail(payload.Email, 3, now) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "该邮箱注册尝试次数过多，请稍后再试"})
		return
	}
	c.Next()
}

// AgentPublicRegisterRateLimit limits public agent self-registration attempts per IP.
func AgentPublicRegisterRateLimit(rdb *redis.Client) gin.HandlerFunc {
	const ipLimit = 30
	return func(c *gin.Context) {
		if rdb == nil {
			if !defaultMemoryRegisterLimiter.allowIP(c.ClientIP(), ipLimit, time.Now()) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Agent 注册尝试过于频繁，请稍后再试"})
				return
			}
			c.Next()
			return
		}
		ctx := context.Background()
		ip := c.ClientIP()
		banKey := fmt.Sprintf("ban:agent-register:ip:%s", ip)
		if rdb.Exists(ctx, banKey).Val() > 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "当前 IP 因频繁 Agent 注册已被临时封禁，请稍后再试"})
			return
		}
		ipKey := fmt.Sprintf("rl:agent-register:ip:%s", ip)
		if n, err := rdb.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				rdb.Expire(ctx, ipKey, time.Minute)
			}
			if n > ipLimit {
				rdb.Set(ctx, banKey, "1", 15*time.Minute)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Agent 注册尝试过于频繁，请稍后再试"})
				return
			}
		}
		c.Next()
	}
}

func readRegisterPayload(c *gin.Context) struct {
	Email    string `json:"email" form:"email"`
	Username string `json:"username" form:"username"`
} {
	var payload struct {
		Email    string `json:"email" form:"email"`
		Username string `json:"username" form:"username"`
	}
	if c.Request != nil && c.Request.Body != nil {
		if bodyBytes, err := io.ReadAll(c.Request.Body); err == nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			_ = json.Unmarshal(bodyBytes, &payload)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}
	return payload
}
