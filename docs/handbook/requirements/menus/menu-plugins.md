# 菜单需求：插件管理（`/plugins`）

> **前端固定路由**（`web/src/modules/core/routes.tsx`），不在 `DefaultCatalog()` 种子菜单中；侧栏由 `plugin-path.ts` 映射到 `core` 插件。

## 1. 定位

- **路由**：`/plugins`，`PluginsPage`。  
- **目标**：查看已注册业务插件及当前 `plugins.enabled` 启停状态。

## 2. API

| 能力 | 路径 |
|------|------|
| 插件清单 | `GET /api/v1/plugins` |

## 3. 说明

- 插件启停由 **`configs/config.yaml`** 的 `plugins.enabled` 控制，需重启后端生效。  
- 详见 [docs/plugins.md](../../../plugins.md)。
