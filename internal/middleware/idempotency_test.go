package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func newIdemTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	var mu sync.Mutex
	hits := 0
	r.POST("/write", Idempotency(nil), func(c *gin.Context) {
		mu.Lock()
		hits++
		mu.Unlock()
		c.JSON(http.StatusCreated, gin.H{"code": 201, "message": "success", "data": map[string]int{"hits": hits}})
	})
	return r
}

func doIdemWrite(t *testing.T, r *gin.Engine, path, key string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(idempotencyHeader, key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIdempotencyReplaysCachedSuccess(t *testing.T) {
	r := newIdemTestEngine()

	first := doIdemWrite(t, r, "/write", "k-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", first.Code)
	}

	second := doIdemWrite(t, r, "/write", "k-1")
	if second.Code != http.StatusCreated {
		t.Fatalf("replay status = %d, want 201 (cached)", second.Code)
	}
	if second.Header().Get("X-Idempotency-Replay") != "true" {
		t.Fatalf("replay header missing: %v", second.Header())
	}
	if !strings.Contains(second.Body.String(), `"hits":1`) {
		t.Fatalf("replay body should be the cached first response, got %s", second.Body.String())
	}
}

func TestIdempotencyPassthroughWithoutKey(t *testing.T) {
	r := newIdemTestEngine()
	for i := 0; i < 3; i++ {
		if w := doIdemWrite(t, r, "/write", ""); w.Code != http.StatusCreated {
			t.Fatalf("request without key status = %d, want 201", w.Code)
		}
	}
	last := doIdemWrite(t, r, "/write", "")
	if !strings.Contains(last.Body.String(), `"hits":4`) {
		t.Fatalf("requests without key should not be deduped, got %s", last.Body.String())
	}
}

func TestIdempotencyRejectsInFlightDuplicate(t *testing.T) {
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.Use(ErrorHandler())
	acquired := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	r.POST("/slow", Idempotency(nil), func(c *gin.Context) {
		once.Do(func() { close(acquired) })
		<-release
		c.JSON(http.StatusOK, gin.H{"code": 200})
	})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- doIdemWrite(t, r, "/slow", "k-inflight")
	}()
	<-acquired

	dup := doIdemWrite(t, r, "/slow", "k-inflight")
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate in-flight status = %d, want 409", dup.Code)
	}
	if !strings.Contains(dup.Body.String(), "11004") {
		t.Fatalf("conflict body should carry error_code 11004, got %s", dup.Body.String())
	}

	close(release)
	if w := <-done; w.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", w.Code)
	}
}

func TestIdempotencyRejectsOverlongKey(t *testing.T) {
	r := newIdemTestEngine()
	w := doIdemWrite(t, r, "/write", strings.Repeat("x", 200))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("overlong key status = %d, want 400", w.Code)
	}
}

func TestIdempotencyReleasesKeyOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	calls := 0
	r.POST("/flaky", Idempotency(nil), func(c *gin.Context) {
		calls++
		if calls == 1 {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "boom"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
	})

	first := doIdemWrite(t, r, "/flaky", "k-retry")
	if first.Code != http.StatusInternalServerError {
		t.Fatalf("first status = %d, want 500", first.Code)
	}
	second := doIdemWrite(t, r, "/flaky", "k-retry")
	if second.Code != http.StatusOK {
		t.Fatalf("retry after failure status = %d, want 200 (key released)", second.Code)
	}
}
