# M-09 CI/CD

| 项 | 内容 |
|----|------|
| 文档编号 | M-09 |
| 模块名称 | CI/CD |
| 插件 | `cicd` |
| 前端 | `/cicd/*` |
| 后端 | `internal/service/cicd` |
| 状态 | 已对齐源码 |

## 1. 目标

项目维度管理应用服务、Jenkins CI 构建、制品（MinIO）、多级审批 CD 发布与工单历史。

## 2. 功能与菜单

| 菜单 | 能力 |
|------|------|
| `/cicd/services` | 应用服务、CI 配置、发布配置、触发构建/发布 |
| `/cicd/todo` | 待办审批/执行 |
| `/cicd/approval-flow` | 审批流配置 |
| `/cicd/build-records` | CI 打包记录与日志 |
| `/cicd/release-records` | CD 历史工单与日志 |

## 3. 接口规格

Base：`/api/v1/projects/:id/cicd`

### 3.1 应用服务

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/services` | 列表/创建 |
| GET/PUT/DELETE | `/services/:serviceId` | 详情/更新/删除 |
| GET/PUT | `/services/:serviceId/ci-config` | CI 配置 |
| GET/POST/PUT/DELETE | `/services/:serviceId/deploy-configs` | 发布配置 |
| GET | `/services/:serviceId/artifacts` | MinIO 制品 |
| POST | `/services/:serviceId/builds` | 触发 CI |
| POST | `/services/:serviceId/releases` | 触发 CD |

### 3.2 构建记录

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/build-runs`、`/:runId` | 列表/详情 |
| GET | `/build-runs/:runId/log` | 构建日志 |
| DELETE | `/build-runs/:runId` | 删除 |

### 3.3 审批流与发布工单

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/PUT | `/approval-flow` | 审批流 |
| GET | `/release-runs`、`/:runId` | 工单列表/详情 |
| GET | `/release-runs/:runId/approval-steps` | 步骤 |
| POST | `/release-runs/:runId/approve\|reject\|execute\|terminate` | 单笔操作 |
| POST | `/release-runs/batch-*` | 批量审批/执行/终止 |
| GET | `/release-runs/:runId/log` | 发布日志 |
| DELETE | `/release-runs/:runId` | 删除工单 |

## 4. 数据模型

| 表（逻辑名） | 说明 |
|--------------|------|
| `cicd_services` 等 | 应用服务 |
| CI 配置 / 发布配置表 | 服务附属配置 |
| `cicd_build_runs` | 构建记录 |
| `cicd_release_runs` + 审批步骤 | 发布工单 |
| 审批流配置表 | 项目级阶段 |

（精确表名以 `internal/model` cicd 定义为准。）

## 5. 依赖

| 依赖 | 用途 |
|------|------|
| Jenkins | CI 构建 |
| MinIO | 制品 |
| 邮件/用户 | 发布通知 |
| 字典 | Jenkins/MinIO 连接配置 |

## 6. 相关文档

- [cicd.md](../../../cicd.md)
- [menu-cicd.md](../menus/menu-cicd.md)
- [K8S-CD-HELM-CHECKOUT.md](../../cicd/K8S-CD-HELM-CHECKOUT.md)
