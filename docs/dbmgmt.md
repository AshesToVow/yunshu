# 数据库管理（dbmgmt）插件说明

**最后更新**: 2026-07-12  
**插件名**: `dbmgmt`（须与 `project` 同时启用）

---

## 1. 能力概览

| 能力 | 说明 |
|------|------|
| 实例管理 | MySQL / PostgreSQL 接入、探活、SSH 隧道（可选） |
| 元数据浏览 | 库 / 表 / 列树形导航 |
| SQL 查询 | 只读 SELECT/SHOW/DESCRIBE/EXPLAIN，查询历史（`db_sql_executions`） |
| SQL 审核 | 变更 SQL / SQL 文件导入，goInception 预检，系统/人工审核，工单执行 |
| 平台授权 | 库表级 `db_access_grants`、查询行数上限 |
| 权限申请 | 新建库、库/表级权限、应用用户（CREATE USER / GRANT / 回收） |
| 审批工单 | 三阶段可配置；SQL 工单支持回滚、OSC 控制 |
| 审计日志 | 查询/执行/授权/工单/应用用户等操作审计 |

菜单结构对齐 **smartdbs** 四类模块，详见 [handbook/requirements/menus/menu-dbmgmt.md](handbook/requirements/menus/menu-dbmgmt.md)。

**代码入口**：

| 层级 | 路径 |
|------|------|
| 插件 | `internal/plugins/dbmgmt/plugin.go` |
| 服务 | `internal/service/dbmgmt/` |
| 路由 | `internal/router/register_dbmgmt_routes.go`（前缀 `/api/v1/projects/:id/dbmgmt/...`） |
| 字典 | `internal/dictconfig/dbmgmt.go` |
| 前端 | `web/src/modules/dbmgmt/routes.tsx` |

---

## 2. 启用步骤

1. `configs/config.yaml`：

```yaml
plugins:
  enabled:
    - core
    - project
    - dbmgmt

dbmgmt:
  query_timeout_seconds: 30
  max_result_rows: 1000
  max_import_file_mb: 50
  prod_force_approval: true
  forbid_self_approve: true
  goinception_enabled: true
  goinception_host: 127.0.0.1
  goinception_port: 4000
  goinception_backup: true
```

2. `go run . migrate && go run . seed && go run . server`
3. **系统管理 → 数据字典** 配置（优先级高于 YAML，分类「数据库管理」）：

| dict_type | 含义 |
|-----------|------|
| `dbmgmt_query_timeout_seconds` | 查询/执行超时（秒） |
| `dbmgmt_max_rows` | 查询结果最大行数 |
| `dbmgmt_max_import_file_mb` | SQL 文件导入大小上限（MB） |
| `dbmgmt_prod_force_approval` | 生产实例是否强制走审批（含 low 风险，禁止直执） |
| `dbmgmt_forbid_self_approve` | 提交人不可审批自己的工单/申请（超管豁免） |
| `dbmgmt_goinception_enabled` | 是否启用 goInception |
| `dbmgmt_goinception_host` / `dbmgmt_goinception_port` | goInception 地址 |
| `dbmgmt_goinception_backup` | 变更是否默认备份 |
| `dbmgmt_approval_sla_hours` | 审批 SLA（小时） |
| `dbmgmt_ping_interval_seconds` | 实例探活间隔 |
| `dbmgmt_max_concurrent_per_instance` | 单实例最大并发连接 |

4. 重启后端；在 **授权管理** 为业务角色勾选 `/api/v1/projects/*/dbmgmt/*` 相关 API。

---

## 3. 前端路由（smartdbs 对齐）

