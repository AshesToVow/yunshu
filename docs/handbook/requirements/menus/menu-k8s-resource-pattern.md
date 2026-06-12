# 通用说明：Kubernetes 列表/详情类菜单页

以下路由共用的交互与权限模式。**完整 catalog 与 component 名见 `internal/menu/catalog.go`**。

| 路由 | Component |
|------|-----------|
| `/namespaces` | `namespaces-page` |
| `/nodes` | `nodes-page` |
| `/deployments` | `deployments-page` |
| `/statefulsets` | `statefulsets-page` |
| `/daemonsets` | `daemonsets-page` |
| `/cronjobs` | `cronjobs-page` |
| `/jobs` | `jobs-page` |
| `/configmaps` | `configmaps-page` |
| `/secrets` | `secrets-page` |
| `/k8s-services` | `k8s-services-page` |
| `/persistentvolumes` | `persistentvolumes-page` |
| `/persistentvolumeclaims` | `persistentvolumeclaims-page` |
| `/storageclasses` | `storageclasses-page` |
| `/ingresses` | `ingresses-page` |
| `/ingress-classes` | `ingress-classes-page` |
| `/network-policies` | `network-policies-page` |
| `/serviceaccounts` | `serviceaccounts-page` |
| `/horizontal-pod-autoscalers` | `horizontal-pod-autoscalers-page` |
| `/events` | `events-page` |

专用说明见各菜单文档；`/pods`、`/clusters`、`/component-status`、`/k8s/event-forward`、`/cluster-api-resources`、RBAC 路由有独立文档。

## 1. 共同前提

- **集群必选**：页面需选择已注册集群（`k8s_clusters`），请求中携带 `cluster_id`（或等价查询参数，以前端服务为准）。  
- **命名空间**：多数工作负载与核心资源需 `namespace`；集群级资源（如 Node、ClusterRole）无 ns。  
- **中间件**：`Authorize` + `K8sScopeAuthorize`——除 API 权限外，需满足 **K8s 三元策略**（集群/命名空间/路径）；部分 GET 在仅有三元策略时也可放行（见权限手册）。

## 2. 典型能力矩阵

| 能力 | 常见 HTTP | 说明 |
|------|-----------|------|
| 列表 | `GET` 集合路径 | 分页/关键字因资源而异 |
| 详情/YAML | `GET .../detail` | 查询参数指定 name/ns |
| 应用 YAML | `POST .../apply` | Body 为清单 |
| 删除 | `DELETE` | 常带 name、namespace 查询参数 |
| 扩缩容/重启 | `POST .../scale`、`.../restart` | 工作负载类 |

具体路径以 **`docs/apipost/permission-system.openapi.yaml`** 为准。

## 3. 注意事项

- **只读角色**：勿开放 `apply`/`delete` 的 API 权限。  
- **生产变更**：建议配合变更窗口与审计（操作日志）。  
- **大集群**：列表默认分页，避免一次拉全量。

各菜单文档在「专用 API」一节只写与本资源不同的部分，其余默认符合本节。
