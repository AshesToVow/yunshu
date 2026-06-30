# CI/CD 插件说明

**最后更新**: 2026-06-30  
**插件名**: `cicd`（须与 `project` 同时启用）

---

## 1. 能力概览

| 能力 | 说明 |
|------|------|
| 应用服务 | 项目内 CI/CD 服务定义（标识符、Jenkins Job 名、制品类型等） |
| CI 配置 | Jenkinsfile 仓库/分支、构建参数 |
| 发布配置 | SSH / K8s / MinIO 等多目标发布模板 |
| CI 打包 | 触发 Jenkins Job，同步构建状态与控制台日志 |
| CD 发布 | 多级审批流 + 执行发布（SSH 脚本 / K8s apply / MinIO 制品） |
| 待办与批量 | 待审批/待执行工单列表，支持批量通过/驳回/执行 |
| 总览图表 | 首页「项目上线趋势」「按人发布统计」读 `cicd_release_runs` |

**代码入口**：

- 插件：`internal/plugins/cicd/plugin.go`（`StartWorkers` → `RunSyncWorker` 同步 Jenkins 状态）
- 服务：`internal/service/cicd/service.go`
- 路由：`internal/router/register_cicd_routes.go`（前缀 `/api/v1/projects/:id/cicd/...`）
- 前端：`web/src/modules/cicd/routes.tsx`

---

## 2. 启用步骤

1. `configs/config.yaml`：

```yaml
plugins:
  enabled:
    - core
    - project
    - cicd
    # ...
```

2. `go run . migrate && go run . server`（创建 `cicd_*` 表）
3. **系统管理 → 数据字典** 配置（优先级高于 YAML 默认值，见 `internal/dictconfig/cicd.go`）：

| dict_type（示例） | 含义 |
|-------------------|------|
| `cicd_enabled` | 是否启用 CI/CD |
| `cicd_jenkins_base_url` | Jenkins 地址 |
| `cicd_jenkins_username` / `cicd_jenkins_api_token` | Jenkins 凭据 |
| `cicd_jenkins_job_folder` | Job 目录（可选） |
| `cicd_jenkinsfile_*` | Jenkinsfile 仓库与分支 |
| `cicd_minio_*` | 制品 MinIO（与备份 `minio_*` **独立**） |

4. 重启后端使字典与插件 Worker 生效。

---

## 3. API 前缀

所有接口在项目作用域下，需 **项目成员** 权限：

```text
GET/POST   /api/v1/projects/:id/cicd/services
GET/PUT    /api/v1/projects/:id/cicd/services/:serviceId/ci-config
GET/POST   /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs
POST       /api/v1/projects/:id/cicd/services/:serviceId/builds
POST       /api/v1/projects/:id/cicd/services/:serviceId/releases
GET        /api/v1/projects/:id/cicd/build-runs
GET        /api/v1/projects/:id/cicd/release-runs
GET/PUT    /api/v1/projects/:id/cicd/approval-flow
POST       /api/v1/projects/:id/cicd/release-runs/:runId/approve|reject|execute|terminate
```

完整列表见 `internal/router/register_cicd_routes.go` 或 OpenAPI。

---

## 4. 审批与通知

- 审批阶段定义：`cicd_approval_flow_stages`（项目级 `GET/PUT .../approval-flow`）
- 工单步骤：`cicd_release_approval_steps`，按阶段顺序推进
- 邮件通知：`internal/service/cicd/notify_email.go` 通过 **`UserRepository`** 解析用户邮箱（不直查 DB）

---

## 5. 后台 Worker

`cicd` 插件 `StartWorkers` 启动 `Service.RunSyncWorker`：

- 间隔：`config.cicd.run_sync_interval_seconds`（默认 15s，可被字典覆盖）
- 作用：轮询 Jenkins，更新 `cicd_build_runs` 状态与日志片段

---

## 6. 与其它模块关系

| 模块 | 关系 |
|------|------|
| `project` | 路由挂载、成员鉴权、服务器列表（发布目标） |
| `overview` | 读 `cicd_release_runs` 做图表；**cicd 未启用时返回空数据** |
| `dictconfig` | Jenkins / MinIO 运行期配置 |
| `backup` | MinIO 键名空间独立（`minio_*` vs `cicd_minio_*`） |

---

## 7. 前端菜单

内置 catalog（`internal/menu/catalog.go`）：

| 路径 | 页面 |
|------|------|
| `/cicd/services` | 应用服务 |
| `/cicd/todo` | 待办列表 |
| `/cicd/approval-flow` | 审批管理 |
| `/cicd/build-records` | CI 打包记录 |
| `/cicd/release-records` | CD 历史工单 |

路径规则：`web/src/modules/plugin-path.ts` 与 `internal/plugin/path_filter.go` 须同步（`/cicd` 与 `/api/v1/projects/*/cicd/*` 归属 `cicd` + `project`）。

---

## 8. 相关文档

- [plugins.md](./plugins.md) — 插件机制
- [CONTRIBUTING.md](../CONTRIBUTING.md) §5 — 路径规则
- [handbook/requirements/menus/menu-cicd.md](./handbook/requirements/menus/menu-cicd.md) — 菜单级需求
