package cmd

import "yunshu/internal/model"

func seedPermissionsAI() []model.Permission {
	return []model.Permission{
		{Name: "AI 状态查询", Resource: "/api/v1/ai/status", Action: "GET", Description: "Get AI status"},
		{Name: "AI 连通测试", Resource: "/api/v1/ai/ping", Action: "POST", Description: "Ping AI provider"},
		{Name: "AI 运维对话", Resource: "/api/v1/ai/chat", Action: "POST", Description: "AI ops assistant chat"},
		{Name: "AI 运维对话流式", Resource: "/api/v1/ai/chat/stream", Action: "POST", Description: "AI ops assistant chat SSE progress"},
		{Name: "AI 会话列表", Resource: "/api/v1/ai/sessions", Action: "GET", Description: "List AI chat sessions"},
		{Name: "AI 创建会话", Resource: "/api/v1/ai/sessions", Action: "POST", Description: "Create AI chat session"},
		{Name: "AI 会话详情", Resource: "/api/v1/ai/sessions/:id", Action: "GET", Description: "Get AI chat session"},
		{Name: "AI 更新会话", Resource: "/api/v1/ai/sessions/:id", Action: "PATCH", Description: "Update AI chat session"},
		{Name: "AI 删除会话", Resource: "/api/v1/ai/sessions/:id", Action: "DELETE", Description: "Delete AI chat session"},
		{Name: "AI 清空会话消息", Resource: "/api/v1/ai/sessions/:id/clear", Action: "POST", Description: "Clear AI chat session messages"},
		{Name: "AI 发起调查", Resource: "/api/v1/ai/investigations", Action: "POST", Description: "Start AI investigation"},
		{Name: "AI 调查列表", Resource: "/api/v1/ai/investigations", Action: "GET", Description: "List AI investigations"},
		{Name: "AI 调查详情", Resource: "/api/v1/ai/investigations/:id", Action: "GET", Description: "Get AI investigation"},
		{Name: "AI Pod 排障分析", Resource: "/api/v1/ai/k8s/pod-diagnose", Action: "POST", Description: "AI analyze pod diagnose"},
		{Name: "AI CI 构建失败分析", Resource: "/api/v1/ai/cicd/build-fail", Action: "POST", Description: "AI analyze CI build failure"},
		{Name: "AI 告警解释", Resource: "/api/v1/ai/alert/explain", Action: "POST", Description: "AI explain alert fingerprint delivery"},
		{Name: "AI 审批列表", Resource: "/api/v1/ai/approvals", Action: "GET", Description: "List AI tool approvals"},
		{Name: "AI 审批审核", Resource: "/api/v1/ai/approvals/:id/review", Action: "POST", Description: "Review AI tool approval"},
		{Name: "AI 审批执行", Resource: "/api/v1/ai/approvals/:id/execute", Action: "POST", Description: "Execute approved AI tool"},
		{Name: "AI 知识库同步", Resource: "/api/v1/ai/knowledge/sync", Action: "POST", Description: "Sync AI knowledge base to ES"},
		{Name: "AI 知识库向量化", Resource: "/api/v1/ai/knowledge/embed", Action: "POST", Description: "Sync AI knowledge chunk embeddings"},
		{Name: "AI 能力中心概览", Resource: "/api/v1/ai/center/overview", Action: "GET", Description: "AI capability center overview"},
		{Name: "AI 能力中心重载种子", Resource: "/api/v1/ai/center/reseed", Action: "POST", Description: "Reseed AI center from data/ai"},
		{Name: "AI Prompt 列表", Resource: "/api/v1/ai/center/prompts", Action: "GET", Description: "List AI prompts"},
		{Name: "AI Prompt 版本", Resource: "/api/v1/ai/center/prompts/:id/versions", Action: "GET", Description: "List prompt versions"},
		{Name: "AI Prompt 发布", Resource: "/api/v1/ai/center/prompts/:id/publish", Action: "POST", Description: "Publish prompt version"},
		{Name: "AI Prompt 回滚", Resource: "/api/v1/ai/center/prompts/:id/versions/:vid/rollback", Action: "POST", Description: "Rollback prompt version"},
		{Name: "AI 模型列表", Resource: "/api/v1/ai/center/models", Action: "GET", Description: "List LLM models"},
		{Name: "AI 创建模型", Resource: "/api/v1/ai/center/models", Action: "POST", Description: "Create LLM model"},
		{Name: "AI 更新模型", Resource: "/api/v1/ai/center/models/:id", Action: "PUT", Description: "Update LLM model"},
		{Name: "AI 删除模型", Resource: "/api/v1/ai/center/models/:id", Action: "DELETE", Description: "Delete LLM model"},
		{Name: "AI 设默认模型", Resource: "/api/v1/ai/center/models/:id/default", Action: "POST", Description: "Set default LLM model"},
		{Name: "AI Tool 列表", Resource: "/api/v1/ai/center/tools", Action: "GET", Description: "List AI tools"},
		{Name: "AI Tool 更新", Resource: "/api/v1/ai/center/tools/:id", Action: "PATCH", Description: "Update AI tool"},
		{Name: "AI 故障案例", Resource: "/api/v1/ai/center/cases", Action: "GET", Description: "List incident cases"},
		{Name: "AI SOP 列表", Resource: "/api/v1/ai/center/sops", Action: "GET", Description: "List SOPs"},
		{Name: "AI 知识库列表", Resource: "/api/v1/ai/center/knowledge-bases", Action: "GET", Description: "List knowledge bases"},
		{Name: "AI 评估用例", Resource: "/api/v1/ai/center/eval/cases", Action: "GET", Description: "List eval cases"},
		{Name: "AI 评估运行", Resource: "/api/v1/ai/center/eval/run", Action: "POST", Description: "Run eval suite"},
	}
}

