# 日志与错误码规范

**最后更新**: 2026-08-22

本文说明 Yunshu 后端 **slog 日志** 与 **统一业务错误码** 的约定，供后端开发与前端联调使用。

---

## 1. 设计原则

- **只用标准库 `log/slog`**，不再使用已删除的 `logutil`、`logger.Biz()` 等二次封装。
- **三文件分流**：`info.log`（Info/Warn）、`error.log`（Error+）、`sql.log`（GORM SQL）。
- **生产级轮转**：文件输出使用 [lumberjack](https://github.com/natefinch/lumberjack) 按大小/天数自动轮转与压缩。
- **错误只打一处**：Service 层 `Pass`/`Reject` 不打日志；HTTP 边界 `LogAPI` 统一记录；真 500 用 `Internal*` 在 Service 打一次。
- **前端联调看 `error_code`**：HTTP JSON 与日志字段对齐，便于按码做提示与埋点。

---

## 2. 日志文件与配置

| 文件 | 级别 | 内容 |
|------|------|------|
| `logs/info.log` | Info、Warn | 访问日志、启动信息、可恢复告警 |
| `logs/error.log` | Error+ | API 失败、panic、内部异常 |
| `logs/sql.log` | Debug/Info（GORM） | SQL 语句、慢查询 |

`configs/config.yaml`：

```yaml
log:
  level: info           # 见下文「级别语义」
  format: json          # 生产建议 json；text | json
  output: both          # console | file | both
  file_path: ./logs
  max_size_mb: 100      # 单文件上限（MB），超出后轮转
  max_age_days: 30      # 保留天数
  max_backups: 10       # 保留备份文件数
  compress: true        # 轮转后 gzip 压缩
```

启动时 `logger.Init(app.Logger)` 会调用 `slog.SetDefault`，业务代码直接使用 `slog` 即可自动分流。

### 2.1 环境变量（`LOG_*`）

Viper 将 `log.*` 映射为环境变量（`.` → `_`，自动大写）：

| 环境变量 | 配置键 | 说明 | 示例 |
|----------|--------|------|------|
| `LOG_LEVEL` | `log.level` | 日志级别语义见下节 | `info` |
| `LOG_FORMAT` | `log.format` | `text` 或 `json` | `json` |
| `LOG_OUTPUT` | `log.output` | `console` / `file` / `both` | `both` |
| `LOG_FILE_PATH` | `log.file_path` | 日志目录 | `./logs` |
| `LOG_MAX_SIZE_MB` | `log.max_size_mb` | 单文件上限 MB | `100` |
| `LOG_MAX_AGE_DAYS` | `log.max_age_days` | 保留天数 | `30` |
| `LOG_MAX_BACKUPS` | `log.max_backups` | 备份文件数 | `10` |
| `LOG_COMPRESS` | `log.compress` | `true` / `false` | `true` |

容器部署时可在 `.env` 或 Deployment `env` 中覆盖，无需改 `config.yaml`。

### 2.2 `log.level` 级别语义

| 值 | info.log | error.log | sql.log |
|----|----------|-----------|---------|
| `debug` | Info+ | Error+ | **Debug+**（输出全部 SQL） |
| `info`（默认） | Info+ | Error+ | Info+（慢 SQL ≥200ms 等） |
| `warn` | Info+ | Error+ | Info+ |
| `error` | Info+ | Error+ | Info+ |

**注意**：`log.level=debug` 主要影响 **GORM SQL 详细程度**；应用业务代码目前以 Info/Warn/Error 为主，几乎无 Debug 级日志。排障访问链路与 API 错误时，重点看 `info.log` 与 `error.log`。

### 2.3 轮转与 retention

- 每个通道独立文件：`info.log`、`error.log`、`sql.log`。
- 达到 `max_size_mb` 时自动轮转，旧文件命名为 `info-2026-08-22T15-04-05.000.log` 等。
- 超过 `max_age_days` 或备份数超过 `max_backups` 的旧文件会被删除。
- `compress: true` 时对已轮转文件执行 gzip，节省磁盘。

---

## 3. 如何打日志

### 3.1 普通业务日志

```go
import "log/slog"

slog.Info("Started MySQL backup scheduler", "component", "mysql.backup", "tick_spec", spec)
slog.Warn("list pods failed", "component", "k8s.node", "error", err, "cluster_id", id)
slog.Error("Recovered HTTP panic", "component", "http.recovery", "panic", rec)
```

**约定字段**

| 字段 | 说明 |
|------|------|
| `component` | 模块名，如 `http.auth`、`mysql.backup`、`alert` |
| `request_id` | 由 `RequestLogger` 写入 context，经 `logger.With(ctx, …)` 自动带上 |
| `user_id` / `username` | 鉴权中间件写入 context 后自动带上 |
| `error` | 底层 `error` 值（结构化输出） |

域内可保留一行 helper，仅返回带 `component` 的 Logger：

```go
func alertLog() *slog.Logger {
    return slog.Default().With("component", "alert")
}
```

### 3.2 HTTP 请求上下文

```go
import logx "yunshu/internal/pkg/logger"

logx.With(c.Request.Context(), "component", "http.auth").Warn("parse token failed", "error", err)
```

中间件会在 context 中注入 `request_id`（响应头 `X-Request-ID` 同源）与当前用户。

### 3.3 访问日志

`middleware/request_logger.go` 每条 HTTP 请求自动写一条 `component=http.access`，含 `method`、`path`、`status`、`latency_ms`、`client_ip` 等，**无需 Handler 重复打**。

### 3.4 SQL

GORM 通过 `logger.NewGormLogger(logger.SQL, cfg.Log.Level)` 写入 `sql.log`，业务代码不要手写 SQL 日志。

---

## 4. 错误码与 HTTP 响应

### 4.1 JSON 格式

失败响应（`ErrorHandler` / `AbortWithError`）：

```json
{
  "code": 401,
  "reason": "Unauthorized",
  "message": "登录会话已过期，请重新登录",
  "error_code": "10010",
  "metadata": {}
}
```

| 字段 | 含义 | 前端用法 |
|------|------|----------|
| `code` | **HTTP 状态码**（200 段外的语义层） | 与 axios/fetch `status` 一致 |
| `reason` | 机器可读原因（OneX 风格） | 分支逻辑、监控标签 |
| `message` | **给用户看的文案** | 直接 Toast / 表单提示 |
| `error_code` | **业务数字码（字符串）** | 稳定联调键、i18n 映射、埋点 |
| `metadata` | 可选扩展（字段校验详情等） | 表单高亮、详情页 |

成功响应仍为：

```json
{ "code": 200, "message": "success", "data": { ... } }
```

### 4.2 业务码分段（`internal/pkg/constants`）

| 区间 | 域 | 示例 |
|------|-----|------|
| `10001`–`10015` | 通用 / 鉴权 | `10010` 会话过期、`10003` 无权限 |
| `10901`–`10902` | 可变文案 500/429 | `ErrInternalWithMsg` |
| `11001`–`11003` | 请求校验 | 正则、参数非法 |
| `11020`–`11024` | 可变文案 4xx | `ErrBadRequestWithMsg` 等 |
| `20001`+ | 按功能域 | `20001` 用户、`22001` 告警、`23001` 项目… |

完整定义见 `internal/pkg/constants/constant.go`。

### 4.3 后端如何返回错误

**Handler（推荐）**

```go
import "yunshu/internal/pkg/response"

// 提交给 ErrorHandler，自动 JSON + 打日志
response.Abort(c, constants.ErrLoginSessionExpired)
```

**Service**

```go
import bizerrors "yunshu/internal/pkg/errors"

// 包装下层错误，不打日志
return nil, bizerrors.Pass(ctx, "user", "GetByID", err)

// 真内部错误：打一次日志并返回 500
return nil, bizerrors.InternalCtx(ctx, err, "user:Create")
```

**预定义产品错误**

```go
return constants.ErrNotFound  // error_code=10004
return constants.ErrBadRequestWithMsg("集群名称不能为空") // error_code=11020
```

### 4.4 日志与响应的对应关系

一次失败的 API 调用，典型只有 **两条** 相关日志：

1. **error.log**（或 info.log）：`LogAPI` — `error_code`、`path`、`method`、`reason`
2. **info.log**：访问日志 — `status=4xx/5xx`、`latency_ms`

Service 里 `Pass` 返回的 4xx **不会**在 Service 层再打一遍。

日志示例（text）：

```text
level=ERROR msg="API request rejected" error_code=10010 reason=Unauthorized http_status=401 method=GET path=/api/v1/users request_id=...
level=WARN  msg="HTTP request completed" status=401 latency_ms=12 component=http.access request_id=...
```

---

## 5. 前端联调清单

1. **优先用 `error_code` 做稳定映射**（比 `message` 可靠；`message` 可能随产品文案调整）。
2. **401 系**（`10002`–`10015`）：跳转登录或刷新 token；`10010` 会话过期可统一处理。
3. **403**（`10003`、`10012` 等）：提示无权限，勿重试。
4. **11020–11024**：展示 `message`（后端已拼好用户可读文案）。
5. **10006 / 10901 / 50001**：通用「服务异常」，可带 `X-Request-ID` 给运维查 `error.log`。
6. 请求头建议带 **`X-Request-ID`**（可选）；未带时服务端生成并在响应头回传。

**TypeScript 示例**

```ts
interface ApiErrorBody {
  code: number;
  reason: string;
  message: string;
  error_code?: string;
  metadata?: Record<string, unknown>;
}

function handleApiError(body: ApiErrorBody) {
  switch (body.error_code) {
    case "10010":
    case "10002":
      redirectToLogin();
      break;
    case "10003":
      toast.error(body.message || "无权限");
      break;
    default:
      toast.error(body.message || "操作失败");
  }
}
```

---

## 6. 代码位置索引

| 用途 | 路径 |
|------|------|
| 三文件 Logger | `internal/pkg/logger/` |
| context → request_id/user | `internal/pkg/logger/context.go` |
| `logger.With(ctx, …)` | `internal/pkg/logger/with.go` |
| 业务错误类型 | `internal/pkg/errors/biz.go` |
| Pass / Internal | `internal/pkg/errors/constructors.go` |
| HTTP 边界打日志 | `internal/pkg/errors/api_log.go` |
| ErrorHandler | `internal/middleware/error_handler.go` |
| 访问日志 | `internal/middleware/request_logger.go` |
| 错误码常量 | `internal/pkg/constants/constant.go` |

**已删除（勿引用）**：`internal/pkg/logutil`、`logger.Biz()`、`svclog`。

---

## 7. 新增错误码流程

1. 在 `internal/pkg/constants/constant.go` 对应域段增加 `ErrXxx = BizError(httpStatus, bizCode, "用户可见文案")`。
2. Handler/Service 返回该常量或 `ErrXxxWithMsg`。
3. 前端在错误码表增加 `error_code` → 展示/行为映射。
4. 若需 OpenAPI 说明，重新生成 `docs/swagger/`（`tools/genopenapi`）。

---

## 8. 相关文档

- [CODEBASE-MAP.md](CODEBASE-MAP.md) — 代码地图与包职责
- [backend-architecture-complete.md](backend-architecture-complete.md) — 后端架构总览