| 分组 | 路径 | 说明 |
|------|------|------|
| **资源申请** | `/dbmgmt/apply/database` | 新建库申请 |
| | `/dbmgmt/apply/query` | 平台查询权限申请 |
| | `/dbmgmt/apply/app-user` | 应用用户权限（新用户/加权限/加 IP/回收） |
| | `/dbmgmt/apply/query-grants` | 已生效查询授权列表 |
| **资源管理** | `/dbmgmt/instances` | 实例列表 |
| | `/dbmgmt/instances/:id` | 实例详情（DB / 用户 Tab） |
| | `/dbmgmt/access-requests/all` | 库表级权限申请 |
| **SQL 操作** | `/dbmgmt/sql/query` | 只读查询 + 查询历史 |
| | `/dbmgmt/sql/audit` | SQL 变更 / 文件变更审核 |
| **工单管理** | `/dbmgmt/workflow/pending` | 待审核（权限 + 应用用户 + SQL） |
| | `/dbmgmt/workflow/history` | 历史工单 |
| | `/dbmgmt/workflow/tickets/:ticketId` | 工单详情（SQL 下载、执行日志、回滚） |
| | `/dbmgmt/approval-flow` | 审批流配置 |
| **其他** | `/dbmgmt/audit` | 审计日志 |
| | `/dbmgmt/grants` | 完整授权管理（菜单隐藏，管理员） |

**旧路径重定向**（兼容书签）：

| 旧路径 | 新路径 |
|--------|--------|
| `/dbmgmt/console` | `/dbmgmt/sql/query` 或 `/dbmgmt/sql/audit` |
| `/dbmgmt/todo` | `/dbmgmt/workflow/pending` |
| `/dbmgmt/tickets` | `/dbmgmt/workflow/history` |
| `/dbmgmt/access-requests` | `/dbmgmt/access-requests/all` |

---

## 4. 运维操作手册

### 4.1 纳管数据库实例

1. 选择项目 → **资源管理 → 实例管理 → 新建**
2. 填写驱动（mysql/postgres）、主机、端口、管理员账号密码
3. 点击 **探活** 确认 `online`
4. 生产实例建议开启 `require_ticket_for_dml`；变更走 SQL 审核

### 4.2 平台查询权限

1. **资源申请 → 查询权限申请**：选择实例、库/表、勾选 SELECT
2. 审批通过后写入 `db_access_grants`，可在 **SQL 查询** 页执行只读 SQL
3. 查询历史记录在 `db_sql_executions`；审计摘要在 **审计日志**

### 4.3 SQL 变更（审核页）

1. **SQL 操作 → SQL 审核**：选择实例、目标库
2. 填写变更描述；选择 **系统审核** 或 **人工审核**、是否备份
3. 点击 **sql检测**（goInception 预检），通过后再 **提交**
4. 低风险可能直接执行；否则进入 **待审核** → 审批 → **执行**
5. **注意**：
   - SELECT/SHOW 等只读语句请在 **SQL 查询** 页执行
   - 启用 goInception 时支持多语句；未启用时仅支持单条或走 SQL 文件导入
   - 库名/表名/字段请用反引号 `` ` ``，不要用单引号

### 4.4 应用用户权限

| 申请类型 | 说明 |
|----------|------|
| 新用户创建 | CREATE USER + GRANT；密码由平台托管，可在实例详情查看 |
| 已存在用户新增权限 | 对已有 `user@host` 追加 GRANT |
| 已存在用户新增 IP | 为新 `@host` 创建账号并 GRANT USAGE 等 |
| 权限回收 | 仅展示用户已有权限，勾选后 REVOKE |

**MySQL 管理员账号要求**：实例配置的管理员（如 `root`）必须对目标库具备 **GRANT OPTION**。平台从云枢服务器 IP 连接 MySQL 时，需对 **`root@'<平台IP>'`** 单独授权，不能只授 `root@'%'`（若存在更具体的 host 条目会优先匹配）。

```sql
-- 示例：平台服务器 10.10.10.1 连接 MySQL 10.10.10.103
GRANT ALL PRIVILEGES ON *.* TO 'root'@'10.10.10.1' WITH GRANT OPTION;
FLUSH PRIVILEGES;
```

库级授权时，SUPER/PROCESS 等全局权限会自动拆分为 `ON *.*` 的独立 GRANT 语句。

### 4.5 工单回滚

1. 历史工单详情 → **回滚** Tab（需 goInception 开启备份）
2. 预览回滚 SQL → 提交回滚工单 → 审批 → 执行

---

## 5. API 前缀（摘要）

完整列表见 `internal/router/register_dbmgmt_routes.go` 或 OpenAPI。

```text
# 实例
GET/POST   /api/v1/projects/:id/dbmgmt/instances
POST       .../instances/:instanceId/ping
GET        .../instances/:instanceId/metadata/databases|tables|columns

