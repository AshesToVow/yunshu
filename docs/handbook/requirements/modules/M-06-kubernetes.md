# M-06 Kubernetes 控制台

| 项 | 内容 |
|----|------|
| 文档编号 | M-06 |
| 模块名称 | Kubernetes 控制台 |
| 插件 | `k8s` |
| 前端 | `/kubernetes/*`、`/kubernetes-crd/*`、动态菜单组件 |
| 后端 | `internal/service/k8s`、`register_k8s_routes.go`、Kom SDK |
| 状态 | 已对齐源码（含长期凭证与门禁加固） |

## 1. 目标

纳管多集群，提供命名空间/节点/工作负载/配置/网络/存储/RBAC/Event/Helm/拓扑等控制台能力。

权限叠加：

1. **Casbin API**
2. **集群档位 / 能力包**（快捷三档或勾选 `read`/`exec`/`restart`/…）
3. **命名空间黑白名单**
4. **凭证意图**（只读/可写 kubeconfig；平台集群授权）
5. **高危确认**（`confirm=true`）与写路径门禁

## 2. 功能范围（菜单）

集群、Namespace、Node、组件状态、Pod、Deployment/StatefulSet/DaemonSet、Job/CronJob、ConfigMap/Secret、Service、PV/PVC/StorageClass、Ingress/IngressClass、NetworkPolicy、RBAC 四件套、Event、Event 转发、ServiceAccount、API 资源发现、HPA、Helm Release/Chart、资源拓扑、CRD/CR。

通用列表模式见 [menu-k8s-resource-pattern.md](../menus/menu-k8s-resource-pattern.md)。

## 3. 接口规格（模式）

多数资源遵循：

| 能力 | 典型路径 | Query |
|------|----------|-------|
| 列表 | `GET /api/v1/<resource>` | `cluster_id`, `namespace?`, `keyword?` |
| 详情 | `GET /api/v1/<resource>/detail` | `cluster_id,namespace,name` |
| YAML 应用 | `POST /api/v1/<resource>/apply` | body: manifest（经 NS 策略 + ApplyManifest） |
| 预检 | `POST .../preview-apply` | 文本 diff + apiserver `DryRun=All` |
| 删除 | `DELETE ...` | 名称 + 删除选项 |

### 集群管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/v1/clusters`、`/:id` | 集群 CRUD；可写/只读 kubeconfig、高危确认开关 |
| PUT/GET | `/api/v1/clusters/:id/status` | 启停 / 连接状态 |
| GET | `/api/v1/clusters/:id/namespaces` | 命名空间 |
| GET | `/api/v1/clusters/:id/component-statuses` | 组件 |
| GET | `/api/v1/clusters/:id/api-resources` | API 发现 |

### Secret

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/secrets/detail` | 默认脱敏（`redacted=true`） |
| GET | `/api/v1/secrets/reveal` | 明文揭示（需 admin；审计 redact） |

### 高危运维

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/nodes/drain` | Drain；`dry_run` / `confirm` |
| POST | `/api/v1/rbac/apply` | RBAC YAML；须 `confirm` |
| DELETE | `/api/v1/helm/releases` | 卸载；Query `confirm=true` |
| POST | `/api/v1/pods/exec`、GET `/pods/exec/ws` | 终端 |

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

| 表 / 字段 | 说明 |
|-----------|------|
| `k8s_clusters` | 连接信息；`kubeconfig`（可写）、`kubeconfig_readonly`（可选）、`direct_config` 均 AES-GCM 加密 |
| `impersonate_*` | **已废弃**（列保留兼容，运行时忽略） |
| `require_destructive_confirm` | 高危操作是否强制 `confirm` |
| K8s 档位相关授权表 | 与 `/k8s-scoped-policies` 对应 |
| Event 转发规则/设置表 | Event Forward |

运行时资源数据主要在 **目标集群 API**，不落业务库。须配置 `security.encryption_key`，否则拒绝密封新凭证。

## 5. 权限与凭证模型

1. Casbin：HTTP path + method  
2. 集群档位：`readonly` / `readonly_exec` / `admin`  
3. Namespace 黑/白名单（白名单激活时按允许 NS 分别 List，避免全集群拉取）  
4. AccessIntent：`read` → 优先只读 kubeconfig；`write`/`exec` → 可写凭证  
5. `assertK8sWritable`：只读意图禁止变更  

连接与排障：[yunshu-cluster-connect.md](../../../../deploy/k8s/yunshu-cluster-connect.md)。  
详见 [casbin-and-k8s-triple-policy.md](../../permissions/casbin-and-k8s-triple-policy.md)。

## 6. 相关文档

- [R-04-kubernetes-console.md](../R-04-kubernetes-console.md)
- [menus/_INDEX.md](../menus/_INDEX.md) Kubernetes 章节
- 根目录 [README.md](../../../../README.md)「纳管 Kubernetes 集群」
