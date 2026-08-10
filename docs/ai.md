# Yunshu AI 模块

## 概述

`ai` 插件：多模型 Provider、运维助手（Tool Calling + RAG）、场景分析、高危操作审批。`esmgmt` 插件：Elasticsearch 连接与集群管理。

## 启用

`plugins.enabled` 包含 `ai` / `esmgmt`；字典 `ai_enabled=true` 并配置 API Key。

## AI API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ai/status` | 状态（脱敏） |
| POST | `/api/v1/ai/ping` | 连通测试 |
| POST | `/api/v1/ai/chat` | 助手（tools + RAG） |
| POST | `/api/v1/ai/k8s/pod-diagnose` | Pod AI 分析 |
| POST | `/api/v1/ai/cicd/build-fail` | CI 失败分析 |
| POST | `/api/v1/ai/alert/explain` | 告警解释 |
| GET | `/api/v1/ai/approvals` | 审批列表 |
| POST | `/api/v1/ai/approvals/:id/review` | 审批 |
| POST | `/api/v1/ai/approvals/:id/execute` | 执行已批准操作 |
| POST | `/api/v1/ai/knowledge/sync` | 同步知识库到 ES |

Chat 默认只读工具；`enable_write_tools=true` 时写操作只创建审批单。

## 排障剧本

`internal/ai/runbooks/`：CrashLoopBackOff / ImagePullBackOff / PendingUnschedulable。

## RAG

优先 ES `yunshu-ai-kb-*`，失败回退内嵌文档关键词匹配。

## esmgmt

`/api/v1/esmgmt/*`：连接、health、索引、节点、受限 REST 代理。不替代日志平台检索。
