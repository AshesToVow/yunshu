# 菜单需求：K8s Event 转发（`/k8s/event-forward`）

> **Catalog**：`internal/menu/catalog.go` · Component: `k8s-event-forward-page` · Plugin: `k8s`

## 1. 定位

- **路由**：`/k8s/event-forward`，`k8s-event-forward-page`。  
- **目标**：配置集群 Event 转发规则与 Worker 参数，将 K8s Event 推送到外部 Webhook。

## 2. 主要 API（前缀 `/api/v1/k8s/event-forward`）

| 能力 | 路径 |
|------|------|
| 规则列表/创建 | `GET/POST /rules` |
| 规则详情/更新/删除 | `GET/PUT/DELETE /rules/:id` |
| Worker 参数 | `GET/PUT /settings` |

## 3. 权限

- Casbin API 权限；规则 CRUD 后 Worker 热启动（见 `k8s/eventforward` 包）。

## 4. 注意事项

- 多实例部署时 Webhook 目标建议使用可达的内网地址。  
- 未匹配规则的 Event 不会被标记为已处理，便于排查。
