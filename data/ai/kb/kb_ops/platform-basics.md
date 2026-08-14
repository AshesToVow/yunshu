# 运维基础知识（种子）

## Yunshu 平台能力概览

Yunshu 提供 K8s、CI/CD、告警、日志平台、CMDB、数据库管理与 AI 运维助手。
AI 通过 Tool 查询真实数据；写操作创建审批单，不会立即变更生产。

## 模块与工具速查

| 场景 | 关键工具 |
|------|----------|
| Pod 异常 | diagnose_pod, get_pod_logs, list_events |
| 构建失败 | list_cicd_builds, get_cicd_build_log |
| 告警未收到 | list_alerts, explain_alert |
| 日志为空 | search_logs |
| 磁盘/内存/负载 | linux.disk.check, linux.mem.check, linux.load.check |

## 缺参数时

- 缺 cluster_id：`list_clusters` 或请用户选择
- 缺 project_id：请用户选择项目（日志/CI/告警必需）
- 缺资源名：list_pods / list_deployments 或向用户索要
