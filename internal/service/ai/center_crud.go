package ai

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// --- Prompt meta CRUD ---

type PromptUpsertRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name" binding:"required,max=128"`
	Type    string `json:"type"` // system|diagnosis|generation
	Scene   string `json:"scene"`
	Enabled *bool  `json:"enabled"`
	Remark  string `json:"remark"`
	Content   string `json:"content"`
	Changelog string `json:"changelog"`
}

type PromptDetail struct {
	model.AiPrompt
	CurrentVersion *model.AiPromptVersion `json:"current_version,omitempty"`
}

func (s *Service) GetPrompt(ctx context.Context, id uint) (*PromptDetail, error) {
	row, err := s.repo.GetPromptByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("Prompt 不存在")
	}
	out := &PromptDetail{AiPrompt: *row}
	ver, err := s.repo.GetCurrentPromptVersion(ctx, id)
	if err == nil {
		out.CurrentVersion = ver
	}
	return out, nil
}

func (s *Service) CreatePrompt(ctx context.Context, userID uint, req PromptUpsertRequest) (*PromptDetail, error) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return nil, constants.ErrBadRequestWithMsg("code 与 name 必填")
	}
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = "system"
	}
	row := model.AiPrompt{
		Code: code, Name: name, Type: typ,
		Scene: strings.TrimSpace(req.Scene), Enabled: true,
		Remark: strings.TrimSpace(req.Remark), CreatedBy: userID,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	var ver *model.AiPromptVersion
	content := strings.TrimSpace(req.Content)
	if content != "" {
		ver = &model.AiPromptVersion{
			Version: 1, Content: content,
			Changelog: coalesce(req.Changelog, "create"), IsCurrent: true, CreatedBy: userID,
		}
	}
	if err := s.repo.CreatePromptWithVersionTx(ctx, &row, ver); err != nil {
		return nil, err
	}
	return s.GetPrompt(ctx, row.ID)
}

func (s *Service) UpdatePrompt(ctx context.Context, id uint, req PromptUpsertRequest) (*PromptDetail, error) {
	if _, err := s.repo.GetPromptByID(ctx, id); err != nil {
		return nil, constants.ErrNotFoundWithMsg("Prompt 不存在")
	}
	updates := map[string]any{
		"name":   strings.TrimSpace(req.Name),
		"remark": strings.TrimSpace(req.Remark),
		"scene":  strings.TrimSpace(req.Scene),
	}
	if t := strings.TrimSpace(req.Type); t != "" {
		updates["type"] = t
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if c := strings.TrimSpace(req.Code); c != "" {
		updates["code"] = c
	}
	if err := s.repo.UpdatePrompt(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.GetPrompt(ctx, id)
}

func (s *Service) DeletePrompt(ctx context.Context, id uint) error {
	if err := s.repo.DeletePromptCascade(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("Prompt 不存在")
		}
		return err
	}
	return nil
}

// --- SOP CRUD ---

type SOPUpsertRequest struct {
	Code            string `json:"code"`
	Title           string `json:"title" binding:"required,max=256"`
	Scenario        string `json:"scenario"`
	Preconditions   string `json:"preconditions"`
	InputParams     string `json:"input_params"`
	CheckSteps      string `json:"check_steps"`
	ExecSteps       string `json:"exec_steps"`
	VerifySteps     string `json:"verify_steps"`
	ExceptionHandle string `json:"exception_handle"`
	Rollback        string `json:"rollback"`
	Risk            string `json:"risk"`
	ApprovalNeeded  *bool  `json:"approval_needed"`
	Enabled         *bool  `json:"enabled"`
}

func (s *Service) GetSOP(ctx context.Context, id uint) (*model.AiSOP, error) {
	row, err := s.repo.GetSOPByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("SOP 不存在")
	}
	return row, nil
}

