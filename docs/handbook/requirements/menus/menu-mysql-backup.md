# 菜单需求：MySQL 备份（`/mysql-backup`）

> **Catalog**：`internal/menu/catalog.go` · Component: `mysql-backup-page` · Plugin: `backup`（需同时启用 `project`）

## 1. 定位

- **路由**：`/mysql-backup`，`mysql-backup-page`（项目管理下）。  
- **目标**：在项目作用域内配置 MySQL 备份实例、执行备份任务、预签名下载。

## 2. 主要 API（前缀 `/api/v1/projects/:id/mysql-backup`）

| 能力 | 方法 |
|------|------|
| 实例列表/创建 | `GET/POST .../instances` |
| 实例更新/删除 | `PUT/DELETE .../instances/:instanceId` |
| 连通测试 | `POST .../instances/:instanceId/ping` |
| 远端检查 | `POST .../instances/:instanceId/check-remote` |
| 执行备份 | `POST .../instances/:instanceId/run` |
| 任务列表 | `GET .../jobs` |
| 预签名下载 | `GET .../jobs/:jobId/presign` |
| mysqldump 选项 | `GET .../mysqldump-options` |

## 3. 权限

- 项目成员中间件 + Casbin；备份执行属高危操作，需严格控制角色。

## 4. 注意事项

- 备份文件存储依赖 MinIO 等配置（见数据字典 / 环境变量）。  
- `backup` 插件未启用时菜单可能被前端过滤，API 亦不可用。
