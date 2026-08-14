# 日志检索排障知识（种子）

## 工具

`search_logs`：需要 `project_id` + `keyword`，可加 namespace/pod/cluster。

## 为空时检查

1. keyword 是否过宽或拼写错误
2. 时间范围是否覆盖采集窗口
3. Loggie / 集群采集规则是否启用
4. Kafka Topic / ES 索引前缀是否含 clusterId、projectId

禁止建议删除生产 ES 索引。
