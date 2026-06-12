package logutil

import (
	"context"

	logx "yunshu/internal/pkg/logger"
)

// WithRequestID delegates to logger.WithRequestID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return logx.WithRequestID(ctx, requestID)
}

// WithUser delegates to logger.WithUser.
func WithUser(ctx context.Context, userID uint, username string) context.Context {
	return logx.WithUser(ctx, userID, username)
}

func contextAttrs(ctx context.Context) []any {
	return logx.ContextAttrs(ctx)
}
