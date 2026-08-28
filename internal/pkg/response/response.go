package response

import (
	"context"
	"net/http"

	bizerrors "yunshu/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorBody struct {
	Code      int            `json:"code"`
	Reason    string         `json:"reason"`
	Message   string         `json:"message"`
	ErrorCode string         `json:"error_code,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: http.StatusOK, Message: "success", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: http.StatusCreated, Message: "success", Data: data})
}

// Abort submits err to ErrorHandler middleware (preferred).
func Abort(c *gin.Context, err error) {
	if err == nil || c == nil {
		return
	}
	// 使用请求链路 ctx 归一化错误，兜底 internal error 日志才能携带 request_id / user
	_ = c.Error(bizerrors.EnsureCtx(RequestContext(c), err))
	c.Abort()
}

// RequestContext 返回请求链路 ctx；c.Request 为空（单测手工构造 gin.Context）时回退到 Background。
func RequestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}

// Error is deprecated: use Abort. Kept so existing handlers delegate to middleware.
func Error(c *gin.Context, err error) {
	Abort(c, err)
}
