# Yunshu AI 模块

## 概述

`ai` 插件：多模型、运维助手（Tool Calling + RAG + SSE 进度）、**调查闭环**、场景分析、高危审批（含 Policy）、**AI 运维能力中心**（Prompt/知识/案例/SOP/Tool/Evaluation）。

## 启用

`plugins.enabled` 含 `ai`；字典 `ai_enabled=true` 并配置 API Key（或能力中心 `ai_llm_models`）。

## 能力中心（去硬编码）

权威数据在 **MySQL + `data/ai/**`**：

| 路径 | 内容 |
|------|------|
| `data/ai/prompts/` | Prompt 版本种子 |
| `data/ai/kb/` | 知识库 Markdown |
| `data/ai/cases/` | 故障案例 YAML |
| `data/ai/sops/` | SOP YAML |
| `data/ai/tools/` | 脚本工具（`tool.yaml` + `run.py`/`run.sh`） |
| `data/ai/eval/` | Evaluation 用例 |

首次对话或访问能力中心会幂等 seed。也可 `POST /api/v1/ai/center/reseed`。

**禁止**再把业务 Prompt/知识 `go:embed` 进运行时（旧 embed 仅作空库过渡回退）。

## 菜单

- `/ai/assistant` 运维助手（SSE 工具轨迹）
- `/ai/investigations` AI 调查记录
- `/ai/approvals` 操作审批
- `/ai/center` 能力中心（Prompt/Tools/案例/SOP/KB/Eval/向量化）

## 主要 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/ai/chat` | 助手（同步） |
| POST | `/api/v1/ai/chat/stream` | 助手 SSE 进度（progress/rag/tool/reply/done） |
| POST | `/api/v1/ai/k8s/generate-yaml` | 按自然语言生成 K8s YAML（创建页回填，不直接 apply） |
| POST | `/api/v1/ai/investigations` | 发起调查（alert\|pod\|cicd\|chat） |
| GET | `/api/v1/ai/investigations` | 调查列表 |
| GET | `/api/v1/ai/investigations/:id` | 调查详情 |
| POST | `/api/v1/ai/knowledge/sync` | DB→ES 同步 |
| POST | `/api/v1/ai/knowledge/embed` | KB chunks 向量化（混合 RAG） |
| GET | `/api/v1/ai/center/overview` | 能力中心统计 |
| POST | `/api/v1/ai/center/reseed` | 从 data/ai 重载种子 |

另有 sessions、approvals、场景分析（pod-diagnose / build-fail / alert-explain）。

## 业务入口

| 页面 | 能力 |
|------|------|
| 告警历史 | AI解读（投递解释）+ **AI调查**（完整闭环） |
| Pod 排障抽屉 | AI 分析 + **AI 调查** |
| CI 构建详情 | AI 分析 + **AI 调查** |
| K8s 资源创建（YamlCrud / Pod） | **AI 生成 YAML**（描述→编辑器，人工核对后 apply） |
| 运维助手 | 流式对话 + 工具/RAG 证据面板 |

## Tool 运行时

- `builtin`：K8s / 日志 / CI / 告警 / **CMDB 服务器** / **DB 实例** / **ES 连接**
- `script`：沙箱 `data/ai/tools`；写操作走审批
- **Policy**：扩缩容 replicas>50 需 reason 含 `emergency`；禁止删 `kube-system`/`kube-public`/`yunshu-logging` Pod；写操作须有 namespace

## RAG

1. DB 案例 + chunks + SOP（词法；有 Embedding 时混合语义）
2. ES `yunshu-ai-kb-*`
3. 回退内嵌模块文档

能力中心「向量化 KB Chunks」写入 `ai_kb_chunks.embedding`。

## 种子覆盖矩阵

| 模块 | 案例 | SOP | KB | 脚本工具 | Builtin |
|------|------|-----|----|----------|---------|
| k8s | CrashLoop/ImagePull/Pending/OOM | ✓ | kb_k8s | — | 强 |
| cicd | 构建失败 | ✓ | kb_cicd | — | list/get/log |
| alert | 未收到 | ✓ | kb_alert | — | list/explain |
| log | 检索为空 | ✓ | kb_log | — | search_logs |
| linux | 磁盘打满 | ✓ | kb_linux | disk/mem/load | — |
| cmdb | 服务器离线 | ✓ | kb_cmdb | — | list_servers/get_server |
| dbmgmt | 连接失败 | ✓ | — | — | list_db_instances |
| esmgmt | 连接不可达 | ✓ | kb_esmgmt | — | list_es_connections |

## 部署注意

1. 重启服务 AutoMigrate（含 `ai_investigations`）；启动 PostMigrate 会将 AI 文本表转为 **utf8mb4**（修复调查存 emoji 报 1366）
2. 重新 seed 权限（含 chat/stream、investigations、knowledge/embed、**k8s/generate-yaml**）
3. 菜单同步后可见「AI 调查」
4. **必须保证运行时可读取 `data/ai`**；能力中心 **reseed** 以加载 `generation/k8s-yaml` Prompt
5. 先 reseed → sync ES →（可选）向量化
6. 调查为同步阻塞调用，耗时受 LLM/采集影响，前端超时约 180s
7. 若库表仍为 utf8，也可手动：`ALTER TABLE ai_investigations CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;`