func (s *Service) CreateSOP(ctx context.Context, userID uint, req SOPUpsertRequest) (*model.AiSOP, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, constants.ErrBadRequestWithMsg("title 必填")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "sop-" + strings.ReplaceAll(strings.ToLower(title), " ", "-")
		if len(code) > 60 {
			code = code[:60]
		}
	}
	row := model.AiSOP{
		Code: code, Title: title,
		Scenario: req.Scenario, Preconditions: req.Preconditions, InputParams: req.InputParams,
		CheckSteps: req.CheckSteps, ExecSteps: req.ExecSteps, VerifySteps: req.VerifySteps,
		ExceptionHandle: req.ExceptionHandle, Rollback: req.Rollback, Risk: req.Risk,
		Enabled: true, CreatedBy: userID,
	}
	if req.ApprovalNeeded != nil {
		row.ApprovalNeeded = *req.ApprovalNeeded
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateSOP(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateSOP(ctx context.Context, id uint, req SOPUpsertRequest) (*model.AiSOP, error) {
	row, err := s.repo.GetSOPByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("SOP 不存在")
	}
	updates := map[string]any{
		"title": strings.TrimSpace(req.Title), "scenario": req.Scenario,
		"preconditions": req.Preconditions, "input_params": req.InputParams,
		"check_steps": req.CheckSteps, "exec_steps": req.ExecSteps,
		"verify_steps": req.VerifySteps, "exception_handle": req.ExceptionHandle,
		"rollback": req.Rollback, "risk": req.Risk,
	}
	if c := strings.TrimSpace(req.Code); c != "" {
		updates["code"] = c
	}
	if req.ApprovalNeeded != nil {
		updates["approval_needed"] = *req.ApprovalNeeded
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateSOP(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetSOPByID(ctx, row.ID)
}

func (s *Service) DeleteSOP(ctx context.Context, id uint) error {
	if err := s.repo.DeleteSOP(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("SOP 不存在")
		}
		return err
	}
	return nil
}

// --- Incident Case CRUD ---

type IncidentCaseUpsertRequest struct {
	CaseID       string   `json:"case_id"`
	Title        string   `json:"title" binding:"required,max=256"`
	Category     string   `json:"category"`
	Technology   string   `json:"technology"`
	Symptom      string   `json:"symptom"`
	Environment  string   `json:"environment"`
	Diagnosis    string   `json:"diagnosis"`
	RootCause    string   `json:"root_cause"`
	Solution     string   `json:"solution"`
	Verification string   `json:"verification"`
	Risk         string   `json:"risk"`
	RelatedTools string   `json:"related_tools"`
	RelatedSOP   string   `json:"related_sop"`
	Source       string   `json:"source"`
	Confidence   *float64 `json:"confidence"`
	Enabled      *bool    `json:"enabled"`
}

func (s *Service) GetIncidentCase(ctx context.Context, id uint) (*model.AiIncidentCase, error) {
	row, err := s.repo.GetIncidentCaseByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("案例不存在")
	}
	return row, nil
}

func (s *Service) CreateIncidentCase(ctx context.Context, userID uint, req IncidentCaseUpsertRequest) (*model.AiIncidentCase, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, constants.ErrBadRequestWithMsg("title 必填")
	}
	caseID := strings.TrimSpace(req.CaseID)
	if caseID == "" {
		caseID = "CASE-" + strings.ToUpper(strings.ReplaceAll(title, " ", "-"))
		if len(caseID) > 60 {
			caseID = caseID[:60]
		}
	}
	row := model.AiIncidentCase{
		CaseID: caseID, Title: title, Category: req.Category, Technology: req.Technology,
		Symptom: req.Symptom, Environment: req.Environment, Diagnosis: req.Diagnosis,
		RootCause: req.RootCause, Solution: req.Solution, Verification: req.Verification,
		Risk: req.Risk, RelatedTools: req.RelatedTools, RelatedSOP: req.RelatedSOP,
		Source: coalesce(req.Source, "manual"), Confidence: 0.8, Enabled: true, CreatedBy: userID,
	}
	if req.Confidence != nil {
		row.Confidence = *req.Confidence
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateIncidentCase(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateIncidentCase(ctx context.Context, id uint, req IncidentCaseUpsertRequest) (*model.AiIncidentCase, error) {
	row, err := s.repo.GetIncidentCaseByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("案例不存在")
	}
	updates := map[string]any{
		"title": titleOr(req.Title, row.Title), "category": req.Category, "technology": req.Technology,
		"symptom": req.Symptom, "environment": req.Environment, "diagnosis": req.Diagnosis,
		"root_cause": req.RootCause, "solution": req.Solution, "verification": req.Verification,
		"risk": req.Risk, "related_tools": req.RelatedTools, "related_sop": req.RelatedSOP,
		"source": req.Source,
	}
	if c := strings.TrimSpace(req.CaseID); c != "" {
		updates["case_id"] = c
	}
	if req.Confidence != nil {
		updates["confidence"] = *req.Confidence
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateIncidentCase(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetIncidentCaseByID(ctx, id)
}

func (s *Service) DeleteIncidentCase(ctx context.Context, id uint) error {
	if err := s.repo.DeleteIncidentCase(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("案例不存在")
		}
		return err
	}
	return nil
}

func titleOr(v, fallback string) string {
	if t := strings.TrimSpace(v); t != "" {
		return t
	}
	return fallback
}

// --- Knowledge Base + Documents ---

type KBUpsertRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name" binding:"required,max=128"`
	Category string `json:"category"`
	Remark   string `json:"remark"`
	Enabled  *bool  `json:"enabled"`
}

type KBDocUpsertRequest struct {
	KBID       uint     `json:"kb_id"`
	Title      string   `json:"title" binding:"required,max=256"`
	Source     string   `json:"source"`
	Version    string   `json:"version"`
	Content    string   `json:"content"`
	MetaJSON   string   `json:"meta_json"`
	Confidence *float64 `json:"confidence"`
	Enabled    *bool    `json:"enabled"`
}

func (s *Service) CreateKnowledgeBase(ctx context.Context, req KBUpsertRequest) (*model.AiKnowledgeBase, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("name 必填")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "kb_" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		if len(code) > 60 {
			code = code[:60]
		}
	}
	row := model.AiKnowledgeBase{
		Code: code, Name: name, Category: coalesce(req.Category, "ops"),
		Remark: req.Remark, Enabled: true,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateKnowledgeBase(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateKnowledgeBase(ctx context.Context, id uint, req KBUpsertRequest) (*model.AiKnowledgeBase, error) {
	if _, err := s.repo.GetKnowledgeBaseByID(ctx, id); err != nil {
		return nil, constants.ErrNotFoundWithMsg("知识库不存在")
	}
	updates := map[string]any{
		"name": strings.TrimSpace(req.Name), "category": req.Category, "remark": req.Remark,
	}
	if c := strings.TrimSpace(req.Code); c != "" {
		updates["code"] = c
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateKnowledgeBase(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetKnowledgeBaseByID(ctx, id)
}

func (s *Service) DeleteKnowledgeBase(ctx context.Context, id uint) error {
	if err := s.repo.DeleteKnowledgeBaseCascade(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("知识库不存在")
		}
		return err
	}
	return nil
}

func (s *Service) ListKBDocuments(ctx context.Context, kbID uint) ([]model.AiKbDocument, error) {
	return s.repo.ListKBDocuments(ctx, kbID, 500)
}

func (s *Service) GetKBDocument(ctx context.Context, id uint) (*model.AiKbDocument, error) {
	row, err := s.repo.GetKBDocumentByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("文档不存在")
	}
	return row, nil
}

func (s *Service) CreateKBDocument(ctx context.Context, userID uint, req KBDocUpsertRequest) (*model.AiKbDocument, error) {
	if req.KBID == 0 || strings.TrimSpace(req.Title) == "" {
		return nil, constants.ErrBadRequestWithMsg("kb_id 与 title 必填")
	}
	if _, err := s.repo.GetKnowledgeBaseByID(ctx, req.KBID); err != nil {
		return nil, constants.ErrBadRequestWithMsg("知识库不存在")
	}
	row := model.AiKbDocument{
		KBID: req.KBID, Title: strings.TrimSpace(req.Title), Source: coalesce(req.Source, "manual"),
		Version: coalesce(req.Version, "v1"), Content: req.Content, MetaJSON: req.MetaJSON,
		Confidence: 0.8, Enabled: true, CreatedBy: userID,
	}
	if req.Confidence != nil {
		row.Confidence = *req.Confidence
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateKBDocument(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateKBDocument(ctx context.Context, id uint, req KBDocUpsertRequest) (*model.AiKbDocument, error) {
	if _, err := s.repo.GetKBDocumentByID(ctx, id); err != nil {
		return nil, constants.ErrNotFoundWithMsg("文档不存在")
	}
	updates := map[string]any{
		"title": strings.TrimSpace(req.Title), "source": req.Source, "version": req.Version,
		"content": req.Content, "meta_json": req.MetaJSON,
	}
	if req.KBID > 0 {
		updates["kb_id"] = req.KBID
	}
	if req.Confidence != nil {
		updates["confidence"] = *req.Confidence
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateKBDocument(ctx, id, updates); err != nil {
		return nil, err
	}
	_ = s.repo.DeleteKBDocumentChunks(ctx, id)
	return s.repo.GetKBDocumentByID(ctx, id)
}

func (s *Service) DeleteKBDocument(ctx context.Context, id uint) error {
	if err := s.repo.DeleteKBDocumentCascade(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("文档不存在")
		}
		return err
	}
	return nil
}

// --- Tool full update / create script / delete ---

type ToolUpsertRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Module              string `json:"module"`
	Runtime             string `json:"runtime"`
	HandlerKey          string `json:"handler_key"`
	ScriptLang          string `json:"script_lang"`
	ScriptPath          string `json:"script_path"`
	TimeoutSec          int    `json:"timeout_sec"`
	InputSchemaJSON     string `json:"input_schema_json"`
	Permission          string `json:"permission"`
	RiskLevel           string `json:"risk_level"`
	RequireConfirmation *bool  `json:"require_confirmation"`
	AuditRequired       *bool  `json:"audit_required"`
	Enabled             *bool  `json:"enabled"`
	Remark              string `json:"remark"`
}

func (s *Service) GetTool(ctx context.Context, id uint) (*model.AiToolDef, error) {
	row, err := s.repo.GetToolByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("工具不存在")
	}
	return row, nil
}

func (s *Service) CreateTool(ctx context.Context, req ToolUpsertRequest) (*model.AiToolDef, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("name 必填")
	}
	runtime := coalesce(strings.TrimSpace(req.Runtime), "script")
	row := model.AiToolDef{
		Name: name, Description: req.Description, Module: coalesce(req.Module, "custom"),
		Runtime: runtime, HandlerKey: req.HandlerKey, ScriptLang: req.ScriptLang, ScriptPath: req.ScriptPath,
		TimeoutSec: req.TimeoutSec, InputSchemaJSON: req.InputSchemaJSON,
		Permission: coalesce(req.Permission, "READ_ONLY"), RiskLevel: coalesce(req.RiskLevel, "LOW"),
		AuditRequired: true, Enabled: true, Remark: req.Remark,
	}
	if row.TimeoutSec <= 0 {
		row.TimeoutSec = 30
	}
	if req.RequireConfirmation != nil {
		row.RequireConfirmation = *req.RequireConfirmation
	}
	if req.AuditRequired != nil {
		row.AuditRequired = *req.AuditRequired
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateTool(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateTool(ctx context.Context, id uint, req ToolUpsertRequest) (*model.AiToolDef, error) {
	if _, err := s.repo.GetToolByID(ctx, id); err != nil {
		return nil, constants.ErrNotFoundWithMsg("工具不存在")
	}
	updates := map[string]any{
		"description": req.Description, "module": req.Module, "handler_key": req.HandlerKey,
		"script_lang": req.ScriptLang, "script_path": req.ScriptPath,
		"input_schema_json": req.InputSchemaJSON, "remark": req.Remark,
	}
	if n := strings.TrimSpace(req.Name); n != "" {
		updates["name"] = n
	}
	if r := strings.TrimSpace(req.Runtime); r != "" {
		updates["runtime"] = r
	}
	if p := strings.TrimSpace(req.Permission); p != "" {
		updates["permission"] = p
	}
	if rl := strings.TrimSpace(req.RiskLevel); rl != "" {
		updates["risk_level"] = rl
	}
	if req.TimeoutSec > 0 {
		updates["timeout_sec"] = req.TimeoutSec
	}
	if req.RequireConfirmation != nil {
		updates["require_confirmation"] = *req.RequireConfirmation
	}
	if req.AuditRequired != nil {
		updates["audit_required"] = *req.AuditRequired
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateTool(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetToolByID(ctx, id)
}

func (s *Service) DeleteTool(ctx context.Context, id uint) error {
	row, err := s.repo.GetToolByID(ctx, id)
	if err != nil {
		return constants.ErrNotFoundWithMsg("工具不存在")
	}
	if strings.EqualFold(row.Runtime, "builtin") {
		return constants.ErrBadRequestWithMsg("内置工具不可删除，可禁用")
	}
	return s.repo.DeleteToolRow(ctx, row)
}

// --- Eval Case CRUD ---

type EvalCaseUpsertRequest struct {
	Suite          string `json:"suite"`
	CaseCode       string `json:"case_code"`
	Title          string `json:"title"`
	InputQuestion  string `json:"input_question" binding:"required"`
	ExpectKeywords string `json:"expect_keywords"`
	ForbidKeywords string `json:"forbid_keywords"`
	ExpectTools    string `json:"expect_tools"`
	ExpectRisk     string `json:"expect_risk"`
	ScoreWeight    int    `json:"score_weight"`
	Enabled        *bool  `json:"enabled"`
}

func (s *Service) GetEvalCase(ctx context.Context, id uint) (*model.AiEvalCase, error) {
	row, err := s.repo.GetEvalCaseByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("评估用例不存在")
	}
	return row, nil
}

func (s *Service) CreateEvalCase(ctx context.Context, req EvalCaseUpsertRequest) (*model.AiEvalCase, error) {
	q := strings.TrimSpace(req.InputQuestion)
	if q == "" {
		return nil, constants.ErrBadRequestWithMsg("input_question 必填")
	}
	code := strings.TrimSpace(req.CaseCode)
	if code == "" {
		code = "EVAL-" + strings.ReplaceAll(strings.ToUpper(coalesce(req.Title, "NEW")), " ", "-")
		if len(code) > 60 {
			code = code[:60]
		}
	}
	row := model.AiEvalCase{
		Suite: coalesce(req.Suite, "default"), CaseCode: code, Title: req.Title,
		InputQuestion: q, ExpectKeywords: req.ExpectKeywords, ForbidKeywords: req.ForbidKeywords,
		ExpectTools: req.ExpectTools, ExpectRisk: req.ExpectRisk, ScoreWeight: req.ScoreWeight, Enabled: true,
	}
	if row.ScoreWeight <= 0 {
		row.ScoreWeight = 10
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.CreateEvalCase(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateEvalCase(ctx context.Context, id uint, req EvalCaseUpsertRequest) (*model.AiEvalCase, error) {
	row, err := s.repo.GetEvalCaseByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("评估用例不存在")
	}
	updates := map[string]any{
		"suite": coalesce(req.Suite, row.Suite), "title": req.Title,
		"input_question": strings.TrimSpace(req.InputQuestion),
		"expect_keywords": req.ExpectKeywords, "forbid_keywords": req.ForbidKeywords,
		"expect_tools": req.ExpectTools, "expect_risk": req.ExpectRisk,
	}
	if c := strings.TrimSpace(req.CaseCode); c != "" {
		updates["case_code"] = c
	}
	if req.ScoreWeight > 0 {
		updates["score_weight"] = req.ScoreWeight
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := s.repo.UpdateEvalCase(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.repo.GetEvalCaseByID(ctx, id)
}

func (s *Service) DeleteEvalCase(ctx context.Context, id uint) error {
	if err := s.repo.DeleteEvalCase(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("评估用例不存在")
		}
		return err
	}
	return nil
}
