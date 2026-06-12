package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/response"
)

// ErrorHandler writes JSON for errors submitted via c.Error (see response.Abort).
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		err := bizerrors.Ensure(c.Errors.Last().Err)
		response.LogHTTPError(c, err)
		biz, ok := bizerrors.As(err)
		if !ok {
			c.JSON(http.StatusInternalServerError, response.ErrorBody{
				Code: http.StatusInternalServerError, Reason: "InternalError",
				Message: "internal server error", ErrorCode: "10006",
			})
			c.Abort()
			return
		}
		body := biz.ToBody()
		c.JSON(biz.HTTPStatus(), response.ErrorBody{
			Code: body.Code, Reason: body.Reason, Message: body.Message,
			ErrorCode: body.ErrorCode, Metadata: body.Metadata,
		})
		c.Abort()
	}
}

// AbortWithError writes an error response immediately (for auth middleware on route groups).
func AbortWithError(c *gin.Context, err error) {
	if c == nil || err == nil || c.Writer.Written() {
		return
	}
	err = bizerrors.Ensure(err)
	response.LogHTTPError(c, err)
	if biz, ok := bizerrors.As(err); ok {
		body := biz.ToBody()
		c.JSON(biz.HTTPStatus(), response.ErrorBody{
			Code: body.Code, Reason: body.Reason, Message: body.Message,
			ErrorCode: body.ErrorCode, Metadata: body.Metadata,
		})
	} else {
		c.JSON(http.StatusInternalServerError, response.ErrorBody{
			Code: http.StatusInternalServerError, Reason: "InternalError",
			Message: "internal server error", ErrorCode: "10006",
		})
	}
	c.Abort()
}
