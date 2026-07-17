# 模块级开发需求文档索引

本文档体系按**业务模块**输出标准开发需求（目标、功能、接口、入参/出参、请求方式、数据库、依赖），与源码、菜单文档、OpenAPI 对齐。

> **说明**：`.codegraph/` 目录仅存放本机 CodeGraph 索引数据（已 gitignore），**不提交文档**。模块需求正文统一维护于本目录。

## 文档约定

| 约定 | 说明 |
|------|------|
| 编号 | `M-00`～`M-10` |
| 接口权威源 | `go run ./tools/genopenapi` → `docs/apipost/permission-system.openapi.yaml`；对照 `web/src/constants/api-catalog.ts` |
| 权限种子 | `cmd/seed.go` 中 `permissions` |
| 菜单 | `internal/menu/catalog.go`；菜单级细文档见 [../menus/_INDEX.md](../menus/_INDEX.md) |
| 数据库 | GORM 模型 `internal/model`；关系见 [../../database/schema-and-relationships.md](../../database/schema-and-relationships.md) |
| HTTP 约定 | [../../api/http-api-conventions.md](../../api/http-api-conventions.md) |
| 模板 | [_TEMPLATE.md](./_TEMPLATE.md) |

## 模块一览

| 编号 | 模块 | 插件 | 文档 |
|------|------|------|------|
| M-00 | 平台约定与横切能力 | 核心 | [M-00-platform-conventions.md](./M-00-platform-conventions.md) |
| M-01 | 认证与身份 | 核心 | [M-01-auth-identity.md](./M-01-auth-identity.md) |
| M-02 | 系统管理 | 核心 | [M-02-system-admin.md](./M-02-system-admin.md) |
| M-03 | 项目管理与 CMDB | `project` / `cmdb` | [M-03-project-cmdb.md](./M-03-project-cmdb.md) |
| M-04 | 日志平台（Loggie + ES） | `project` | [M-04-log-platform.md](./M-04-log-platform.md) |
| M-05 | 告警通知 | `alert` | [M-05-alert-notify.md](./M-05-alert-notify.md) |
| M-06 | Kubernetes 控制台 | `k8s` | [M-06-kubernetes.md](./M-06-kubernetes.md) |
| M-07 | 数据库管理 | `dbmgmt` | [M-07-dbmgmt.md](./M-07-dbmgmt.md) |
| M-08 | MySQL 备份 | `backup` | [M-08-mysql-backup.md](./M-08-mysql-backup.md) |
| M-09 | CI/CD | `cicd` | [M-09-cicd.md](./M-09-cicd.md) |
| M-10 | 总览仪表盘 | 核心（图表可依赖 cicd） | [M-10-overview-dashboard.md](./M-10-overview-dashboard.md) |

## 与旧文档映射

| 旧域文档 | 模块文档 |
|----------|----------|
| R-01-auth-and-identity | M-01 |
| R-02-project-management | M-03（+ M-04 日志部分） |
| R-03-alert-and-monitor | M-05 |
| R-04-kubernetes-console | M-06 |
| R-05-system-administration | M-02 |
| R-06-log-platform-and-agent | **M-04**（已从 gRPC Agent 演进为 Loggie+ES） |

菜单级页面需求仍保留在 `../menus/`，模块文档侧重**跨页面的领域边界与接口/表清单**。

## 维护流程

1. 增删路由 → 更新 `register_*.go`、`api-catalog.ts`、`cmd/seed.go` → 重跑 genopenapi  
2. 同步修改本目录对应 `M-XX-*.md` 的「接口规格」「数据模型」  
3. 若菜单变化 → 更新 `catalog.go` 与 `menus/_INDEX.md`
