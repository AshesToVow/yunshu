package apperror

import bizerrors "yunshu/internal/pkg/errors"

// NewBiz constructs a business error (prefer constants.BizError).
func NewBiz(httpStatus, bizCode int, reason, message string) error {
	return bizerrors.NewBiz(httpStatus, bizCode, reason, message)
}

// WithMetadata attaches metadata to a BizError.
func WithMetadata(err error, md map[string]any) error {
	biz, ok := IsAppError(err)
	if !ok {
		return err
	}
	return biz.WithMetadata(md)
}
