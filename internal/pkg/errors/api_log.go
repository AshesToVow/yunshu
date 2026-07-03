package errors

import (
	"context"

	logx "yunshu/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

// LogAPI 在 HTTP 边界记录业务错误（与前端 error_code 对齐，每条请求只记一次）。
func (e *BizError) LogAPI(ctx context.Context, level string, c *gin.Context) {
	if e == nil || e.logged {
		return
	}
	attrs := []any{
		"error_code", e.ErrorCode,
		"reason", e.Reason,
		"http_status", e.HTTPStatus(),
		"component", "api",
	}
	if e.Operation != "" {
		attrs = append(attrs, "operation", e.Operation)
	}
	if e.Component != "" {
		attrs = append(attrs, "service", e.Component)
	}
	if c != nil {
		attrs = append(attrs, "method", c.Request.Method)
		if route := c.FullPath(); route != "" {
			attrs = append(attrs, "path", route)
		} else if c.Request.URL != nil {
			attrs = append(attrs, "path", c.Request.URL.Path)
		}
	}
	if e.Cause != nil {
		attrs = append(attrs, "error", e.Cause)
	}
	log := logx.With(ctx)
	if level == "warn" {
		log.Warn("API request rejected", attrs...)
	} else {
		log.Error("API request failed", attrs...)
	}
	e.logged = true
}
