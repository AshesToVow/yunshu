# 数据库管理（dbmgmt）

## 概述

`dbmgmt` 插件提供项目级多类型数据库（MySQL / PostgreSQL）接入、SQL 查询与审核、库表级授权、权限申请与审批工单、SQL 执行审计等能力。菜单结构对齐 **smartdbs** 四类模块。

- **插件名**：`dbmgmt`（须与 `project` 同时启用）
- **API 前缀**：`/api/v1/projects/:id/dbmgmt/...`
- **完整说明**：[docs/dbmgmt.md](../../../dbmgmt.md)

## 菜单（smartdbs 对齐）

| 分组 | 路径 | 页面组件 | 说明 |
|------|------|----------|------|
| **资源申请** | `/dbmgmt/apply/database` | `dbmgmt-database-apply-page` | 数据库创建申请 |
| | `/dbmgmt/apply/query` | `dbmgmt-query-apply-page` | 平台查询权限申请（SELECT + 行数上限） |
| | `/dbmgmt/apply/app-user` | `dbmgmt-app-user-apply-page` | 应用用户（新用户/加权限/加 IP/回收） |
| | `/dbmgmt/apply/query-grants` | `dbmgmt-query-grants-page` | 已生效查询授权列表 |
| **资源管理** | `/dbmgmt/instances` | `dbmgmt-instances-page` | 实例 CRUD、探活 |
| | `/dbmgmt/instances/:id` | `dbmgmt-instance-detail-page` | 详情 · DB 管理 · 用户管理 |
| | `/dbmgmt/access-requests/all` | `dbmgmt-access-requests-page` | 库/表级权限申请 |
| **SQL 操作** | `/dbmgmt/sql/query` | `dbmgmt-console-page`（mode=query） | 只读查询 · 查询历史 |
| | `/dbmgmt/sql/audit` | `dbmgmt-console-page`（mode=audit） | SQL 变更 / 文件变更 · goInception |
| **工单管理** | `/dbmgmt/workflow/pending` | `dbmgmt-todo-page` | 待审核（权限 + 应用用户 + SQL） |
| | `/dbmgmt/workflow/history` | `dbmgmt-tickets-page` | 历史工单 |
| | `/dbmgmt/workflow/tickets/:ticketId` | `dbmgmt-ticket-detail-page` | 工单详情（执行日志、回滚、OSC） |
| | `/dbmgmt/approval-flow` | `dbmgmt-approval-flow-page` | 审批流配置 |
| **其他** | `/dbmgmt/audit` | `dbmgmt-audit-page` | 审计日志 |
| | `/dbmgmt/grants` | `dbmgmt-grants-page` | 完整授权管理（菜单隐藏） |

旧路径 `/dbmgmt/console`、`/dbmgmt/todo`、`/dbmgmt/tickets`、`/dbmgmt/access-requests` 自动重定向至新路径（见 [routes.tsx](../../../../web/src/modules/dbmgmt/routes.tsx)）。

## 配置

`configs/config.yaml`：

```yaml
plugins:
  enabled:
    - project
    - dbmgmt

dbmgmt:
  query_timeout_seconds: 30
  max_result_rows: 1000
  max_import_file_mb: 50
  goinception_enabled: true
  goinception_host: 127.0.0.1
  goinception_port: 4000
  goinception_backup: true
  prod_force_approval: true
  forbid_self_approve: true
```

**数据字典**（分类「数据库管理」，优先级高于 YAML）：

| dict_type | 含义 |
|-----------|------|
| `dbmgmt_query_timeout_seconds` | 查询/执行超时（秒） |
| `dbmgmt_max_rows` | 查询结果最大行数 |
| `dbmgmt_max_import_file_mb` | SQL 文件导入大小上限（MB） |
| `dbmgmt_prod_force_approval` | 生产实例是否强制走审批（含 low 风险） |
| `dbmgmt_forbid_self_approve` | 提交人不可审批自己的工单/申请 |
| `dbmgmt_goinception_enabled` | 是否启用 goInception |
| `dbmgmt_goinception_host` / `dbmgmt_goinception_port` | goInception 地址 |
| `dbmgmt_goinception_backup` | 变更是否默认备份 |
| `dbmgmt_approval_sla_hours` | 审批 SLA（小时） |
| `dbmgmt_ping_interval_seconds` | 实例探活间隔 |
| `dbmgmt_max_concurrent_per_instance` | 单实例最大并发连接 |

定义见 `internal/dictconfig/dbmgmt.go`。

## API 摘要

路由注册：`internal/router/register_dbmgmt_routes.go`。

