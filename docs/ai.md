# Yunshu AI 模块

## 概述

`ai` 插件：多模型、运维助手（Tool Calling + RAG）、场景分析、高危审批、**AI 运维能力中心**（Prompt/知识/案例/SOP/Tool/Evaluation）。

## 启用

`plugins.enabled` 含 `ai`；字典 `ai_enabled=true` 并配置 API Key（或后续 `ai_llm_models`）。

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

- `/ai/assistant` 运维助手
- `/ai/approvals` 操作审批
- `/ai/center` 能力中心（Prompt/Tools/案例/SOP/KB/Eval）

## 主要 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/ai/chat` | 助手（读 DB Prompt + RAG + Tool 注册表） |
| GET | `/api/v1/ai/center/overview` | 能力中心统计 |
| POST | `/api/v1/ai/center/reseed` | 从 data/ai 重载种子 |
| GET/POST | `/api/v1/ai/center/prompts*` | Prompt 与版本 |
| GET/PATCH | `/api/v1/ai/center/tools*` | Tool 注册表 |
| GET | `/api/v1/ai/center/cases` `/sops` `/knowledge-bases` | 案例/SOP/KB |
| POST | `/api/v1/ai/center/eval/run` | Evaluation（`live=true` 走真实 Chat） |
| POST | `/api/v1/ai/knowledge/sync` | DB→ES 同步 |

另有 sessions、approvals、场景分析接口（见历史文档）。

## Tool 运行时

- `builtin`：现有 Go 服务（K8s/日志/CI/告警等）
- `script`：Python2.7+/3、Shell、Go 二进制；根目录沙箱 `data/ai/tools`；写操作仍走审批

样例脚本工具：

- `data/ai/tools/linux/disk_check/` → `linux.disk.check`
- `data/ai/tools/linux/mem_check/` → `linux.mem.check`
- `data/ai/tools/linux/load_check/` → `linux.load.check`

环境变量 `YUNSHU_AI_PYTHON` 可指定解释器。脚本探测的是 AI 运行环境本机；远端主机请用服务器操作台。

## RAG

优先 DB 故障案例 + chunks + SOP，其次 ES `yunshu-ai-kb-*`，再回退内嵌模块文档。

## 种子覆盖矩阵（持续补充）

| 模块 | 案例 | SOP | KB | 脚本工具 | Builtin |
|------|------|-----|----|----------|---------|
| k8s | CrashLoop/ImagePull/Pending/OOM | ✓ | kb_k8s | — | 强 |
| cicd | 构建失败 | ✓ | kb_cicd | — | list/get/log |
| alert | 未收到 | ✓ | kb_alert | — | list/explain |
| log | 检索为空 | ✓ | kb_log | — | search_logs |
| linux | 磁盘打满 | ✓ | kb_linux | disk/mem/load | — |
| cmdb/db/esmgmt | — | — | — | — | 暂无 |

## 部署注意

1. 重启服务 AutoMigrate 新表
2. 重新 seed 权限（能力中心 API）
3. 菜单同步后可见「AI 能力中心」
4. 可选：`POST /ai/center/reseed` 与 `POST /ai/knowledge/sync`（新种子幂等导入）
