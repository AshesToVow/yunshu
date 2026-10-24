package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
)

const idempotencyHeader = "Idempotency-Key"

const (
	idemKeyPrefix   = "idem:"
	idemInflightVal = "INFLIGHT"
	idemDonePrefix  = "DONE:"
	idemInflightTTL = 5 * time.Minute
	idemDoneTTL     = 24 * time.Hour
	idemMaxKeyLen   = 128
	idemMaxBodySize = 512 * 1024
)

// Idempotency 对携带 Idempotency-Key 头的写请求（POST/PUT/PATCH）提供幂等保护：
//   - 未携带该头或非写方法：完全透传，不影响存量客户端；
//   - 首次请求占位后正常执行，成功响应（<400）缓存 24h，同 Key 重试直接重放响应；
//   - 处理中（占位未完成）的同 Key 重复请求返回 409；失败响应释放 Key 允许重试。
//
// Redis 不可用时回退进程内存储（单实例有效）。SSE/WebSocket 等流式响应不缓存。
func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	var store idemStore = newMemoryIdemStore()
	if rdb != nil {
		store = &redisIdemStore{rdb: rdb}
	}
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader(idempotencyHeader))
		if key == "" || !isIdemWriteMethod(c.Request.Method) {
			c.Next()
			return
		}
		if len(key) > idemMaxKeyLen {
			response.Error(c, constants.ErrBadRequestWithMsg("Idempotency-Key 长度不能超过 128 字符"))
			c.Abort()
			return
		}
		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			c.Next()
			return
		}
		id := idemStorageID(c, key)
		ctx := c.Request.Context()

		state, err := store.state(ctx, id)
		if err != nil {
			// 存储不可用：失败开放（可用性优先），避免 Redis 故障阻断写操作
			c.Next()
			return
		}
		if state == idemInflightVal {
			response.Error(c, constants.ErrIdempotencyConflict)
			c.Abort()
			return
		}
		if strings.HasPrefix(state, idemDonePrefix) {
			replayIdemResponse(c, state[len(idemDonePrefix):])
			c.Abort()
			return
		}

		ok, err := store.tryAcquire(ctx, id, idemInflightTTL)
		if err != nil || !ok {
			response.Error(c, constants.ErrIdempotencyConflict)
			c.Abort()
			return
		}

		captured := &idemCaptureWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = captured
		c.Next()

		status := c.Writer.Status()
		ct := c.Writer.Header().Get("Content-Type")
		if status >= 400 || strings.Contains(ct, "text/event-stream") || captured.body.Len() > idemMaxBodySize {
			_ = store.release(ctx, id)
			return
		}
		_ = store.complete(ctx, id, []byte(strconv.Itoa(status)+":"+captured.body.String()), idemDoneTTL)
	}
}

func isIdemWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func idemStorageID(c *gin.Context, key string) string {
	h := sha256.Sum256([]byte(c.Request.Method + "|" + c.Request.URL.Path + "|" + key))
	return idemKeyPrefix + hex.EncodeToString(h[:])
}

func replayIdemResponse(c *gin.Context, payload string) {
	status := http.StatusOK
	body := payload
	if i := strings.Index(payload, ":"); i > 0 {
		if s, err := strconv.Atoi(payload[:i]); err == nil && s >= 200 && s < 400 {
			status = s
			body = payload[i+1:]
		}
	}
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.Header().Set("X-Idempotency-Replay", "true")
	c.Status(status)
	_, _ = c.Writer.WriteString(body)
}

type idemStore interface {
	tryAcquire(ctx context.Context, id string, ttl time.Duration) (bool, error)
	state(ctx context.Context, id string) (string, error)
	complete(ctx context.Context, id string, body []byte, ttl time.Duration) error
	release(ctx context.Context, id string) error
}

type redisIdemStore struct{ rdb *redis.Client }

func (s *redisIdemStore) tryAcquire(ctx context.Context, id string, ttl time.Duration) (bool, error) {
	ok, err := s.rdb.SetNX(ctx, id, idemInflightVal, ttl).Result()
	return ok, err
}

func (s *redisIdemStore) state(ctx context.Context, id string) (string, error) {
	return s.rdb.Get(ctx, id).Result()
}

func (s *redisIdemStore) complete(ctx context.Context, id string, body []byte, ttl time.Duration) error {
	return s.rdb.Set(ctx, id, idemDonePrefix+string(body), ttl).Err()
}

func (s *redisIdemStore) release(ctx context.Context, id string) error {
	return s.rdb.Del(ctx, id).Err()
}

type idemEntry struct {
	val     string
	expires time.Time
}

type memoryIdemStore struct {
	mu      sync.Mutex
	entries map[string]idemEntry
}

func newMemoryIdemStore() *memoryIdemStore {
	return &memoryIdemStore{entries: make(map[string]idemEntry)}
}

func (s *memoryIdemStore) pruneLocked(now time.Time) {
	for k, e := range s.entries {
		if now.After(e.expires) {
			delete(s.entries, k)
		}
	}
}

func (s *memoryIdemStore) tryAcquire(_ context.Context, id string, ttl time.Duration) (bool, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	if _, ok := s.entries[id]; ok {
		return false, nil
	}
	s.entries[id] = idemEntry{val: idemInflightVal, expires: now.Add(ttl)}
	return true, nil
}

func (s *memoryIdemStore) state(_ context.Context, id string) (string, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok || now.After(e.expires) {
		delete(s.entries, id)
		return "", nil
	}
	return e.val, nil
}

func (s *memoryIdemStore) complete(_ context.Context, id string, body []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[id] = idemEntry{val: idemDonePrefix + string(body), expires: time.Now().Add(ttl)}
	return nil
}

func (s *memoryIdemStore) release(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
	return nil
}

type idemCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *idemCaptureWriter) Write(b []byte) (int, error) {
	if w.body.Len() < idemMaxBodySize {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *idemCaptureWriter) WriteString(s string) (int, error) {
	if w.body.Len() < idemMaxBodySize {
		w.body.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}
