package logger

import "context"

// ContextAttrs returns slog key-value pairs extracted from ctx (request_id, user, etc.).
func ContextAttrs(ctx context.Context) []any {
	return attrsFromContext(ctx)
}
