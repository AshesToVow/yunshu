package server

import (
	"context"
	"log/slog"
	"net"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
)

type RuntimeServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func Start(addr string, impl *LogPlatformServer, internalToken string, maxRecvBytes, maxSendBytes int) (*RuntimeServer, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(internalAuthUnaryInterceptor(internalToken), unaryLogInterceptor),
	}
	if maxRecvBytes > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(maxRecvBytes))
	}
	if maxSendBytes > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(maxSendBytes))
	}
	s := grpc.NewServer(opts...)
	pb.RegisterProjectServerServiceServer(s, impl)
	pb.RegisterLogSourceServiceServer(s, impl)
	pb.RegisterAgentRuntimeServiceServer(s, impl)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().With("component", "grpc").Error("serve panic", "recover", r)
			}
		}()
		_ = s.Serve(lis)
	}()
	return &RuntimeServer{grpcServer: s, listener: lis}, nil
}

func (s *RuntimeServer) Stop(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().With("component", "grpc").Error("graceful stop panic", "recover", r)
			}
		}()
		s.grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
	case <-done:
	}
}