| 分组 | 方法 | 路径（相对 `/api/v1/projects/:id/dbmgmt`） |
|------|------|---------------------------------------------|
| 实例 | GET/POST | `/instances` |
| | GET/PUT/DELETE | `/instances/:instanceId` |
| | POST | `/instances/:instanceId/ping` |
| 元数据 | GET | `/instances/:instanceId/metadata/databases\|tables\|columns` |
| SQL | POST | `/instances/:instanceId/query\|check\|execute\|import` |
| 授权 | GET/POST | `/grants` |
| | PUT/DELETE | `/grants/:grantId` |
| | GET | `/grants/effective` |
| 权限申请 | GET/POST | `/access-requests` |
| | POST | `/access-requests/:requestId/approve\|reject` |
| 应用用户 | GET/POST | `/app-user-requests` |
| | POST | `/app-user-requests/:requestId/approve\|reject` |
| MySQL 用户 | GET | `/instances/:instanceId/mysql-users` |
| | GET | `/instances/:instanceId/mysql-user-privileges` |
| | GET | `/instances/:instanceId/accounts/:accountId/password` |
| 工单 | GET | `/tickets`、`/tickets/:ticketId`、`/tickets/:ticketId/steps` |
| | POST | `/tickets/:ticketId/approve\|reject\|execute` |
| 回滚 | GET | `/tickets/:ticketId/rollback`、`/rollback/preview` |
| | POST | `/tickets/:ticketId/rollback/submit` |
| OSC | GET/POST | `/tickets/:ticketId/osc`、`/osc/:sqlsha1/control` |
| 审批流 | GET/PUT | `/approval-flow` |
| 历史/审计 | GET | `/executions`、`/audit-logs` |

## 数据表

| 表名 | 说明 |
|------|------|
| `db_instances` | 数据库实例 |
| `db_access_grants` | 平台用户库表级授权 |
| `db_access_requests` / `db_access_request_steps` | 权限申请与审批步骤 |
| `db_app_user_requests` / `db_app_user_request_steps` | 应用用户申请 |
| `db_sql_tickets` / `db_sql_ticket_steps` | SQL 工单 |
| `db_sql_executions` | SQL 执行历史（查询/变更） |
| `db_audit_logs` | 统一审计日志 |
| `db_instance_accounts` | 平台托管 MySQL 账号密码 |
| `db_approval_flow_stages` | 项目级审批流 |

模型：`internal/model/dbmgmt.go`。

## 权限模型（与 smartdbs 对照）

| smartdbs | yunshu dbmgmt |
|----------|---------------|
| 平台查询权限（QueryPrivileges） | `apply/query` → `db_access_grants`（仅 select） |
| 数据库创建申请 | `apply/database` → 审批后 CREATE DATABASE |
| 应用用户 GRANT | `apply/app-user` → 审批后 CREATE USER / GRANT / REVOKE |
| SQL 上线工单 | `sql/audit` → SQL 工单 + goInception |
| 待审核 / 历史工单 | `workflow/pending` / `workflow/history` |

写操作校验 `db_access_grants` 有效权限：`connect` / `query` / `dml` / `ddl` / `import` / `manage`。

## 审批流

默认三阶段（可配置用户组）：DBA → 安全 → 运维。全部通过后：

- **权限申请** → 写入 `db_access_grants`（新建库则 `CREATE DATABASE`）
- **应用用户申请** → 逐条执行 CREATE USER / GRANT / REVOKE SQL
- **SQL 工单** → `pending_execution`，审批人/提交人可执行

## 运维注意事项

1. **MySQL 管理员 GRANT OPTION**：平台从云枢服务器 IP 连接时，须对 **`管理员@'<平台IP>'`** 授权，不能只授 `@'%'`。
2. **只读 vs 变更**：SELECT/SHOW 等在 **SQL 查询** 页；DDL/DML 在 **SQL 审核** 页。
3. **多语句**：启用 goInception 后支持；未启用时仅单条或 SQL 文件导入。
4. **库级应用用户**：SUPER/PROCESS/GRANT 等全局权限自动拆分为 `ON *.*` 的独立 GRANT。
5. **审计 vs 查询历史**：`db_audit_logs`（审计摘要）与 `db_sql_executions`（完整 SQL 历史）分离。

## 与 MySQL 备份联动

MySQL 实例列表/详情可跳转 `/mysql-backup`（`backup` 插件）。

## 前端代码入口

| 文件 | 说明 |
|------|------|
| `web/src/modules/dbmgmt/routes.tsx` | 路由与旧路径重定向 |
| `web/src/modules/dbmgmt/dbmgmt-console.ts` | SQL 检测错误格式化 |
| `web/src/pages/dbmgmt-*.tsx` | 各功能页 |

插件路径规则（须与后端同步）：`web/src/modules/plugin-path.ts`、`internal/plugin/path_filter.go`。

## 后续规划

- 列级脱敏
