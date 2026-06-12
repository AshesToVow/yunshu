# 菜单需求：历史告警 / 事件列表（`/alert-events`）

> **已废弃侧栏入口**（2026-06）：seed 会将此菜单设为 `hidden`，并重定向至 `/alert-monitor-platform?tab=config&cfg=history`。请使用 **告警监控平台 → 策略与联调 → 历史**。

## 1. 定位

- **路由**：`/alert-events`（历史书签可能仍存在）。  
- **目标**：按时间/集群/关键字查询 **已落库** 的告警发送记录，用于排障与审计。

## 2. 主要 API

- `GET /api/v1/alerts/events`：分页、筛选。  
- （可选）`GET /api/v1/alerts/history/stats`：看板统计。

## 3. 数据表

- `alert_events`（含成功失败、HTTP 状态、命中策略名、monitor_pipeline 等）。

## 4. 注意事项

- 与 **实时** Prometheus 告警列表不同；本页为**平台侧投递历史**。  
- 大时间范围查询需限制 page_size 或后端超时。
