package server

import (
	"context"
	"net/http"

	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/logutil"
)

func logGRPCError(ctx context.Context, method string, err error) {
	if err == nil || bizerrors.IsAlreadyLogged(err) {
		return
	}
	log := logutil.GRPCCtx(ctx, "grpc.server")
	biz, ok := bizerrors.As(bizerrors.Ensure(err))
	if !ok {
		log.Error("gRPC request failed", "method", method, "error", err)
		return
	}
	attrs := []any{"method", method, "error_code", biz.ErrorCode, "reason", biz.Reason, "http_status", biz.HTTPStatus()}
	if biz.Cause != nil {
		attrs = append(attrs, "error", biz.Cause)
	}
	if biz.HTTPStatus() >= http.StatusInternalServerError {
		log.Error("gRPC request failed", attrs...)
		return
	}
	log.Warn("gRPC request rejected", attrs...)
}
