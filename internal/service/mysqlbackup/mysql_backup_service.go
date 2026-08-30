// MySQL 备份服务装配：依赖注入、加密器与调度状态。
// 具体行为按职责分布在同包：instance_service.go（实例 CRUD/凭据）、job_runner.go（任务编排）、
// runner_mysqldump.go / runner_xtrabackup.go（两条执行链路）、artifact.go（对象存储命名与下载）、
// logging.go（结构化日志与远端日志轮询）、shell.go（SSH 与命令工具）、dto.go（DTO 与转换）。
package mysqlbackup

import (
	"context"
	"crypto/cipher"
	"fmt"
	"strings"
	"sync"

	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/objectstore"
)

// ObjectStoreFactory 创建 MinIO 客户端；由装配层注入，避免 Service 直拿 *gorm.DB 读字典。
type ObjectStoreFactory func(ctx context.Context) (*objectstore.Client, error)

// SchedulerConfigResolver 解析备份调度字典配置。
type SchedulerConfigResolver func(ctx context.Context) dictconfig.MysqlBackupSchedulerConfig

type MysqlBackupService struct {
	backupRepo     interfaces.MysqlBackupRepository
	serverRepo     interfaces.ServerRepository
	projectRepo    interfaces.ProjectRepository
	userRepo       interfaces.UserRepository
	newObjectStore ObjectStoreFactory
	resolveSched   SchedulerConfigResolver
	aead           cipher.AEAD
	mailer         mailer.Sender
	appName        string
	schedMu        sync.Mutex
	schedRunning   map[uint]bool
	jobCancels     sync.Map // jobID -> context.CancelFunc
}

func NewMysqlBackupService(
	backupRepo interfaces.MysqlBackupRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userRepo interfaces.UserRepository,
	newObjectStore ObjectStoreFactory,
	resolveSched SchedulerConfigResolver,
	encryptionKey string,
	emailSender mailer.Sender,
	appName string,
) (*MysqlBackupService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	if newObjectStore == nil {
		newObjectStore = func(context.Context) (*objectstore.Client, error) {
			return nil, fmt.Errorf("object store factory not configured")
		}
	}
	if resolveSched == nil {
		resolveSched = func(context.Context) dictconfig.MysqlBackupSchedulerConfig {
			return dictconfig.ResolveMysqlBackupSchedulerConfig(context.Background(), nil, dictconfig.DefaultMysqlBackupSchedulerDictTypes())
		}
	}
	return &MysqlBackupService{
		backupRepo:     backupRepo,
		serverRepo:     serverRepo,
		projectRepo:    projectRepo,
		userRepo:       userRepo,
		newObjectStore: newObjectStore,
		resolveSched:   resolveSched,
		aead:           aead,
		mailer:         emailSender,
		appName:        strings.TrimSpace(appName),
		schedRunning:   make(map[uint]bool),
	}, nil
}
