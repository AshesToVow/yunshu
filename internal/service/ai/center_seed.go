package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"yunshu/internal/model"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// EnsureCenterSeed 从 data/ai 导入种子（幂等：已有同 code/name 则跳过）。
func (s *Service) EnsureCenterSeed(ctx context.Context) error {
	root := s.dataRoot()
	if err := s.seedPromptsFromDir(ctx, filepath.Join(root, "prompts")); err != nil {
		slog.Warn("ai seed prompts", "err", err)
	}
	if err := s.seedKnowledgeFromDir(ctx, filepath.Join(root, "kb")); err != nil {
		slog.Warn("ai seed kb", "err", err)
	}
	if err := s.seedCasesFromDir(ctx, filepath.Join(root, "cases")); err != nil {
		slog.Warn("ai seed cases", "err", err)
	}
	if err := s.seedSOPsFromDir(ctx, filepath.Join(root, "sops")); err != nil {
		slog.Warn("ai seed sops", "err", err)
	}
	if err := s.seedToolsFromDir(ctx, filepath.Join(root, "tools")); err != nil {
		slog.Warn("ai seed tools", "err", err)
	}
	if err := s.seedBuiltinToolDefs(ctx); err != nil {
		slog.Warn("ai seed builtin tools", "err", err)
	}
	if err := s.seedEvalFromDir(ctx, filepath.Join(root, "eval")); err != nil {
		slog.Warn("ai seed eval", "err", err)
	}
	return nil
}

func (s *Service) dataRoot() string {
	if s.dataDir != "" {
		return s.dataDir
	}
	return filepath.Join("data", "ai")
}

func (s *Service) seedPromptsFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		code := e.Name() // system_ops-agent → normalize
		code = strings.ReplaceAll(code, "_", "/")
		name := code
		typ := "system"
		scene := ""
		switch {
		case strings.HasPrefix(code, "system/"):
			typ = "system"
		case strings.HasPrefix(code, "diagnosis/"):
			typ = "diagnosis"
			scene = strings.TrimPrefix(code, "diagnosis/")
		}
		var prompt model.AiPrompt
		err := s.db.WithContext(ctx).Where("code = ?", code).First(&prompt).Error
		if err == gorm.ErrRecordNotFound {
			prompt = model.AiPrompt{Code: code, Name: name, Type: typ, Scene: scene, Enabled: true}
			if err := s.db.WithContext(ctx).Create(&prompt).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		var verCount int64
		_ = s.db.WithContext(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", prompt.ID).Count(&verCount).Error
		if verCount > 0 {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name(), "v1.md"))
		if err != nil {
			continue
		}
		ver := model.AiPromptVersion{
			PromptID:  prompt.ID,
			Version:   1,
			Content:   string(body),
			Changelog: "seed",
			IsCurrent: true,
		}
		if err := s.db.WithContext(ctx).Create(&ver).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) seedKnowledgeFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		code := e.Name()
		var kb model.AiKnowledgeBase
		err := s.db.WithContext(ctx).Where("code = ?", code).First(&kb).Error
		if err == gorm.ErrRecordNotFound {
			cat := strings.TrimPrefix(code, "kb_")
			kb = model.AiKnowledgeBase{Code: code, Name: code, Category: cat, Enabled: true}
			if err := s.db.WithContext(ctx).Create(&kb).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		files, _ := os.ReadDir(filepath.Join(dir, e.Name()))
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
				continue
			}
			src := code + "/" + f.Name()
			var cnt int64
			_ = s.db.WithContext(ctx).Model(&model.AiKbDocument{}).Where("kb_id = ? AND source = ?", kb.ID, src).Count(&cnt).Error
			if cnt > 0 {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name(), f.Name()))
			if err != nil {
				continue
			}
			doc := model.AiKbDocument{
				KBID: kb.ID, Title: f.Name(), Source: src, Version: "v1",
				Enabled: true, Confidence: 0.8, Content: string(raw),
			}
			if err := s.db.WithContext(ctx).Create(&doc).Error; err != nil {
				return err
			}
			_ = s.rechunkDocument(ctx, &doc)
		}
	}
	return nil
}

type caseYAML struct {
	CaseID       string  `yaml:"case_id"`
	Title        string  `yaml:"title"`
	Category     string  `yaml:"category"`
	Technology   string  `yaml:"technology"`
	Symptom      string  `yaml:"symptom"`
	Environment  string  `yaml:"environment"`
	Diagnosis    string  `yaml:"diagnosis"`
	RootCause    string  `yaml:"root_cause"`
	Solution     string  `yaml:"solution"`
	Verification string  `yaml:"verification"`
	Risk         string  `yaml:"risk"`
	RelatedTools string  `yaml:"related_tools"`
	RelatedSOP   string  `yaml:"related_sop"`
	Source       string  `yaml:"source"`
	Confidence   float64 `yaml:"confidence"`
}

