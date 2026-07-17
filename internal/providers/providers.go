package providers

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"yunshu/internal/config"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/database"
)

func NewConfig(configPath string) (*config.Config, error) {
	return config.Load(configPath)
}

func NewLogger(cfg *config.Config) (*logx.Logger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for logger")
	}
	l := logx.New(cfg.Log)
	return l, nil
}

func NewDB(cfg *config.Config, logger *logx.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for database")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required for database")
	}
	return database.Open(cfg.Database, logger, cfg.Log.Level)
}

func NewRedis(cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for redis")
	}
	rCfg := cfg.Redis
	client := redis.NewClient(&redis.Options{
		Addr: rCfg.Addr, Password: rCfg.Password, DB: rCfg.DB, PoolSize: rCfg.PoolSize,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}
