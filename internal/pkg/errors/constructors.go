package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	logx "yunshu/internal/pkg/logger"
)

func Internal(err error, operation string) *BizError {
	b := &BizError{
		Code: 50001, Message: "internal server error", Reason: "InternalError",
		ErrorCode: "50001", Cause: err, StatusCode: http.StatusInternalServerError, Operation: operation,
	}
	b.logBiz(context.Background(), "error")
	return b
}

func InternalCtx(ctx context.Context, err error, operation string) *BizError {
	b := &BizError{
		Code: 50001, Message: "internal server error", Reason: "InternalError",
		ErrorCode: "50001", Cause: err, StatusCode: http.StatusInternalServerError, Operation: operation,
	}
	b.logBiz(ctx, "error")
	return b
}

func InferHTTPStatus(code int) int {
	if code >= 10001 && code <= 10999 {
		switch code {
		case 10002, 10008, 10009, 10010, 10011, 10013, 10014:
			return http.StatusUnauthorized
		case 10003, 10012:
			return http.StatusForbidden
		case 10004:
			return http.StatusNotFound
		case 10005:
			return http.StatusConflict
		case 10007:
			return http.StatusTooManyRequests
		case 10006, 10901:
			return http.StatusInternalServerError
		default:
			return http.StatusBadRequest
		}
	}
	switch {
	case code >= 40001 && code < 40100:
		return http.StatusBadRequest
	case code >= 40101 && code < 40200:
		return http.StatusUnauthorized
	case code >= 40301 && code < 40400:
		return http.StatusForbidden
	case code >= 40401 && code < 40500:
		return http.StatusNotFound
	case code >= 40901 && code < 41000:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// Pass 包装下层错误并附加 component/operation，不在 Service 层打日志（由 HTTP LogAPI 统一记录）。
func Pass(ctx context.Context, component, operation string, err error, attrs ...any) error {
	_ = attrs
	if err == nil {
		return nil
	}
	var biz *BizError
	if stderrors.As(err, &biz) {
		cp := *biz
		if component != "" {
			cp.Component = component
		}
		if operation != "" {
			cp.Operation = operation
		}
		return &cp
	}
	return &BizError{
		Code: 50001, Message: "operation failed", Reason: "InternalError", ErrorCode: "50001",
		Cause: err, StatusCode: http.StatusInternalServerError, Operation: operation, Component: component,
	}
}

// Reject 包装客户端错误，不在 Service 层打日志。
func Reject(ctx context.Context, component, operation string, err error, attrs ...any) error {
	_ = attrs
	if err == nil {
		return nil
	}
	var biz *BizError
	if stderrors.As(err, &biz) {
		cp := *biz
		if component != "" {
			cp.Component = component
		}
		if operation != "" {
			cp.Operation = operation
		}
		return &cp
	}
	return &BizError{
		Code: 40001, Message: err.Error(), Reason: "BadRequest", ErrorCode: "40001",
		Cause: err, StatusCode: http.StatusBadRequest, Operation: operation, Component: component,
	}
}

func Warn(ctx context.Context, component, operation, msg string, attrs ...any) {
	logx.With(ctx, "component", component, "operation", operation).Warn(msg, attrs...)
}

// InternalMsg logs and returns a 500 BizError (component + operation for structured logs).
func InternalMsg(ctx context.Context, component, operation, msg string, attrs ...any) error {
	_ = attrs
	return internalCtx(ctx, component, operation, fmt.Errorf("%s", msg))
}

// Internalf logs a wrapped error with component/operation (msgFmt should include %w or %v for err).
func Internalf(ctx context.Context, component, operation string, err error, msgFmt string, args ...any) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(msgFmt, append(args, err)...)
	return internalCtx(ctx, component, operation, fmt.Errorf("%s: %w", msg, err))
}

func internalCtx(ctx context.Context, component, operation string, cause error) error {
	b := &BizError{
		Code: 50001, Message: "internal server error", Reason: "InternalError",
		ErrorCode: "50001", Cause: cause, StatusCode: http.StatusInternalServerError,
		Operation: operation, Component: component,
	}
	b.logBiz(ctx, "error")
	return MarkLogged(b)
}
