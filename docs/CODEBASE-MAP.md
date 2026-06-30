# Yunshu 后端代码地图（与源码同步）

**文档版本**: v1.4  
**最后更新**: 2026-06-30  
**适用分支**: 插件化 + 目录拆分 + 仓储化 + `bizerrors` 统一之后  

本文是阅读后端源码的**入口索引**。若与代码冲突，以 `internal/` 源码为准。

---

## 1. 分层与请求路径

```text
HTTP/gRPC 请求
  → internal/middleware/     Auth / Casbin / K8sScope / ErrorHandler / Audit
  → internal/handler/        参数绑定、调用 Service、abortService 出错
  → internal/service/        领域逻辑（按域子包，见 §2）
  → internal/interfaces/     Repository 接口
  → internal/repository/     GORM 实现
  → internal/model/          表结构与实体
```

**装配入口**（谁创建 Service/Repository）：

| 步骤 | 文件 | 说明 |
|------|------|------|
| 进程启动 | `cmd/server.go` | migrate、插件 Worker、gRPC、HTTP |
| 基础设施 | `internal/bootstrap/app.go` | DB / Redis / Casbin / Gin |
| 插件 | `internal/plugin/` + `internal/plugins/all/` | 编译期注册；`config.plugins.enabled` 启停 |
| 路由绑定 | `internal/router/plugin_bind.go` | `RegisterCoreRoutes` / `RegisterK8sRoutes` / … |
| Wire（部分） | `internal/providers/wire.go` | Config / Logger / DB / Redis |
| 路由依赖 | `internal/router/wire.go` | `InitializeRouteDeps` |
| 仓储 | `internal/router/repositories.go` | `newRouteRepositories(db)` |
| 服务 | `internal/router/route_services.go` | `buildRouteServices` 手工 `NewXxx` |
| Handler | `internal/router/router_deps.go` | `assembleRouteDeps` |

Handler 普遍 `import "yunshu/internal/service"`，通过 **`internal/service/exports.go`** 类型别名访问子包实现（如 `service.AlertService` → `alert.AlertService`）。

---

## 2. Service 层目录（按业务域）

```text
internal/service/
├── exports.go                 # 门面：type alias + 构造函数 re-export（给 handler/router）
├── alert/                     # 告警：摄入、聚合、投递、监控规则、订阅、静默…
├── cmdb/                      # 服务器资产、云同步、SSH/Web 终端（原 project 子域）
├── k8s/                       # K8s 资源与运行时
│   └── eventforward/          # 集群 Event → Webhook 转发 Worker/Admin
├── project/                   # 项目、成员、服务、日志源、Agent
├── system/                    # 用户、角色、权限、认证、菜单、字典…
├── logplatform/               # Log Agent、发现、内存日志 Broker
├── mysqlbackup/               # MySQL 备份调度与执行
├── overview/                  # 首页总览指标（CI 图表依赖 cicd 插件）
├── cicd/                      # Jenkins CI/CD、发布审批、制品
│
internal/menu/                 # 内置菜单 catalog（DefaultCatalog）+ seed 同步（Sync）
internal/plugin/               # 插件注册、path_filter（与 web/plugin-path.ts 同步）
```

### 域 → 典型入口文件

| 域 | 包 import | 核心类型 | 建议首读文件 |
|----|-----------|----------|--------------|
| 认证 | `service` / `system` | `AuthService` | `system/auth_service.go` |
| 用户权限 | `system` | `UserService`, `RoleService` | `system/user_service.go` |
| 项目 | `project` | `ProjectMgmtService` | `project/project_mgmt_core.go` |
| CMDB | `cmdb` | `CMDBService` | `cmdb/servers.go` |
| 告警 | `alert` | `AlertService`, `CloudExpiryRuleService` | `alert/alert_service_core.go` → `alert_ingest_pipeline.go` |
| K8s | `k8s` | `K8sRuntimeService`, `K8sPodService` | `k8s/k8s_runtime_service.go` |
| 日志 Agent | `logplatform` | `LogAgentService` | `logplatform/log_agent_service.go` |
| 备份 | `mysqlbackup` | `MysqlBackupService` | `mysqlbackup/mysql_backup_service.go` |
| CI/CD | `cicd` | `cicd.Service` | `cicd/service.go` |
| 总览 | `overview` | `OverviewService` | `overview/overview_service.go` |

### 跨域依赖（读代码时注意）

- `alert` → `cmdb`：云到期评估拉云实例（`cmdb.CloudProviderByName`、云账号解密）
- `alert` → `alert`：`CloudExpiryRuleService`、`ShouldEvalCloudExpiryByCron`（`internal/pkg/cronutil`）
- `alert` 后台任务：`plugins/alert/workers.go` → `AlertService.RunBackgroundWorkers`（Prometheus enrich、抑制 pruning、**云到期调度**）
- `project` → `logplatform`：日志 SSE/导出使用 `AgentLogBroker`（LogSource CRUD **直调 Service**，不经 gRPC 回环）
- `cicd` → `dictconfig` + `UserRepository`：Jenkins/MinIO 字典配置；发布通知邮件查用户
- `mysqlbackup` / `cmdb` → `sshserver`：SSH 凭据解密与 Dial
- `system` → `alert`：`DepartmentService` / `UserService` 依赖 `AlertRuleAssigneeService`
- `overview` → `k8s`：总览 Pod 聚合使用 `K8sRuntimeService`
- `overview` → `cicd`：项目上线/工单图表读 `cicd_release_runs`（cicd 插件未启用时返回空数据）

---

## 3. 初学者阅读顺序（约 2～4 小时）

