package ai

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// --- Prompt meta CRUD ---

type PromptUpsertRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name" binding:"required,max=128"`
	Type    string `json:"type"` // system|diagnosis|generation
	Scene   string `json:"scene"`
	Enabled *bool  `json:"enabled"`
	Remark  string `json:"remark"`
	// 创建时可同时写入首版内容
	Content   string `json:"content"`
	Changelog string `json:"changelog"`
}

type PromptDetail struct {
	model.AiPrompt
	CurrentVersion *model.AiPromptVersion `json:"current_version,omitempty"`
}

func (s *Service) GetPrompt(ctx context.Context, id uint) (*PromptDetail, error) {
	var row model.AiPrompt
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("Prompt 不存在")
	}
	out := &PromptDetail{AiPrompt: row}
	var ver model.AiPromptVersion
	if err := s.db.WithContext(ctx).Where("prompt_id = ? AND is_current = ?", id, true).First(&ver).Error; err == nil {
		out.CurrentVersion = &ver
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
	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Create(&row).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	content := strings.TrimSpace(req.Content)
	if content != "" {
		ver := model.AiPromptVersion{
			PromptID: row.ID, Version: 1, Content: content,
			Changelog: coalesce(req.Changelog, "create"), IsCurrent: true, CreatedBy: userID,
		}
		if err := tx.Create(&ver).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return s.GetPrompt(ctx, row.ID)
}

func (s *Service) UpdatePrompt(ctx context.Context, id uint, req PromptUpsertRequest) (*PromptDetail, error) {
	var row model.AiPrompt
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	// code 一般不改；若传入且不同则允许（需唯一）
	if c := strings.TrimSpace(req.Code); c != "" && c != row.Code {
		updates["code"] = c
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetPrompt(ctx, id)
}

func (s *Service) DeletePrompt(ctx context.Context, id uint) error {
	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Where("prompt_id = ?", id).Delete(&model.AiPromptVersion{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	res := tx.Delete(&model.AiPrompt{}, id)
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return constants.ErrNotFoundWithMsg("Prompt 不存在")
	}
	return tx.Commit().Error
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
	var row model.AiSOP
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("SOP 不存在")
	}
	return &row, nil
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateSOP(ctx context.Context, id uint, req SOPUpsertRequest) (*model.AiSOP, error) {
	var row model.AiSOP
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("SOP 不存在")
	}
	updates := map[string]any{
		"title":            strings.TrimSpace(req.Title),
		"scenario":         req.Scenario,
		"preconditions":    req.Preconditions,
		"input_params":     req.InputParams,
		"check_steps":      req.CheckSteps,
		"exec_steps":       req.ExecSteps,
		"verify_steps":     req.VerifySteps,
		"exception_handle": req.ExceptionHandle,
		"rollback":         req.Rollback,
		"risk":             req.Risk,
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteSOP(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AiSOP{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("SOP 不存在")
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
	var row model.AiIncidentCase
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("案例不存在")
	}
	return &row, nil
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateIncidentCase(ctx context.Context, id uint, req IncidentCaseUpsertRequest) (*model.AiIncidentCase, error) {
	var row model.AiIncidentCase
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteIncidentCase(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AiIncidentCase{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("案例不存在")
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
	KBID       uint    `json:"kb_id"`
	Title      string  `json:"title" binding:"required,max=256"`
	Source     string  `json:"source"`
	Version    string  `json:"version"`
	Content    string  `json:"content"`
	MetaJSON   string  `json:"meta_json"`
	Confidence *float64 `json:"confidence"`
	Enabled    *bool   `json:"enabled"`
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateKnowledgeBase(ctx context.Context, id uint, req KBUpsertRequest) (*model.AiKnowledgeBase, error) {
	var row model.AiKnowledgeBase
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteKnowledgeBase(ctx context.Context, id uint) error {
	tx := s.db.WithContext(ctx).Begin()
	var docs []model.AiKbDocument
	_ = tx.Where("kb_id = ?", id).Find(&docs).Error
	for _, d := range docs {
		_ = tx.Where("document_id = ?", d.ID).Delete(&model.AiKbChunk{}).Error
	}
	_ = tx.Where("kb_id = ?", id).Delete(&model.AiKbDocument{}).Error
	_ = tx.Where("kb_id = ?", id).Delete(&model.AiKbChunk{}).Error
	res := tx.Delete(&model.AiKnowledgeBase{}, id)
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return constants.ErrNotFoundWithMsg("知识库不存在")
	}
	return tx.Commit().Error
}

func (s *Service) ListKBDocuments(ctx context.Context, kbID uint) ([]model.AiKbDocument, error) {
	var rows []model.AiKbDocument
	q := s.db.WithContext(ctx).Order("id DESC")
	if kbID > 0 {
		q = q.Where("kb_id = ?", kbID)
	}
	err := q.Limit(500).Find(&rows).Error
	return rows, err
}

func (s *Service) GetKBDocument(ctx context.Context, id uint) (*model.AiKbDocument, error) {
	var row model.AiKbDocument
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("文档不存在")
	}
	return &row, nil
}

func (s *Service) CreateKBDocument(ctx context.Context, userID uint, req KBDocUpsertRequest) (*model.AiKbDocument, error) {
	if req.KBID == 0 || strings.TrimSpace(req.Title) == "" {
		return nil, constants.ErrBadRequestWithMsg("kb_id 与 title 必填")
	}
	var kb model.AiKnowledgeBase
	if err := s.db.WithContext(ctx).First(&kb, req.KBID).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateKBDocument(ctx context.Context, id uint, req KBDocUpsertRequest) (*model.AiKbDocument, error) {
	var row model.AiKbDocument
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	// 内容变更后清旧切片，等待重新 embed
	_ = s.db.WithContext(ctx).Where("document_id = ?", id).Delete(&model.AiKbChunk{}).Error
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteKBDocument(ctx context.Context, id uint) error {
	tx := s.db.WithContext(ctx).Begin()
	_ = tx.Where("document_id = ?", id).Delete(&model.AiKbChunk{}).Error
	res := tx.Delete(&model.AiKbDocument{}, id)
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return constants.ErrNotFoundWithMsg("文档不存在")
	}
	return tx.Commit().Error
}

// --- Tool full update / create script / delete ---

type ToolUpsertRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Module              string `json:"module"`
	Runtime             string `json:"runtime"` // builtin|script
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
	var row model.AiToolDef
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("工具不存在")
	}
	return &row, nil
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateTool(ctx context.Context, id uint, req ToolUpsertRequest) (*model.AiToolDef, error) {
	var row model.AiToolDef
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteTool(ctx context.Context, id uint) error {
	var row model.AiToolDef
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return constants.ErrNotFoundWithMsg("工具不存在")
	}
	if strings.EqualFold(row.Runtime, "builtin") {
		return constants.ErrBadRequestWithMsg("内置工具不可删除，可禁用")
	}
	return s.db.WithContext(ctx).Delete(&row).Error
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
	var row model.AiEvalCase
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("评估用例不存在")
	}
	return &row, nil
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateEvalCase(ctx context.Context, id uint, req EvalCaseUpsertRequest) (*model.AiEvalCase, error) {
	var row model.AiEvalCase
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).First(&row, id).Error
	return &row, nil
}

func (s *Service) DeleteEvalCase(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AiEvalCase{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("评估用例不存在")
	}
	return nil
}
