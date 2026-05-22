package providers

import (
	"yunshu/internal/config"
	logx "yunshu/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Infra struct {
	Config *config.Config
	Logger *logx.Logger
	DB     *gorm.DB
	Redis  *redis.Client
}