# SQL
POST       .../instances/:instanceId/query
POST       .../instances/:instanceId/check
POST       .../instances/:instanceId/execute
POST       .../instances/:instanceId/import

# 授权与申请
GET/POST   .../dbmgmt/grants
GET/POST   .../dbmgmt/access-requests
GET/POST   .../dbmgmt/app-user-requests
POST       .../access-requests/:requestId/approve|reject
POST       .../app-user-requests/:requestId/approve|reject

# 工单
GET/POST   .../dbmgmt/tickets
POST       .../tickets/:ticketId/approve|reject|execute
GET        .../tickets/:ticketId/rollback
POST       .../tickets/:ticketId/rollback/submit

# 审计
GET        .../dbmgmt/executions
GET        .../dbmgmt/audit-logs
GET/PUT    .../dbmgmt/approval-flow
```

---

## 6. 数据表

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

模型定义：`internal/model/dbmgmt.go`

---

## 7. 权限与插件路径

- API 鉴权：Casbin + 项目成员（`project_access` 中间件）
- 写操作：检查 `db_access_grants` 有效权限（connect/query/dml/ddl/import/manage）
- 插件路径规则（须与前端同步）：
  - 后端：`internal/plugin/path_filter.go`
  - 前端：`web/src/modules/plugin-path.ts`
  - 规则：`/dbmgmt` + `/api/v1/projects/*/dbmgmt/*` 需同时启用 `dbmgmt` + `project`

---

## 8. 排障

| 现象 | 排查 |
|------|------|
| SQL 检测报「禁止多语句」 | 确认 `dbmgmt_goinception_enabled=true` 且 goInception 可达；**仅 MySQL 直连实例**支持 |
| 提交报「须启用至少一级审批」 | 所有环境均需配置审批流；空审批流不再自动通过权限申请/SQL/应用用户 |
| 查询报「无法解析 SQL 涉及的表」 | 复杂 SQL 仅整库（或 `*`）查询授权可放行；表级授权请用简单 `FROM/JOIN` 或申请库级权限 |
| 元数据看不到某些库/表 | 已按 `db_access_grants` 过滤；无授权库不会出现在树中 |
| 应用用户审批 1044 | 实例管理员对目标库无 GRANT OPTION；检查 `root@'<平台IP>'` 而非仅 `root@'%'` |
| GRANT 语法错误 1064 | 库级勿把 SUPER/GRANT 与库权限混在一条语句（已自动拆分，需新版本） |
| 审计日志只有查询 | 新版本已记录工单/授权/应用用户；历史数据无回填 |
| goInception 不可用 | SSH 隧道实例与 **PostgreSQL** 均不走 goInception，仅本地风险规则 + 人工审批 |

### PostgreSQL 说明

- 支持：纳管、探活、元数据、只读查询、平台库表授权、SQL 工单（人工审批）。
- **不支持**：goInception 预检/代执行/备份回滚/OSC；MySQL 应用账号 CREATE USER/GRANT 流程。
- PG 变更请依赖审批流与本地风险评估；勿期望 Inception 语法报告。

---

## 9. 相关文档

- [menu-dbmgmt.md](handbook/requirements/menus/menu-dbmgmt.md) — 菜单级需求
- [plugins.md](plugins.md) — 插件机制
- [logging.md](logging.md) — 日志与错误码
- MySQL 备份联动：`/mysql-backup`（`backup` 插件）
