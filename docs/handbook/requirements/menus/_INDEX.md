# 菜单级需求文档索引

以下文档与 **内置菜单 catalog**（`internal/menu/catalog.go`）、**seed 同步**（`menu.Sync` → `cmd/seed.go`）、**前端路由**（`web/src/modules/*/routes.tsx`、`plugin-path.ts`、`dynamic-menu-page` 兜底）对齐。  
OpenAPI 全量路由见 **`docs/apipost/permission-system.openapi.yaml`**（由 `go run ./tools/genopenapi` 从 `internal/router/register_*.go` 生成）。

## 一级入口与固定路由

| 路由 | Component | 文档 |
|------|-----------|------|
| `/` 总览页面 | `dashboard-page` | [menu-root-dashboard.md](./menu-root-dashboard.md) |
| `/login` 登录页 | — | [menu-login.md](./menu-login.md) |
| `/plugins` 插件管理 | `plugins-page` | [menu-plugins.md](./menu-plugins.md) |
| `/personal-settings` | — | [menu-personal-settings.md](./menu-personal-settings.md) |
| `/server-console` | — | [menu-server-console.md](./menu-server-console.md) |

## 告警通知（`/alert-notify`）

| 路由 | Component | 文档 |
|------|-----------|------|
| `/alert-channels` | `alert-channels-page` | [menu-alert-channels.md](./menu-alert-channels.md) |
| `/alert-monitor-platform` | `alert-monitor-platform-page` | [menu-alert-monitor-platform.md](./menu-alert-monitor-platform.md) |
| `/alert-duty` | `alert-duty-page` | [menu-alert-duty.md](./menu-alert-duty.md) |

### 已隐藏 / 重定向

| 路由 | 重定向 | 文档 |
|------|--------|------|
| `/alert-config-center` | 监控平台「策略与联调」 | [menu-alert-config-center.md](./menu-alert-config-center.md) |
| `/alert-events` | 监控平台「历史」 | [menu-alert-events.md](./menu-alert-events.md) |

## 项目管理（`/project-management`）

| 路由 | Component | 文档 |
|------|-----------|------|
| `/projects` | `projects-page` | [menu-projects.md](./menu-projects.md) |
| `/project-members` | `project-members-page` | [menu-project-members.md](./menu-project-members.md) |
| `/project-servers` | `project-servers-page` | [menu-project-servers.md](./menu-project-servers.md) |
| `/project-services` | `project-services-page` | [menu-project-services.md](./menu-project-services.md) |
| `/project-log-sources` | `project-log-sources-page` | [menu-project-log-sources.md](./menu-project-log-sources.md) |
| `/project-logs` | `project-logs-page` | [menu-project-logs.md](./menu-project-logs.md) |
| `/agent-list` | `agent-list-page` | [menu-agent-list.md](./menu-agent-list.md) |
| `/mysql-backup` | `mysql-backup-page` | [menu-mysql-backup.md](./menu-mysql-backup.md) |

## 数据库管理（`/dbmgmt`，插件 `dbmgmt` + `project`）

完整说明见 [docs/dbmgmt.md](../../../dbmgmt.md) 与 [menu-dbmgmt.md](./menu-dbmgmt.md)。

### 资源申请

| 路由 | Component |
|------|-----------|
| `/dbmgmt/apply/database` | `dbmgmt-database-apply-page` |
| `/dbmgmt/apply/query` | `dbmgmt-query-apply-page` |
| `/dbmgmt/apply/app-user` | `dbmgmt-app-user-apply-page` |
| `/dbmgmt/apply/query-grants` | `dbmgmt-query-grants-page` |

### 资源管理

| 路由 | Component |
|------|-----------|
| `/dbmgmt/instances` | `dbmgmt-instances-page` |
| `/dbmgmt/instances/:id` | `dbmgmt-instance-detail-page` |
| `/dbmgmt/access-requests/all` | `dbmgmt-access-requests-page` |

### SQL 操作

| 路由 | Component |
|------|-----------|
| `/dbmgmt/sql/query` | `dbmgmt-console-page`（mode=query） |
| `/dbmgmt/sql/audit` | `dbmgmt-console-page`（mode=audit） |

### 工单管理

| 路由 | Component |
|------|-----------|
| `/dbmgmt/workflow/pending` | `dbmgmt-todo-page` |
| `/dbmgmt/workflow/history` | `dbmgmt-tickets-page` |
| `/dbmgmt/workflow/tickets/:ticketId` | `dbmgmt-ticket-detail-page` |
| `/dbmgmt/approval-flow` | `dbmgmt-approval-flow-page` |

### 其他

| 路由 | Component |
|------|-----------|
| `/dbmgmt/audit` | `dbmgmt-audit-page` |
| `/dbmgmt/grants` | `dbmgmt-grants-page`（菜单隐藏） |

### 旧路径重定向

| 旧路由 | 新路由 |
|--------|--------|
| `/dbmgmt/console` | `/dbmgmt/sql/query` 或 `/dbmgmt/sql/audit` |
| `/dbmgmt/todo` | `/dbmgmt/workflow/pending` |
| `/dbmgmt/tickets` | `/dbmgmt/workflow/history` |
| `/dbmgmt/access-requests` | `/dbmgmt/access-requests/all` |

## CI/CD（`/cicd`，插件 `cicd` + `project`）

| 路由 | Component | 文档 |
|------|-----------|------|
| `/cicd/services` | `cicd-services-page` | [menu-cicd.md](./menu-cicd.md) |
| `/cicd/todo` | `cicd-todo-page` | 同上 |
| `/cicd/approval-flow` | `cicd-approval-flow-page` | 同上 |
| `/cicd/build-records` | `cicd-build-records-page` | 同上 |
| `/cicd/release-records` | `cicd-release-records-page` | 同上 |