func (s *Service) seedCasesFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var c caseYAML
		if err := yaml.Unmarshal(raw, &c); err != nil {
			continue
		}
		if c.CaseID == "" {
			continue
		}
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiIncidentCase{}).Where("case_id = ?", c.CaseID).Count(&cnt).Error
		if cnt > 0 {
			continue
		}
		row := model.AiIncidentCase{
			CaseID: c.CaseID, Title: c.Title, Category: c.Category, Technology: c.Technology,
			Symptom: c.Symptom, Environment: c.Environment, Diagnosis: c.Diagnosis,
			RootCause: c.RootCause, Solution: c.Solution, Verification: c.Verification,
			Risk: c.Risk, RelatedTools: c.RelatedTools, RelatedSOP: c.RelatedSOP,
			Source: c.Source, Confidence: c.Confidence, Enabled: true,
		}
		if row.Confidence <= 0 {
			row.Confidence = 0.8
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type sopYAML struct {
	Code            string `yaml:"code"`
	Title           string `yaml:"title"`
	Scenario        string `yaml:"scenario"`
	Preconditions   string `yaml:"preconditions"`
	InputParams     string `yaml:"input_params"`
	CheckSteps      string `yaml:"check_steps"`
	ExecSteps       string `yaml:"exec_steps"`
	VerifySteps     string `yaml:"verify_steps"`
	ExceptionHandle string `yaml:"exception_handle"`
	Rollback        string `yaml:"rollback"`
	Risk            string `yaml:"risk"`
	ApprovalNeeded  bool   `yaml:"approval_needed"`
}

func (s *Service) seedSOPsFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var c sopYAML
		if err := yaml.Unmarshal(raw, &c); err != nil {
			continue
		}
		if c.Code == "" {
			continue
		}
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiSOP{}).Where("code = ?", c.Code).Count(&cnt).Error
		if cnt > 0 {
			continue
		}
		row := model.AiSOP{
			Code: c.Code, Title: c.Title, Scenario: c.Scenario, Preconditions: c.Preconditions,
			InputParams: c.InputParams, CheckSteps: c.CheckSteps, ExecSteps: c.ExecSteps,
			VerifySteps: c.VerifySteps, ExceptionHandle: c.ExceptionHandle, Rollback: c.Rollback,
			Risk: c.Risk, ApprovalNeeded: c.ApprovalNeeded, Enabled: true,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type toolYAML struct {
	Name                string         `yaml:"name"`
	Description         string         `yaml:"description"`
	Runtime             string         `yaml:"runtime"`
	ScriptLang          string         `yaml:"script_lang"`
	Entry               string         `yaml:"entry"`
	Module              string         `yaml:"module"`
	Permission          string         `yaml:"permission"`
	RiskLevel           string         `yaml:"risk_level"`
	TimeoutSec          int            `yaml:"timeout_sec"`
	RequireConfirmation bool           `yaml:"require_confirmation"`
	InputSchema         map[string]any `yaml:"input_schema"`
}

func (s *Service) seedToolsFromDir(ctx context.Context, dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "tool.yaml" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var t toolYAML
		if err := yaml.Unmarshal(raw, &t); err != nil || t.Name == "" {
			return nil
		}
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiToolDef{}).Where("name = ?", t.Name).Count(&cnt).Error
		if cnt > 0 {
			return nil
		}
		relDir, _ := filepath.Rel(s.dataRoot(), filepath.Dir(path))
		schema, _ := json.Marshal(t.InputSchema)
		row := model.AiToolDef{
			Name: t.Name, Description: t.Description, Module: t.Module,
			Runtime: "script", ScriptLang: t.ScriptLang,
			ScriptPath: filepath.ToSlash(filepath.Join(relDir, t.Entry)),
			TimeoutSec: t.TimeoutSec, InputSchemaJSON: string(schema),
			Permission: coalesce(t.Permission, "READ_ONLY"),
			RiskLevel:  coalesce(t.RiskLevel, "LOW"),
			RequireConfirmation: t.RequireConfirmation,
			AuditRequired: true, Enabled: true,
		}
		if row.TimeoutSec <= 0 {
			row.TimeoutSec = 30
		}
		return s.db.WithContext(ctx).Create(&row).Error
	})
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) == "" {
		return b
	}
	return a
}

