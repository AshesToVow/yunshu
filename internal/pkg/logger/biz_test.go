package logger

import (
	"errors"
	"net/http"
	"testing"

	"yunshu/internal/config"
	bizerrors "yunshu/internal/pkg/errors"
)

func TestBizOpInternalVsClient(t *testing.T) {
	SetDefault(New(config.LogConfig{Level: "debug", Output: "console"}))
	b := Biz("test")

	b.Op("noop", nil)
	b.Op("client", bizerrors.NewBiz(http.StatusBadRequest, 11020, "BadRequest", "bad"))
	b.Op("server", bizerrors.NewBiz(http.StatusInternalServerError, 50001, "InternalError", "boom"))
}

func TestBizDisabledWithoutDefault(t *testing.T) {
	SetDefault(nil)
	b := Biz("test")
	b.Error("should not panic")
	b.Op("x", errors.New("e"))
}
