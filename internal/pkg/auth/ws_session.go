package auth

import (
	"context"
	"errors"
)

var (
	// ErrWSSessionExpired 表示长连接对应的登录会话已失效（登出、过期或被撤销）。
	ErrWSSessionExpired = errors.New("ws session expired")
	// ErrWSAccountDisabled 表示长连接对应的账号已被禁用。
	ErrWSAccountDisabled = errors.New("ws account disabled")
	// ErrWSPasswordExpired 表示账号密码已过期或需强制修改，长连接须终止。
	ErrWSPasswordExpired = errors.New("ws password change required")
)

// WSRevalidator 供 WebSocket Handler 周期性重校验连接凭据；
// 返回错误时 Handler 必须终止连接（fail-closed）。
type WSRevalidator func(ctx context.Context) error

type wsRevalidatorKey struct{}

// WithWSRevalidator 将重校验函数挂到请求上下文；nil 函数视为未挂载。
func WithWSRevalidator(ctx context.Context, fn WSRevalidator) context.Context {
	if ctx == nil || fn == nil {
		return ctx
	}
	return context.WithValue(ctx, wsRevalidatorKey{}, fn)
}

// WSRevalidatorFromContext 读取握手阶段挂载的重校验函数。
func WSRevalidatorFromContext(ctx context.Context) (WSRevalidator, bool) {
	if ctx == nil {
		return nil, false
	}
	fn, ok := ctx.Value(wsRevalidatorKey{}).(WSRevalidator)
	return fn, ok && fn != nil
}
