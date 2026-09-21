package dbmgmt

import (
	"context"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/dbconn"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/pagination"
	workflowsvc "yunshu/internal/service/workflow"

	"crypto/cipher"

	"golang.org/x/crypto/ssh"
)

// DbmgmtConfigResolver 解析运行期字典覆盖后的 dbmgmt 配置（由 Wire 注入，避免 Service 持有 *gorm.DB）。
type DbmgmtConfigResolver func(ctx context.Context) config.DbmgmtConfig

// Service 数据库管理插件核心服务。
type Service struct {
	repo          interfaces.DbmgmtRepository
	serverRepo    interfaces.ServerRepository
	projectRepo   interfaces.ProjectRepository
	memberRepo    interfaces.ProjectMemberRepository
	userGroupRepo interfaces.UserGroupRepository
	userRepo      interfaces.UserRepository
	dutyRepo      interfaces.AlertDutyRepository
	workflow      *workflowsvc.Service
	resolveCfg    DbmgmtConfigResolver
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
	memberRepo interfaces.ProjectMemberRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	dutyRepo interfaces.AlertDutyRepository,
	workflow *workflowsvc.Service,
	resolveCfg DbmgmtConfigResolver,
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
	if resolveCfg == nil {
		resolveCfg = func(context.Context) config.DbmgmtConfig { return cfg }
	}
	if workflow == nil {
		workflow = workflowsvc.NewService(nil, userGroupRepo, dutyRepo, userRepo)
	}
	return &Service{
		repo:          repo,
		serverRepo:    serverRepo,
		projectRepo:   projectRepo,
		memberRepo:    memberRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		dutyRepo:      dutyRepo,
		workflow:      workflow,
		resolveCfg:    resolveCfg,
		aead:          aead,
		mailer:        emailSender,
		appName:       strings.TrimSpace(appName),
		cfg:           cfg,
		instSem:       make(map[uint]chan struct{}),
	}, nil
}

func (s *Service) resolvedConfig(ctx context.Context) config.DbmgmtConfig {
	if s.resolveCfg == nil {
		return s.cfg
	}
	return s.resolveCfg(ctx)
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
