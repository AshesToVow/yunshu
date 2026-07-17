# M-03 项目管理与 CMDB

| 项 | 内容 |
|----|------|
| 文档编号 | M-03 |
| 模块名称 | 项目管理与 CMDB（服务器/云账号/终端） |
| 插件 | `project`、`cmdb` |
| 前端 | `/project-management/*`、`/server-console` |
| 后端 | `internal/service/project`、`internal/service/cmdb` |
| 状态 | 已对齐源码 |

## 1. 目标

以**项目**为租户管理成员、服务器资产、云账号同步、SSH/Web 终端；为日志平台（M-04）、备份（M-08）、dbmgmt（M-07）、CI/CD（M-09）提供服务器与项目边界。

## 2. 功能需求

| ID | 功能 | 说明 |
|----|------|------|
| F-01 | 项目 CRUD | `code` 唯一 |
| F-02 | 项目成员 | owner/成员角色 |
| F-04 | 服务器分组/CRUD | 分组树 + 主机 |
| F-05 | 连通性测试/同步/导入导出 | SSH 探测 |
| F-06 | 云账号与实例同步 | 云厂商 API |
| F-07 | 命令执行 / WebSocket 终端 | 需 ws-ticket |

> 服务与日志源配置见 **M-04**（菜单已整合为「服务与日志源」）。

## 3. 接口规格

### 3.1 项目与成员

| 方法 | 路径 | Query/Body | 结果 |
|------|------|------------|------|
| GET | `/api/v1/projects` | `page,page_size,keyword` | 分页 |
| POST | `/api/v1/projects` | `{ name, code, ... }` | 项目；创建者自动 owner |
| PUT/DELETE | `/api/v1/projects/:id` | — | 更新/删除 |
| GET | `/api/v1/projects/:id/members` | — | 成员列表 |
| POST | `/api/v1/projects/:id/members` | `{ user_id, role }` | 添加 |
| PUT | `/api/v1/projects/:id/members/:memberId` | `{ role }` | 改角色 |
| DELETE | `/api/v1/projects/:id/members/:memberId` | — | 移除 |

### 3.2 服务器与分组

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:id/server-groups/tree` | 分组树 |
| POST/PUT/DELETE | `.../server-groups`、`.../:groupId` | 分组维护 |
| GET | `/api/v1/projects/:id/servers` | 分页；`keyword` 等 |
| POST | `/api/v1/projects/:id/servers` | Upsert（含凭据加密存储） |
| GET/DELETE | `.../servers/:serverId` | 详情/删除 |
| POST | `.../servers/:serverId/exec` | Body：`{ command }` → stdout/stderr/exit |
| GET | `.../servers/:serverId/terminal/ws` | WebSocket 终端 |
| POST | `.../servers/test`、`.../test/batch`、`.../sync` | 探测/批量/同步 |
| GET/POST | `.../servers/export`、`import`、`import-template` | 导入导出 |

### 3.3 云账号

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/api/v1/projects/:id/cloud-accounts` | 列表/创建 |
| PUT | `.../cloud-accounts/:accountId` | 更新 |
| PUT | `.../cloud-accounts/:accountId/sync` | 同步云实例 |

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `projects` | 项目租户 |
| `project_members` | `(project_id,user_id)` 唯一 |
| `server_groups` | 分组树 |
| `servers` | 主机：host/port/os 等 |
| `server_credentials` | SSH 凭据（加密） |
| `cloud_accounts` | 云账号密钥（加密） |

## 5. 依赖与安全

| 项 | 说明 |
|----|------|
| AES-GCM | SSH/云密钥加解密（`security.encryption_key`） |
| SSH | `internal/pkg/sshserver` |
| 审计 | 写操作走 OpAudit |

## 6. 验收

- [ ] 跨项目访问服务器返回业务错误  
- [ ] 凭据不明文回显  
- [ ] 终端需有效 JWT + ws-ticket  

## 7. 相关文档

- [R-02-project-management.md](../R-02-project-management.md)
- [menu-projects.md](../menus/menu-projects.md)、[menu-project-servers.md](../menus/menu-project-servers.md)
