# 需求说明：告警与监控平台

## 1. 目标

统一 **Prometheus 数据源**、**静默（含批量）**、**监控规则（PromQL）**、**订阅树路由**、**通知通道**、**处理人/部门**、**值班班次**；支持 Alertmanager Webhook 接入与历史事件查询。

与实现对齐的详细需求与设计（模型、API、`ReceiveAlertmanager` 流水线、Redis key）：见 [R-alert-platform-detailed-design.md](../../requirements/R-alert-platform-detailed-design.md)。

## 2. 路径定位（混合架构）

| 路径 | 定位 | 说明 |
|------|------|------|
| **Prometheus + Alertmanager → Webhook** | **主评估路径** | 生产核心告警（节点/应用/SLO 等）应写在 Prometheus 规则文件并由 AM 推送；云枢负责订阅树路由、通道、静默/抑制、值班与历史 |
| **平台内监控规则（PromQL 评估器）** | **轻量补充** | 适合快速试验、少量临时规则、或尚未纳入 GitOps 的场景；**不**替代 Prometheus 规则文件管理 |
| **云资源到期等** | **非 Prom 自研** | 无 Prometheus 指标时继续由平台评估 |

**不要下线**：订阅树、接收组、通道、静默、抑制、值班、历史事件——它们是统一通知中枢，与评估路径无关。

规则模板（`GET /alerts/rule-templates` → `POST /alerts/monitor-rules/from-template`）仅用于**一键创建平台监控规则**，按 `cpu` / `disk` / `memory` / `availability` 分组，并写入 `labels.category`；**不会**下发或管理 Prometheus `rules/*.yml`。

## 3. 功能结构

```
告警通知 / 监控
├── 告警通道：钉钉/企邮/Webhook 等，字典项辅助密钥
├── 告警监控平台（Tab）
│   ├── 数据源
│   ├── 静默：单条编辑 + 从活跃告警批量创建
│   ├── 监控规则：手工新建 / 从模板创建；启用/停用筛选
│   ├── 处理人：用户 + 可选部门子树（IM @）+ 恢复通知；邮件不展开部门
│   ├── 值班：按规则维度的班次表；值班时段标题加「值班」前缀
│   ├── 云到期：非 Prom 场景
│   └── PromQL 查询 / 原生告警视图
├── 订阅树配置：匹配标签/正则、节点静默窗口、接收组与通道关联
└── 历史告警记录：事件列表与统计
```

## 4. 注意事项

| 项 | 说明 |
|----|------|
| 主路径 | 生产核心告警优先 Prometheus+AM；平台规则为补充，勿把全量生产规则迁入平台评估器 |
| 规则模板 | 内置分组包 → 创建 `alert_monitor_rules`；带 `category`（cpu/disk/…）便于订阅树按类路由 |
| 项目绑定 | 规则项目从数据源推导；**组织部门**通过用户 `department_id` 关联，项目仅可选 `owner_department_id` 标注，**不**自动决定成员 |
| 规则启用 | `enabled=false` 不评估 PromQL；列表 Tab 可筛「全部/启用/停用」 |
| 处理人邮件 | 显式用户邮箱 + 额外邮箱 + 值班班次邮箱；**不**因选部门而向部门子树全员发邮件 |
| 处理人部门 | 手动选择、可清空保存；用于钉钉/企微 @（项目成员 ∩ 部门子树） |
| 静默 | 时间区间、matcher 与 Alertmanager 语义对齐；批量静默对多条分别创建 |
| 值班 | `alert_duty_blocks` 挂 `monitor_rule_id`；当前时刻命中班次时通知标题前缀 `值班` |
| 订阅树 | 节点 `match_labels_json` / `match_regex_json` 为 JSON 字符串，需合法；节点引用接收组，接收组再绑定通道；与 Prometheus/平台规则 labels 对齐约定见 `docs/alert-subscription-labels-chain.md` |

## 5. 相关表

`alert_channels`、`alert_subscription_nodes`、`alert_receiver_groups`、`alert_datasources`、`alert_silences`、`alert_monitor_rules`、`alert_rule_assignees`、`alert_duty_blocks`、`alert_events`。
