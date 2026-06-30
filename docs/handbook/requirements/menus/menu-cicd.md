# 菜单需求：CI/CD（`/cicd/*`）

> **插件**：须同时启用 `cicd` + `project`。详见 [docs/cicd.md](../../../cicd.md)。

## 1. 定位

- **父菜单**：`/cicd`（catalog 名称「CI/CD」）  
- **目标**：项目维度 Jenkins 打包、制品发布、多级审批与执行记录。

## 2. 子菜单

| 路由 | Component | 说明 |
|------|-----------|------|
| `/cicd/services` | `cicd-services-page` | 应用服务、CI/发布配置、触发构建/发布 |
| `/cicd/todo` | `cicd-todo-page` | 待审批/待执行工单 |
| `/cicd/approval-flow` | `cicd-approval-flow-page` | 项目级审批阶段定义 |
| `/cicd/build-records` | `cicd-build-records-page` | CI 打包记录与日志 |
| `/cicd/release-records` | `cicd-release-records-page` | CD 工单历史与详情 |

## 3. 主要 API

前缀 **`/api/v1/projects/:id/cicd`**（需项目成员），完整路由见 `internal/router/register_cicd_routes.go`。

## 4. 数据表

- `cicd_services`、`cicd_ci_configs`、`cicd_deploy_configs`
- `cicd_build_runs`、`cicd_release_runs`
- `cicd_approval_flow_stages`、`cicd_release_approval_steps`

## 5. 注意事项

- Jenkins / MinIO 等连接信息优先读 **数据字典**（`cicd_*` dict_type），修改后需确认后端已加载。  
- 禁用 `cicd` 插件后侧栏隐藏；总览 CI 图表为空属预期。  
- 邮件通知收件人解析走 `UserRepository`，用户须配置有效邮箱。
