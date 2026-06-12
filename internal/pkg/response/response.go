package response

import (
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
	_ = c.Error(bizerrors.Ensure(err))
	c.Abort()
}

// Error is deprecated: use Abort. Kept so existing handlers delegate to middleware.
func Error(c *gin.Context, err error) {
	Abort(c, err)
}
