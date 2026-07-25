# Yunshu 后端架构重构实施报告

**最后更新**: 2026-07-12  
**状态**: ✅ 主干已落地（`go build ./...` / `go test ./internal/... -short` 通过）  
**代码地图**: [CODEBASE-MAP.md](CODEBASE-MAP.md)

---

## 实施状态快照

| # | 项 | 状态 | 说明 |
|---|-----|------|------|
| 1 | 日志 `logutil` | ✅ | 已删除 `svclog`；HTTP/Service/Worker 组件化 |
| 2 | 错误 `bizerrors` | ✅ | 已删除 `svcerr`、`pkg/apperror`；HTTP + gRPC 统一 |
| 3 | Wire 基础设施 | ✅ | `providers.InitializeInfra/Core` |
| 4 | 路由 Wire | ✅ | `InitializeRouteDeps`：Repo + Service + **全部 Handler** 由 Wire 注入 |
| 5 | Repository | ✅ | 告警 / 系统 / 项目 / K8s 策略 / 总览 / 事件转发等主路径 |
| 6 | Service 目录拆分 | ✅ | `alert/` `k8s/` `project/` `system/` `logplatform/` `mysqlbackup/` `overview/` `cicd/` `cmdb/` **`dbmgmt/`** + `exports.go` |
| 7 | Alert 域 | ✅ | 主链路在 `alert` 包；Redis 状态 `NewRedisAlertStateService` |
| 8 | K8s Event 转发 | ✅ | `k8s/eventforward/`；仓储 `K8sEventForwardRepository` |
| 9 | 死代码清理 | ✅ | 删除根目录迁移残留；移除 `log-files` / `log-units` stub API |
| 10 | 业务插件化 | ✅ | `core/k8s/alert/project/cmdb/backup/cicd` **`dbmgmt`** |
| 11 | CMDB 拆分 | ✅ | 服务器/云账号/终端/云 SDK → `service/cmdb` |
| 12 | 菜单 catalog | ✅ | `internal/menu/catalog.go` + `Sync` |
| 13 | seed 优化 | ✅ | 事务、Permission 批量 upsert、Casbin `AddPolicies` |
| 14 | P1 共享包 | ✅ | `cronutil`、`sshserver`；LogSource CRUD 去 gRPC 回环 |
| 15 | P2 域整理 | ✅ | `CloudExpiryRuleService` → `alert/`；告警 Worker → `plugins/alert/workers.go`；`dictconfig.Parse*` 去重 |
| 16 | P3 清理 | ✅ | 日志 stub 路由删除；CICD `notify_email` 改 `UserRepository` |

---

## 当前目录结构（摘要）

```text
internal/service/
├── exports.go          # handler 用 service.Xxx 别名
├── alert/              # 含 CloudExpiryRuleService、云到期评估 Worker
├── cmdb/               # 服务器资产 + 云 Provider SDK
├── cicd/               # Jenkins CI/CD
├── k8s/eventforward/
├── project/
├── system/
├── logplatform/
├── mysqlbackup/
├── dbmgmt/               # 实例、SQL 查询/审核、授权工单、审计
└── overview/

internal/pkg/
├── cronutil/           # Cron 校验与调度 Worker
├── sshserver/          # SSH 凭据解密与连接
└── ...

internal/menu/          # 内置菜单 catalog + seed 同步
internal/plugin/        # 插件注册表 + Runtime（AlertSvc/CicdSvc 等）
internal/plugins/       # 各业务插件 init 注册 + StartWorkers
```

详见 [backend-architecture-complete.md §3.3](backend-architecture-complete.md) 与 [CODEBASE-MAP.md](CODEBASE-MAP.md)。

---

## 仍待优化（非阻塞）

| 项 | 说明 |
|----|------|
| 全量 Wire Handler 装配 | ✅ | `HandlerSet` + `routeHandlers`；`assembleRouteDeps` 仅拼中间件与 RouteDeps |
| Handler 直引子包 | 弱化 `exports.go`，新代码优先 `import alert` 等 |
| `mysqlbackup.db` | ✅ | Service 不再持有 `*gorm.DB`；经 Wire 注入 ObjectStoreFactory / SchedulerConfigResolver |
| 插件菜单与 Casbin 联动 | ✅ | 菜单 status 双向同步；授权树按插件过滤；冲突分析 + 清理禁用插件策略 |
| 告警 ingest 统一模型 | ✅ | `CanonicalIngressAlert` 为唯一内部模型；AM Webhook 仅作适配；监控/云到期直接构造（含 `CloudExpiryExtension`） |
| 巡检报告模板 + 对象存储 | ✅ | 多版式（default/compact/executive）；MinIO 优先落盘；Excel/PDF 导出；保留天数清理 |

---

## 验证命令

```bash
go build ./...
go test ./internal/... -short
cd internal/providers && go generate   # 或 go run github.com/google/wire/cmd/wire
cd internal/router && go generate
```

---

## 相关文档

- [backend-architecture-complete.md](backend-architecture-complete.md) — 完整技术文档  
- [cicd.md](cicd.md) — CI/CD 插件说明  
- [dbmgmt.md](dbmgmt.md) — 数据库管理插件说明  
- [code-review-report.md](code-review-report.md) — 历史审查记录（部分条目已解决，见文首说明）  
- [architecture-diagrams.md](architecture-diagrams.md) — 架构图

---

*Generated / maintained with backend refactor | 2026-05-22*
