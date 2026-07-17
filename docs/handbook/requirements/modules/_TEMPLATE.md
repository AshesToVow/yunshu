# 模块开发需求文档模板（标准）

> 复制本模板新建 `M-XX-<slug>.md`。接口以 `docs/apipost/permission-system.openapi.yaml` 与 `web/src/constants/api-catalog.ts` 为准；表以 `internal/model` + `docs/handbook/database/schema-and-relationships.md` 为准。

| 项 | 内容 |
|----|------|
| 文档编号 | M-XX |
| 模块名称 | |
| 插件开关 | `plugins.enabled` 中的键（无则写「核心，常开」） |
| 前端入口 | 菜单 path / component |
| 后端包 | `internal/service/<pkg>`、`internal/handler`、`internal/router/register_*.go` |
| 版本/日期 | |
| 状态 | 草案 / 已对齐源码 |

## 1. 背景与目标

- **业务目标**：
- **非目标**：

## 2. 角色与权限

| 角色/能力 | 说明 |
|-----------|------|
| | Casbin `permissions.resource` + `action` |

## 3. 功能需求

| ID | 功能 | 优先级 | 说明 |
|----|------|--------|------|
| F-01 | | P0 | |

## 4. 接口规格

**Base**：`/api/v1`  
**鉴权**：除特别说明外均为 `Authorization: Bearer <token>`  
**统一响应**：见 [http-api-conventions.md](../../api/http-api-conventions.md)

### 4.x `METHOD /path`

| 项 | 说明 |
|----|------|
| 请求方式 | GET / POST / PUT / DELETE |
| 鉴权 | 是 / 否 |
| Path 参数 | |
| Query 参数 | |
| Body | JSON 字段表 |
| 成功 `data` | |
| 错误 | 常见 `error_code` / HTTP |
| 权限资源 | 与 seed `permissions` 一致 |

## 5. 数据模型

| 表名 | 说明 | 关键字段 |
|------|------|----------|
| | | |

## 6. 外部依赖

| 依赖 | 用途 |
|------|------|
| | MySQL / Redis / ES / K8s / Jenkins / MinIO / SSH |

## 7. 非功能与约束

- 超时、幂等、审计、插件 path 同步（`path_filter.go` ↔ `plugin-path.ts`）

## 8. 验收标准

- [ ] …

## 9. 相关文档

- 菜单级：`../menus/menu-*.md`
- 设计/运维：