1. `cmd/server.go` — 启动流程  
2. `internal/router/router.go` + `plugin_bind.go` — 插件路由如何注册  
3. `internal/menu/catalog.go` + `cmd/seed.go` — 菜单与权限种子数据  
4. 选一个简单 API：**字典条目**  
   `handler` → `service.DictEntryService` → `repository` → `model`  
5. 打开 `internal/service/exports.go`，理解 `service.Xxx` 与真实子包的对应关系  
6. 再读 **告警 Webhook**（复杂）：`handler/alert_handler.go` → `alert/alert_service_webhook.go` → `alert_ingest_pipeline.go`  
7. 中间件链：`middleware/auth.go` → `casbin.go` → `error_handler.go`  

### 3.1 二次开发入口

外部贡献者请先读 **[CONTRIBUTING.md](../CONTRIBUTING.md)**。  
插件菜单/API 路径规则须同时维护 `internal/plugin/path_filter.go` 与 `web/src/modules/plugin-path.ts`。

---

## 4. 错误与日志（当前标准）

| 用途 | 包 | 示例 |
|------|-----|------|
| 业务错误 | `internal/pkg/errors` (`bizerrors`) | `bizerrors.Pass(ctx, "user", "GetByID", err)` |
| HTTP 响应 | `internal/middleware/error_handler.go` | `bizerrors.Ensure` + JSON |
| gRPC 状态 | `internal/pkg/errors/grpc.go` | `bizerrors.ToGRPCStatus(err)` |
| Handler 出错 | `internal/handler/alert_abort.go` | `abortService(c, err)` |
| 结构化日志 | `internal/pkg/logutil` | `logutil.HTTP("http.auth").Warn(...)` |
| 底层 Logger | `internal/pkg/logger` | bootstrap 初始化 |

**已删除（勿再引用）**：`internal/service/svcerr`、`svclog`、`internal/pkg/apperror`。  
**已移除 API**：`GET .../projects/:id/log-files`、`GET .../log-units`（日志文件浏览改走 Agent **discovery**）。

---

## 4.1 共享工具包（`internal/pkg/`）

| 包 | 用途 | 典型调用方 |
|----|------|------------|
| `cronutil` | Cron 校验、`ShouldRunAfterLast` / `ShouldRunWithDayAnchor`、`RunWorker` | 云到期调度、MySQL 备份定时 |
| `sshserver` | 凭据解密 → `ssh.ClientConfig`、Dial | CMDB 终端、MySQL 备份远程执行 |
| `dictconfig` | 字典覆盖 YAML（mail、cicd、minio、parse 工具） | mail、cicd、mysqlbackup、bootstrap |
| `mysqlbackup` | mysqldump/xtrabackup/innobackupex 命令构建 | `service/mysqlbackup` |
| `logutil` | HTTP/Worker 组件化日志 | 全模块 |
| `errors` (`bizerrors`) | 业务错误 Pass/Ensure、gRPC 映射 | Service / Handler |

云到期调度要点（`alert/alert_cloud_expiry_scheduler.go`）：

- **规则 Cron**（控制台 `eval_cron_spec`，如 `0 */2 * * *`）决定**何时拉云评估**
- **内置轮询** `0 * * * * *`（每分钟）仅检查是否到点，与 `alert.monitor_eval_cron_spec`（PromQL 内置规则）无关

---

## 5. 依赖注入（Wire）现状

| 范围 | 状态 |
|------|------|
| Config / Logger / DB / Redis | ✅ `providers.InitializeInfra` |
| `routeRepositories` | ✅ Wire 调用 `newRouteRepositories` |
| `routeServices` | 🟡 **`buildRouteServices` 手工拼装**（约 50+ Service） |
| 全量 Service ProviderSet | ⬜ 未做（可选优化） |

---

## 6. 仓储化现状（摘要）

告警、系统、项目、K8s 策略、总览、事件转发等 **主路径已走 `interfaces.*Repository`**。  
仍直接持有 `*gorm.DB` 的少数点：

- `mysqlbackup.MysqlBackupService` — MinIO 配置、`dictconfig` 解析  
- `system.DepartmentService` — 部分事务经 `repo.DB()`  
- `k8s/eventforward.Manager` — 字典热加载  

新功能：**禁止在 Service 中新增裸 SQL**；通过 Repository 或现有 `interfaces` 扩展。

---

## 7. 种子数据与菜单（2026-06）

| 项 | 位置 | 说明 |
|----|------|------|
| 内置菜单定义 | `internal/menu/catalog.go` | `DefaultCatalog()`，支持任意层级 `Children` |
| 菜单 DB 同步 | `internal/menu/sync.go` | `menu.Sync`：按 `(parent_id, path)` upsert + 历史补丁 |
| seed 入口 | `cmd/seed.go` | 事务：Permission 批量 OnConflict、Casbin `AddPolicies`；admin 密码**仅首次创建** |
| 菜单树 API | `internal/repository/menu_repository.go` | 单次 `ListAll` + O(n) 建树；Service 侧 60s 缓存 |

---

## 8. 相关文档

| 文档 | 说明 |
|------|------|
| [plugins.md](./plugins.md) | 业务插件启停与 CMDB 拆分 |
| [cicd.md](./cicd.md) | CI/CD 插件：Jenkins、审批、字典配置 |
| [backend-architecture-complete.md](./backend-architecture-complete.md) | 完整技术说明（架构、模块、部署） |
| [refactoring-report.md](./refactoring-report.md) | 重构实施状态与后续项 |
| [architecture-diagrams.md](./architecture-diagrams.md) | 架构图集 |
| [handbook/00-architecture-analysis-and-optimization.md](./handbook/00-architecture-analysis-and-optimization.md) | 功能域与 SQL 优化 |

---

*维护：插件、菜单 catalog 或 seed 行为变更时，请同步更新本文、`plugins.md` 与 `backend-architecture-complete.md` §3.3。*
