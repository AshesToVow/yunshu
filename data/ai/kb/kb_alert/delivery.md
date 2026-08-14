# 告警投递排障知识（种子）

## 工具链

1. `list_alerts`（可带 project_id）确认事件与 status
2. 取 `fingerprint` → `explain_alert` 看投递、跳过、抑制原因

## 常见原因

- 订阅标签不匹配 / 未启用
- 渠道 HTTP 失败
- 静默窗 / 抑制
- Alertmanager 入站缺少 project_id / datasource 导致路由弱匹配

## 注意

总览「Firing」为全局计数；项目页按 `project_id` 过滤，二者口径不同。
