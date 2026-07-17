# M-08 MySQL 备份

| 项 | 内容 |
|----|------|
| 文档编号 | M-08 |
| 模块名称 | MySQL 备份 |
| 插件 | `backup` |
| 前端 | `/mysql-backup` |
| 后端 | `internal/service/mysqlbackup` |
| 状态 | 已对齐源码 |

## 1. 目标

按项目配置备份实例（mysqldump / xtrabackup / innobackupex），经 SSH 在目标机执行，产物上传 MinIO，支持任务列表、停止、预签名下载与 Cron 调度。

## 2. 功能需求

| ID | 功能 |
|----|------|
| F-01 | 备份实例 CRUD、连通测试、远端工具检查 |
| F-02 | 触发备份任务、列表、停止、删除记录 |
| F-03 | 预签名下载 |
| F-04 | mysqldump 选项字典 |

## 3. 接口规格

Base：`/api/v1/projects/:id/mysql-backup`

| 方法 | 路径 | 入参 | 结果 |
|------|------|------|------|
| GET | `/instances` | 分页 | 实例列表 |
| POST | `/instances` | 实例配置（主机/账号/备份类型/路径/MinIO…） | 实例 |
| PUT/DELETE | `/instances/:instanceId` | — | 更新/删除 |
| POST | `/instances/:instanceId/ping` | — | 连通结果 |
| POST | `/instances/:instanceId/check-remote` | — | 远端 xtrabackup 等检查 |
| POST | `/instances/:instanceId/run` | 可选参数 | 创建 job |
| GET | `/jobs` | 过滤 | 任务列表 |
| POST | `/jobs/:jobId/stop` | — | 停止运行中任务 |
| DELETE | `/jobs/:jobId` | — | 删记录 |
| GET | `/jobs/:jobId/presign` | — | 预签名 URL |
| GET | `/mysqldump-options` | — | 选项列表 |

## 4. 数据模型

| 表 | 说明 |
|----|------|
| MySQL 备份实例表 | 实例配置（见 `internal/model` mysqlbackup 相关） |
| 备份任务表 | 状态、日志、对象键 |

## 5. 依赖

| 依赖 | 用途 |
|------|------|
| SSH | 远端执行 |
| MinIO/S3 | 制品存储 |
| Cron | 定时调度（服务内） |

## 6. 相关文档

- [menu-mysql-backup.md](../menus/menu-mysql-backup.md)
