package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const metadataInternalGRPCToken = "x-internal-grpc-token"

// internalAuthUnaryInterceptor 保护 Project/LogSource gRPC；AgentRuntime 沿用各自 token/register 校验。
func internalAuthUnaryInterceptor(internalToken string) grpc.UnaryServerInterceptor {
	token := strings.TrimSpace(internalToken)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if token == "" || isAgentRuntimeMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		if !internalTokenMatches(ctx, token) {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid internal grpc token")
		}
		return handler(ctx, req)
	}
}

func isAgentRuntimeMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/logplatform.v1.AgentRuntimeService/")
}

func internalTokenMatches(ctx context.Context, want string) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, key := range []string{metadataInternalGRPCToken, "authorization"} {
		for _, v := range md.Get(key) {
			v = strings.TrimSpace(v)
			if v == want {
				return true
			}
			if len(v) > 7 && strings.EqualFold(v[:7], "bearer ") {
				if strings.TrimSpace(v[7:]) == want {
					return true
				}
			}
		}
	}
	return false
}
