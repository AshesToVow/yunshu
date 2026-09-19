# 日志分析与快速排障（种子）

## 工具矩阵

| 工具 | 用途 |
|------|------|
| `analyze_logs` | 检索 + 级别/服务/Pod 统计 + 高频错误签名 + 样例行（优先用于「帮我看看报错」） |
| `search_logs` | 原始命中列表；支持 keyword/level/from/to/service_name/namespace/pod/container/collector_mode |
| `list_log_sources` | 核对项目是否配置了日志源路径 |
| `list_loggie_status` | 主机 Loggie Agent 健康与流水线 |
| `list_cluster_log_rules` | K8s 侧采集规则（需 cluster_id） |
| `get_pod_logs` | kubectl 实时容器日志（与平台 ES 日志互补） |

## 推荐流程

1. 确认 `project_id`（助手页选择项目）
2. 业务报错 / 异常排查：`analyze_logs`（keyword=Exception 或 level=ERROR，可加 namespace/service_name）
3. 需要原文翻页对照：`search_logs` 用同一过滤条件
4. 结果为空：`list_log_sources` →（host）`list_loggie_status` /（k8s）`list_cluster_log_rules`
5. 仍无线索：核对时间窗口 from/to、索引租户前缀、Kafka Topic

## 整理原则

- 先看 `top_error_signatures` 是否同一类异常
- 对照 `level_counts` 是否 ERROR 激增
- `samples` 只作证据摘录，勿编造未出现的堆栈
- 禁止建议删除生产 ES 索引或 Kafka Topic
