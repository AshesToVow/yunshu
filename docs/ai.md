# Yunshu AI 模块

## 概述

`ai` 插件：多模型、运维助手（Tool Calling + RAG + SSE 进度）、**调查闭环**（采集→分析→写动作挂审批）、场景分析、高危审批（含 Policy）、**AI 运维能力中心**（Prompt/知识/案例/SOP/Tool/Evaluation）。

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

- `/ai/assistant` 运维助手（SSE 工具轨迹；可发起 chat 调查、取消发送）
- `/ai/investigations` AI 调查记录（支持 awaiting_approval；详情轮询进行中状态）
- `/ai/approvals` 操作审批
- `/ai/center` 能力中心（Prompt/Tools/案例/SOP/KB/Eval 运行历史/向量化）

助手菜单入口绑定权限（任一即可）：`POST /ai/chat`、`POST /ai/chat/stream`、`GET /ai/sessions`。

## 主要 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/ai/chat` | 助手（同步） |
| POST | `/api/v1/ai/chat/stream` | 助手 SSE 进度（progress/rag/tool/reply/done） |
| GET/PATCH | `/api/v1/ai/sessions*` | 会话；PATCH 可写 `namespace`→`context_json` |
| POST | `/api/v1/ai/k8s/generate-yaml` | 按自然语言生成 K8s YAML（创建页回填，不直接 apply） |
| POST | `/api/v1/ai/investigations` | 发起调查（`alert\|pod\|cicd\|chat\|incident`） |
| GET | `/api/v1/ai/investigations` | 调查列表 |
| GET | `/api/v1/ai/investigations/:id` | 调查详情 |
| GET | `/api/v1/ai/approvals` | 审批列表 |
| POST | `/api/v1/ai/approvals/:id/review` | 审批 |
| POST | `/api/v1/ai/approvals/:id/execute` | 执行已批准单（含 K8s 写 / 静默 / 脚本） |
| POST | `/api/v1/ai/knowledge/sync` | DB→ES 同步 |
| POST | `/api/v1/ai/knowledge/embed` | KB chunks 向量化（混合 RAG） |
| GET | `/api/v1/ai/center/overview` | 能力中心统计 |
| POST | `/api/v1/ai/center/reseed` | 从 data/ai 重载种子 |
| POST | `/api/v1/ai/center/eval/run` | 跑评估套件（`live` 可选） |
| GET | `/api/v1/ai/center/eval/runs` | 评估运行历史 |
| GET | `/api/v1/ai/center/eval/runs/:id` | 评估运行详情（含各用例结果） |
| POST | `/api/v1/ai/logs/analyze` | 日志场景分析 |
| POST | `/api/v1/ai/logs/pipeline-adjust` | Loggie Pipeline 建议 |

另有场景分析：pod-diagnose / build-fail / alert-explain。

## 业务入口

| 页面 | 能力 |
|------|------|
| 告警历史 | AI解读 + **AI调查**（静默需显式申请，走审批） |
| Pod 排障抽屉 | AI 分析 + **AI 调查** |
| CI 构建详情 | AI 分析 + **AI 调查** |
| K8s 资源创建（YamlCrud / Pod） | **AI 生成 YAML**（描述→编辑器，人工核对后 apply） |
| 项目日志 / Loggie | `/ai/logs/analyze`、`pipeline-adjust` |
| 运维助手 | 流式对话 + 工具/RAG 证据面板 + **发起调查** |

嵌入入口在调用前检查 `ai_enabled`（`useAiEnabled`）。

## Tool 运行时

- `builtin`：K8s / **日志** / CI / 告警 / **Prometheus 监控** / **CMDB 服务器** / **DB 实例** / **ES 连接**
- `script`：沙箱 `data/ai/tools`；**WRITE/HIGH → 创建审批单**，批准后 `ExecuteApproval` 执行
- **写工具统一审批**：`scale_deployment` / `restart_deployment` / `delete_pod` / **`create_alert_silence`**（不再立即静默）
- **Policy**：扩缩容 replicas>50 需 reason 含 `emergency`；禁止删 `kube-system`/`kube-public`/`yunshu-logging` Pod；写操作须有 namespace

## 调查闭环

1. 采集（确定性）→ 分析（LLM）→ 报告
2. 若报告/场景产生写动作（如告警静默、K8s 写工具）：创建 `ai_tool_approvals`，调查状态 → **`awaiting_approval`**，写入 `approval_id`
3. 人工在「AI 操作审批」或统一 workflow 待办中审核并执行
4. 无写动作时状态为 `done`

## 日志分析（助手）

在运维助手选择项目后可对话触发：

1. `analyze_logs`：级别/服务/Pod 统计 + 高频错误签名 + 样例
2. `search_logs`：原始命中
3. 为空时：`list_log_sources` → `list_loggie_status` / `list_cluster_log_rules`

## RAG

1. DB 案例 + chunks + SOP（词法；有 Embedding 时混合语义）
2. ES `yunshu-ai-kb-*`
3. 回退内嵌模块文档

## 种子覆盖矩阵

| 模块 | 案例 | SOP | KB | 脚本工具 | Builtin |
|------|------|-----|----|----------|---------|
| k8s | CrashLoop/ImagePull/Pending/OOM | ✓ | kb_k8s | — | 强 |
| cicd | 构建失败 | ✓ | kb_cicd | — | list/get/log |
| alert | 未收到 | ✓ | kb_alert | — | list/explain/**silence(审批)** |
| monitor | PromQL 空 | ✓ | kb_ops | — | datasources/query/active_alerts |
| log | 检索为空 + 错误整理 | ✓ | kb_log | — | search/analyze/sources |
| linux | 磁盘打满 | ✓ | kb_linux | disk/mem/load | probe |
| cmdb | 服务器离线 | ✓ | kb_cmdb | — | list/get/test |
| dbmgmt | 连接失败 | ✓ | **kb_dbmgmt** | — | list_db_instances |
| esmgmt | 连接不可达 | ✓ | kb_esmgmt | — | list_es_connections |

## 部署注意

1. 重启服务 AutoMigrate（含 `ai_investigations`）；启动 PostMigrate 会将 AI 文本表转为 **utf8mb4**
2. **重新 seed 权限**（含 chat/stream、investigations、eval/runs、knowledge/embed、k8s/generate-yaml）
3. 菜单同步后可见「AI 调查」；助手入口含 stream 权限
4. **必须保证运行时可读取 `data/ai`**；能力中心 **reseed** 加载新 Prompt/KB/SOP/Tool/Eval（含 monitor、kb_dbmgmt、CASE-013）
5. 先 reseed → sync ES →（可选）向量化
6. 调查多为同步阻塞，前端超时约 180s；`collecting`/`analyzing`/`awaiting_approval` 详情页会轮询
7. 系统 Prompt（`system/ops-agent`）若库中内容与种子不一致，reseed **会更新**当前版本内容；仅「已有且内容相同」时跳过
8. 不含 MCP（扩展点预留，未实现协议桥）
9. `seed` 会增量补齐：已有 AI 能力中心/Eval 权限的角色自动获得 `eval/runs`；已有 `/ai/chat` 的角色自动获得 `/ai/chat/stream`
10. 调查挂接审批后：驳回→`cancelled`；执行成功→`done`；执行失败→`failed`；仅「已批准未执行」仍 `awaiting_approval`
11. 告警调查 **不默认** 创建静默审批；需报告/助手显式申请 `create_alert_silence`
12. 写工具硬策略（保护 NS、replicas>50）在 **调查建单** 与 **审批执行** 两处均校验
