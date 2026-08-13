# Yunshu AI 模块

## 概述

`ai` 插件：多模型 Provider、运维助手（Tool Calling + RAG）、场景分析、高危操作审批、MySQL 会话持久化。`esmgmt` 插件：Elasticsearch 连接与集群管理。

## 启用

`plugins.enabled` 包含 `ai` / `esmgmt`；字典 `ai_enabled=true` 并配置 API Key。

## AI API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ai/status` | 状态（脱敏） |
| POST | `/api/v1/ai/ping` | 连通测试 |
| POST | `/api/v1/ai/chat` | 助手（tools + RAG；可带 `session_id` 落库） |
| GET | `/api/v1/ai/sessions` | 会话列表 |
| POST | `/api/v1/ai/sessions` | 创建会话 |
| GET | `/api/v1/ai/sessions/:id` | 会话详情（含消息） |
| PATCH | `/api/v1/ai/sessions/:id` | 更新会话元数据 |
| DELETE | `/api/v1/ai/sessions/:id` | 删除会话 |
| POST | `/api/v1/ai/sessions/:id/clear` | 清空会话消息 |
| POST | `/api/v1/ai/k8s/pod-diagnose` | Pod AI 分析 |
| POST | `/api/v1/ai/cicd/build-fail` | CI 失败分析 |
| POST | `/api/v1/ai/alert/explain` | 告警解释 |
| GET | `/api/v1/ai/approvals` | 审批列表 |
| POST | `/api/v1/ai/approvals/:id/review` | 审批 |
| POST | `/api/v1/ai/approvals/:id/execute` | 执行已批准操作 |
| POST | `/api/v1/ai/knowledge/sync` | 同步知识库到 ES |

Chat 默认只读工具；`enable_write_tools=true` 时写操作只创建审批单。未传 `session_id` 时会自动创建会话并返回 `session_id`。

## 会话持久化（MySQL）

- 表：`ai_chat_sessions`、`ai_chat_messages`（GORM AutoMigrate）
- 按 `user_id` 隔离；每轮对话追加 user/assistant 消息
- 助手页以服务端会话为准（跨浏览器/设备可恢复）

## 工具（按模块）

| 模块 | 只读工具 | 写工具（审批） |
|------|----------|----------------|
| K8s | `list_clusters` `list_namespaces` `list_pods` `get_pod_detail` `get_pod_logs` `diagnose_pod` `run_diagnose_runbook` `list_deployments` `list_events` `list_runbooks` | `scale_deployment` `restart_deployment` `delete_pod` |
| 日志 | `search_logs` | — |
| CI/CD | `list_cicd_builds` `get_cicd_build` `get_cicd_build_log` | — |
| 告警 | `list_alerts` `explain_alert` | — |

## 排障剧本

`internal/ai/runbooks/`：CrashLoopBackOff / ImagePullBackOff / PendingUnschedulable。

## RAG（按功能模块）

- 内嵌文档：`internal/ai/knowledge`（ai / k8s / cicd / alert / log / esmgmt / cmdb / dbmgmt）+ runbooks + prompts
- 检索时按问题关键词推断模块并加权
- ES 索引 `yunshu-ai-kb-*`（同步写入 `yunshu-ai-kb-v1`，含 `module` 字段）；失败回退内嵌关键词匹配

## esmgmt

`/api/v1/esmgmt/*`：连接、health、索引、节点、受限 REST 代理。不替代日志平台检索。
