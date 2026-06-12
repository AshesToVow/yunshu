package server

import bizerrors "yunshu/internal/pkg/errors"

func toStatusErr(err error) error {
	return bizerrors.ToGRPCStatus(err)
}