func (s *Service) seedBuiltinToolDefs(ctx context.Context) error {
	builtins := []model.AiToolDef{
		{Name: "list_clusters", Description: "列出 K8s 集群", Module: "k8s", Runtime: "builtin", HandlerKey: "list_clusters", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "list_pods", Description: "列出 Pod", Module: "k8s", Runtime: "builtin", HandlerKey: "list_pods", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "get_pod_detail", Description: "Pod 详情", Module: "k8s", Runtime: "builtin", HandlerKey: "get_pod_detail", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "get_pod_logs", Description: "Pod 日志", Module: "k8s", Runtime: "builtin", HandlerKey: "get_pod_logs", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 60},
		{Name: "diagnose_pod", Description: "Pod 诊断", Module: "k8s", Runtime: "builtin", HandlerKey: "diagnose_pod", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 60},
		{Name: "run_diagnose_runbook", Description: "排障剧本", Module: "k8s", Runtime: "builtin", HandlerKey: "run_diagnose_runbook", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 60},
		{Name: "list_deployments", Description: "列出 Deployment", Module: "k8s", Runtime: "builtin", HandlerKey: "list_deployments", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "list_namespaces", Description: "列出命名空间", Module: "k8s", Runtime: "builtin", HandlerKey: "list_namespaces", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "list_events", Description: "列出 Events", Module: "k8s", Runtime: "builtin", HandlerKey: "list_events", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "list_runbooks", Description: "列出剧本", Module: "k8s", Runtime: "builtin", HandlerKey: "list_runbooks", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 10},
		{Name: "search_logs", Description: "检索项目日志", Module: "log", Runtime: "builtin", HandlerKey: "search_logs", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 60},
		{Name: "list_cicd_builds", Description: "列出构建", Module: "cicd", Runtime: "builtin", HandlerKey: "list_cicd_builds", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "get_cicd_build", Description: "构建详情", Module: "cicd", Runtime: "builtin", HandlerKey: "get_cicd_build", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "get_cicd_build_log", Description: "构建日志", Module: "cicd", Runtime: "builtin", HandlerKey: "get_cicd_build_log", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 60},
		{Name: "list_alerts", Description: "列出告警", Module: "alert", Runtime: "builtin", HandlerKey: "list_alerts", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "explain_alert", Description: "解释告警投递", Module: "alert", Runtime: "builtin", HandlerKey: "explain_alert", Permission: "READ_ONLY", RiskLevel: "LOW", Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "scale_deployment", Description: "扩缩容（审批）", Module: "k8s", Runtime: "builtin", HandlerKey: "scale_deployment", Permission: "WRITE", RiskLevel: "HIGH", RequireConfirmation: true, Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "restart_deployment", Description: "重启 Deployment（审批）", Module: "k8s", Runtime: "builtin", HandlerKey: "restart_deployment", Permission: "WRITE", RiskLevel: "HIGH", RequireConfirmation: true, Enabled: true, AuditRequired: true, TimeoutSec: 30},
		{Name: "delete_pod", Description: "删除 Pod（审批）", Module: "k8s", Runtime: "builtin", HandlerKey: "delete_pod", Permission: "WRITE", RiskLevel: "HIGH", RequireConfirmation: true, Enabled: true, AuditRequired: true, TimeoutSec: 30},
	}
	for _, b := range builtins {
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiToolDef{}).Where("name = ?", b.Name).Count(&cnt).Error
		if cnt > 0 {
			continue
		}
		// 填入与 tools.go 一致的 schema 可后续完善；Chat 仍用代码侧完整 schema
		if err := s.db.WithContext(ctx).Create(&b).Error; err != nil {
			return err
		}
	}
	return nil
}

type evalYAML struct {
	CaseCode       string `yaml:"case_code"`
	Title          string `yaml:"title"`
	Suite          string `yaml:"suite"`
	InputQuestion  string `yaml:"input_question"`
	ExpectKeywords string `yaml:"expect_keywords"`
	ForbidKeywords string `yaml:"forbid_keywords"`
	ExpectTools    string `yaml:"expect_tools"`
	ExpectRisk     string `yaml:"expect_risk"`
	ScoreWeight    int    `yaml:"score_weight"`
}

func (s *Service) seedEvalFromDir(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var c evalYAML
		if err := yaml.Unmarshal(raw, &c); err != nil || c.CaseCode == "" {
			continue
		}
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiEvalCase{}).Where("case_code = ?", c.CaseCode).Count(&cnt).Error
		if cnt > 0 {
			continue
		}
		suite := c.Suite
		if suite == "" {
			suite = "default"
		}
		w := c.ScoreWeight
		if w <= 0 {
			w = 10
		}
		row := model.AiEvalCase{
			Suite: suite, CaseCode: c.CaseCode, Title: c.Title, InputQuestion: c.InputQuestion,
			ExpectKeywords: c.ExpectKeywords, ForbidKeywords: c.ForbidKeywords,
			ExpectTools: c.ExpectTools, ExpectRisk: c.ExpectRisk, ScoreWeight: w, Enabled: true,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
