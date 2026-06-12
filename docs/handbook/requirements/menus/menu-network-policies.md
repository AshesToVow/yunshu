# 菜单需求：网络策略（`/network-policies`）

> **Catalog**：`internal/menu/catalog.go` · Component: `network-policies-page` · Plugin: `k8s`

## 1. 定位

- **路由**：`/network-policies`，`network-policies-page`。  
- **目标**：管理 Kubernetes `NetworkPolicy` 资源（列表、YAML 详情、apply、删除）。

## 2. 专用 API

| 能力 | 路径 |
|------|------|
| 列表 | `GET /api/v1/network-policies` |
| 详情 | `GET /api/v1/network-policies/detail` |
| 应用 YAML | `POST /api/v1/network-policies/apply` |
| 删除 | `DELETE /api/v1/network-policies` |

## 3. 权限

- **K8s 三元策略** + Casbin；变更类操作通常需 **admin** 档位。

## 4. 其它

- 列表/集群/命名空间等共性交互见 [menu-k8s-resource-pattern.md](./menu-k8s-resource-pattern.md)。
