# 告警 Redis 降级策略

本文描述 Yunshu 告警链路在 **Redis 不可用 / 抖动 / 仅加速缓存失效** 时的实际行为与 SLA。实现以源码为准。

关联：`docs/requirements/R-alert-platform-detailed-design.md` §13。

## 1. 总原则

| 能力类别 | Redis 不可用时 | 说明 |
|----------|----------------|------|
| **外发投递（AM Webhook / 已入队告警）** | **尽量继续** | 节流/静默窗口等「加速态」跳过，优先保证通知可达 |
| **恢复判定（firing_delivered）** | **降级读库** | DB `alert_firing_deliveries` 为最终依据；Redis 仅作缓存 |
| **平台内置监控规则 for 状态** | **进程内简化状态机** | `evaluateMonitorRuleNoRedis`；多副本会重复评估风险升高 |
| **Webhook 异步队列** | **同步处理** | 无 Redis 时不入队，请求内跑完流水线（注意 AM 超时） |
| **Leader 锁（监控 tick / 云到期）** | **单机视为持有** | 多副本无锁时可能并行评估 |

## 2. firing_delivered（已落地）

- **写入**：成功 firing 通道投递后 → `Mark` DB + Set Redis（TTL ≥ `aggregate_ttl_seconds`，至少 7 天）。
- **读取**：先 Redis，未命中再查 DB。
- **清除**：resolved 流程结束后清 Redis + DB。
- **效果**：长时间 firing 后 Redis TTL 过期，仍可凭 DB 正确发送恢复，避免「只知故障不知恢复」。

表：`alert_firing_deliveries(fingerprint PK, updated_at)`。

## 3. 节流 / 静默 / 去重

| 机制 | 无 Redis |
|------|----------|
| `decideFiringGroupTiming` / peek | 视为 `shouldSend=true`（不节流） |
| 订阅树 silence 窗口 SetNX | 跳过静默占位，可能多发 |
| fingerprint 计数去重 | 跳过；可能重复留痕 |
| `resolved:sent` SetNX | 返回 first=true，可能重复发恢复 |

以上属于 **非核心加速态关闭**：宁可多通知，也不因 Redis 丢告警。

## 4. 平台监控规则

- **有 Redis**：`pending` / `for_seconds` 状态机 + 全局 Leader 锁。
- **无 Redis**：`evaluateMonitorRuleNoRedis` 进程内 map；**不适合多副本生产**，应保证至少一副本 Redis 可用或单实例部署。

## 5. 排障入口

- 历史事件：`GET /api/v1/alerts/events?fingerprint=`
- 指纹追溯：`GET /api/v1/alerts/events/by-fingerprint?fingerprint=`  
  返回 `firing_delivered`（redis|db|none）、跳过原因汇总、最近留痕列表。
- 事件 `error_message` 与策略分类见前端 `alert-event-reasons` / 后端 `applyAlertEventCategoryFilter`。

## 6. 运维建议

1. 生产 Redis 高可用（Sentinel/Cluster）；监控连接失败率与队列长度。
2. 告警 API 副本水平扩展时，**必须**有 Redis，否则内置规则与节流语义劣化。
3. 新权限需 seed：`/api/v1/alerts/events/by-fingerprint` GET。
