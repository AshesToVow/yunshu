package logger

import (
	"context"
	"log/slog"
)

// routeHandler 将 slog 记录按级别分流：Info/Warn → info.log，Error+ → error.log。
type routeHandler struct {
	info slog.Handler
	err  slog.Handler
}

func (h *routeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level >= slog.LevelError {
		return h.err.Enabled(ctx, level)
	}
	return h.info.Enabled(ctx, level)
}

func (h *routeHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return h.err.Handle(ctx, r)
	}
	return h.info.Handle(ctx, r)
}

func (h *routeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &routeHandler{info: h.info.WithAttrs(attrs), err: h.err.WithAttrs(attrs)}
}

func (h *routeHandler) WithGroup(name string) slog.Handler {
	return &routeHandler{info: h.info.WithGroup(name), err: h.err.WithGroup(name)}
}
