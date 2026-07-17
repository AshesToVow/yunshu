package logger

import (
	"context"
	"log/slog"
)

// With 返回带 context 字段（request_id、user 等）及额外 attrs 的 slog 记录器。
func With(ctx context.Context, attrs ...any) *slog.Logger {
	l := slog.Default()
	if ctx != nil {
		if a := ContextAttrs(ctx); len(a) > 0 {
			l = l.With(a...)
		}
	}
	if len(attrs) > 0 {
		l = l.With(attrs...)
	}
	return l
}
