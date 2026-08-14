package ai

import (
	"context"
	"strings"

	"yunshu/internal/ai/prompts"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// loadPromptContent 优先读 DB 当前版本；空库则 seed 后重试；仍失败回退 embed（仅过渡）。
func (s *Service) loadPromptContent(ctx context.Context, code string, vars map[string]string) (string, error) {
	s.ensureSeed()
	code = strings.TrimSpace(code)
	// 兼容旧名
	switch code {
	case "system_ops_assistant":
		code = "system/ops-agent"
	case "k8s_pod_diagnose":
		code = "diagnosis/k8s-pod"
	case "cicd_build_fail":
		code = "diagnosis/cicd-build-fail"
	case "alert_explain":
		code = "diagnosis/alert-explain"
	}
	body, err := s.loadPromptFromDB(ctx, code)
	if err != nil || strings.TrimSpace(body) == "" {
		// 过渡回退：旧 embed 文件名
		legacy := map[string]string{
			"system/ops-agent":            "system_ops_assistant",
			"diagnosis/k8s-pod":           "k8s_pod_diagnose",
			"diagnosis/cicd-build-fail":   "cicd_build_fail",
			"diagnosis/alert-explain":     "alert_explain",
		}
		name := legacy[code]
		if name == "" {
			name = code
		}
		return prompts.Load(name, vars)
	}
	out := body
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out, nil
}

func (s *Service) loadPromptFromDB(ctx context.Context, code string) (string, error) {
	var p model.AiPrompt
	if err := s.db.WithContext(ctx).Where("code = ? AND enabled = ?", code, true).First(&p).Error; err != nil {
		return "", err
	}
	var ver model.AiPromptVersion
	err := s.db.WithContext(ctx).Where("prompt_id = ? AND is_current = ?", p.ID, true).First(&ver).Error
	if err != nil {
		return "", err
	}
	return ver.Content, nil
}

// GetCurrentPrompt 管理/调试用。
func (s *Service) GetCurrentPrompt(ctx context.Context, code string) (*model.AiPrompt, *model.AiPromptVersion, error) {
	s.ensureSeed()
	var p model.AiPrompt
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&p).Error; err != nil {
		return nil, nil, constants.ErrNotFoundWithMsg("Prompt 不存在")
	}
	var ver model.AiPromptVersion
	if err := s.db.WithContext(ctx).Where("prompt_id = ? AND is_current = ?", p.ID, true).First(&ver).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &p, nil, nil
		}
		return nil, nil, err
	}
	return &p, &ver, nil
}
