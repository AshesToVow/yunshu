package apperror

import bizerrors "yunshu/internal/pkg/errors"

// AppError is deprecated; use errors.BizError.
type AppError = bizerrors.BizError

// IsAppError reports whether err is a BizError.
func IsAppError(err error) (*AppError, bool) {
	return bizerrors.As(err)
}
