---
name: yunshu-go-architecture
description: >-
  Yunshu backend Clean Architecture gates: Handler→Service→Repo, ban new
  Service *gorm.DB fields, soft-delete/Tx conventions, RouteDeps narrow
  interfaces. Use when writing or reviewing Go service/repository/handler,
  Wire, or plugin routes in this repo.
---

# Yunshu Go Architecture

写/改 `internal/service`、`internal/repository`、`internal/handler`、`internal/router`、插件路由时必须遵守。

配套：`use-modern-go`、`golang-error-handling`、`golang-code-style`。

## 分层（硬门禁）

```
Handler → Service → Repository(interface) → GORM
```

| 层 | 允许 | 禁止 |
|----|------|------|
| Handler | 解析/鉴权上下文、调 Service、`ServeJSON`/`ServeQuery` | 持有或调用 `*gorm.DB` |
| Service | 业务编排、依赖 **Repo interface** | **新建**字段 `db *gorm.DB`；跨域硬编码具体 Repo 实现 |
| Repository | GORM / SQL | 业务规则（ACL 判定等应在 Service） |

### Service 与 `*gorm.DB`

- **禁止**新增 Service 结构体字段 `db *gorm.DB`（或等价直接持 DB）。
- **例外**（须注释说明原因）：
  - 字典 / 极短工具型服务
  - 已有遗留文件：仅允许修 bug，**不得扩面**新增直查表逻辑；新能力走 Repo
- 多表写优先 Repo 的 `Transaction` / `WithTx`（参考 `AlertSubscriptionRepository`），或 Service 边界 `db.WithContext(ctx).Transaction` 且内部只用 `tx` 构造的 Repo。

### 跨域依赖注入

- AI / CICD / Alert 跨域依赖经 Wire **单一 provider** 注入：`provideAIService`、`provideCicdService`（含 K8s hooks）、`provideAlertService`（含 `attachAlertLogSearch`）。
- **禁止**在 `assembleRouteDeps` 再加 `SetXxxDeps`；勿对同类型注册双 provider。
- 插件 HTTP：窄接口在 `internal/routedeps`；`*router.RouteDeps` 实现 `routedeps.Bundle`。`plugin.Runtime` **无** `Deps` 字段（避免 plugin↔handler 环）；`router.Register` 经 `SetRouteBinder` 闭包注入 Bundle。
- Worker：`plugin.As[*Concrete](rt.Xxx)` 取槽位。
- 新增插件路由：在 `routedeps` 加窄接口并并入 `Bundle`，在 `router` 实现 getter，并在 `bindPluginRoutes` 分发。

## Soft-delete

| 实体类型 | 规则 |
|----------|------|
| 可配置业务实体（用户、项目、告警静默等） | 默认 `DeletedAt gorm.DeletedAt` + `json:"-"` + `gorm:"index"` |
| 业务唯一键需可复用 | 复合唯一索引**包含** `deleted_at`（参考 `internal/model/role.go`） |
| 全局唯一种子行（权限等） | 唯一索引可不含 `deleted_at`；复活用 upsert 清 `deleted_at`（参考 `permission.go`） |
| 审计 / append-only | **无** soft-delete（`OperationLog`、`DbAuditLog`、`DbSqlExecution`、`AlertAck`） |
| dbmgmt `db_*` | 当前无 `DeletedAt` → **硬删**；勿假设 GORM 软删 |
| 跨表 Join | 软删表须显式 `deleted_at IS NULL`（raw SQL / `Joins`） |
| 物理清理 | 仅 `Unscoped()` / DBA，业务路径禁止 |

## 事务（Tx）

1. Handler **不开**事务。
2. 多表写：`db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })`，始终带 `ctx`。
3. 优先 Repo 层包装：`Transaction(ctx, fn)` 内用 `tx` 重建同接口 Repo，避免 Service 散落裸 `tx`。
4. dbmgmt 审批链路：优先走 workflow 引擎已有 Tx；新增多表写须显式 Transaction 或并入 workflow。
5. 两步创建可用补偿式软删回滚（项目+成员等），须注释；能上 Tx 则上 Tx。

## Handler 文件组织

胖 Handler 用**同 struct、多文件**拆分（参考 `CicdHandler` + `cicd_registry_handler.go`、`PodHandler` + `pod_exec_ws.go`）。

- 不拆类型、不改 Wire/路由签名，除非域真正独立（如 alert 多 Handler）。
- 新能力按域落到对应 `*_handler.go`，避免继续堆进单文件。

## 自检清单

- [ ] 新 Service 无 `*gorm.DB` 字段（或已标例外）
- [ ] 新持久化经 Repo interface
- [ ] 模型 soft-delete 选择符合上表
- [ ] 多表写有 Tx 或明确补偿策略
- [ ] 未新增 `SetXxxDeps` / 未扩大 `Runtime` 的 `any` 面
