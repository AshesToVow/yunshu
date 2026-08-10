# Yunshu AI 模块

## 概述

`ai` 为编译期业务插件：通过数据字典配置多模型 Provider 与总开关，提供运维助手对话、K8s Pod 排障、CI 构建失败分析与告警解释。不自研 Agent/RAG 框架，仅封装 OpenAI 兼容 Chat Completions 与 Anthropic Messages。

## 启用插件

`configs/config.yaml`：

```yaml
plugins:
  enabled:
    # ...
    - ai

ai:
  enabled: false
  default_provider: openai_compat
  timeout_sec: 60
  max_tokens: 2048
  openai:
    base_url: https://api.openai.com/v1
    api_key: ""
    model: gpt-4o-mini
  deepseek:
    base_url: https://api.deepseek.com/v1
    api_key: ""
    model: deepseek-chat
  anthropic:
    base_url: https://api.anthropic.com
    api_key: ""
    model: claude-sonnet-4-20250514
```

字典 `ai_*`（`status=1`）优先覆盖 YAML。

## 字典清单

| dict_type | 说明 |
|-----------|------|
| `ai_enabled` | 总开关 true/false |
| `ai_default_provider` | `openai_compat` / `deepseek` / `anthropic` |
| `ai_timeout_sec` / `ai_max_tokens` | 调用参数 |
| `ai_openai_base_url` / `ai_openai_api_key` / `ai_openai_model` | OpenAI 兼容（通义/vLLM/LiteLLM 等也可填此组） |
| `ai_deepseek_*` | DeepSeek |
| `ai_anthropic_*` | Claude（Anthropic Messages） |

密钥类 seed 默认 `status=0`；启用后填写并设 `status=1`。

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ai/status` | 启用状态、默认 Provider、模型（脱敏） |
| POST | `/api/v1/ai/ping` | 连通测试 |
| POST | `/api/v1/ai/chat` | 运维助手 |
| POST | `/api/v1/ai/k8s/pod-diagnose` | Pod 诊断 AI 分析 |
| POST | `/api/v1/ai/cicd/build-fail` | CI 构建失败 AI 分析 |
| POST | `/api/v1/ai/alert/explain` | 告警指纹投递 AI 解释 |

未启用 `ai_enabled` 时返回明确错误。

### CI 构建失败

请求体：`project_id`、`run_id`（可选 `provider`）。后端拉取构建元数据、阶段摘要与 Console 尾部后分析。

### 告警解释

请求体：`fingerprint`（必填），可选 `project_id`、`window_hours`。复用确定性指纹追溯，并附带质量报告摘要。

## 安全边界

- 只分析与建议，不自动执行 kubectl / Jenkins / 告警配置变更
- Prompt 不注入密钥；日志与 JSON 有截断
- 审计记录场景、Provider、token 用量概要

## Prompt 与 Skill

- 运行时模板：`internal/ai/prompts/`（embed）
- 研发 Agent Skill：`.agents/skills/yunshu-ai-ops`、`.agents/skills/yunshu-k8s-diagnose`
