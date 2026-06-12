# 菜单需求：ServiceAccount 管理（`/serviceaccounts`）

> **Catalog**：`internal/menu/catalog.go` · Component: `serviceaccounts-page` · Plugin: `k8s`

## 1. 定位

- **路由**：`/serviceaccounts`，`serviceaccounts-page`。  
- **目标**：管理命名空间内 `ServiceAccount`（列表、YAML、apply、删除）。

## 2. 专用 API

| 能力 | 路径 |
|------|------|
| 列表 | `GET /api/v1/serviceaccounts` |
| 详情 | `GET /api/v1/serviceaccounts/detail` |
| 应用 YAML | `POST /api/v1/serviceaccounts/apply` |
| 删除 | `DELETE /api/v1/serviceaccounts` |

## 3. 权限

- **K8s 三元策略** + Casbin。

## 4. 其它

- 共性交互见 [menu-k8s-resource-pattern.md](./menu-k8s-resource-pattern.md)。
