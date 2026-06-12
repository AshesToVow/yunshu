package errors

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// LogAPI logs HTTP API layer errors.
func (e *BizError) LogAPI(ctx context.Context, level string, c *gin.Context) {
	if e == nil || e.logged {
		return
	}
	log := slog.Default().With("layer", "api", "component", "http.api")
	attrs := []any{"error_code", e.ErrorCode, "reason", e.Reason, "http_status", e.HTTPStatus()}
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
	if level == "warn" {
		log.Warn("API request rejected", attrs...)
	} else {
		log.Error("API request failed", attrs...)
	}
	e.logged = true
}
