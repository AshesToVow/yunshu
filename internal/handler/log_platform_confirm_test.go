package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newConfirmTestContext(t *testing.T, target, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	return c
}

func TestRequireDestructiveConfirm(t *testing.T) {
	cases := []struct {
		name       string
		target     string
		url        string
		body       string
		wantOK     bool
		wantReason string
	}{
		{
			name:       "JSON 精确匹配",
			target:     "yunshu-agent-1-2026.10.26",
			url:        "/api/v1/log-platform/es-indices/yunshu-agent-1-2026.10.26",
			body:       `{"confirm":"yunshu-agent-1-2026.10.26","reason":"过期清理"}`,
			wantOK:     true,
			wantReason: "过期清理",
		},
		{
			name:       "query 回退",
			target:     "yunshu-k8s-c1-p1-2026.10.26",
			url:        "/api/v1/log-platform/kafka-topics/x?confirm=yunshu-k8s-c1-p1-2026.10.26&reason=manual",
			wantOK:     true,
			wantReason: "manual",
		},
		{
			name:   "confirm 不匹配",
			target: "idx-a",
			url:    "/x",
			body:   `{"confirm":"idx-b","reason":"manual"}`,
		},
		{
			name:   "缺少原因",
			target: "idx-a",
			url:    "/x",
			body:   `{"confirm":"idx-a","reason":"  "}`,
		},
		{
			name:   "JSON 格式错误",
			target: "idx-a",
			url:    "/x",
			body:   `{"confirm":`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newConfirmTestContext(t, tc.url, tc.body)
			reason, ok := requireDestructiveConfirm(c, tc.target, "索引")
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if ok && reason != tc.wantReason {
				t.Fatalf("reason=%q want %q", reason, tc.wantReason)
			}
			if !ok && len(c.Errors) == 0 {
				t.Fatal("expected error recorded on context")
			}
		})
	}
}

func TestRequireCleanupConfirm(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		body       string
		wantOK     bool
		wantReason string
	}{
		{
			name:       "cleanup 大小写不敏感",
			url:        "/x",
			body:       `{"confirm":"CleanUp","reason":"手动清理"}`,
			wantOK:     true,
			wantReason: "手动清理",
		},
		{
			name: "错误关键字",
			url:  "/x",
			body: `{"confirm":"delete","reason":"手动清理"}`,
		},
		{
			name: "缺少原因",
			url:  "/x",
			body: `{"confirm":"cleanup"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newConfirmTestContext(t, tc.url, tc.body)
			reason, ok := requireCleanupConfirm(c)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if ok && reason != tc.wantReason {
				t.Fatalf("reason=%q want %q", reason, tc.wantReason)
			}
			if !ok && len(c.Errors) == 0 {
				t.Fatal("expected error recorded on context")
			}
		})
	}
}
