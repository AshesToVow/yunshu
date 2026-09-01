# Yunshu 业务插件（GVA 风格）

## 概述

业务功能以**编译期插件**形式组织：每个插件在 `init()` 中向全局注册表登记，由 `config.yaml` 的 `plugins.enabled` 控制是否在运行时加载路由、迁移与后台任务。

与 [gin-vue-admin 插件](https://www.gin-vue-admin.com/) 类似：**同一二进制、按配置启停模块**，而非 `plugin.Open` 动态 `.so`。

## 内置插件

| 插件名 | 包路径 | 职责 |
|--------|--------|------|
| `core` | `internal/plugins/core` | 认证、用户/角色/权限、菜单、字典、审计日志 |
| `k8s` | `internal/plugins/k8s` | 多集群 K8s 资源、K8s 权限策略、总览、事件转发 Worker |
| `alert` | `internal/plugins/alert` | 告警摄入、规则、值班、订阅、云到期 |
| `project` | `internal/plugins/project` | 多租户项目、成员、服务配置、日志 Agent 与日志流 |
| `cmdb` | `internal/plugins/cmdb` | CMDB 服务器资产：主机、分组、云账号、SSH/Web 终端 |
| `backup` | `internal/plugins/backup` | MySQL 备份调度与任务 |
| `cicd` | `internal/plugins/cicd` | CI/CD：Jenkins 打包、MinIO/SSH 发布、执行记录 |
| `dbmgmt` | `internal/plugins/dbmgmt` | 数据库管理：实例纳管、SQL 查询/审核、授权工单、goInception |
| `inspect` | `internal/plugins/inspect` | 项目级 Prometheus 巡检报告与定时调度 |

## 配置

```yaml
plugins:
  enabled:
    - core
    - k8s
    - alert
    - project
    - cmdb
    - backup
    - cicd
    - dbmgmt
    - inspect
```

- 省略 `plugins` 或 `enabled` 为空：启用上述默认全集。
- **CMDB 与 project**：服务器 API 仍挂在 `/api/v1/projects/:id/servers`（功能不变）；`cmdb` 负责路由与表迁移，`project` 负责项目上下文。使用服务器管理时请**同时启用** `project` + `cmdb`。
- **dbmgmt 与 project**：数据库 API 挂在 `/api/v1/projects/:id/dbmgmt/...`；须**同时启用** `project` + `dbmgmt`。详见 [dbmgmt.md](dbmgmt.md)。
- **inspect 与 project**：巡检 API 挂在 `/api/v1/projects/:id/inspect/...`；须**同时启用** `project` + `inspect`，并配置项目的 Prometheus `alert_datasources`。

## 新增插件步骤

1. 新建 `internal/plugins/<name>/plugin.go`，`init()` 中 `plugin.Register(&module{})`。
2. 实现 `plugin.Module`：`Models`、`PostMigrate`、`StartWorkers`（可选，如 alert 云到期 / cicd Jenkins 同步 / backup 调度）等。
3. 在 `internal/plugins/all/all.go` 增加 `_ "yunshu/internal/plugins/<name>"`。
4. 路由实现放在 `internal/router/register_<name>_routes.go`，并在 `router/plugin_bind.go` 注册。
5. 将插件名加入 `config.yaml` 的 `plugins.enabled`（及 `plugin.DefaultEnabled()`）。

### 完整清单（含前端）

| 步骤 | 位置 |
|------|------|
| 后端插件注册 | `internal/plugins/<name>/plugin.go` + `all/all.go` |
| 路由 | `register_<name>_routes.go` + `plugin_bind.go`（含 `RegisterCicdRoutes`） |
| 权限种子 | `cmd/seed.go` |
| 菜单 | `internal/menu/catalog.go` → `go run . seed` |
| 前端路由 | `web-pro/src/modules/<name>/routes.tsx` 或动态菜单 |
| **插件路径规则** | `internal/plugin/path_filter.go` **与** `web-pro/src/modules/plugin-path.ts`（必须同步） |
| OpenAPI | `go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml` |

详见 [CONTRIBUTING.md](../CONTRIBUTING.md)。

## CMDB 模块结构

CMDB 不是空脚手架，业务代码已从原 `project` 包完整迁移：

```text
internal/plugins/cmdb/plugin.go           # 插件注册 + 数据模型迁移
internal/service/cmdb/                    # 服务器/分组/云账号/SSH/终端 业务逻辑
  service.go                              # CMDBService 构造与加密
  servers.go                              # 主机 CRUD、连通性、导入导出、云同步
  cloud_provider_*.go                     # 阿里云/腾讯云/京东云 Provider
internal/handler/cmdb_handler.go          # HTTP + WebSocket 终端
internal/router/register_cmdb_routes.go   # 路由（仍挂在 /projects/:id/...）
web/src/modules/cmdb/routes.tsx           # 前端静态路由
web/src/modules/plugin-path.ts            # 菜单 path → cmdb 映射
```

API 路径与行为与拆分前一致；`project` 插件保留项目/成员/日志，`cmdb` 负责服务器资产。

## dbmgmt 模块结构

菜单对齐 **smartdbs** 四类：资源申请 / 资源管理 / SQL 操作 / 工单管理。

```text
internal/plugins/dbmgmt/plugin.go           # 插件注册 + db_* 表迁移
internal/service/dbmgmt/                    # 实例、SQL、授权、工单、审计
internal/handler/dbmgmt_handler.go          # HTTP 入口
internal/router/register_dbmgmt_routes.go   # /projects/:id/dbmgmt/*
internal/dictconfig/dbmgmt.go               # 数据字典覆盖
web/src/modules/dbmgmt/routes.tsx           # 前端路由 + 旧路径重定向
web/src/pages/dbmgmt-*.tsx                  # 各功能页
```

完整运维手册与 API 摘要见 [dbmgmt.md](dbmgmt.md)。

## API

```http
GET /api/v1/plugins
Authorization: Bearer <token>
```

前端通过 `PluginProvider` 拉取该接口，按 `enabled` 懒加载 `web/src/modules/*/routes.tsx` 并过滤侧栏菜单。管理页：`/plugins`。

## 内置菜单

平台侧栏内置菜单不在插件内重复定义，统一由 **`internal/menu/catalog.go`**（`DefaultCatalog`）维护，经 `go run . seed` → `menu.Sync` 写入 `menus` 表。详见 [handbook/requirements/menus/_INDEX.md](handbook/requirements/menus/_INDEX.md)。

## 依赖装配

插件**不**自行 Wire 装配；HTTP 依赖由 `router.InitializeRouteDeps` 统一构建，通过 `plugin.Runtime.Deps`（`*router.RouteDeps`）注入路由绑定。
