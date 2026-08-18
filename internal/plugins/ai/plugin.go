package ai

import (
	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "ai" }
func (m *module) Description() string { return "AI 运维助手：多模型接入、K8s 排障增强、对话分析" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/ai"},
		APIPrefixes:      []string{"/api/v1/ai"},
		DependsOn:        []string{},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.AiToolApproval{},
		&model.AiChatSession{},
		&model.AiChatMessage{},
		&model.AiLLMModel{},
		&model.AiPrompt{},
		&model.AiPromptVersion{},
		&model.AiKnowledgeBase{},
		&model.AiKbDocument{},
		&model.AiKbChunk{},
		&model.AiIncidentCase{},
		&model.AiSOP{},
		&model.AiToolDef{},
		&model.AiEvalCase{},
		&model.AiEvalRun{},
		&model.AiEvalResult{},
		&model.AiAuditEvent{},
	}
}
