# M-05 告警通知

| 项 | 内容 |
|----|------|
| 文档编号 | M-05 |
| 模块名称 | 告警通知 |
| 插件 | `alert` |
| 前端 | `/alert-notify/*`（通道、监控平台、值班、维护窗） |
| 后端 | `internal/service/alert` |
| 状态 | 已对齐源码 |

## 1. 目标

接收 Alertmanager Webhook / 监控规则评估，经订阅路由、静默、值班、处理人后投递多通道（钉钉/邮件等）；提供历史事件与配置中心能力。

## 2. 功能需求

| ID | 功能 | 菜单/入口 |
|----|------|-----------|
| F-01 | 告警通道 CRUD、测试、模板预览 | `/alert-channels` |
| F-02 | 监控平台（数据源、规则、静默、订阅、历史） | `/alert-monitor-platform` |
| F-03 | 值班班次 | `/alert-duty` |
| F-04 | 维护窗口 | `/alert-maintenance` |
| F-05 | 外部 Webhook 摄入 | Alertmanager |

## 3. 接口规格（摘要）

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/alerts/webhook/alertmanager` | 否（可配 token） | 外部推送 |
| GET/POST/PUT/DELETE | `/api/v1/alerts/channels`、`/:id` | 是 | 通道 |
| POST | `/api/v1/alerts/channels/:id/test` | 是 | 测试投递 |
| POST | `/api/v1/alerts/channels/preview-template` | 是 | 模板预览 |
| GET/POST/PUT/DELETE | `/api/v1/alerts/datasources` 等 | 是 | 数据源 |
| … | `/api/v1/alerts/...` 规则/静默/订阅/值班/事件 | 是 | 详见 OpenAPI 与 api-catalog「告警中心」 |

完整字段与路由逻辑见详细设计文档（下节）。

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `alert_channels` | 通知通道 |
| `alert_datasources` | Prometheus 等，绑定 `project_id` |
| `alert_monitor_rules` | 监控规则 |
| `alert_rule_assignees` | 处理人 |
| `alert_silences` | 静默 |
| `alert_duty_blocks` | 值班 |
| `alert_subscription_nodes` / `alert_receiver_groups` / `alert_subscription_matches` | 订阅路由 |
| `alert_events` | 历史事件 |
| 维护窗相关表 | 维护窗口 |

## 5. 依赖

| 依赖 | 用途 |
|------|------|
| Prometheus | 规则评估/enrich（配置） |
| 钉钉/邮件 Webhook | 投递 |
| CMDB | 云到期规则拉实例（可选） |
| 后台 Worker | `plugins/alert/workers.go` |

## 6. 相关文档

- [R-03-alert-and-monitor.md](../R-03-alert-and-monitor.md)
- [R-alert-platform-detailed-design.md](../../../requirements/R-alert-platform-detailed-design.md)
- [alert-routing-and-delivery-guide.md](../../../alert-routing-and-delivery-guide.md)
- [alert-notify-guide.md](../../../alert-notify-guide.md)
