---
name: yunshu-ai-ops
description: Yunshu 平台 AI 运维助手相关开发与排障。覆盖 ai 插件、字典配置（ai_*）、OpenAI 兼容/Anthropic 客户端、运维助手 Chat、K8s/CI/CD/告警 AI 分析 API 与前端入口。在修改 AI 接入、Prompt、权限菜单或助手 UI 时使用。
---

# Yunshu AI Ops

## 范围

- 插件：`internal/plugins/ai`
- 配置：`internal/config/ai.go` + 字典 `ai_*`（`internal/dictconfig/ai.go`）
- 客户端：`internal/pkg/llm`
- 服务：`internal/service/ai`
- Prompt：`internal/ai/prompts/`（embed）
- API：
  - `/api/v1/ai/status|ping|chat`
  - `/api/v1/ai/k8s/pod-diagnose`
  - `/api/v1/ai/cicd/build-fail`
  - `/api/v1/ai/alert/explain`
- 前端：
  - `web/src/pages/ai-assistant-page.tsx`
  - Pod 排障 Drawer「AI 分析」
  - CI 打包记录「AI 分析」
  - 告警指纹追溯 Modal「AI 解释」
  - `web/src/services/ai.ts`

## 约定

- 总开关：`ai_enabled`；未启用时 API 返回明确中文错误
- 禁止用户消息覆盖 system prompt
- 不自动执行集群/流水线/告警配置写操作；只分析与建议
- CI 分析需项目 CI/CD 访问权限（`AssertCicdAccess`）
- 密钥走字典 mask；seed 默认密钥项 `status=0`
- 审计：记录 scene / provider / token 用量（slog）

## 改动检查清单

1. 字典类型是否已 seed / mask / category / `dict-types.ts`
2. Wire：`provideAIConfig`、`aisvc.NewService`（含 cicd/alert 依赖）、`NewAIHandler`、`RouteDeps.aiHandler`
3. 权限 seed：`/api/v1/ai/*`
4. 菜单：`/ai/assistant` + 插件 path `/ai`
5. `go generate` 更新 `wire_gen.go` 后能编译
