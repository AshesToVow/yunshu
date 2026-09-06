# 日志检索排障知识（种子）

## 工具

- `search_logs`：需要 `project_id`；可选 keyword / level / from / to / service_name / namespace / pod / container / collector_mode / cluster_id
- `analyze_logs`：同样过滤条件，返回级别统计、高频错误签名与样例（推荐先用）
- `list_log_sources` / `list_loggie_status` / `list_cluster_log_rules`：采集链路诊断

## 为空时检查

1. keyword 是否过宽或拼写错误；可试 level=ERROR
2. 时间范围 from/to 是否覆盖采集窗口
3. Loggie / 集群采集规则是否启用（对应工具）
4. Kafka Topic / ES 索引前缀是否含 clusterId、projectId

禁止建议删除生产 ES 索引。

详见同目录 `analyze-errors.md`。
