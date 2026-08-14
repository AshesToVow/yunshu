package ai

import (
	"context"
	"encoding/json"
	"time"

	"yunshu/internal/model"
)

func (s *Service) recordAudit(userID, sessionID uint, action, tool, risk string, ok bool, detail string) {
	if s.db == nil {
		return
	}
	detailJSON, _ := json.Marshal(map[string]any{"detail": truncateStr(detail, 2000)})
	ev := model.AiAuditEvent{
		UserID:     userID,
		SessionID:  sessionID,
		Action:     action,
		ToolName:   tool,
		RiskLevel:  risk,
		OK:         ok,
		DetailJSON: string(detailJSON),
		CreatedAt:  time.Now(),
	}
	_ = s.db.Create(&ev).Error
}

// --- Admin list helpers (MVP) ---

func (s *Service) ListPrompts(ctx context.Context) ([]model.AiPrompt, error) {
	s.ensureSeed()
	var rows []model.AiPrompt
	err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (s *Service) ListPromptVersions(ctx context.Context, promptID uint) ([]model.AiPromptVersion, error) {
	var rows []model.AiPromptVersion
	err := s.db.WithContext(ctx).Where("prompt_id = ?", promptID).Order("version DESC").Find(&rows).Error
	return rows, err
}

type PromptPublishRequest struct {
	Content   string `json:"content" binding:"required"`
	Changelog string `json:"changelog"`
}

func (s *Service) PublishPromptVersion(ctx context.Context, promptID, userID uint, req PromptPublishRequest) (*model.AiPromptVersion, error) {
	var maxVer int
	_ = s.db.WithContext(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
		Select("COALESCE(MAX(version),0)").Scan(&maxVer)
	_ = s.db.WithContext(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
		Update("is_current", false).Error
	ver := model.AiPromptVersion{
		PromptID: promptID, Version: maxVer + 1, Content: req.Content,
		Changelog: req.Changelog, IsCurrent: true, CreatedBy: userID,
	}
	if err := s.db.WithContext(ctx).Create(&ver).Error; err != nil {
		return nil, err
	}
	return &ver, nil
}

func (s *Service) RollbackPromptVersion(ctx context.Context, promptID, versionID uint) error {
	var ver model.AiPromptVersion
	if err := s.db.WithContext(ctx).Where("id = ? AND prompt_id = ?", versionID, promptID).First(&ver).Error; err != nil {
		return err
	}
	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).Update("is_current", false).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&ver).Update("is_current", true).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *Service) ListLLMModels(ctx context.Context) ([]model.AiLLMModel, error) {
	s.ensureSeed()
	var rows []model.AiLLMModel
	err := s.db.WithContext(ctx).Order("is_default DESC, id ASC").Find(&rows).Error
	for i := range rows {
		rows[i].HasAPIKey = rows[i].APIKeyEnc != ""
		rows[i].APIKeyEnc = ""
	}
	return rows, err
}

func (s *Service) ListTools(ctx context.Context) ([]model.AiToolDef, error) {
	s.ensureSeed()
	var rows []model.AiToolDef
	err := s.db.WithContext(ctx).Order("module ASC, name ASC").Find(&rows).Error
	return rows, err
}

func (s *Service) UpdateToolEnabled(ctx context.Context, id uint, enabled bool) error {
	return s.db.WithContext(ctx).Model(&model.AiToolDef{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *Service) ListIncidentCases(ctx context.Context) ([]model.AiIncidentCase, error) {
	s.ensureSeed()
	var rows []model.AiIncidentCase
	err := s.db.WithContext(ctx).Order("id DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (s *Service) ListSOPs(ctx context.Context) ([]model.AiSOP, error) {
	s.ensureSeed()
	var rows []model.AiSOP
	err := s.db.WithContext(ctx).Order("id DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (s *Service) ListKnowledgeBases(ctx context.Context) ([]model.AiKnowledgeBase, error) {
	s.ensureSeed()
	var rows []model.AiKnowledgeBase
	err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (s *Service) ListEvalCases(ctx context.Context) ([]model.AiEvalCase, error) {
	s.ensureSeed()
	var rows []model.AiEvalCase
	err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (s *Service) ReseedCenter(ctx context.Context) error {
	return s.EnsureCenterSeed(ctx)
}

func (s *Service) CenterOverview(ctx context.Context) map[string]any {
	s.ensureSeed()
	count := func(model any) int64 {
		var n int64
		_ = s.db.WithContext(ctx).Model(model).Count(&n).Error
		return n
	}
	return map[string]any{
		"prompts":     count(&model.AiPrompt{}),
		"llm_models":  count(&model.AiLLMModel{}),
		"tools":       count(&model.AiToolDef{}),
		"cases":       count(&model.AiIncidentCase{}),
		"sops":        count(&model.AiSOP{}),
		"kb":          count(&model.AiKnowledgeBase{}),
		"eval_cases":  count(&model.AiEvalCase{}),
		"sessions":    count(&model.AiChatSession{}),
		"data_root":   s.dataRoot(),
	}
}
