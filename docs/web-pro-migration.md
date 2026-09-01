# Yunshu 前端（web-pro）

## 状态：正式唯一前端

业务源码已全部位于 **`web-pro/src/`**。原 `web/`（Vite）目录已移除。

## 开发

```bash
cd web-pro
npm install
npm run dev    # :8000
```

## 架构要点

- **Pro 壳层**：`app.tsx` + `pages/dynamic-menu/`
- **业务页**：`src/pages/*-page.tsx` + `src/modules/*/routes.tsx`
- **原生 Pro 页**（逐步增加）：`/workflow/inbox`、`/login-logs`、`/operation-logs`
- **插件菜单**：`fetchProLayoutMenu()` + `modules/filter-menu.ts`

## 部署

- 产物：`web-pro/dist/`
- Docker：`web-pro/Dockerfile.frontend`（`docker compose` 已配置）

## 新增页面

1. 创建 `src/pages/foo-page.tsx` 并 export `FooPage`
2. 更新 `src/utils/legacy-page-registry.ts`（或后续自动化脚本）
3. 后端菜单 `component` 填 `foo-page`

## 原生 Pro 页（PageContainer + ProTable / ModalForm）

| component | 路由 | 说明 |
|-----------|------|------|
| `workflow-inbox-page` | `/workflow/inbox` | 我的待办 |
| `login-logs-page` | `/login-logs` | 登录日志 |
| `operation-logs-page` | `/operation-logs` | 操作日志 |
| `users-page` | `/users` | 用户（新建/编辑/角色/重置密码/导出） |
| `departments-page` | `/departments` | 部门树 + ModalForm |
| `roles-page` | `/roles` | 角色模板 + 分配用户 |
| `banned-ips-page` | `/banned-ips` | 封禁 IP |
| `menus-page` | `/menus` | 菜单树 + 入口权限绑定 |
| `permissions-page` | `/permissions` | 接口能力 + K8s 范围 + 补全 |
| `policies-page` | `/policies` | 授权编排（PageContainer + LegacyShell） |
| `cicd-release-records-page` | `/cicd/release-records` | CD 历史工单 + 详情/日志 |
| `cicd-build-records-page` | `/cicd/build-records` | CI 打包记录 ProTable + 详情/日志/AI |
| `cicd-services-page` | `/cicd/services` | 应用服务（PageContainer + LegacyShell，复用 drawers） |
| `cicd-registries-page` | `/cicd/registries` | 镜像仓库注册中心 + 清理策略 |
| `cicd-image-browser-page` | `/cicd/image-browser` | Harbor/Registry 镜像浏览 |
| `dbmgmt-instances-page` | `/dbmgmt/instances` | 数据库实例 ProTable + 表单 |
| `dict-entries-page` | `/dict-entries` | 数据字典（PageContainer） |
| `projects-page` | `/projects` | 项目列表（PageContainer + LegacyShell） |
| `dbmgmt-tickets-page` / `dbmgmt-workflow-history-page` | `/dbmgmt/workflow/history` | SQL 工单 ProTable |
| `dbmgmt-grants-page` | `/dbmgmt/grants` | 授权管理 |
| `dbmgmt-query-grants-page` | `/dbmgmt/apply/query-grants` | SQL 查询权限（同 grants 页 preset） |
| `esmgmt-connections-page` | `/esmgmt/connections` | ES 连接管理 |
| `user-groups-page` | `/user-groups` | 用户组（PageContainer） |
| `registrations-page` | `/registrations` | 注册审核 ProTable |
| `alert-channels-page` | `/alert-channels` | Webhook 告警通道（PageContainer） |
| `alert-duty-page` | `/alert-duty` | 值班总览甘特（PageContainer） |
| `alert-maintenance-page` | `/alert-maintenance` | 维护窗口 ProTable |
| `ai-approvals-page` | `/ai/approvals` | AI Tool 审批 ProTable |
| `ai-investigations-page` | `/ai/investigations` | AI 调查列表 + 详情 |
| `personal-settings-page` | `/personal-settings` | 个人设置（PageContainer + LegacyShell） |
| `plugins-page` | `/plugins` | 业务插件列表（LegacyShell） |
| `dashboard-page` | `/dashboard` | 运营总览 |
| `dbmgmt-audit-page` | `/dbmgmt/audit` | DB 审计日志 ProTable |
| `alert-quality-page` | `/alert-quality` | 告警质量治理 |
| `alert-events-page` | `/alert-events` | 重定向至监控平台历史 |
| `workflow-definitions-page` | `/workflow/definitions` | 审批流配置 |
| `dbmgmt-access-requests-page` | `/dbmgmt/access-requests/all` | 权限申请列表 |
| `dbmgmt-query-apply-page` | `/dbmgmt/apply/query` | 查询权限申请 |
| `dbmgmt-database-apply-page` | `/dbmgmt/apply/database` | 数据库创建申请 |
| `dbmgmt-app-user-apply-page` | `/dbmgmt/apply/app-user` | 应用用户权限申请 |
| `esmgmt-overview-page` | `/esmgmt/overview` | ES 集群概览 |
| `ai-center-page` | `/ai/center` | AI 运维能力中心 |
| `platform-templates-page` | `/platform-templates` | 模板中心 |
| `workflow-tickets-page` | `/workflow/tickets` | 跨域工单列表 |
| `cluster-page` | `/clusters` | 集群列表（PageContainer） |
| `pod-page` | `/pods` | Pod 管理（PageContainer） |
| `namespaces-page` 等 YamlCrud | `/namespaces` 等 | 共享 `YamlCrudPage` → PageContainer；Deployments/StatefulSets/Nodes 含 LegacyShell |
| `events-page` | `/events` | Event 事件 |
| `helm-charts-page` / `helm-releases-page` | `/helm/charts` · `/helm/releases` | Harbor Chart / Release |
| `component-status-page` | `/component-status` | 控制面组件状态 |
| `cluster-api-resources-page` | `/cluster-api-resources` | API 资源发现 |
| `crs-page` | `/crs` | CR 实例 |
| `k8s-resource-topology-page` | `/k8s-resource-topology` | 资源拓扑 |
| `k8s-scoped-policies-page` | `/k8s-scoped-policies` | 集群访问档位 |
| `k8s-cr-templates-page` | `/k8s-cr-templates` | CR/YAML 模板库 |
| `k8s-event-forward-page` | `/k8s/event-forward` | Event 转发 |
| `project-servers-page` | `/project-servers` | 服务器管理 |
| `project-inspect-page` | `/project-inspect` | 项目巡检 |
| `project-members-page` | `/project-members` | 项目成员 |
| `project-logs-page` | `/project-logs` | 日志检索 |
| `project-collect-config-page` | `/project-services` | 服务与日志采集 |
| `service-catalog-page` / `service-portrait-page` | `/service-catalog` · `/service-portrait` | 服务目录 / 画像 |
| `log-retention-page` / `loggie-status-page` | `/log-retention` · `/loggie-status` | 日志保留 / Agent |
| `server-console-page` | `/server-console` | SSH 控制台 |
| `mysql-backup-page` | `/mysql-backup` | MySQL 备份（LegacyShell） |
| `dbmgmt-sql-query-page` / `dbmgmt-sql-audit-page` | `/dbmgmt/sql/query` · `/dbmgmt/sql/audit` | SQL 控制台 |
| `esmgmt-console-page` | `/esmgmt/console` | ES REST 控制台 |
| `ai-assistant-page` | `/ai/assistant` | AI 运维助手 |
| `alert-monitor-platform-page` | `/alert-monitor-platform/:tab` | 告警平台 |
| `alert-config-center-page` | `/alert-config-center` | 重定向至告警平台 |
| `cicd-todo-page` / `dbmgmt-todo-page` | `/workflow/inbox?domain=…` | 旧待办 → 统一 inbox |
| `cicd-approval-flow-page` / `dbmgmt-approval-flow-page` | `/workflow/definitions?domain=…` | 旧审批流 → 统一 definitions |
| `project-cluster-log-page` | `/project-services?tab=cluster-log` | 集群采集并入采集页 |
| `dbmgmt-ticket-detail-page` | `/dbmgmt/workflow/tickets/:ticketId` | SQL 工单详情（PageContainer） |
| `dbmgmt-instance-detail-page` | `/dbmgmt/instances/:instanceId` | 实例详情（参数路由，不走菜单重定向） |

菜单 `component` 注册表已全部对齐；带路径参数的详情页仅通过静态路由进入。