# 菜单需求：菜单管理（`/menus`）

## 1. 定位

- **路由**：`/menus`，`MenusPage`。  
- **目标**：维护**动态侧边栏菜单树**（名称、路径、图标、`component` 懒加载名、排序、显隐）。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 树展示 | `GET /api/v1/menus/tree`（登录后拉取；Repository 单次查全表 O(n) 建树；Service 缓存 60s）。 |
| 新建 | `POST /api/v1/menus`。 |
| 批量状态 | `PUT /api/v1/menus/status`。 |
| 更新 | `PUT /api/v1/menus/:id`。 |
| 删除 | `DELETE /api/v1/menus/:id`。 |

## 3. 内置菜单与 seed

- **单一数据源**：`internal/menu/catalog.go` → `DefaultCatalog()`（支持任意层级子菜单）。  
- **同步**：`go run . seed` 调用 `internal/menu/sync.go` → `menu.Sync`，按 `(parent_id, path)` upsert。  
- **不再**在 `GET /menus/tree` 热路径执行 `ensure*` 补菜单逻辑。

## 4. 数据表

- `menus`。

## 5. 注意事项

- **`component`** 必须与 `web/src/pages/**/*-page.tsx` 或 `web/src/modules/*/routes.tsx` 的 loader 命名一致（如 `projects-page`）。  
- 父级目录可仅作分组，`component` 可空。  
- 修改内置菜单结构：改 catalog 后执行 `seed`；用户刷新或重新登录以拉新树。
