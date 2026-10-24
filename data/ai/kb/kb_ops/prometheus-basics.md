# Prometheus / 告警数据源

## 助手可用工具

- `list_alert_datasources`：列出项目绑定的监控数据源
- `query_prometheus` / `query_prometheus_range`：执行 PromQL
- `list_prometheus_active_alerts`：当前 firing 告警
- `get_alert_detail`：告警详情

## 排查空结果

1. 先查数据源是否启用、URL 是否可达
2. 用 `up` 验证连通，再换业务指标
3. 注意时间参数与 step；瞬时查询与区间查询语义不同
