# 菜单需求：API 资源发现（`/cluster-api-resources`）

> **Catalog**：`internal/menu/catalog.go` · Component: `cluster-api-resources-page` · Plugin: `k8s`

## 1. 定位

- **路由**：`/cluster-api-resources`，`cluster-api-resources-page`。  
- **目标**：查看选定集群的 **Discovery API 资源列表**（类似 `kubectl api-resources`）。

## 2. 专用 API

| 能力 | 路径 |
|------|------|
| 资源发现 | `GET /api/v1/clusters/:id/api-resources` |

## 3. 权限

- 需选择集群 ID；走 Casbin + **K8s 三元策略**（只读 GET）。

## 4. 注意事项

- 与 CRD/CR 菜单配合使用，便于确认 GVR 与 scope（Namespaced/Cluster）。
