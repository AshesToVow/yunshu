# 告警平台整改方案：夜莺式引擎 + Consul + Telegraf

> 状态：**P0 / P1 / P2 完成**  
> 配置样例：[`deploy/monitoring/`](../../deploy/monitoring/)

## 目标

采集在机房（Telegraf/PGW/blackbox + Consul → Prom/VM），规则与通知在 Yunshu。无 Alertmanager。

## 已落地

| 项 | 说明 |
|----|------|
| P0 | 拆 AM、规则中心主路径、采集样例与模板 |
| 序列指纹 | 同规则多序列独立 pending/firing |
| Consul 目录 | 监控对象 Tab + sync API |
| **当前/历史告警** | `alert_cur_events` / `alert_his_events`；屏蔽不入当前；事件台三分栏 |
| **YAML 导入** | `POST /monitor-rules/import-prometheus-yaml` + 规则中心 UI |
| **P2 数据源** | `prometheus` / `victoria`（Prom 兼容 API）；连通检测与评测共用 |
| **P2 质量** | `GET /quality-report` 按项目过滤 + 当前 firing；告警平台「质量」Tab |
| **P2 AI** | 当前告警行内「AI 解读」→ `POST /ai/alert/explain` |

## API 增量

- `GET /api/v1/alerts/cur-events`
- `GET /api/v1/alerts/his-events`
- `POST /api/v1/alerts/monitor-rules/import-prometheus-yaml`
- Consul：`/alerts/consul-endpoints*`、`/alerts/monitor-objects`
- `GET /api/v1/alerts/quality-report`（`project_id`、`cur_firing_count`）
- `POST /api/v1/ai/alert/explain`

## 后续（可选）

- ES 日志类规则评测（非 Prom API）
- 通知规则语义再拆分

## 修订

| 日期 | 说明 |
|------|------|
| 2026-08-15 | P0 + P1（指纹、Consul、当前/历史、YAML 导入） |
| 2026-08-15 | P2（VM 数据源、质量看板、当前告警 AI 解读） |
