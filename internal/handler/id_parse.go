package handler

import (
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"

	"github.com/gin-gonic/gin"
)

// parseUintParam parses a uint path parameter and returns a BadRequest app error on failure.
func parseUintParam(c *gin.Context, key string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, constants.ErrInvalidRequestParam
	}
	return uint(id), nil
}

func parseInt64Param(c *gin.Context, key string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, constants.ErrInvalidRequestParam
	}
	return id, nil
}

func parseUintQuery(c *gin.Context, key string) (uint, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, constants.ErrBadRequestWithMsg(key + " 必填")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, constants.ErrInvalidRequestParam
	}
	return uint(id), nil
}

// parseOptionalUintQuery 空字符串返回 0，不报错。
func parseOptionalUintQuery(c *gin.Context, key string) uint {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}
