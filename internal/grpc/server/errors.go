package server

import (
	bizerrors "yunshu/internal/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusErr(err error) error {
	if err == nil {
		return nil
	}
	if biz, ok := bizerrors.As(err); ok {
		switch biz.HTTPStatus() {
		case 400:
			return status.Error(codes.InvalidArgument, biz.Message)
		case 401:
			return status.Error(codes.Unauthenticated, biz.Message)
		case 403:
			return status.Error(codes.PermissionDenied, biz.Message)
		case 404:
			return status.Error(codes.NotFound, biz.Message)
		case 409:
			return status.Error(codes.AlreadyExists, biz.Message)
		default:
			return status.Error(codes.Internal, biz.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
