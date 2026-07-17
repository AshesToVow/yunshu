# M-02 系统管理

| 项 | 内容 |
|----|------|
| 文档编号 | M-02 |
| 模块名称 | 系统管理（用户/组织/RBAC/菜单/字典/审计） |
| 前端 | `/system/*` |
| 后端 | `internal/service/system` |
| 状态 | 已对齐源码 |

## 1. 目标

平台级账号、组织、API 能力项、Casbin 授权、菜单、数据字典、登录/操作审计、封禁 IP、注册审核。

## 2. 功能与菜单

| 菜单 path | 能力 |
|-----------|------|
| `/users` | 用户 CRUD、角色分配、导入模板 |
| `/user-groups` | 用户组 |
| `/departments` | 部门树 |
| `/roles` | 角色模板 |
| `/permissions` | API 能力项元数据 |
| `/policies` | 角色绑定能力（Casbin） |
| `/k8s-scoped-policies` | K8s 集群档位（见 M-06） |
| `/registrations` | 注册审核 |
| `/menus` | 菜单树 |
| `/dict-entries` | 数据字典 |
| `/login-logs` / `/operation-logs` | 审计 |
| `/banned-ips` | 封禁 IP |

## 3. 接口规格（摘要）

### 3.1 用户

| 方法 | 路径 | 入参要点 | 结果 |
|------|------|----------|------|
| GET | `/api/v1/users` | `page,page_size,keyword` | 分页列表 |
| POST | `/api/v1/users` | 创建字段 | 用户 |
| GET/PUT/DELETE | `/api/v1/users/:id` | — | 详情/更新/删除 |
| PUT | `/api/v1/users/:id/roles` | `{ role_ids: [] }` | ok |
| GET | `/api/v1/users/import-template` | — | 文件 |

### 3.2 部门

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/departments/tree` | 树 |
| POST/PUT/DELETE | `/api/v1/departments`、`/:id` | CRUD，PUT 可迁移子树 |

### 3.3 角色 / 权限 / 策略

| 方法 | 路径 | 说明 |
|------|------|------|
| CRUD | `/api/v1/roles`、`/api/v1/permissions` | 角色与 API 能力项 |
| GET | `/api/v1/policies` | 策略列表 |
| POST | `/api/v1/policies` | 授予：`role_id` + `permission_id` |
| DELETE | `/api/v1/policies` | 撤销（body 同授予） |

### 3.4 菜单 / 字典 / 审计 / 封禁 / 注册

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/v1/menus...` | 菜单树维护 |
| GET/POST/PUT/DELETE | `/api/v1/dict/entries...` | 字典条目 |
| GET/DELETE | `/api/v1/login-logs`、`/operation-logs` | 列表与删除；支持批量 delete POST |
| CRUD | `/api/v1/security/banned-ips` | 封禁 IP |
| GET | `/api/v1/registrations` | 申请列表 |
| POST | `/api/v1/registrations/:id/review` | 审核通过/拒绝 |

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `users` / `roles` / `user_roles` | 账号与角色 |
| `user_groups` 及相关 | 用户组 |
| `departments` | 部门树 |
| `permissions` | resource + action |
| `casbin_rule` | Casbin 策略存储 |
| `menus` / `menu_permission_bindings` | 菜单与入口权限 |
| `dict_entries` | 字典 |
| `login_logs` / `operation_logs` | 审计 |
| `banned_ips`（或等价表名） | 封禁 |
| `registration_requests` | 注册 |

## 5. 约束

- `super-admin` 角色受保护，删除/降权需特殊规则。  
- 新增 API 必须同步 seed 权限与 api-catalog。  
- 菜单 `component` 须对应 `web/src/pages/*-page.tsx` 导出。

## 6. 相关文档

- [R-05-system-administration.md](../R-05-system-administration.md)
- [casbin-and-k8s-triple-policy.md](../../permissions/casbin-and-k8s-triple-policy.md)
