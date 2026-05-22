package apperror

import bizerrors "yunshu/internal/pkg/errors"

// MarkLogged marks the error as logged at the service layer.
func MarkLogged(err error) error {
	return bizerrors.MarkLogged(err)
}

// AlreadyLogged reports whether the error was already logged.
func AlreadyLogged(err error) bool {
	return bizerrors.IsAlreadyLogged(err)
}
