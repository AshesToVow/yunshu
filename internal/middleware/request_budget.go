package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DefaultRequestBudget 常规请求的总时长预算。
// 外部组件（SMTP/ES/Prometheus 等）不可用时，ctx 预算让 handler 内所有
// 支持 context 的调用按时取消并释放 DB 连接，避免 goroutine 与连接池被拖死。
const DefaultRequestBudget = 60 * time.Second

// budgetExemptSuffixes 长耗时/流式端点不设预算（导出、SSE、WS、终端）。
var budgetExemptSuffixes = []string{
	"/chat/stream",
	"/logs/stream",
	"/stream",
	"/terminal/ws",
	"/export",
	"/export.csv",
}

func IsRequestBudgetExempt(path string) bool {
	for _, suffix := range budgetExemptSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// RequestBudget 为非流式请求附加 context 超时预算。
func RequestBudget(timeout time.Duration) gin.HandlerFunc {
	if timeout <= 0 {
		timeout = DefaultRequestBudget
	}
	return func(c *gin.Context) {
		if IsRequestBudgetExempt(c.Request.URL.Path) {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
