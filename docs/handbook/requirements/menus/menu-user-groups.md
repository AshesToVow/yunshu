# 菜单需求：用户组管理（`/user-groups`）

> **Catalog**：`internal/menu/catalog.go` · Component: `user-groups-page` · Plugin: `core`

## 1. 定位

- **路由**：`/user-groups`，`user-groups-page`（系统管理 → 用户组管理）。  
- **目标**：维护用户组及成员，用于批量授权与告警触达等场景。

## 2. 主要 API

| 能力 | 路径 |
|------|------|
| 列表 | `GET /api/v1/user-groups` |
| 新建 | `POST /api/v1/user-groups` |
| 详情 | `GET /api/v1/user-groups/:id` |
| 更新 | `PUT /api/v1/user-groups/:id` |
| 删除 | `DELETE /api/v1/user-groups/:id` |
| 成员 | `PUT /api/v1/user-groups/:id/users` |

## 3. 权限

- 各接口需在 Casbin 中授权；种子见 `cmd/seed.go` → `defaultPermissions`。

## 4. 注意事项

- 删除用户组前确认无业务策略依赖其成员列表。
