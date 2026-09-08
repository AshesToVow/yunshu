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
)

type Service struct {
	repo          interfaces.CicdRepository
	workflowRepo  interfaces.WorkflowRepository
	serverRepo    interfaces.ServerRepository
	projectRepo   interfaces.ProjectRepository
	userGroupRepo interfaces.UserGroupRepository
	userRepo      interfaces.UserRepository
	memberRepo    interfaces.ProjectMemberRepository
	dutyRepo      interfaces.AlertDutyRepository
	catalogRepo   interfaces.ServiceCatalogRepository
	nsEnsurer     K8sNamespaceEnsurer
	mailer        mailer.Sender
	appName       string
	resolveConfig func(ctx context.Context) config.CicdConfig
	resolveMinio  func(ctx context.Context) dictconfig.MinioConfig
	syncMu        sync.Mutex
	// optional post-release verify hooks
	workloadReadyCheck func(ctx context.Context, clusterID, namespace, kind, name string) (*bool, string)
	errorLogSampler    func(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string)
	k8sRolloutUndo     K8sRolloutUndoFn
	k8sProgressive     K8sProgressiveFns
}

func NewService(
	repo interfaces.CicdRepository,
	workflowRepo interfaces.WorkflowRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	memberRepo interfaces.ProjectMemberRepository,
	dutyRepo interfaces.AlertDutyRepository,
	catalogRepo interfaces.ServiceCatalogRepository,
	resolveConfig func(context.Context) config.CicdConfig,
	resolveMinio func(context.Context) dictconfig.MinioConfig,
	emailSender mailer.Sender,
	appName string,
	nsEnsurer K8sNamespaceEnsurer,
) *Service {
	return &Service{
		repo:          repo,
		workflowRepo:  workflowRepo,
		serverRepo:    serverRepo,
		projectRepo:   projectRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		memberRepo:    memberRepo,
		dutyRepo:      dutyRepo,
		catalogRepo:   catalogRepo,
		nsEnsurer:     nsEnsurer,
		mailer:        emailSender,
		appName:       strings.TrimSpace(appName),
		resolveConfig: resolveConfig,
		resolveMinio:  resolveMinio,
	}
}

func (s *Service) SetWorkloadReadyCheck(fn func(ctx context.Context, clusterID, namespace, kind, name string) (*bool, string)) {
	s.workloadReadyCheck = fn
}

func (s *Service) SetErrorLogSampler(fn func(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string)) {
	s.errorLogSampler = fn
}

func (s *Service) resolvedConfig(ctx context.Context) config.CicdConfig {
	if s.resolveConfig != nil {
		return s.resolveConfig(ctx)
	}
	return config.DefaultCicdConfig()
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
