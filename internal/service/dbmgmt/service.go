package dbmgmt

import (
	"context"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/dbconn"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/pagination"

	"crypto/cipher"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Service 数据库管理插件核心服务。
type Service struct {
	repo          interfaces.DbmgmtRepository
	serverRepo    interfaces.ServerRepository
	projectRepo   interfaces.ProjectRepository
	userGroupRepo interfaces.UserGroupRepository
	userRepo      interfaces.UserRepository
	db            *gorm.DB
	aead          cipher.AEAD
	mailer        mailer.Sender
	appName       string
	cfg           config.DbmgmtConfig

	instSem map[uint]chan struct{}
	semMu   sync.Mutex
}

func NewService(
	repo interfaces.DbmgmtRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	db *gorm.DB,
	encryptionKey string,
	emailSender mailer.Sender,
	appName string,
	cfg config.DbmgmtConfig,
) (*Service, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	dbconn.SetDecryptFunc(cryptox.DecryptString)
	if cfg.QueryTimeoutSeconds <= 0 {
		cfg = config.DefaultDbmgmtConfig()
	}
	return &Service{
		repo:          repo,
		serverRepo:    serverRepo,
		projectRepo:   projectRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		db:            db,
		aead:          aead,
		mailer:        emailSender,
		appName:       strings.TrimSpace(appName),
		cfg:           cfg,
		instSem:       make(map[uint]chan struct{}),
	}, nil
}

func (s *Service) resolvedConfig(ctx context.Context) config.DbmgmtConfig {
	base := s.cfg
	if s.db == nil {
		return base
	}
	return dictconfig.ResolveDbmgmtConfig(ctx, s.db, base)
}

func (s *Service) acquireInstance(instanceID uint) func() {
	s.semMu.Lock()
	ch, ok := s.instSem[instanceID]
	if !ok {
		max := s.cfg.MaxConcurrentPerInstance
		if max <= 0 {
			max = 5
		}
		ch = make(chan struct{}, max)
		s.instSem[instanceID] = ch
	}
	s.semMu.Unlock()
	ch <- struct{}{}
	return func() { <-ch }
}

type sshDialer struct {
	s *Service
}

func (d sshDialer) DialServer(ctx context.Context, serverID uint) (*ssh.Client, error) {
	cli, _, err := d.s.dialSSH(ctx, serverID)
	return cli, err
}

func (s *Service) dialSSH(ctx context.Context, serverID uint) (*ssh.Client, interface{}, error) {
	return dialServerSSH(ctx, s.aead, s.serverRepo, serverID)
}

func paginate[T any](list []T, total int64, page, pageSize int) *pagination.Result[T] {
	page, pageSize = pagination.Normalize(page, pageSize)
	return &pagination.Result[T]{List: list, Total: total, Page: page, PageSize: pageSize}
}

func (s *Service) RunBackgroundWorkers(ctx context.Context) {
	interval := time.Duration(s.cfg.PingIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	slaTicker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	defer slaTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runPeriodicPing(ctx)
		case <-slaTicker.C:
			s.syncApprovalReminders(ctx)
		}
	}
}

func (s *Service) runPeriodicPing(ctx context.Context) {
	list, err := s.repo.ListAllInstances(ctx)
	if err != nil {
		return
	}
	for _, inst := range list {
		_, _ = s.pingInstance(ctx, inst.ProjectID, inst.ID)
	}
}
