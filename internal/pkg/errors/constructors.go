package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"log/slog"
)

func NotFound(resource string, id any) *BizError {
	return &BizError{
		Code: 40401, Message: fmt.Sprintf("%s [%v] not found", resource, id),
		Reason: "NotFound", ErrorCode: "40401", StatusCode: http.StatusNotFound,
	}
}

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

func Pass(ctx context.Context, component, operation string, err error) error {
	if err == nil {
		return nil
	}
	var biz *BizError
	if stderrors.As(err, &biz) {
		if biz.logged || IsAlreadyLogged(err) {
			return err
		}
		cp := *biz
		cp.Component, cp.Operation = component, operation
		cp.logBiz(ctx, "error")
		return MarkLogged(&cp)
	}
	biz = &BizError{
		Code: 50001, Message: "operation failed", Reason: "InternalError", ErrorCode: "50001",
		Cause: err, StatusCode: http.StatusInternalServerError, Operation: operation, Component: component,
	}
	biz.logBiz(ctx, "error")
	return MarkLogged(biz)
}

func Reject(ctx context.Context, component, operation string, err error) error {
	if err == nil {
		return nil
	}
	var biz *BizError
	if stderrors.As(err, &biz) {
		if !biz.logged {
			cp := *biz
			cp.Component, cp.Operation = component, operation
			cp.logBiz(ctx, "warn")
			return MarkLogged(&cp)
		}
		return err
	}
	biz = &BizError{
		Code: 40001, Message: err.Error(), Reason: "BadRequest", ErrorCode: "40001",
		Cause: err, StatusCode: http.StatusBadRequest, Operation: operation, Component: component,
	}
	biz.logBiz(ctx, "warn")
	return MarkLogged(biz)
}

func Warn(ctx context.Context, component, operation, msg string) {
	slog.Default().Warn(msg, "component", component, "operation", operation)
}
