package handler

import (
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// abortService maps service-layer errors (BizError, constants.Err*, etc.) to middleware Abort.
func abortService(c *gin.Context, err error) {
	if err == nil || c == nil {
		return
	}
	response.Abort(c, bizerrors.Ensure(err))
}
