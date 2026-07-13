package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Doctor 检查 gRPC 连通性与 Token 有效性（不触发 PublicRegister，避免轮换 Token）。
func Doctor(cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}
	if cfg.ServerID == 0 {
		return fmt.Errorf("server-id is required")
	}
	token := strings.TrimSpace(cfg.Token)
	tokenFile := effectiveTokenFile(cfg)
	if token == "" && tokenFile != "" {
		if saved, err := loadTokenFile(tokenFile); err == nil {
			token = saved
		}
	}
	if token == "" {
		if strings.TrimSpace(cfg.RegisterSecret) != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			conn, err := grpc.NewClient(cfg.GrpcServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return fmt.Errorf("grpc dial failed: %w", err)
			}
			defer conn.Close()
			fmt.Printf("OK grpc dial %s (register-secret present; use --token or --token-file to verify runtime-config)\n", cfg.GrpcServer)
			_ = ctx
			return nil
		}
		return fmt.Errorf("token, token-file, or register-secret is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(cfg.GrpcServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc dial failed: %w", err)
	}
	defer conn.Close()
	cli := pb.NewAgentRuntimeServiceClient(conn)

	bundle, err := fetchRuntimeConfig(ctx, cli, token)
	if err != nil {
		return fmt.Errorf("GetRuntimeConfig failed: %w", err)
	}
	fmt.Printf("OK runtime-config project_id=%d sources=%d discovery_roots=%d grpc=%s server_id=%d\n",
		bundle.ProjectID, len(bundle.Sources), len(bundle.Roots), cfg.GrpcServer, cfg.ServerID)
	for _, s := range bundle.Sources {
		fmt.Printf("  - source id=%d type=%s path=%s\n", s.LogSourceID, s.LogType, s.Path)
	}
	return nil
}
