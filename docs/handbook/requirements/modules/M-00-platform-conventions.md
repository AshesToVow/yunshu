# M-00 平台约定与横切能力

| 项 | 内容 |
|----|------|
| 文档编号 | M-00 |
| 模块名称 | 平台约定（HTTP / 鉴权中间件 / 插件 / 错误） |
| 插件 | 核心，常开 |
| 状态 | 已对齐源码（2026-07） |

## 1. 目标

统一所有业务模块的请求入口、响应格式、鉴权、权限、审计与插件边界，保证二次开发可预期。

## 2. 请求链路

```text
Client → Gin (/api/v1)
  → Auth（JWT，公开路由除外）
  → Casbin Authorize（permissions 资源+方法）
  → （K8s 路由）K8sScope 三元策略
  → OpAudit（写操作审计）
  → Handler → Service → Repository → MySQL/Redis/外系统
```

入口：`cmd/server.go`、`internal/router/router.go`、`internal/router/plugin_bind.go`。

## 3. 统一响应

| 场景 | HTTP | body |
|------|------|------|
| 成功 | 200 | `{ "code": 200, "message": "success", "data": ... }` |
| 业务错误 | 4xx | `{ "code", "error_code", "message" }` |
| 未登录 | 401 | Token 无效/缺失 |
| 无权限 | 403 | Casbin 拒绝 |

分页 `data` 常见：`{ list, total, page, page_size }`。

## 4. 横切接口

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/v1/health` | 否 | 健康检查 |
| GET | `/api/v1/overview` | 是 | 总览指标（见 M-10） |

## 5. 插件

| 配置 | 文件 |
|------|------|
| `plugins.enabled` | `configs/config.yaml` |
| 后端 path 过滤 | `internal/plugin/path_filter.go` |
| 前端 path 过滤 | `web/src/modules/plugin-path.ts` |

业务插件示例：`project`、`cmdb`、`alert`、`k8s`、`dbmgmt`、`backup`、`cicd`。

## 6. 数据与依赖

| 组件 | 用途 |
|------|------|
| MySQL/PostgreSQL | 主库（GORM AutoMigrate） |
| Redis | 会话/验证码/缓存（按配置） |
| Casbin | `casbin_rule` 策略表 |

## 7. 相关文档

- [http-api-conventions.md](../../api/http-api-conventions.md)
- [casbin-and-k8s-triple-policy.md](../../permissions/casbin-and-k8s-triple-policy.md)
- [plugins.md](../../../plugins.md)
- [CODEBASE-MAP.md](../../../CODEBASE-MAP.md)
