package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestRegisterOpsEndpoints(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	middleware.RegisterOpsEndpoints(eng, nil, nil, time.Now())

	for _, path := range []string{"/livez", "/metrics"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		eng.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz without db should be 503, got %d", w.Code)
	}
}
