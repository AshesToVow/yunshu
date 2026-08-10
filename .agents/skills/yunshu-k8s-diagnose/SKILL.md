---
name: yunshu-k8s-diagnose
description: Yunshu K8s Pod 排障诊断与 AI 增强。覆盖确定性 Diagnose API、AI 分析接口、Prompt 模板与 Pod 页「AI 分析」交互。在改 Pod 排障逻辑、诊断上下文或 AI 分析展示时使用。
---

# Yunshu K8s Diagnose

## 数据流

1. 确定性诊断：`K8sPodService.Diagnose` → 容器状态 / Events / 日志片段 / hints
2. AI 增强：`AIService.AnalyzePodDiagnose` 复用 Diagnose 结果，填充 `k8s_pod_diagnose` Prompt，解析 JSON（`ai_summary` / `root_causes` / `actions`）
3. 前端：Pod 排障 Drawer 先展示确定性结果，再可选「AI 分析」

## 关键文件

- `internal/service/k8s/k8s_pod_diagnose.go`
- `internal/service/ai/service.go`（`AnalyzePodDiagnose`）
- `internal/ai/prompts/k8s_pod_diagnose.md`
- `web/src/pages/pod-page.tsx`
- `web/src/services/ai.ts`（`analyzePodDiagnoseAI`）

## 约定

- AI 失败不得吞掉已有 Diagnose 结果；前端应保留确定性诊断
- Prompt 上下文截断（避免超大 JSON）
- 输出优先结构化 JSON；解析失败时回退展示 `raw_reply` / 原 summary
- 权限：`POST /api/v1/ai/k8s/pod-diagnose`；依赖 `ai` + `k8s` 插件
