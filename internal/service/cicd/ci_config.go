package cicd

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/jenkins"

	"gorm.io/gorm"
)

// --- CI Config ---

type CiConfigUpsertRequest struct {
	GitURL           string `json:"git_url" binding:"required,max=512"`
	RefType          string `json:"ref_type" binding:"omitempty,oneof=branch tag"`
	RefName          string `json:"ref_name" binding:"required,max=128"`
	BuildType        string `json:"build_type" binding:"required,max=32"`
	LanguageType     string `json:"language_type" binding:"omitempty,max=32"`
	BuildShell       string `json:"build_shell" binding:"omitempty,max=512"`
	BuildPath        string `json:"build_path" binding:"omitempty,max=256"`
	ProjectName      string `json:"project_name" binding:"omitempty,max=128"`
	Version          string `json:"version" binding:"omitempty,max=64"`
	NodeVersion      string `json:"node_version" binding:"omitempty,max=32"`
	NpmInstallMode   string `json:"npm_install_mode" binding:"omitempty,max=16"`
	CleanNpmCache    bool   `json:"clean_npm_cache"`
	CleanNodeModules bool   `json:"clean_node_modules"`
	JavaToolName     string `json:"java_tool_name" binding:"omitempty,max=64"`
	ServerPort       string `json:"server_port" binding:"omitempty,max=16"`
	PackConfigPaths  string `json:"pack_config_paths" binding:"omitempty,max=512"`
	Description      string `json:"description" binding:"omitempty,max=512"`
}

func (s *Service) FindCiConfig(ctx context.Context, projectID, serviceID uint) (*model.CicdCiConfig, bool, error) {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return nil, false, err
	}
	var row model.CicdCiConfig
	if err := s.db.WithContext(ctx).Where("service_id = ?", serviceID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &row, true, nil
}

// CiConfigView GET ci-config 响应（未配置时 configured=false，不返回 404）。
type CiConfigView struct {
	Configured bool                `json:"configured"`
	Config     *model.CicdCiConfig `json:"config,omitempty"`
}

func (s *Service) GetCiConfigView(ctx context.Context, projectID, serviceID uint) (*CiConfigView, error) {
	row, ok, err := s.FindCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	return &CiConfigView{Configured: ok, Config: row}, nil
}

func (s *Service) requireCiConfig(ctx context.Context, projectID, serviceID uint) (*model.CicdCiConfig, error) {
	row, ok, err := s.FindCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	if !ok || row == nil {
		return nil, constants.ErrBadRequestWithMsg("请先配置 CI 信息")
	}
	return row, nil
}

// CiConfigUpsertResult 保存 CI 配置结果（DB 保存与 Jenkins 同步解耦）。
type CiConfigUpsertResult struct {
	Config           *model.CicdCiConfig `json:"config"`
	JenkinsSync      *JenkinsSyncResult  `json:"jenkins_sync,omitempty"`
	JenkinsSyncError string              `json:"jenkins_sync_error,omitempty"`
}

func (s *Service) UpsertCiConfig(ctx context.Context, projectID, serviceID uint, req CiConfigUpsertRequest) (*CiConfigUpsertResult, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	refType := strings.TrimSpace(req.RefType)
	if refType == "" {
		refType = model.CicdRefTypeBranch
	}
	var row model.CicdCiConfig
	err = s.db.WithContext(ctx).Where("service_id = ?", serviceID).First(&row).Error
	isNew := err != nil
	row.ServiceID = serviceID
	row.GitURL = strings.TrimSpace(req.GitURL)
	row.RefType = refType
	row.RefName = strings.TrimSpace(req.RefName)
	row.BuildType = strings.TrimSpace(req.BuildType)
	lt := strings.ToLower(strings.TrimSpace(req.LanguageType))
	if lt == "" {
		lt = model.CicdLanguageCustom
	}
	row.LanguageType = lt
	row.BuildShell = strings.TrimSpace(req.BuildShell)
	row.BuildPath = strings.TrimSpace(req.BuildPath)
	row.ProjectName = strings.TrimSpace(req.ProjectName)
	if row.ProjectName == "" {
		row.ProjectName = svc.Identifier
	}
	row.Version = strings.TrimSpace(req.Version)
	row.NodeVersion = strings.TrimSpace(req.NodeVersion)
	if row.NodeVersion == "" {
		row.NodeVersion = model.DefaultNodeToolName
	}
	row.NpmInstallMode = strings.TrimSpace(req.NpmInstallMode)
	row.CleanNpmCache = req.CleanNpmCache
	row.CleanNodeModules = req.CleanNodeModules
	row.JavaToolName = strings.TrimSpace(req.JavaToolName)
	if row.JavaToolName == "" {
		row.JavaToolName = "jdk8"
	}
	row.ServerPort = strings.TrimSpace(req.ServerPort)
	row.PackConfigPaths = strings.TrimSpace(req.PackConfigPaths)
	row.Description = strings.TrimSpace(req.Description)
	if isNew {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	}
	syncResult, err := s.syncJenkinsJob(ctx, svc, &row)
	if err != nil {
		return &CiConfigUpsertResult{
			Config:           &row,
			JenkinsSyncError: jenkins.HumanizeAPIError(err),
		}, nil
	}
	return &CiConfigUpsertResult{Config: &row, JenkinsSync: syncResult}, nil
}
