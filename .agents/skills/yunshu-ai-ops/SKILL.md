---
name: yunshu-ai-ops
description: Yunshu 平台 AI 运维助手与能力中心。覆盖模型/Prompt/RAG/Tool（builtin+script）/审批/Evaluation/data/ai 种子。修改 AI 接入、管理台或助手时使用。
---

# Yunshu AI Ops

## 范围

- 运行时：`internal/service/ai`（Chat、tools、rag、session、approval、script_runner、center_*）
- 数据：`data/ai/**` + MySQL 能力中心表（禁止业务内容 go:embed）
- API：`/api/v1/ai/*` 与 `/api/v1/ai/center/*`
- 前端：助手、审批、`ai-center-page`

## 约定

- Prompt/知识/案例/SOP/Tool/Eval 以 DB + `data/ai` 为准
- Tool：`builtin` 或 `script`（python27/go/shell），沙箱+风险+审批
- 默认只读；写工具 → `ai_tool_approvals`
- RAG：案例优先 → ES → 回退
- Chat Temperature≈0.2；禁止用户覆盖 system
- 种子覆盖见 `docs/ai.md` 矩阵；补充内容优先改 `data/ai/**` 后 reseed
