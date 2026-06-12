# Yunshu 后端架构重构实施报告

**最后更新**: 2026-06-11  
**状态**: ✅ 主干已落地（`go build ./...` / `go test ./internal/... -short` 通过）  
**代码地图**: [CODEBASE-MAP.md](CODEBASE-MAP.md)

---

## 实施状态快照

| # | 项 | 状态 | 说明 |
|---|-----|------|------|
| 1 | 日志 `logutil` | ✅ | 已删除 `svclog`；HTTP/Service/Worker 组件化 |
| 2 | 错误 `bizerrors` | ✅ | 已删除 `svcerr`、`pkg/apperror`；HTTP + gRPC 统一 |
| 3 | Wire 基础设施 | ✅ | `providers.InitializeInfra/Core` |
| 4 | 路由 Wire | 🟡 | `InitializeRouteDeps`：Repo ✅；Service 仍 `buildRouteServices` 手工装配 |
| 5 | Repository | ✅ | 告警 / 系统 / 项目 / K8s 策略 / 总览 / 事件转发等主路径 |
| 6 | Service 目录拆分 | ✅ | `alert/` `k8s/` `project/` `system/` `logplatform/` `mysqlbackup/` `overview/` + `exports.go` 门面 |
| 7 | Alert 域 | ✅ | 主链路在 `alert` 包；Redis 状态 `NewRedisAlertStateService`；无根目录重复实现 |
| 8 | K8s Event 转发 | ✅ | `k8s/eventforward/`；仓储 `K8sEventForwardRepository` |
| 9 | 死代码清理 | ✅ | 删除根目录 `alert_service_core.go` 等迁移残留 |
| 10 | 业务插件化 | ✅ | `internal/plugin` + `plugins.enabled`；`core/k8s/alert/project/cmdb/backup` |
| 11 | CMDB 拆分 | ✅ | 服务器/云账号/终端 → `service/cmdb` + `register_cmdb_routes.go` |
| 12 | 菜单 catalog | ✅ | `internal/menu/catalog.go` + `Sync`；`Tree()` 不再热路径维护 |
| 13 | seed 优化 | ✅ | 事务、Permission 批量 upsert、Casbin `AddPolicies`、admin 密码不重复重置 |

---

## 当前目录结构（摘要）

```text
internal/service/
├── exports.go          # handler 用 service.Xxx 别名
├── alert/
├── cmdb/               # 服务器资产（从 project 拆出）
├── k8s/eventforward/
├── project/
├── system/
├── logplatform/
├── mysqlbackup/
└── overview/

internal/menu/          # 内置菜单 catalog + seed 同步
internal/plugin/        # 插件注册表
internal/plugins/       # 各业务插件 init 注册
```

详见 [backend-architecture-complete.md §3.3](backend-architecture-complete.md) 与 [CODEBASE-MAP.md](CODEBASE-MAP.md)。

---

## 仍待优化（非阻塞）

| 项 | 说明 |
|----|------|
| 全量 Wire Service | `route_services.go` ~50 个 `NewXxx` 可逐步改为 ProviderSet |
| Handler 直引子包 | 弱化 `exports.go`，新代码优先 `import alert` 等 |
| `mysqlbackup.db` | MinIO / dictconfig 仍直用 `*gorm.DB` |
| 插件菜单与 Casbin 联动 | 按 `plugins.enabled` 过滤侧栏；角色菜单授权待完善 |
| 前端菜单 Context | 减少多处重复 `getMenuTree()` |

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
- [code-review-report.md](code-review-report.md) — 历史审查记录（部分条目已解决，见文首说明）  
- [architecture-diagrams.md](architecture-diagrams.md) — 架构图

---

*Generated / maintained with backend refactor | 2026-05-22*
