package logutil

import (
	"context"
	"log/slog"
)

// Component is a structured logger with layer/component and optional context fields.
type Component struct {
	log *slog.Logger
}

func (c *Component) logger() *slog.Logger {
	if c == nil || c.log == nil {
		return Default()
	}
	return c.log
}

// W binds context fields (request_id, user_id, username) for subsequent log calls.
func (c *Component) W(ctx context.Context) *Component {
	if c == nil {
		return &Component{log: Ctx(ctx)}
	}
	attrs := contextAttrs(ctx)
	if len(attrs) == 0 {
		return c
	}
	return &Component{log: c.logger().With(attrs...)}
}

func (c *Component) Info(msg string, attrs ...any)  { c.logger().Info(msg, attrs...) }
func (c *Component) Warn(msg string, attrs ...any)  { c.logger().Warn(msg, attrs...) }
func (c *Component) Error(msg string, attrs ...any) { c.logger().Error(msg, attrs...) }
func (c *Component) Debug(msg string, attrs ...any) { c.logger().Debug(msg, attrs...) }

func (c *Component) Infow(msg string, keyvals ...any)  { c.Info(msg, keyvals...) }
func (c *Component) Warnw(msg string, keyvals ...any)  { c.Warn(msg, keyvals...) }
func (c *Component) Debugw(msg string, keyvals ...any) { c.Debug(msg, keyvals...) }

func (c *Component) Errorw(err error, msg string, keyvals ...any) {
	if err != nil {
		keyvals = append(keyvals, "error", err)
	}
	c.Error(msg, keyvals...)
}

// Service returns layer=service.
func Service(name string) *Component {
	return &Component{log: Default().With("layer", "service", "component", name)}
}

// ServiceCtx returns layer=service with context fields.
func ServiceCtx(ctx context.Context, name string) *Component {
	return Service(name).W(ctx)
}

// ServiceComponent wraps an existing slog.Logger as service layer.
func ServiceComponent(l *slog.Logger, name string) *Component {
	if l == nil {
		return Service(name)
	}
	return &Component{log: l.With("layer", "service", "component", name)}
}

// WorkerComponent wraps slog.Logger as worker layer.
func WorkerComponent(l *slog.Logger, name string) *Component {
	if l == nil {
		return Worker(name)
	}
	return &Component{log: l.With("layer", "worker", "component", name)}
}

// HTTPComponent wraps slog.Logger as http layer.
func HTTPComponent(l *slog.Logger, name string) *Component {
	if l == nil {
		return HTTP(name)
	}
	return &Component{log: l.With("layer", "http", "component", name)}
}