## 系统管理（`/system`）

| 路由 | Component | 文档 |
|------|-----------|------|
| `/users` | `users-page` | [menu-users.md](./menu-users.md) |
| `/user-groups` | `user-groups-page` | [menu-user-groups.md](./menu-user-groups.md) |
| `/departments` | `departments-page` | [menu-departments.md](./menu-departments.md) |
| `/roles` | `roles-page` | [menu-roles.md](./menu-roles.md) |
| `/permissions` | `permissions-page` | [menu-permissions.md](./menu-permissions.md) |
| `/policies` | `policies-page` | [menu-policies.md](./menu-policies.md) |
| `/k8s-scoped-policies` | `k8s-scoped-policies-page` | [menu-k8s-scoped-policies.md](./menu-k8s-scoped-policies.md) |
| `/registrations` | `registrations-page` | [menu-registrations.md](./menu-registrations.md) |
| `/menus` | `menus-page` | [menu-menus.md](./menu-menus.md) |
| `/dict-entries` | `dict-entries-page` | [menu-dict-entries.md](./menu-dict-entries.md) |
| `/login-logs` | `login-logs-page` | [menu-login-logs.md](./menu-login-logs.md) |
| `/operation-logs` | `operation-logs-page` | [menu-operation-logs.md](./menu-operation-logs.md) |
| `/banned-ips` | `banned-ips-page` | [menu-banned-ips.md](./menu-banned-ips.md) |

## Kubernetes 容器管理（`/kubernetes`）

共性模式：[menu-k8s-resource-pattern.md](./menu-k8s-resource-pattern.md)

| 路由 | Component | 文档 |
|------|-----------|------|
| `/clusters` | `cluster-page` | [menu-clusters.md](./menu-clusters.md) |
| `/namespaces` | `namespaces-page` | [menu-namespaces.md](./menu-namespaces.md) |
| `/nodes` | `nodes-page` | [menu-nodes.md](./menu-nodes.md) |
| `/component-status` | `component-status-page` | [menu-component-status.md](./menu-component-status.md) |
| `/pods` | `pod-page` | [menu-pods.md](./menu-pods.md) |
| `/deployments` | `deployments-page` | [menu-deployments.md](./menu-deployments.md) |
| `/statefulsets` | `statefulsets-page` | [menu-statefulsets.md](./menu-statefulsets.md) |
| `/daemonsets` | `daemonsets-page` | [menu-daemonsets.md](./menu-daemonsets.md) |
| `/cronjobs` | `cronjobs-page` | [menu-cronjobs.md](./menu-cronjobs.md) |
| `/jobs` | `jobs-page` | [menu-jobs.md](./menu-jobs.md) |
| `/configmaps` | `configmaps-page` | [menu-configmaps.md](./menu-configmaps.md) |
| `/secrets` | `secrets-page` | [menu-secrets.md](./menu-secrets.md) |
| `/k8s-services` | `k8s-services-page` | [menu-k8s-services.md](./menu-k8s-services.md) |
| `/persistentvolumes` | `persistentvolumes-page` | [menu-persistentvolumes.md](./menu-persistentvolumes.md) |
| `/persistentvolumeclaims` | `persistentvolumeclaims-page` | [menu-persistentvolumeclaims.md](./menu-persistentvolumeclaims.md) |
| `/storageclasses` | `storageclasses-page` | [menu-storageclasses.md](./menu-storageclasses.md) |
| `/ingresses` | `ingresses-page` | [menu-ingresses.md](./menu-ingresses.md) |
| `/ingress-classes` | `ingress-classes-page` | [menu-ingress-classes.md](./menu-ingress-classes.md) |
| `/network-policies` | `network-policies-page` | [menu-network-policies.md](./menu-network-policies.md) |
| `/rbac/roles` | `rbac-roles-page` | [menu-rbac-roles.md](./menu-rbac-roles.md) |
| `/rbac/rolebindings` | `rbac-rolebindings-page` | [menu-rbac-rolebindings.md](./menu-rbac-rolebindings.md) |
| `/rbac/clusterroles` | `rbac-clusterroles-page` | [menu-rbac-clusterroles.md](./menu-rbac-clusterroles.md) |
| `/rbac/clusterrolebindings` | `rbac-clusterrolebindings-page` | [menu-rbac-clusterrolebindings.md](./menu-rbac-clusterrolebindings.md) |
| `/events` | `events-page` | [menu-events.md](./menu-events.md) |
| `/k8s/event-forward` | `k8s-event-forward-page` | [menu-k8s-event-forward.md](./menu-k8s-event-forward.md) |
| `/serviceaccounts` | `serviceaccounts-page` | [menu-serviceaccounts.md](./menu-serviceaccounts.md) |
| `/cluster-api-resources` | `cluster-api-resources-page` | [menu-cluster-api-resources.md](./menu-cluster-api-resources.md) |
| `/horizontal-pod-autoscalers` | `horizontal-pod-autoscalers-page` | [menu-horizontal-pod-autoscalers.md](./menu-horizontal-pod-autoscalers.md) |

## Kubernetes CRD 管理（`/kubernetes-crd`）

| 路由 | Component | 文档 |
|------|-----------|------|
| `/crds` | `crds-page` | [menu-crds.md](./menu-crds.md) |
| `/crs` | `crs-page` | [menu-crs.md](./menu-crs.md) |

## 其它（已废弃侧栏）

| 路由 | 重定向 | 文档 |
|------|--------|------|
| `/runtime-config` | `/dict-entries` | [menu-runtime-config.md](./menu-runtime-config.md) |
