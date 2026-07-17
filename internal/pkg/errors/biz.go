package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"

	logx "yunshu/internal/pkg/logger"
)

// BizError is the unified business error type (OneX: code, reason, message, error_code, metadata).
type BizError struct {
	Code       int            `json:"code"`
	Message    string         `json:"message"`
	Reason     string         `json:"reason"`
	ErrorCode  string         `json:"error_code"`
	Cause      error          `json:"-"`
	StatusCode int            `json:"-"`
	Operation  string         `json:"operation,omitempty"`
	Component  string         `json:"component,omitempty"`
	Attrs      map[string]any `json:"attrs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	logged     bool
}

func (e *BizError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("business error (code=%d)", e.Code)
}

func (e *BizError) Unwrap() error { return e.Cause }

// NewBiz constructs a business error (used by constants and services).
func NewBiz(httpStatus, bizCode int, reason, message string) *BizError {
	if reason == "" {
		reason = "Unknown"
	}
	return &BizError{
		Code:       bizCode,
		Message:    message,
		Reason:     reason,
		ErrorCode:  strconv.Itoa(bizCode),
		StatusCode: httpStatus,
	}
}

// WithMetadata returns a copy with metadata attached.
func (e *BizError) WithMetadata(md map[string]any) *BizError {
	if e == nil {
		return nil
	}
	cp := *e
	if len(md) == 0 {
		return &cp
	}
	cp.Metadata = make(map[string]any, len(md))
	for k, v := range md {
		cp.Metadata[k] = v
	}
	return &cp
}

// HTTPStatus returns the HTTP status code for this error.
func (e *BizError) HTTPStatus() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}
	return InferHTTPStatus(e.Code)
}

// Ensure normalizes any error to *BizError.
func Ensure(err error) error {
	if err == nil {
		return nil
	}
	var biz *BizError
	if stderrors.As(err, &biz) {
		return err
	}
	if IsAlreadyLogged(err) {
		if b, ok := As(stderrors.Unwrap(err)); ok {
			return b
		}
	}
	return Internal(err, "handler")
}

type loggedMarker struct{ cause error }

func (e *loggedMarker) Error() string { return e.cause.Error() }
func (e *loggedMarker) Unwrap() error { return e.cause }

// MarkLogged marks the error as already logged at the service layer.
func MarkLogged(err error) error {
	if err == nil {
		return nil
	}
	var m *loggedMarker
	if stderrors.As(err, &m) {
		return err
	}
	return &loggedMarker{cause: err}
}

// IsAlreadyLogged reports whether the error was already logged.
func IsAlreadyLogged(err error) bool {
	var m *loggedMarker
	return stderrors.As(err, &m)
}

func (e *BizError) logBiz(ctx context.Context, level string) {
	if e == nil || e.logged {
		return
	}
	attrs := []any{
		"error_code", e.ErrorCode,
		"reason", e.Reason,
		"http_status", e.HTTPStatus(),
	}
	if e.Operation != "" {
		attrs = append(attrs, "operation", e.Operation)
	}
	if e.Component != "" {
		attrs = append(attrs, "component", e.Component)
	}
	if e.Cause != nil {
		attrs = append(attrs, "error", e.Cause)
	}
	log := logx.With(ctx)
	if level == "warn" {
		log.Warn(e.Message, attrs...)
	} else {
		log.Error(e.Message, attrs...)
	}
	e.logged = true
}
