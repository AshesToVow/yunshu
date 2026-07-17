# M-06 Kubernetes 控制台

| 项 | 内容 |
|----|------|
| 文档编号 | M-06 |
| 模块名称 | Kubernetes 控制台 |
| 插件 | `k8s` |
| 前端 | `/kubernetes/*`、`/kubernetes-crd/*` |
| 后端 | `internal/service/k8s`、`register_k8s_routes.go` |
| 状态 | 已对齐源码 |

## 1. 目标

纳管多集群，提供命名空间/节点/工作负载/配置/网络/存储/RBAC/Event/Helm/拓扑等控制台能力；权限叠加 **Casbin API + 集群档位 + 命名空间黑白名单**。

## 2. 功能范围（菜单）

集群、Namespace、Node、组件状态、Pod、Deployment/StatefulSet/DaemonSet、Job/CronJob、ConfigMap/Secret、Service、PV/PVC/StorageClass、Ingress/IngressClass、NetworkPolicy、RBAC 四件套、Event、Event 转发、ServiceAccount、API 资源发现、HPA、Helm Release/Chart、资源拓扑、CRD/CR。

通用列表模式见 [menu-k8s-resource-pattern.md](../menus/menu-k8s-resource-pattern.md)。

## 3. 接口规格（模式）

多数资源遵循：

| 能力 | 典型路径 | Query |
|------|----------|-------|
| 列表 | `GET /api/v1/<resource>` | `cluster_id`, `namespace?`, `keyword?` |
| 详情 | `GET /api/v1/<resource>/detail` | `cluster_id,namespace,name` |
| YAML 应用 | `POST /api/v1/<resource>/apply` | body: manifest |
| 删除 | `DELETE ...` | 名称 + 删除选项 |

### 集群管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/v1/clusters`、`/:id` | 集群 CRUD |
| PUT/GET | `/api/v1/clusters/:id/status` | 启停 / 连接状态 |
| GET | `/api/v1/clusters/:id/namespaces` | 命名空间 |
| GET | `/api/v1/clusters/:id/component-statuses` | 组件 |
| GET | `/api/v1/clusters/:id/api-resources` | API 发现 |

### 工作负载示例（Deployment）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/deployments` | 列表 |
| GET | `/api/v1/deployments/detail` | 详情 |
| POST | `/api/v1/deployments/scale` | 水平扩缩 |
| POST | `/api/v1/deployments/container-resources` | 垂直扩缩 |
| POST | `/api/v1/deployments/restart` | 滚动重启 |

### Event 转发

| 方法 | 路径 | 说明 |
|------|------|------|
| CRUD | `/api/v1/k8s/event-forward/rules` | 转发规则 |
| GET/PUT | `/api/v1/k8s/event-forward/settings` | Worker 参数 |

### Helm

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/DELETE 等 | `/api/v1/helm/releases` | 安装/升级/回滚/卸载 |
| GET | `/api/v1/helm/harbor/charts` | Harbor Chart |

完整清单：`api-catalog.ts`「集群管理」「K8s 工作负载」及 OpenAPI。

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `k8s_clusters` | kubeconfig/连接信息（敏感字段加密） |
| K8s 档位相关授权表 | 与 `/k8s-scoped-policies` 对应 |
| Event 转发规则/设置表 | Event Forward |

运行时资源数据主要在 **目标集群 API**，不落业务库。

## 5. 权限模型

1. Casbin：HTTP path + method  
2. 集群档位：`readonly` / `readonly_exec` / `admin`  
3. Namespace 黑/白名单  

详见 [casbin-and-k8s-triple-policy.md](../../permissions/casbin-and-k8s-triple-policy.md)。

## 6. 相关文档

- [R-04-kubernetes-console.md](../R-04-kubernetes-console.md)
- [menus/_INDEX.md](../menus/_INDEX.md) Kubernetes 章节
