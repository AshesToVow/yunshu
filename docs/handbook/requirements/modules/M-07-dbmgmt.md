# M-07 数据库管理（dbmgmt）

| 项 | 内容 |
|----|------|
| 文档编号 | M-07 |
| 模块名称 | 数据库管理 |
| 插件 | `dbmgmt`（依赖 `project`） |
| 前端 | `/dbmgmt/**` |
| 后端 | `internal/service/dbmgmt` |
| 状态 | 已对齐源码 |

## 1. 目标

在项目下纳管 MySQL/PostgreSQL 实例；提供平台查询授权、应用用户 GRANT、SQL 查询/审核（goInception）、审批工单与审计。

## 2. 功能域（菜单）

| 域 | 路由前缀 |
|----|----------|
| 资源申请 | `/dbmgmt/apply/*`（建库、查询权限、应用用户、查询授权管理） |
| 资源管理 | `/dbmgmt/instances`、`/dbmgmt/access-requests/all` |
| SQL 操作 | `/dbmgmt/sql/query`、`/dbmgmt/sql/audit` |
| 工单 | `/dbmgmt/workflow/*`、`/dbmgmt/approval-flow` |
| 审计 | `/dbmgmt/audit` |
| 授权（可隐藏） | `/dbmgmt/grants` |

## 3. 接口规格（模式）

项目作用域 Base：`/api/v1/projects/:id/dbmgmt/...`

| 能力 | 典型方法/路径 | 入参要点 | 结果 |
|------|---------------|----------|------|
| 实例 CRUD | GET/POST/PUT/DELETE `.../instances`、`.../instances/:instanceId` | 连接信息（密码加密存） | 实例 |
| 探活 | POST `.../instances/:instanceId/ping` | — | ok/错误 |
| 元数据 | GET `.../metadata/databases\|tables\|columns` | 库表过滤 | 列表 |
| 只读查询 | POST `.../instances/:instanceId/query` | SQL + 行数限制 | 结果集 |
| 变更执行 | POST `.../execute` / `import` | SQL/文件；走审核与审批 | 工单或执行结果 |
| 授权/申请/工单 | `.../grants`、`access-requests`、`tickets`、`approval-flow` | 申请单字段 | 分页/详情 |

权威清单：OpenAPI + [docs/dbmgmt.md](../../../dbmgmt.md) + [menu-dbmgmt.md](../menus/menu-dbmgmt.md)。

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `db_instances` | 实例，归属 `project_id` |
| `db_access_grants` | 平台用户库表授权 |
| `db_access_requests` / `_steps` | 权限申请 |
| `db_app_user_requests` / `_steps` | 应用用户申请 |
| `db_sql_tickets` / `_steps` | SQL 工单 |
| `db_sql_executions` | 执行历史 |
| `db_audit_logs` | 审计摘要 |
| `db_instance_accounts` | 托管账号 |
| `db_approval_flow_stages` | 审批流 |

## 5. 外部依赖

| 依赖 | 用途 |
|------|------|
| goInception | SQL 预检/备份/OSC |
| 目标 MySQL/PG | 元数据与执行 |
| 字典配置 | 超时、行数等（`dictconfig`） |

## 6. 验收

- [ ] 无授权不可查库表；元数据树仅展示已授权库/表  
- [ ] 写操作：`require_ticket_for_dml` 开启时一律走工单；须配置至少一级审批流（不再因空流自动通过）  
- [ ] 敏感密码不明文返回  
- [ ] PostgreSQL 无 goInception 依赖，变更走人工审批  

## 7. 相关文档

- [dbmgmt.md](../../../dbmgmt.md)
- [menu-dbmgmt.md](../menus/menu-dbmgmt.md)
