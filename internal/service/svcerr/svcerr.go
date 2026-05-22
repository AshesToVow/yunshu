// Package svcerr wraps internal/pkg/errors for service-layer error logging.
package svcerr

import (
	"context"
	"fmt"

	bizerrors "yunshu/internal/pkg/errors"
)

func Internal(ctx context.Context, component, operation string, err error, msgFmt string, attrs ...any) error {
	_ = attrs
	if err == nil {
		return nil
	}
	return bizerrors.InternalCtx(ctx, err, operation+": "+fmt.Sprintf(msgFmt, err))
}

func Internalf(ctx context.Context, component, operation string, err error, msgFmt string, args ...any) error {
	if err == nil {
		return nil
	}
	return bizerrors.InternalCtx(ctx, err, operation+": "+fmt.Sprintf(msgFmt, append(args, err)...))
}

func InternalMsg(ctx context.Context, component, operation, msg string, attrs ...any) error {
	_ = attrs
	return bizerrors.InternalCtx(ctx, fmt.Errorf("%s", msg), operation+": "+msg)
}

func InternalFmt(ctx context.Context, component, operation, msgFmt string, args ...any) error {
	return InternalMsg(ctx, component, operation, fmt.Sprintf(msgFmt, args...))
}

func Warn(ctx context.Context, component, operation, msg string, attrs ...any) {
	_ = attrs
	bizerrors.Warn(ctx, component, operation, msg)
}

func Reject(ctx context.Context, component, operation string, err error, attrs ...any) error {
	_ = attrs
	return bizerrors.Reject(ctx, component, operation, err)
}

func Pass(ctx context.Context, component, operation string, err error, attrs ...any) error {
	_ = attrs
	return bizerrors.Pass(ctx, component, operation, err)
}
