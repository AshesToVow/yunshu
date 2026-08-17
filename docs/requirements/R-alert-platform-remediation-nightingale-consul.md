# 告警平台整改方案：夜莺式引擎 + Consul + Telegraf

> 状态：**P0 / P1 / P2 完成**  
> 配置样例：[`deploy/monitoring/`](../../deploy/monitoring/)  
> **路径说明（必读）：[`deploy/monitoring/ALERT-PATH.md`](../../deploy/monitoring/ALERT-PATH.md)**

## 目标

采集在机房（Telegraf/PGW/blackbox + Consul → Prom/VM），**规则与通知在 Yunshu**。无 Alertmanager 主路径。

```text
采集 → Prometheus（TSDB）→ Yunshu 规则中心 → 事件台 / 渠道
         （旧）Prom rules → Alertmanager   ← 遗留，与平台互不同步
```

| 看什么 | 去哪看 |
|--------|--------|
| 平台规则是否告警 | Yunshu **事件台** |
| Prom 本地 rules | Prometheus **Alerts** |
| 首次投递等待 | `config.yaml` 的 `alert.group_wait_seconds` + 规则 **评估间隔**（不是 AM 的 30s） |

## 已落地

| 项 | 说明 |
|----|------|
| P0 | 拆 AM 主路径、规则中心、采集样例与模板 |
| 序列指纹 | 同规则多序列独立 pending/firing |
| Consul 目录 | 监控对象 Tab + sync API |
| 当前/历史告警 | `alert_cur_events` / `alert_his_events`；事件台三分栏 |
| YAML 导入 | `POST /monitor-rules/import-prometheus-yaml` |
| P2 | prometheus/victoria 数据源、质量看板、当前告警 AI 解读 |
| 运维手册 | Consul ACL、统一注册脚本、**ALERT-PATH** |

## API 增量

- `GET /api/v1/alerts/cur-events` / `his-events`
- `POST /api/v1/alerts/monitor-rules/import-prometheus-yaml`
- Consul：`/alerts/consul-endpoints*`、`/alerts/monitor-objects`
- `GET /api/v1/alerts/quality-report`
- `POST /api/v1/ai/alert/explain`

## 后续（可选）

- 停用机房 Prom `rule_files` + Alertmanager，避免双通道
- ES 日志类规则；通知语义再拆分
- 主路径 `group_wait` 到期后主动补发（减少「等下一轮评估」体感延迟）

## 修订

| 日期 | 说明 |
|------|------|
| 2026-08-15 | P0 + P1 + P2 |
| 2026-08-17 | 补充 ALERT-PATH：主/旧路径、group_wait 与评估间隔、与 AM 无关 |
