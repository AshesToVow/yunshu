package response

import (
	"net/http"

	bizerrors "yunshu/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

const ctxKeyBizErrorLogged = "biz_error_logged"

// LogHTTPError logs API errors once per request (called from ErrorHandler).
func LogHTTPError(c *gin.Context, err error) {
	if err == nil || c == nil || bizerrors.IsAlreadyLogged(err) {
		return
	}
	if _, ok := c.Get(ctxKeyBizErrorLogged); ok {
		return
	}
	c.Set(ctxKeyBizErrorLogged, true)
	biz, ok := bizerrors.As(bizerrors.Ensure(err))
	if !ok {
		return
	}
	level := "warn"
	if biz.HTTPStatus() >= http.StatusInternalServerError {
		level = "error"
	}
	biz.LogAPI(c.Request.Context(), level, c)
}
