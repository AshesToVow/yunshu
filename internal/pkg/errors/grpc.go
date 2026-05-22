package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCStatus maps BizError (or any error via Ensure) to gRPC status for a unified error path with HTTP.
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}
	biz, ok := As(Ensure(err))
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}
	code := grpcCodeForHTTP(biz.HTTPStatus())
	if biz.Message != "" {
		return status.Error(code, biz.Message)
	}
	return status.Error(code, biz.Error())
}

func grpcCodeForHTTP(httpStatus int) codes.Code {
	switch httpStatus {
	case 400:
		return codes.InvalidArgument
	case 401:
		return codes.Unauthenticated
	case 403:
		return codes.PermissionDenied
	case 404:
		return codes.NotFound
	case 409:
		return codes.AlreadyExists
	case 429:
		return codes.ResourceExhausted
	default:
		return codes.Internal
	}
}
