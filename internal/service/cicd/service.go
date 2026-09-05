package cicd

import (
	"context"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/jenkins"
	"yunshu/internal/pkg/mailer"

	"gorm.io/gorm"
)

type Service struct {
	db            *gorm.DB
	serverRepo    interfaces.ServerRepository
	projectRepo   interfaces.ProjectRepository
	userGroupRepo interfaces.UserGroupRepository
	userRepo      interfaces.UserRepository
	memberRepo    interfaces.ProjectMemberRepository
	dutyRepo      interfaces.AlertDutyRepository
	nsEnsurer     K8sNamespaceEnsurer
	mailer        mailer.Sender
	appName       string
	yamlCicd      config.CicdConfig
	syncMu        sync.Mutex
	// optional post-release verify hooks
	workloadReadyCheck func(ctx context.Context, clusterID, namespace, kind, name string) (*bool, string)
	errorLogSampler    func(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string)
	k8sRolloutUndo     K8sRolloutUndoFn
	k8sProgressive     K8sProgressiveFns
}

func NewService(db *gorm.DB, serverRepo interfaces.ServerRepository, projectRepo interfaces.ProjectRepository, userGroupRepo interfaces.UserGroupRepository, userRepo interfaces.UserRepository, memberRepo interfaces.ProjectMemberRepository, dutyRepo interfaces.AlertDutyRepository, yamlCicd config.CicdConfig, emailSender mailer.Sender, appName string, nsEnsurer K8sNamespaceEnsurer) *Service {
	if yamlCicd.RunSyncIntervalSeconds <= 0 {
		yamlCicd.RunSyncIntervalSeconds = 15
	}
	if yamlCicd.ApprovalSlaHours <= 0 {
		yamlCicd.ApprovalSlaHours = 24
	}
	if yamlCicd.ApprovalReminderIntervalHours <= 0 {
		yamlCicd.ApprovalReminderIntervalHours = 4
	}
	return &Service{
		db:            db,
		serverRepo:    serverRepo,
		projectRepo:   projectRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		memberRepo:    memberRepo,
		dutyRepo:      dutyRepo,
		nsEnsurer:     nsEnsurer,
		mailer:        emailSender,
		appName:       strings.TrimSpace(appName),
		yamlCicd:      yamlCicd,
	}
}

func (s *Service) SetWorkloadReadyCheck(fn func(ctx context.Context, clusterID, namespace, kind, name string) (*bool, string)) {
	s.workloadReadyCheck = fn
}

func (s *Service) SetErrorLogSampler(fn func(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string)) {
	s.errorLogSampler = fn
}

func (s *Service) resolvedConfig(ctx context.Context) config.CicdConfig {
	base := s.yamlCicd
	if base.RunSyncIntervalSeconds <= 0 {
		base = config.DefaultCicdConfig()
		base.Jenkins = s.yamlCicd.Jenkins
	}
	return dictconfig.ResolveCicdConfig(ctx, s.db, base, dictconfig.DefaultCicdDictTypes())
}

func (s *Service) jenkinsClient(ctx context.Context) (*jenkins.Client, config.CicdConfig, error) {
	cfg := s.resolvedConfig(ctx)
	if !cfg.Enabled {
		return nil, cfg, constants.ErrBadRequestWithMsg("CI/CD 未启用，请在数据字典配置 cicd_enabled=true")
	}
	if strings.TrimSpace(cfg.Jenkins.BaseURL) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins 地址未配置，请在数据字典设置 cicd_jenkins_base_url")
	}
	if strings.TrimSpace(cfg.Jenkins.Username) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins 用户名未配置，请在数据字典设置 cicd_jenkins_username")
	}
	if strings.TrimSpace(cfg.Jenkins.APIToken) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins API Token 未配置，请在数据字典设置 cicd_jenkins_api_token")
	}
	return jenkins.NewClient(cfg.Jenkins.BaseURL, cfg.Jenkins.Username, cfg.Jenkins.APIToken, cfg.Jenkins.JobFolder), cfg, nil
}

func (s *Service) ensureProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project id required")
	}
	if s.projectRepo == nil {
		return nil
	}
	_, err := s.projectRepo.GetByID(ctx, projectID)
	return err
}
