package cicd

import (
	"context"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

type PipelineTemplateUpsertRequest struct {
	LanguageType string `json:"language_type" binding:"required,max=32"`
	Name         string `json:"name" binding:"required,max=128"`
	ScriptPath   string `json:"script_path" binding:"required,max=256"`
	Description  string `json:"description" binding:"omitempty,max=512"`
	Sort         int    `json:"sort"`
	Status       *int   `json:"status"`
}

func (s *Service) ListPipelineTemplates(ctx context.Context) ([]model.CicdPipelineTemplate, error) {
	s.ensureDefaultPipelineTemplates(ctx)
	var rows []model.CicdPipelineTemplate
	if err := s.db.WithContext(ctx).Where("status = 1").Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.CicdPipelineTemplate{}
	}
	return rows, nil
}

func (s *Service) UpsertPipelineTemplate(ctx context.Context, id uint, req PipelineTemplateUpsertRequest) (*model.CicdPipelineTemplate, error) {
	lt := strings.ToLower(strings.TrimSpace(req.LanguageType))
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	var row model.CicdPipelineTemplate
	if id > 0 {
		if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return nil, constants.ErrNotFound
		}
	}
	row.LanguageType = lt
	row.Name = strings.TrimSpace(req.Name)
	row.ScriptPath = strings.TrimSpace(req.ScriptPath)
	row.Description = strings.TrimSpace(req.Description)
	row.Sort = req.Sort
	row.Status = status
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) ensureDefaultPipelineTemplates(ctx context.Context) {
	defaults := []model.CicdPipelineTemplate{
		{LanguageType: model.CicdLanguageGo, Name: "Go", ScriptPath: "backend.jenkinsfile", Description: "Go 后端（复用 backend.jenkinsfile）", Sort: 1, Status: 1},
		{LanguageType: model.CicdLanguageJava, Name: "Java", ScriptPath: "backend.jenkinsfile", Description: "Java/Maven/Gradle", Sort: 2, Status: 1},
		{LanguageType: model.CicdLanguageFrontend, Name: "前端", ScriptPath: "front.jenkinsfile", Description: "npm/yarn 前端", Sort: 3, Status: 1},
		{LanguageType: model.CicdLanguagePython, Name: "Python", ScriptPath: "backend.jenkinsfile", Description: "Python 后端", Sort: 4, Status: 1},
		{LanguageType: model.CicdLanguageCustom, Name: "自定义", ScriptPath: "", Description: "按服务类型选择 front/backend/k8s Jenkinsfile", Sort: 99, Status: 1},
	}
	for _, d := range defaults {
		var n int64
		_ = s.db.WithContext(ctx).Model(&model.CicdPipelineTemplate{}).Where("language_type = ?", d.LanguageType).Count(&n).Error
		if n == 0 {
			row := d
			_ = s.db.WithContext(ctx).Create(&row).Error
		}
	}
}

// ResolveScriptPathForService 按 language_type 模板或 service_type 解析 Jenkins Script Path。
func (s *Service) ResolveScriptPathForService(ctx context.Context, svc *model.CicdService, ci *model.CicdCiConfig, usesK8s bool) string {
	cfg := s.resolvedConfig(ctx)
	if usesK8s {
		return dictconfig.JenkinsScriptPath(cfg, svc.ServiceType, true)
	}
	lt := model.CicdLanguageCustom
	if ci != nil {
		lt = strings.ToLower(strings.TrimSpace(ci.LanguageType))
	}
	if lt != "" && lt != model.CicdLanguageCustom {
		s.ensureDefaultPipelineTemplates(ctx)
		var tpl model.CicdPipelineTemplate
		if err := s.db.WithContext(ctx).
			Where("language_type = ? AND status = 1", lt).
			First(&tpl).Error; err == nil {
			if v := strings.TrimSpace(tpl.ScriptPath); v != "" {
				return v
			}
		}
	}
	st := ""
	if svc != nil {
		st = svc.ServiceType
	}
	return dictconfig.JenkinsScriptPath(cfg, st, false)
}
