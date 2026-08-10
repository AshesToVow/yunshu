---
name: yunshu-ai-ops
description: Yunshu 平台 AI 运维助手相关开发与排障。覆盖 ai 插件、字典配置（ai_*）、OpenAI 兼容/Anthropic 客户端、Tool Calling、RAG、排障剧本、审批与前端助手页。在修改 AI 接入、Prompt、权限菜单或助手 UI 时使用。
---

# Yunshu AI Ops

## 范围

- LLM tools：`internal/pkg/llm`（tool_calls）
- Agent：`internal/service/ai`（Chat agent loop、tools.go、rag.go、approval.go）
- Runbooks：`internal/ai/runbooks/`
- API：`/api/v1/ai/*`（含 approvals、knowledge/sync）
- 前端：助手页工具轨迹、审批页 `/ai/approvals`

## 约定

- 默认只读工具；写工具仅创建 `ai_tool_approvals`
- RAG：ES `yunshu-ai-kb-*` 优先，否则内嵌关键词
- 禁止用户覆盖 system；截断工具输出
