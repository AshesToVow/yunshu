package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"yunshu/internal/config"
	"yunshu/internal/model"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/logutil"
)

func NewConfig(configPath string) (*config.Config, error) {
	return config.Load(configPath)
}

func NewLogger(cfg *config.Config) (*logx.Logger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for logger")
	}
	l := logx.New(cfg.Log)
	logutil.SetDefaultLogger(l.Info)
	return l, nil
}

func NewDB(cfg *config.Config, logger *logx.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for mysql")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required for mysql")
	}
	mysqlCfg := cfg.MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=%s",
		mysqlCfg.User, mysqlCfg.Password, mysqlCfg.Host, mysqlCfg.Port,
		mysqlCfg.DBName, mysqlCfg.Charset, mysqlCfg.Loc,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logx.NewGormLogger(logger.SQL, cfg.Log.Level),
	})
	if err != nil {
		return nil, err
	}
	if err = db.SetupJoinTable(&model.User{}, "Roles", &model.UserRole{}); err != nil {
		return nil, err
	}
	if err = db.SetupJoinTable(&model.User{}, "Groups", &model.UserGroupUser{}); err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(mysqlCfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(mysqlCfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(mysqlCfg.ConnMaxLifetimeSeconds) * time.Second)
	return db, nil
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
