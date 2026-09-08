package ai

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"yunshu/internal/model"
)

func (s *Service) recordAudit(userID, sessionID uint, action, tool, risk string, ok bool, detail string) {
	if s.repo == nil {
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
	_ = s.repo.CreateAuditEvent(context.Background(), &ev)
}

// --- Admin list helpers (MVP) ---

func (s *Service) ListPrompts(ctx context.Context) ([]model.AiPrompt, error) {
	s.ensureSeed()
	return s.repo.ListPrompts(ctx)
}

func (s *Service) ListPromptVersions(ctx context.Context, promptID uint) ([]model.AiPromptVersion, error) {
	return s.repo.ListPromptVersions(ctx, promptID)
}

type PromptPublishRequest struct {
	Content   string `json:"content" binding:"required"`
	Changelog string `json:"changelog"`
}

func (s *Service) PublishPromptVersion(ctx context.Context, promptID, userID uint, req PromptPublishRequest) (*model.AiPromptVersion, error) {
	return s.repo.PublishPromptVersion(ctx, promptID, userID, req.Content, req.Changelog)
}

func (s *Service) RollbackPromptVersion(ctx context.Context, promptID, versionID uint) error {
	return s.repo.RollbackPromptVersion(ctx, promptID, versionID)
}

func (s *Service) ListLLMModels(ctx context.Context) ([]model.AiLLMModel, error) {
	s.ensureSeed()
	rows, err := s.repo.ListLLMModels(ctx)
	for i := range rows {
		rows[i].HasAPIKey = rows[i].APIKeyEnc != ""
		rows[i].APIKeyEnc = ""
	}
	return rows, err
}

func (s *Service) ListTools(ctx context.Context) ([]model.AiToolDef, error) {
	s.ensureSeed()
	return s.repo.ListTools(ctx)
}

func (s *Service) UpdateToolEnabled(ctx context.Context, id uint, enabled bool) error {
	return s.repo.UpdateToolEnabled(ctx, id, enabled)
}

func (s *Service) ListIncidentCases(ctx context.Context) ([]model.AiIncidentCase, error) {
	s.ensureSeed()
	return s.repo.ListIncidentCases(ctx, 200)
}

func (s *Service) ListSOPs(ctx context.Context) ([]model.AiSOP, error) {
	s.ensureSeed()
	return s.repo.ListSOPs(ctx, 200)
}

func (s *Service) ListKnowledgeBases(ctx context.Context) ([]model.AiKnowledgeBase, error) {
	s.ensureSeed()
	return s.repo.ListKnowledgeBases(ctx)
}

func (s *Service) ListEvalCases(ctx context.Context) ([]model.AiEvalCase, error) {
	s.ensureSeed()
	return s.repo.ListEvalCases(ctx)
}

func (s *Service) ReseedCenter(ctx context.Context) (*CenterSeedReport, error) {
	return s.EnsureCenterSeedReport(ctx)
}

func (s *Service) CenterOverview(ctx context.Context) map[string]any {
	s.ensureSeed()
	root := s.dataRoot()
	rootOK := false
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		rootOK = true
	}
	count := func(fn func(context.Context) (int64, error)) int64 {
		n, _ := fn(ctx)
		return n
	}
	return map[string]any{
		"prompts":      count(s.repo.CountPrompts),
		"llm_models":   count(func(ctx context.Context) (int64, error) { return s.repo.CountEnabledLLMModels(ctx) }),
		"tools":        count(s.repo.CountAllTools),
		"cases":        count(s.repo.CountIncidentCases),
		"sops":         count(s.repo.CountSOPs),
		"kb":           count(s.repo.CountKnowledgeBases),
		"eval_cases":   count(s.repo.CountEvalCases),
		"sessions":     count(s.repo.CountChatSessions),
		"data_root":    root,
		"data_root_ok": rootOK,
	}
}
