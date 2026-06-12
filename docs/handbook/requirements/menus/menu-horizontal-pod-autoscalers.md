# 菜单需求：HPA 弹性伸缩（`/horizontal-pod-autoscalers`）

> **Catalog**：`internal/menu/catalog.go` · Component: `horizontal-pod-autoscalers-page` · Plugin: `k8s`

## 1. 定位

- **路由**：`/horizontal-pod-autoscalers`，`horizontal-pod-autoscalers-page`。  
- **目标**：管理 `HorizontalPodAutoscaler` 资源。

## 2. 专用 API

| 能力 | 路径 |
|------|------|
| 列表 | `GET /api/v1/horizontal-pod-autoscalers` |
| 详情 | `GET /api/v1/horizontal-pod-autoscalers/detail` |
| 应用 YAML | `POST /api/v1/horizontal-pod-autoscalers/apply` |
| 删除 | `DELETE /api/v1/horizontal-pod-autoscalers` |

## 3. 权限

- **K8s 三元策略** + Casbin；apply/delete 通常需 **admin** 档位。

## 4. 其它

- 共性交互见 [menu-k8s-resource-pattern.md](./menu-k8s-resource-pattern.md)。
