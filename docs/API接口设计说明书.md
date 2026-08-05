# Yunshu API 接口设计说明书

| 项 | 内容 |
|----|------|
| 文档编号 | YUNSHU-API-2026-002 |
| 版本 | V2.0.0 |
| Base Path | `/api/v1` |
| OpenAPI | 3.0.3（[`docs/apipost/permission-system.openapi.yaml`](apipost/permission-system.openapi.yaml)） |
| 生成方式 | `go run ./tools/genopenapi` + `python tools/gen_api_design_md.py` |
| 权威来源 | `internal/router/register_*.go` |
| 日期 | 2026-08-05 |

> 本文档接口清单由路由自动生成，与 OpenAPI 对齐；请求 Body 字段细节以实现 Handler DTO 为准。

---

## 1. API 规范

每个接口条目包含：

| 要素 | 说明 |
|------|------|
| 接口编号 | `API-<域前缀>-<序号>` |
| 接口名称 | 中文简述 |
| 请求方式 | GET/POST/PUT/PATCH/DELETE |
| URL | OpenAPI 路径（`{param}` 对应 Gin `:param`） |
| 请求参数 | Path / Query / Header / Body |
| 响应结构 | 统一 `StandardResponse` / `ErrorBody` |
| 错误码 | HTTP + 业务 `error_code`（见 `internal/pkg/constants`） |
| 权限要求 | JWT / Casbin / 项目成员 / K8sScope / 资源 ACL / Webhook |

### 1.1 鉴权

```http
Authorization: Bearer <JWT>
```

- Session：JWT Claims.`TokenID` 须在 Redis 白名单（失败关闭）。
- WebSocket：`POST /api/v1/auth/ws-ticket` → 连接带 `?ticket=`。
- Alertmanager：`X-Alert-Token` 或 Bearer webhook token。

### 1.2 统一响应（兼容 OpenAPI `StandardResponse`）

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

### 1.3 通用错误码（节选）

| error_code | HTTP | 说明 |
|------------|------|------|
| 10001 | 400 | 请求参数无效 |
| 10002 | 401 | 登录失效/凭证无效 |
| 10003 | 403 | 无权执行 |
| 10004 | 404 | 资源不存在 |
| 10005 | 409 | 状态冲突 |
| 10006 | 500 | 服务异常 |
| 10007 | 429 | 过于频繁 |
| 10008–10015 | 401 | 鉴权头/Token/Session/WS ticket |
| 11020–11024 | 4xx | WithMsg 可变文案（参数/未找到/禁止/未授权/冲突） |
| 20001–20014 | 4xx | 认证与账号域 |
| 其他 21xxx+ | — | 按功能域，见 `internal/pkg/constants/constant.go` |

### 1.4 Swagger / OpenAPI 兼容

| 产物 | 路径 |
|------|------|
| **推荐（全量路由）** | [`docs/apipost/permission-system.openapi.yaml`](apipost/permission-system.openapi.yaml) |
| Swagger UI | 配置 `swagger.enabled=true` → `/swagger/index.html` |
| swag 注解集 | `docs/swagger/`（覆盖不全，仅作补充） |

重新生成：

```bash
go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml
python tools/gen_api_design_md.py
```

---

## 2. 接口目录总览

合计 **410** 个路径、**526** 个操作，按 Tag 分组如下。

| Tag | 接口数 | 编号前缀 |
|-----|--------|----------|
| Alerts | 67 | API-ALT-* |
| AlertsWebhook | 1 | API-ALTWH-* |
| Auth | 11 | API-AUTH-* |
| Clusters | 10 | API-CLS-* |
| Configmaps | 4 | API-CONFIG-* |
| Crds | 4 | API-CRDS-* |
| Cronjobs | 9 | API-CRONJO-* |
| Crs | 5 | API-CRS-* |
| Daemonsets | 7 | API-DAEMON-* |
| Departments | 5 | API-DEPT-* |
| Deployments | 12 | API-DEP-* |
| Dict | 6 | API-DICT-* |
| Events | 2 | API-EVENTS-* |
| Helm | 11 | API-HELM-* |
| Horizontal-pod-autoscalers | 4 | API-HORIZO-* |
| Ingresses | 10 | API-INGRES-* |
| Jobs | 7 | API-JOBS-* |
| K8s | 3 | API-K8S-* |
| K8s-namespace-allow-rules | 3 | API-K8SNAM-* |
| K8s-namespace-deny-rules | 3 | API-K8SNAM-* |
| K8s-services | 4 | API-K8SSER-* |
| K8sEventForward | 7 | API-K8SEF-* |
| K8sScopedPolicy | 9 | API-K8SP-* |
| LogPlatform | 10 | API-LOG-* |
| Loggie | 1 | API-LGG-* |
| Login-logs | 4 | API-LOGINL-* |
| Menus | 7 | API-MENU-* |
| Namespaces | 4 | API-NS-* |
| Network-policies | 4 | API-NETWOR-* |
| Nodes | 4 | API-NODE-* |
| Operation-logs | 4 | API-OPERAT-* |
| Overview | 3 | API-OVW-* |
| Permissions | 6 | API-PERM-* |
| Persistentvolumeclaims | 4 | API-PERSIS-* |
| Persistentvolumes | 4 | API-PERSIS-* |
| Plugins | 1 | API-PLG-* |
| Pods | 19 | API-POD-* |
| Policies | 9 | API-POL-* |
| Projects | 185 | API-PRJ-* |
| Rbac | 7 | API-RBAC-* |
| Registrations | 2 | API-REGIST-* |
| Roles | 5 | API-ROLE-* |
| Secrets | 4 | API-SECRET-* |
| Security | 2 | API-SECURI-* |
| Serviceaccounts | 4 | API-SERVIC-* |
| Statefulsets | 9 | API-STATEF-* |
| Storageclasses | 4 | API-STORAG-* |
| System | 1 | API-SYS-* |
| User-groups | 6 | API-USERGR-* |
| Users | 9 | API-USR-* |

---

## 3. 接口明细

### 3.1 Alerts

#### API-ALT-001 查询 v1/alerts/channels

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-001` |
| 接口名称 | 查询 v1/alerts/channels |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/channels` |
| Gin 路径 | `/api/v1/alerts/channels` |
| operationId | `get_api_v1_alerts_channels` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-002 创建/提交 v1/alerts/channels

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-002` |
| 接口名称 | 创建/提交 v1/alerts/channels |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/channels` |
| Gin 路径 | `/api/v1/alerts/channels` |
| operationId | `post_api_v1_alerts_channels` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-003 创建/提交 alerts/channels/preview-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-003` |
| 接口名称 | 创建/提交 alerts/channels/preview-template |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/channels/preview-template` |
| Gin 路径 | `/api/v1/alerts/channels/preview-template` |
| operationId | `post_api_v1_alerts_channels_preview-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels/preview-template, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-004 删除 v1/alerts/channels

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-004` |
| 接口名称 | 删除 v1/alerts/channels |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/channels/{id}` |
| Gin 路径 | `/api/v1/alerts/channels/:id` |
| operationId | `delete_api_v1_alerts_channels_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-005 更新 v1/alerts/channels

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-005` |
| 接口名称 | 更新 v1/alerts/channels |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/channels/{id}` |
| Gin 路径 | `/api/v1/alerts/channels/:id` |
| operationId | `put_api_v1_alerts_channels_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-006 创建/提交 alerts/channels/test

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-006` |
| 接口名称 | 创建/提交 alerts/channels/test |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/channels/{id}/test` |
| Gin 路径 | `/api/v1/alerts/channels/:id/test` |
| operationId | `post_api_v1_alerts_channels_id__test` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/channels/:id/test, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-007 查询 v1/alerts/cloud-expiry-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-007` |
| 接口名称 | 查询 v1/alerts/cloud-expiry-rules |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/cloud-expiry-rules` |
| Gin 路径 | `/api/v1/alerts/cloud-expiry-rules` |
| operationId | `get_api_v1_alerts_cloud-expiry-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/cloud-expiry-rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-008 创建/提交 v1/alerts/cloud-expiry-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-008` |
| 接口名称 | 创建/提交 v1/alerts/cloud-expiry-rules |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/cloud-expiry-rules` |
| Gin 路径 | `/api/v1/alerts/cloud-expiry-rules` |
| operationId | `post_api_v1_alerts_cloud-expiry-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/cloud-expiry-rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-009 创建/提交 alerts/cloud-expiry-rules/evaluate-now

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-009` |
| 接口名称 | 创建/提交 alerts/cloud-expiry-rules/evaluate-now |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/cloud-expiry-rules/evaluate-now` |
| Gin 路径 | `/api/v1/alerts/cloud-expiry-rules/evaluate-now` |
| operationId | `post_api_v1_alerts_cloud-expiry-rules_evaluate-now` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/cloud-expiry-rules/evaluate-now, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-010 删除 v1/alerts/cloud-expiry-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-010` |
| 接口名称 | 删除 v1/alerts/cloud-expiry-rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/cloud-expiry-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/cloud-expiry-rules/:id` |
| operationId | `delete_api_v1_alerts_cloud-expiry-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/cloud-expiry-rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-011 更新 v1/alerts/cloud-expiry-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-011` |
| 接口名称 | 更新 v1/alerts/cloud-expiry-rules |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/cloud-expiry-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/cloud-expiry-rules/:id` |
| operationId | `put_api_v1_alerts_cloud-expiry-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/cloud-expiry-rules/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-012 查询 v1/alerts/datasources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-012` |
| 接口名称 | 查询 v1/alerts/datasources |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/datasources` |
| Gin 路径 | `/api/v1/alerts/datasources` |
| operationId | `get_api_v1_alerts_datasources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-013 创建/提交 v1/alerts/datasources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-013` |
| 接口名称 | 创建/提交 v1/alerts/datasources |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/datasources` |
| Gin 路径 | `/api/v1/alerts/datasources` |
| operationId | `post_api_v1_alerts_datasources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-014 删除 v1/alerts/datasources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-014` |
| 接口名称 | 删除 v1/alerts/datasources |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/datasources/{id}` |
| Gin 路径 | `/api/v1/alerts/datasources/:id` |
| operationId | `delete_api_v1_alerts_datasources_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-015 更新 v1/alerts/datasources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-015` |
| 接口名称 | 更新 v1/alerts/datasources |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/datasources/{id}` |
| Gin 路径 | `/api/v1/alerts/datasources/:id` |
| operationId | `put_api_v1_alerts_datasources_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-016 查询 alerts/datasources/alertmanager-silences

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-016` |
| 接口名称 | 查询 alerts/datasources/alertmanager-silences |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/datasources/{id}/alertmanager-silences` |
| Gin 路径 | `/api/v1/alerts/datasources/:id/alertmanager-silences` |
| operationId | `get_api_v1_alerts_datasources_id__alertmanager-silences` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id/alertmanager-silences, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-017 查询 alerts/datasources/ping

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-017` |
| 接口名称 | 查询 alerts/datasources/ping |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/datasources/{id}/ping` |
| Gin 路径 | `/api/v1/alerts/datasources/:id/ping` |
| operationId | `get_api_v1_alerts_datasources_id__ping` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id/ping, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-018 查询 alerts/datasources/prometheus-alerts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-018` |
| 接口名称 | 查询 alerts/datasources/prometheus-alerts |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/datasources/{id}/prometheus-alerts` |
| Gin 路径 | `/api/v1/alerts/datasources/:id/prometheus-alerts` |
| operationId | `get_api_v1_alerts_datasources_id__prometheus-alerts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id/prometheus-alerts, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-019 创建/提交 alerts/datasources/query

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-019` |
| 接口名称 | 创建/提交 alerts/datasources/query |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/datasources/{id}/query` |
| Gin 路径 | `/api/v1/alerts/datasources/:id/query` |
| operationId | `post_api_v1_alerts_datasources_id__query` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id/query, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-020 创建/提交 alerts/datasources/query_range

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-020` |
| 接口名称 | 创建/提交 alerts/datasources/query_range |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/datasources/{id}/query_range` |
| Gin 路径 | `/api/v1/alerts/datasources/:id/query_range` |
| operationId | `post_api_v1_alerts_datasources_id__query_range` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/datasources/:id/query_range, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-021 查询 v1/alerts/duty-blocks

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-021` |
| 接口名称 | 查询 v1/alerts/duty-blocks |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/duty-blocks` |
| Gin 路径 | `/api/v1/alerts/duty-blocks` |
| operationId | `get_api_v1_alerts_duty-blocks` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-022 创建/提交 v1/alerts/duty-blocks

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-022` |
| 接口名称 | 创建/提交 v1/alerts/duty-blocks |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/duty-blocks` |
| Gin 路径 | `/api/v1/alerts/duty-blocks` |
| operationId | `post_api_v1_alerts_duty-blocks` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-023 查询 alerts/duty-blocks/calendar

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-023` |
| 接口名称 | 查询 alerts/duty-blocks/calendar |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/duty-blocks/calendar` |
| Gin 路径 | `/api/v1/alerts/duty-blocks/calendar` |
| operationId | `get_api_v1_alerts_duty-blocks_calendar` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks/calendar, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-024 创建/提交 alerts/duty-blocks/validate

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-024` |
| 接口名称 | 创建/提交 alerts/duty-blocks/validate |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/duty-blocks/validate` |
| Gin 路径 | `/api/v1/alerts/duty-blocks/validate` |
| operationId | `post_api_v1_alerts_duty-blocks_validate` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks/validate, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-025 删除 v1/alerts/duty-blocks

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-025` |
| 接口名称 | 删除 v1/alerts/duty-blocks |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/duty-blocks/{id}` |
| Gin 路径 | `/api/v1/alerts/duty-blocks/:id` |
| operationId | `delete_api_v1_alerts_duty-blocks_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-026 更新 v1/alerts/duty-blocks

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-026` |
| 接口名称 | 更新 v1/alerts/duty-blocks |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/duty-blocks/{id}` |
| Gin 路径 | `/api/v1/alerts/duty-blocks/:id` |
| operationId | `put_api_v1_alerts_duty-blocks_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-027 创建/提交 alerts/duty-blocks/handoff

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-027` |
| 接口名称 | 创建/提交 alerts/duty-blocks/handoff |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/duty-blocks/{id}/handoff` |
| Gin 路径 | `/api/v1/alerts/duty-blocks/:id/handoff` |
| operationId | `post_api_v1_alerts_duty-blocks_id__handoff` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/duty-blocks/:id/handoff, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-028 查询 v1/alerts/events

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-028` |
| 接口名称 | 查询 v1/alerts/events |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/events` |
| Gin 路径 | `/api/v1/alerts/events` |
| operationId | `get_api_v1_alerts_events` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/events, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-029 查询 alerts/events/by-fingerprint

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-029` |
| 接口名称 | 查询 alerts/events/by-fingerprint |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/events/by-fingerprint` |
| Gin 路径 | `/api/v1/alerts/events/by-fingerprint` |
| operationId | `get_api_v1_alerts_events_by-fingerprint` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/events/by-fingerprint, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-030 查询 alerts/events/grouped

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-030` |
| 接口名称 | 查询 alerts/events/grouped |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/events/grouped` |
| Gin 路径 | `/api/v1/alerts/events/grouped` |
| operationId | `get_api_v1_alerts_events_grouped` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/events/grouped, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-031 查询 alerts/history/stats

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-031` |
| 接口名称 | 查询 alerts/history/stats |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/history/stats` |
| Gin 路径 | `/api/v1/alerts/history/stats` |
| operationId | `get_api_v1_alerts_history_stats` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/history/stats, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-032 查询 v1/alerts/inhibition-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-032` |
| 接口名称 | 查询 v1/alerts/inhibition-rules |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/inhibition-rules` |
| Gin 路径 | `/api/v1/alerts/inhibition-rules` |
| operationId | `get_api_v1_alerts_inhibition-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/inhibition-rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-033 创建/提交 v1/alerts/inhibition-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-033` |
| 接口名称 | 创建/提交 v1/alerts/inhibition-rules |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/inhibition-rules` |
| Gin 路径 | `/api/v1/alerts/inhibition-rules` |
| operationId | `post_api_v1_alerts_inhibition-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/inhibition-rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-034 创建/提交 alerts/inhibition-rules/refresh-cache

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-034` |
| 接口名称 | 创建/提交 alerts/inhibition-rules/refresh-cache |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/inhibition-rules/refresh-cache` |
| Gin 路径 | `/api/v1/alerts/inhibition-rules/refresh-cache` |
| operationId | `post_api_v1_alerts_inhibition-rules_refresh-cache` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/inhibition-rules/refresh-cache, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-035 删除 v1/alerts/inhibition-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-035` |
| 接口名称 | 删除 v1/alerts/inhibition-rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/inhibition-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/inhibition-rules/:id` |
| operationId | `delete_api_v1_alerts_inhibition-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/inhibition-rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-036 更新 v1/alerts/inhibition-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-036` |
| 接口名称 | 更新 v1/alerts/inhibition-rules |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/inhibition-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/inhibition-rules/:id` |
| operationId | `put_api_v1_alerts_inhibition-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/inhibition-rules/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-037 查询 v1/alerts/maintenance-windows

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-037` |
| 接口名称 | 查询 v1/alerts/maintenance-windows |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/maintenance-windows` |
| Gin 路径 | `/api/v1/alerts/maintenance-windows` |
| operationId | `get_api_v1_alerts_maintenance-windows` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/maintenance-windows, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-038 创建/提交 v1/alerts/maintenance-windows

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-038` |
| 接口名称 | 创建/提交 v1/alerts/maintenance-windows |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/maintenance-windows` |
| Gin 路径 | `/api/v1/alerts/maintenance-windows` |
| operationId | `post_api_v1_alerts_maintenance-windows` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/maintenance-windows, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-039 删除 v1/alerts/maintenance-windows

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-039` |
| 接口名称 | 删除 v1/alerts/maintenance-windows |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/maintenance-windows/{id}` |
| Gin 路径 | `/api/v1/alerts/maintenance-windows/:id` |
| operationId | `delete_api_v1_alerts_maintenance-windows_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/maintenance-windows/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-040 更新 v1/alerts/maintenance-windows

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-040` |
| 接口名称 | 更新 v1/alerts/maintenance-windows |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/maintenance-windows/{id}` |
| Gin 路径 | `/api/v1/alerts/maintenance-windows/:id` |
| operationId | `put_api_v1_alerts_maintenance-windows_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/maintenance-windows/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-041 查询 v1/alerts/monitor-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-041` |
| 接口名称 | 查询 v1/alerts/monitor-rules |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/monitor-rules` |
| Gin 路径 | `/api/v1/alerts/monitor-rules` |
| operationId | `get_api_v1_alerts_monitor-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-042 创建/提交 v1/alerts/monitor-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-042` |
| 接口名称 | 创建/提交 v1/alerts/monitor-rules |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/monitor-rules` |
| Gin 路径 | `/api/v1/alerts/monitor-rules` |
| operationId | `post_api_v1_alerts_monitor-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-043 创建/提交 alerts/monitor-rules/from-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-043` |
| 接口名称 | 创建/提交 alerts/monitor-rules/from-template |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/monitor-rules/from-template` |
| Gin 路径 | `/api/v1/alerts/monitor-rules/from-template` |
| operationId | `post_api_v1_alerts_monitor-rules_from-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules/from-template, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-044 删除 v1/alerts/monitor-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-044` |
| 接口名称 | 删除 v1/alerts/monitor-rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/monitor-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/monitor-rules/:id` |
| operationId | `delete_api_v1_alerts_monitor-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-045 更新 v1/alerts/monitor-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-045` |
| 接口名称 | 更新 v1/alerts/monitor-rules |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/monitor-rules/{id}` |
| Gin 路径 | `/api/v1/alerts/monitor-rules/:id` |
| operationId | `put_api_v1_alerts_monitor-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-046 查询 alerts/monitor-rules/assignees

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-046` |
| 接口名称 | 查询 alerts/monitor-rules/assignees |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/monitor-rules/{id}/assignees` |
| Gin 路径 | `/api/v1/alerts/monitor-rules/:id/assignees` |
| operationId | `get_api_v1_alerts_monitor-rules_id__assignees` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules/:id/assignees, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-047 更新 alerts/monitor-rules/assignees

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-047` |
| 接口名称 | 更新 alerts/monitor-rules/assignees |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/monitor-rules/{id}/assignees` |
| Gin 路径 | `/api/v1/alerts/monitor-rules/:id/assignees` |
| operationId | `put_api_v1_alerts_monitor-rules_id__assignees` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/monitor-rules/:id/assignees, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-048 查询 v1/alerts/quality-report

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-048` |
| 接口名称 | 查询 v1/alerts/quality-report |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/quality-report` |
| Gin 路径 | `/api/v1/alerts/quality-report` |
| operationId | `get_api_v1_alerts_quality-report` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/quality-report, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-049 查询 v1/alerts/receiver-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-049` |
| 接口名称 | 查询 v1/alerts/receiver-groups |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/receiver-groups` |
| Gin 路径 | `/api/v1/alerts/receiver-groups` |
| operationId | `get_api_v1_alerts_receiver-groups` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/receiver-groups, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-050 创建/提交 v1/alerts/receiver-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-050` |
| 接口名称 | 创建/提交 v1/alerts/receiver-groups |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/receiver-groups` |
| Gin 路径 | `/api/v1/alerts/receiver-groups` |
| operationId | `post_api_v1_alerts_receiver-groups` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/receiver-groups, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-051 删除 v1/alerts/receiver-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-051` |
| 接口名称 | 删除 v1/alerts/receiver-groups |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/receiver-groups/{id}` |
| Gin 路径 | `/api/v1/alerts/receiver-groups/:id` |
| operationId | `delete_api_v1_alerts_receiver-groups_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/receiver-groups/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-052 更新 v1/alerts/receiver-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-052` |
| 接口名称 | 更新 v1/alerts/receiver-groups |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/receiver-groups/{id}` |
| Gin 路径 | `/api/v1/alerts/receiver-groups/:id` |
| operationId | `put_api_v1_alerts_receiver-groups_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/receiver-groups/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-053 创建/提交 alerts/routing/debug

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-053` |
| 接口名称 | 创建/提交 alerts/routing/debug |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/routing/debug` |
| Gin 路径 | `/api/v1/alerts/routing/debug` |
| operationId | `post_api_v1_alerts_routing_debug` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/routing/debug, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-054 查询 v1/alerts/rule-templates

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-054` |
| 接口名称 | 查询 v1/alerts/rule-templates |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/rule-templates` |
| Gin 路径 | `/api/v1/alerts/rule-templates` |
| operationId | `get_api_v1_alerts_rule-templates` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/rule-templates, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-055 查询 v1/alerts/silences

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-055` |
| 接口名称 | 查询 v1/alerts/silences |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/silences` |
| Gin 路径 | `/api/v1/alerts/silences` |
| operationId | `get_api_v1_alerts_silences` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/silences, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-056 创建/提交 v1/alerts/silences

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-056` |
| 接口名称 | 创建/提交 v1/alerts/silences |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/silences` |
| Gin 路径 | `/api/v1/alerts/silences` |
| operationId | `post_api_v1_alerts_silences` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/silences, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-057 创建/提交 alerts/silences/batch

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-057` |
| 接口名称 | 创建/提交 alerts/silences/batch |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/silences/batch` |
| Gin 路径 | `/api/v1/alerts/silences/batch` |
| operationId | `post_api_v1_alerts_silences_batch` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/silences/batch, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-058 删除 v1/alerts/silences

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-058` |
| 接口名称 | 删除 v1/alerts/silences |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/silences/{id}` |
| Gin 路径 | `/api/v1/alerts/silences/:id` |
| operationId | `delete_api_v1_alerts_silences_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/silences/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-059 更新 v1/alerts/silences

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-059` |
| 接口名称 | 更新 v1/alerts/silences |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/silences/{id}` |
| Gin 路径 | `/api/v1/alerts/silences/:id` |
| operationId | `put_api_v1_alerts_silences_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/silences/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-060 查询 v1/alerts/subscriptions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-060` |
| 接口名称 | 查询 v1/alerts/subscriptions |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/subscriptions` |
| Gin 路径 | `/api/v1/alerts/subscriptions` |
| operationId | `get_api_v1_alerts_subscriptions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-061 创建/提交 v1/alerts/subscriptions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-061` |
| 接口名称 | 创建/提交 v1/alerts/subscriptions |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/subscriptions` |
| Gin 路径 | `/api/v1/alerts/subscriptions` |
| operationId | `post_api_v1_alerts_subscriptions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-062 创建/提交 alerts/subscriptions/clone-from-project

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-062` |
| 接口名称 | 创建/提交 alerts/subscriptions/clone-from-project |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/subscriptions/clone-from-project` |
| Gin 路径 | `/api/v1/alerts/subscriptions/clone-from-project` |
| operationId | `post_api_v1_alerts_subscriptions_clone-from-project` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/clone-from-project, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-063 创建/提交 alerts/subscriptions/migrate-from-policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-063` |
| 接口名称 | 创建/提交 alerts/subscriptions/migrate-from-policies |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/subscriptions/migrate-from-policies` |
| Gin 路径 | `/api/v1/alerts/subscriptions/migrate-from-policies` |
| operationId | `post_api_v1_alerts_subscriptions_migrate-from-policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/migrate-from-policies, POST)`；写操作：`OperationAudit` 审计

---

#### API-ALT-064 查询 alerts/subscriptions/tree

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-064` |
| 接口名称 | 查询 alerts/subscriptions/tree |
| 请求方式 | `GET` |
| URL | `/api/v1/alerts/subscriptions/tree` |
| Gin 路径 | `/api/v1/alerts/subscriptions/tree` |
| operationId | `get_api_v1_alerts_subscriptions_tree` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/tree, GET)`；写操作：`OperationAudit` 审计

---

#### API-ALT-065 删除 v1/alerts/subscriptions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-065` |
| 接口名称 | 删除 v1/alerts/subscriptions |
| 请求方式 | `DELETE` |
| URL | `/api/v1/alerts/subscriptions/{id}` |
| Gin 路径 | `/api/v1/alerts/subscriptions/:id` |
| operationId | `delete_api_v1_alerts_subscriptions_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ALT-066 更新 v1/alerts/subscriptions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-066` |
| 接口名称 | 更新 v1/alerts/subscriptions |
| 请求方式 | `PUT` |
| URL | `/api/v1/alerts/subscriptions/{id}` |
| Gin 路径 | `/api/v1/alerts/subscriptions/:id` |
| operationId | `put_api_v1_alerts_subscriptions_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-ALT-067 创建/提交 alerts/subscriptions/move

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALT-067` |
| 接口名称 | 创建/提交 alerts/subscriptions/move |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/subscriptions/{id}/move` |
| Gin 路径 | `/api/v1/alerts/subscriptions/:id/move` |
| operationId | `post_api_v1_alerts_subscriptions_id__move` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/alerts/subscriptions/:id/move, POST)`；写操作：`OperationAudit` 审计

---

### 3.2 AlertsWebhook

#### API-ALTWH-001 创建/提交 alerts/webhook/alertmanager

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ALTWH-001` |
| 接口名称 | 创建/提交 alerts/webhook/alertmanager |
| 请求方式 | `POST` |
| URL | `/api/v1/alerts/webhook/alertmanager` |
| Gin 路径 | `/api/v1/alerts/webhook/alertmanager` |
| operationId | `post_api_v1_alerts_webhook_alertmanager` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |
| header | `X-Alert-Token` | string | 是* | 与 `Authorization: Bearer` 二选一 |
| body | Alertmanager webhook payload | object | 是 | 标准 AM JSON |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

Webhook Token（请求头 `X-Alert-Token` 或 `Authorization: Bearer <token>`）；无需 JWT

---

### 3.3 Auth

#### API-AUTH-001 创建/提交 v1/auth/email-login

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-001` |
| 接口名称 | 创建/提交 v1/auth/email-login |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/email-login` |
| Gin 路径 | `/api/v1/auth/email-login` |
| operationId | `post_api_v1_auth_email-login` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |
| 401 | 20002 | 账号或密码错误 |
| 429 | 20014 | 登录失败次数过多 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-002 创建/提交 v1/auth/login

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-002` |
| 接口名称 | 创建/提交 v1/auth/login |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/login` |
| Gin 路径 | `/api/v1/auth/login` |
| operationId | `post_api_v1_auth_login` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |
| 401 | 20002 | 账号或密码错误 |
| 429 | 20014 | 登录失败次数过多 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-003 创建/提交 v1/auth/login-code

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-003` |
| 接口名称 | 创建/提交 v1/auth/login-code |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/login-code` |
| Gin 路径 | `/api/v1/auth/login-code` |
| operationId | `post_api_v1_auth_login-code` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |
| 401 | 20002 | 账号或密码错误 |
| 429 | 20014 | 登录失败次数过多 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-004 创建/提交 v1/auth/logout

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-004` |
| 接口名称 | 创建/提交 v1/auth/logout |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/logout` |
| Gin 路径 | `/api/v1/auth/logout` |
| operationId | `post_api_v1_auth_logout` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/auth/logout, POST)`；写操作：`OperationAudit` 审计

---

#### API-AUTH-005 查询 v1/auth/me

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-005` |
| 接口名称 | 查询 v1/auth/me |
| 请求方式 | `GET` |
| URL | `/api/v1/auth/me` |
| Gin 路径 | `/api/v1/auth/me` |
| operationId | `get_api_v1_auth_me` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/auth/me, GET)`；写操作：`OperationAudit` 审计

---

#### API-AUTH-006 更新 v1/auth/me

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-006` |
| 接口名称 | 更新 v1/auth/me |
| 请求方式 | `PUT` |
| URL | `/api/v1/auth/me` |
| Gin 路径 | `/api/v1/auth/me` |
| operationId | `put_api_v1_auth_me` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/auth/me, PUT)`；写操作：`OperationAudit` 审计

---

#### API-AUTH-007 更新 v1/auth/password

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-007` |
| 接口名称 | 更新 v1/auth/password |
| 请求方式 | `PUT` |
| URL | `/api/v1/auth/password` |
| Gin 路径 | `/api/v1/auth/password` |
| operationId | `put_api_v1_auth_password` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/auth/password, PUT)`；写操作：`OperationAudit` 审计

---

#### API-AUTH-008 创建/提交 v1/auth/password-login-code

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-008` |
| 接口名称 | 创建/提交 v1/auth/password-login-code |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/password-login-code` |
| Gin 路径 | `/api/v1/auth/password-login-code` |
| operationId | `post_api_v1_auth_password-login-code` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |
| 401 | 20002 | 账号或密码错误 |
| 429 | 20014 | 登录失败次数过多 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-009 创建/提交 v1/auth/register

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-009` |
| 接口名称 | 创建/提交 v1/auth/register |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/register` |
| Gin 路径 | `/api/v1/auth/register` |
| operationId | `post_api_v1_auth_register` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-010 创建/提交 v1/auth/verification-code

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-010` |
| 接口名称 | 创建/提交 v1/auth/verification-code |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/verification-code` |
| Gin 路径 | `/api/v1/auth/verification-code` |
| operationId | `post_api_v1_auth_verification-code` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

公开接口（无需登录）

---

#### API-AUTH-011 创建/提交 v1/auth/ws-ticket

| 项 | 内容 |
|----|------|
| 接口编号 | `API-AUTH-011` |
| 接口名称 | 创建/提交 v1/auth/ws-ticket |
| 请求方式 | `POST` |
| URL | `/api/v1/auth/ws-ticket` |
| Gin 路径 | `/api/v1/auth/ws-ticket` |
| operationId | `post_api_v1_auth_ws-ticket` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/auth/ws-ticket, POST)`；写操作：`OperationAudit` 审计

---

### 3.4 Clusters

#### API-CLS-001 查询 api/v1/clusters

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-001` |
| 接口名称 | 查询 api/v1/clusters |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters` |
| Gin 路径 | `/api/v1/clusters` |
| operationId | `get_api_v1_clusters` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-002 创建/提交 api/v1/clusters

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-002` |
| 接口名称 | 创建/提交 api/v1/clusters |
| 请求方式 | `POST` |
| URL | `/api/v1/clusters` |
| Gin 路径 | `/api/v1/clusters` |
| operationId | `post_api_v1_clusters` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters, POST)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-003 删除 api/v1/clusters

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-003` |
| 接口名称 | 删除 api/v1/clusters |
| 请求方式 | `DELETE` |
| URL | `/api/v1/clusters/{id}` |
| Gin 路径 | `/api/v1/clusters/:id` |
| operationId | `delete_api_v1_clusters_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id, DELETE)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-004 查询 api/v1/clusters

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-004` |
| 接口名称 | 查询 api/v1/clusters |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters/{id}` |
| Gin 路径 | `/api/v1/clusters/:id` |
| operationId | `get_api_v1_clusters_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-005 更新 api/v1/clusters

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-005` |
| 接口名称 | 更新 api/v1/clusters |
| 请求方式 | `PUT` |
| URL | `/api/v1/clusters/{id}` |
| Gin 路径 | `/api/v1/clusters/:id` |
| operationId | `put_api_v1_clusters_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id, PUT)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-006 查询 v1/clusters/api-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-006` |
| 接口名称 | 查询 v1/clusters/api-resources |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters/{id}/api-resources` |
| Gin 路径 | `/api/v1/clusters/:id/api-resources` |
| operationId | `get_api_v1_clusters_id__api-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id/api-resources, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-007 查询 v1/clusters/component-statuses

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-007` |
| 接口名称 | 查询 v1/clusters/component-statuses |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters/{id}/component-statuses` |
| Gin 路径 | `/api/v1/clusters/:id/component-statuses` |
| operationId | `get_api_v1_clusters_id__component-statuses` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id/component-statuses, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-008 查询 v1/clusters/namespaces

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-008` |
| 接口名称 | 查询 v1/clusters/namespaces |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters/{id}/namespaces` |
| Gin 路径 | `/api/v1/clusters/:id/namespaces` |
| operationId | `get_api_v1_clusters_id__namespaces` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id/namespaces, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-009 查询 v1/clusters/status

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-009` |
| 接口名称 | 查询 v1/clusters/status |
| 请求方式 | `GET` |
| URL | `/api/v1/clusters/{id}/status` |
| Gin 路径 | `/api/v1/clusters/:id/status` |
| operationId | `get_api_v1_clusters_id__status` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id/status, GET)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

#### API-CLS-010 更新 v1/clusters/status

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CLS-010` |
| 接口名称 | 更新 v1/clusters/status |
| 请求方式 | `PUT` |
| URL | `/api/v1/clusters/{id}/status` |
| Gin 路径 | `/api/v1/clusters/:id/status` |
| operationId | `put_api_v1_clusters_id__status` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/clusters/:id/status, PUT)`；K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位；写操作：`OperationAudit` 审计

---

### 3.5 Configmaps

#### API-CONFIG-001 删除 api/v1/configmaps

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CONFIG-001` |
| 接口名称 | 删除 api/v1/configmaps |
| 请求方式 | `DELETE` |
| URL | `/api/v1/configmaps` |
| Gin 路径 | `/api/v1/configmaps` |
| operationId | `delete_api_v1_configmaps` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/configmaps, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CONFIG-002 查询 api/v1/configmaps

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CONFIG-002` |
| 接口名称 | 查询 api/v1/configmaps |
| 请求方式 | `GET` |
| URL | `/api/v1/configmaps` |
| Gin 路径 | `/api/v1/configmaps` |
| operationId | `get_api_v1_configmaps` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/configmaps, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CONFIG-003 创建/提交 v1/configmaps/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CONFIG-003` |
| 接口名称 | 创建/提交 v1/configmaps/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/configmaps/apply` |
| Gin 路径 | `/api/v1/configmaps/apply` |
| operationId | `post_api_v1_configmaps_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/configmaps/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CONFIG-004 查询 v1/configmaps/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CONFIG-004` |
| 接口名称 | 查询 v1/configmaps/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/configmaps/detail` |
| Gin 路径 | `/api/v1/configmaps/detail` |
| operationId | `get_api_v1_configmaps_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/configmaps/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.6 Crds

#### API-CRDS-001 删除 api/v1/crds

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRDS-001` |
| 接口名称 | 删除 api/v1/crds |
| 请求方式 | `DELETE` |
| URL | `/api/v1/crds` |
| Gin 路径 | `/api/v1/crds` |
| operationId | `delete_api_v1_crds` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crds, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRDS-002 查询 api/v1/crds

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRDS-002` |
| 接口名称 | 查询 api/v1/crds |
| 请求方式 | `GET` |
| URL | `/api/v1/crds` |
| Gin 路径 | `/api/v1/crds` |
| operationId | `get_api_v1_crds` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crds, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRDS-003 创建/提交 v1/crds/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRDS-003` |
| 接口名称 | 创建/提交 v1/crds/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/crds/apply` |
| Gin 路径 | `/api/v1/crds/apply` |
| operationId | `post_api_v1_crds_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crds/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRDS-004 查询 v1/crds/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRDS-004` |
| 接口名称 | 查询 v1/crds/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/crds/detail` |
| Gin 路径 | `/api/v1/crds/detail` |
| operationId | `get_api_v1_crds_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crds/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.7 Cronjobs

#### API-CRONJO-001 删除 api/v1/cronjobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-001` |
| 接口名称 | 删除 api/v1/cronjobs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/cronjobs` |
| Gin 路径 | `/api/v1/cronjobs` |
| operationId | `delete_api_v1_cronjobs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-002 查询 api/v1/cronjobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-002` |
| 接口名称 | 查询 api/v1/cronjobs |
| 请求方式 | `GET` |
| URL | `/api/v1/cronjobs` |
| Gin 路径 | `/api/v1/cronjobs` |
| operationId | `get_api_v1_cronjobs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-003 创建/提交 v1/cronjobs/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-003` |
| 接口名称 | 创建/提交 v1/cronjobs/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/cronjobs/apply` |
| Gin 路径 | `/api/v1/cronjobs/apply` |
| operationId | `post_api_v1_cronjobs_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-004 创建/提交 v1/cronjobs/container-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-004` |
| 接口名称 | 创建/提交 v1/cronjobs/container-resources |
| 请求方式 | `POST` |
| URL | `/api/v1/cronjobs/container-resources` |
| Gin 路径 | `/api/v1/cronjobs/container-resources` |
| operationId | `post_api_v1_cronjobs_container-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/container-resources, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-005 查询 v1/cronjobs/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-005` |
| 接口名称 | 查询 v1/cronjobs/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/cronjobs/detail` |
| Gin 路径 | `/api/v1/cronjobs/detail` |
| operationId | `get_api_v1_cronjobs_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-006 查询 v1/cronjobs/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-006` |
| 接口名称 | 查询 v1/cronjobs/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/cronjobs/pods` |
| Gin 路径 | `/api/v1/cronjobs/pods` |
| operationId | `get_api_v1_cronjobs_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-007 创建/提交 v1/cronjobs/suspend

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-007` |
| 接口名称 | 创建/提交 v1/cronjobs/suspend |
| 请求方式 | `POST` |
| URL | `/api/v1/cronjobs/suspend` |
| Gin 路径 | `/api/v1/cronjobs/suspend` |
| operationId | `post_api_v1_cronjobs_suspend` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/suspend, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-008 创建/提交 v1/cronjobs/trigger

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-008` |
| 接口名称 | 创建/提交 v1/cronjobs/trigger |
| 请求方式 | `POST` |
| URL | `/api/v1/cronjobs/trigger` |
| Gin 路径 | `/api/v1/cronjobs/trigger` |
| operationId | `post_api_v1_cronjobs_trigger` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/trigger, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRONJO-009 查询 v1/cronjobs/v2

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRONJO-009` |
| 接口名称 | 查询 v1/cronjobs/v2 |
| 请求方式 | `GET` |
| URL | `/api/v1/cronjobs/v2` |
| Gin 路径 | `/api/v1/cronjobs/v2` |
| operationId | `get_api_v1_cronjobs_v2` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/cronjobs/v2, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.8 Crs

#### API-CRS-001 删除 api/v1/crs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRS-001` |
| 接口名称 | 删除 api/v1/crs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/crs` |
| Gin 路径 | `/api/v1/crs` |
| operationId | `delete_api_v1_crs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crs, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRS-002 查询 api/v1/crs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRS-002` |
| 接口名称 | 查询 api/v1/crs |
| 请求方式 | `GET` |
| URL | `/api/v1/crs` |
| Gin 路径 | `/api/v1/crs` |
| operationId | `get_api_v1_crs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crs, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRS-003 创建/提交 v1/crs/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRS-003` |
| 接口名称 | 创建/提交 v1/crs/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/crs/apply` |
| Gin 路径 | `/api/v1/crs/apply` |
| operationId | `post_api_v1_crs_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crs/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRS-004 查询 v1/crs/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRS-004` |
| 接口名称 | 查询 v1/crs/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/crs/detail` |
| Gin 路径 | `/api/v1/crs/detail` |
| operationId | `get_api_v1_crs_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crs/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-CRS-005 查询 v1/crs/resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-CRS-005` |
| 接口名称 | 查询 v1/crs/resources |
| 请求方式 | `GET` |
| URL | `/api/v1/crs/resources` |
| Gin 路径 | `/api/v1/crs/resources` |
| operationId | `get_api_v1_crs_resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/crs/resources, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.9 Daemonsets

#### API-DAEMON-001 删除 api/v1/daemonsets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-001` |
| 接口名称 | 删除 api/v1/daemonsets |
| 请求方式 | `DELETE` |
| URL | `/api/v1/daemonsets` |
| Gin 路径 | `/api/v1/daemonsets` |
| operationId | `delete_api_v1_daemonsets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-002 查询 api/v1/daemonsets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-002` |
| 接口名称 | 查询 api/v1/daemonsets |
| 请求方式 | `GET` |
| URL | `/api/v1/daemonsets` |
| Gin 路径 | `/api/v1/daemonsets` |
| operationId | `get_api_v1_daemonsets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-003 创建/提交 v1/daemonsets/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-003` |
| 接口名称 | 创建/提交 v1/daemonsets/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/daemonsets/apply` |
| Gin 路径 | `/api/v1/daemonsets/apply` |
| operationId | `post_api_v1_daemonsets_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-004 创建/提交 v1/daemonsets/container-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-004` |
| 接口名称 | 创建/提交 v1/daemonsets/container-resources |
| 请求方式 | `POST` |
| URL | `/api/v1/daemonsets/container-resources` |
| Gin 路径 | `/api/v1/daemonsets/container-resources` |
| operationId | `post_api_v1_daemonsets_container-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets/container-resources, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-005 查询 v1/daemonsets/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-005` |
| 接口名称 | 查询 v1/daemonsets/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/daemonsets/detail` |
| Gin 路径 | `/api/v1/daemonsets/detail` |
| operationId | `get_api_v1_daemonsets_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-006 查询 v1/daemonsets/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-006` |
| 接口名称 | 查询 v1/daemonsets/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/daemonsets/pods` |
| Gin 路径 | `/api/v1/daemonsets/pods` |
| operationId | `get_api_v1_daemonsets_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DAEMON-007 创建/提交 v1/daemonsets/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DAEMON-007` |
| 接口名称 | 创建/提交 v1/daemonsets/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/daemonsets/restart` |
| Gin 路径 | `/api/v1/daemonsets/restart` |
| operationId | `post_api_v1_daemonsets_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/daemonsets/restart, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.10 Departments

#### API-DEPT-001 创建/提交 api/v1/departments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEPT-001` |
| 接口名称 | 创建/提交 api/v1/departments |
| 请求方式 | `POST` |
| URL | `/api/v1/departments` |
| Gin 路径 | `/api/v1/departments` |
| operationId | `post_api_v1_departments` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/departments, POST)`；写操作：`OperationAudit` 审计

---

#### API-DEPT-002 查询 v1/departments/tree

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEPT-002` |
| 接口名称 | 查询 v1/departments/tree |
| 请求方式 | `GET` |
| URL | `/api/v1/departments/tree` |
| Gin 路径 | `/api/v1/departments/tree` |
| operationId | `get_api_v1_departments_tree` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/departments/tree, GET)`；写操作：`OperationAudit` 审计

---

#### API-DEPT-003 删除 api/v1/departments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEPT-003` |
| 接口名称 | 删除 api/v1/departments |
| 请求方式 | `DELETE` |
| URL | `/api/v1/departments/{id}` |
| Gin 路径 | `/api/v1/departments/:id` |
| operationId | `delete_api_v1_departments_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/departments/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-DEPT-004 查询 api/v1/departments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEPT-004` |
| 接口名称 | 查询 api/v1/departments |
| 请求方式 | `GET` |
| URL | `/api/v1/departments/{id}` |
| Gin 路径 | `/api/v1/departments/:id` |
| operationId | `get_api_v1_departments_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/departments/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-DEPT-005 更新 api/v1/departments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEPT-005` |
| 接口名称 | 更新 api/v1/departments |
| 请求方式 | `PUT` |
| URL | `/api/v1/departments/{id}` |
| Gin 路径 | `/api/v1/departments/:id` |
| operationId | `put_api_v1_departments_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/departments/:id, PUT)`；写操作：`OperationAudit` 审计

---

### 3.11 Deployments

#### API-DEP-001 删除 api/v1/deployments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-001` |
| 接口名称 | 删除 api/v1/deployments |
| 请求方式 | `DELETE` |
| URL | `/api/v1/deployments` |
| Gin 路径 | `/api/v1/deployments` |
| operationId | `delete_api_v1_deployments` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-002 查询 api/v1/deployments

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-002` |
| 接口名称 | 查询 api/v1/deployments |
| 请求方式 | `GET` |
| URL | `/api/v1/deployments` |
| Gin 路径 | `/api/v1/deployments` |
| operationId | `get_api_v1_deployments` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-003 创建/提交 v1/deployments/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-003` |
| 接口名称 | 创建/提交 v1/deployments/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/apply` |
| Gin 路径 | `/api/v1/deployments/apply` |
| operationId | `post_api_v1_deployments_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-004 创建/提交 v1/deployments/container-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-004` |
| 接口名称 | 创建/提交 v1/deployments/container-resources |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/container-resources` |
| Gin 路径 | `/api/v1/deployments/container-resources` |
| operationId | `post_api_v1_deployments_container-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/container-resources, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-005 查询 v1/deployments/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-005` |
| 接口名称 | 查询 v1/deployments/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/deployments/detail` |
| Gin 路径 | `/api/v1/deployments/detail` |
| operationId | `get_api_v1_deployments_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-006 查询 v1/deployments/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-006` |
| 接口名称 | 查询 v1/deployments/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/deployments/pods` |
| Gin 路径 | `/api/v1/deployments/pods` |
| operationId | `get_api_v1_deployments_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-007 创建/提交 v1/deployments/preview-apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-007` |
| 接口名称 | 创建/提交 v1/deployments/preview-apply |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/preview-apply` |
| Gin 路径 | `/api/v1/deployments/preview-apply` |
| operationId | `post_api_v1_deployments_preview-apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/preview-apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-008 创建/提交 v1/deployments/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-008` |
| 接口名称 | 创建/提交 v1/deployments/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/restart` |
| Gin 路径 | `/api/v1/deployments/restart` |
| operationId | `post_api_v1_deployments_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/restart, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-009 查询 v1/deployments/rollout-status

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-009` |
| 接口名称 | 查询 v1/deployments/rollout-status |
| 请求方式 | `GET` |
| URL | `/api/v1/deployments/rollout-status` |
| Gin 路径 | `/api/v1/deployments/rollout-status` |
| operationId | `get_api_v1_deployments_rollout-status` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/rollout-status, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-010 创建/提交 v1/deployments/scale

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-010` |
| 接口名称 | 创建/提交 v1/deployments/scale |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/scale` |
| Gin 路径 | `/api/v1/deployments/scale` |
| operationId | `post_api_v1_deployments_scale` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/scale, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-011 查询 v1/deployments/snapshots

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-011` |
| 接口名称 | 查询 v1/deployments/snapshots |
| 请求方式 | `GET` |
| URL | `/api/v1/deployments/snapshots` |
| Gin 路径 | `/api/v1/deployments/snapshots` |
| operationId | `get_api_v1_deployments_snapshots` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/snapshots, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-DEP-012 创建/提交 deployments/snapshots/rollback

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DEP-012` |
| 接口名称 | 创建/提交 deployments/snapshots/rollback |
| 请求方式 | `POST` |
| URL | `/api/v1/deployments/snapshots/rollback` |
| Gin 路径 | `/api/v1/deployments/snapshots/rollback` |
| operationId | `post_api_v1_deployments_snapshots_rollback` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/deployments/snapshots/rollback, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.12 Dict

#### API-DICT-001 查询 v1/dict/entries

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-001` |
| 接口名称 | 查询 v1/dict/entries |
| 请求方式 | `GET` |
| URL | `/api/v1/dict/entries` |
| Gin 路径 | `/api/v1/dict/entries` |
| operationId | `get_api_v1_dict_entries` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/entries, GET)`；写操作：`OperationAudit` 审计

---

#### API-DICT-002 创建/提交 v1/dict/entries

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-002` |
| 接口名称 | 创建/提交 v1/dict/entries |
| 请求方式 | `POST` |
| URL | `/api/v1/dict/entries` |
| Gin 路径 | `/api/v1/dict/entries` |
| operationId | `post_api_v1_dict_entries` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/entries, POST)`；写操作：`OperationAudit` 审计

---

#### API-DICT-003 删除 v1/dict/entries

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-003` |
| 接口名称 | 删除 v1/dict/entries |
| 请求方式 | `DELETE` |
| URL | `/api/v1/dict/entries/{id}` |
| Gin 路径 | `/api/v1/dict/entries/:id` |
| operationId | `delete_api_v1_dict_entries_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/entries/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-DICT-004 更新 v1/dict/entries

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-004` |
| 接口名称 | 更新 v1/dict/entries |
| 请求方式 | `PUT` |
| URL | `/api/v1/dict/entries/{id}` |
| Gin 路径 | `/api/v1/dict/entries/:id` |
| operationId | `put_api_v1_dict_entries_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/entries/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-DICT-005 创建/提交 dict/entries/reveal-value

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-005` |
| 接口名称 | 创建/提交 dict/entries/reveal-value |
| 请求方式 | `POST` |
| URL | `/api/v1/dict/entries/{id}/reveal-value` |
| Gin 路径 | `/api/v1/dict/entries/:id/reveal-value` |
| operationId | `post_api_v1_dict_entries_id__reveal-value` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/entries/:id/reveal-value, POST)`；写操作：`OperationAudit` 审计

---

#### API-DICT-006 查询 v1/dict/options

| 项 | 内容 |
|----|------|
| 接口编号 | `API-DICT-006` |
| 接口名称 | 查询 v1/dict/options |
| 请求方式 | `GET` |
| URL | `/api/v1/dict/options/{dictType}` |
| Gin 路径 | `/api/v1/dict/options/:dictType` |
| operationId | `get_api_v1_dict_options_dictType_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `dictType` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/dict/options/:dictType, GET)`；写操作：`OperationAudit` 审计

---

### 3.13 Events

#### API-EVENTS-001 查询 api/v1/events

| 项 | 内容 |
|----|------|
| 接口编号 | `API-EVENTS-001` |
| 接口名称 | 查询 api/v1/events |
| 请求方式 | `GET` |
| URL | `/api/v1/events` |
| Gin 路径 | `/api/v1/events` |
| operationId | `get_api_v1_events` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/events, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-EVENTS-002 查询 v1/events/grouped

| 项 | 内容 |
|----|------|
| 接口编号 | `API-EVENTS-002` |
| 接口名称 | 查询 v1/events/grouped |
| 请求方式 | `GET` |
| URL | `/api/v1/events/grouped` |
| Gin 路径 | `/api/v1/events/grouped` |
| operationId | `get_api_v1_events_grouped` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/events/grouped, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.14 Helm

#### API-HELM-001 查询 helm/harbor/charts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-001` |
| 接口名称 | 查询 helm/harbor/charts |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/harbor/charts` |
| Gin 路径 | `/api/v1/helm/harbor/charts` |
| operationId | `get_api_v1_helm_harbor_charts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/harbor/charts, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-002 查询 harbor/charts/versions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-002` |
| 接口名称 | 查询 harbor/charts/versions |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/harbor/charts/versions` |
| Gin 路径 | `/api/v1/helm/harbor/charts/versions` |
| operationId | `get_api_v1_helm_harbor_charts_versions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/harbor/charts/versions, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-003 查询 helm/harbor/info

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-003` |
| 接口名称 | 查询 helm/harbor/info |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/harbor/info` |
| Gin 路径 | `/api/v1/helm/harbor/info` |
| operationId | `get_api_v1_helm_harbor_info` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/harbor/info, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-004 删除 v1/helm/releases

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-004` |
| 接口名称 | 删除 v1/helm/releases |
| 请求方式 | `DELETE` |
| URL | `/api/v1/helm/releases` |
| Gin 路径 | `/api/v1/helm/releases` |
| operationId | `delete_api_v1_helm_releases` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-005 查询 v1/helm/releases

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-005` |
| 接口名称 | 查询 v1/helm/releases |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/releases` |
| Gin 路径 | `/api/v1/helm/releases` |
| operationId | `get_api_v1_helm_releases` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-006 查询 helm/releases/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-006` |
| 接口名称 | 查询 helm/releases/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/releases/detail` |
| Gin 路径 | `/api/v1/helm/releases/detail` |
| operationId | `get_api_v1_helm_releases_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-007 查询 helm/releases/history

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-007` |
| 接口名称 | 查询 helm/releases/history |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/releases/history` |
| Gin 路径 | `/api/v1/helm/releases/history` |
| operationId | `get_api_v1_helm_releases_history` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/history, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-008 创建/提交 helm/releases/install

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-008` |
| 接口名称 | 创建/提交 helm/releases/install |
| 请求方式 | `POST` |
| URL | `/api/v1/helm/releases/install` |
| Gin 路径 | `/api/v1/helm/releases/install` |
| operationId | `post_api_v1_helm_releases_install` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/install, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-009 创建/提交 helm/releases/rollback

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-009` |
| 接口名称 | 创建/提交 helm/releases/rollback |
| 请求方式 | `POST` |
| URL | `/api/v1/helm/releases/rollback` |
| Gin 路径 | `/api/v1/helm/releases/rollback` |
| operationId | `post_api_v1_helm_releases_rollback` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/rollback, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-010 创建/提交 helm/releases/upgrade

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-010` |
| 接口名称 | 创建/提交 helm/releases/upgrade |
| 请求方式 | `POST` |
| URL | `/api/v1/helm/releases/upgrade` |
| Gin 路径 | `/api/v1/helm/releases/upgrade` |
| operationId | `post_api_v1_helm_releases_upgrade` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/upgrade, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-HELM-011 查询 helm/releases/values

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HELM-011` |
| 接口名称 | 查询 helm/releases/values |
| 请求方式 | `GET` |
| URL | `/api/v1/helm/releases/values` |
| Gin 路径 | `/api/v1/helm/releases/values` |
| operationId | `get_api_v1_helm_releases_values` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/helm/releases/values, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.15 Horizontal-pod-autoscalers

#### API-HORIZO-001 删除 api/v1/horizontal-pod-autoscalers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HORIZO-001` |
| 接口名称 | 删除 api/v1/horizontal-pod-autoscalers |
| 请求方式 | `DELETE` |
| URL | `/api/v1/horizontal-pod-autoscalers` |
| Gin 路径 | `/api/v1/horizontal-pod-autoscalers` |
| operationId | `delete_api_v1_horizontal-pod-autoscalers` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/horizontal-pod-autoscalers, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-HORIZO-002 查询 api/v1/horizontal-pod-autoscalers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HORIZO-002` |
| 接口名称 | 查询 api/v1/horizontal-pod-autoscalers |
| 请求方式 | `GET` |
| URL | `/api/v1/horizontal-pod-autoscalers` |
| Gin 路径 | `/api/v1/horizontal-pod-autoscalers` |
| operationId | `get_api_v1_horizontal-pod-autoscalers` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/horizontal-pod-autoscalers, GET)`；写操作：`OperationAudit` 审计

---

#### API-HORIZO-003 创建/提交 v1/horizontal-pod-autoscalers/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HORIZO-003` |
| 接口名称 | 创建/提交 v1/horizontal-pod-autoscalers/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/horizontal-pod-autoscalers/apply` |
| Gin 路径 | `/api/v1/horizontal-pod-autoscalers/apply` |
| operationId | `post_api_v1_horizontal-pod-autoscalers_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/horizontal-pod-autoscalers/apply, POST)`；写操作：`OperationAudit` 审计

---

#### API-HORIZO-004 查询 v1/horizontal-pod-autoscalers/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-HORIZO-004` |
| 接口名称 | 查询 v1/horizontal-pod-autoscalers/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/horizontal-pod-autoscalers/detail` |
| Gin 路径 | `/api/v1/horizontal-pod-autoscalers/detail` |
| operationId | `get_api_v1_horizontal-pod-autoscalers_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/horizontal-pod-autoscalers/detail, GET)`；写操作：`OperationAudit` 审计

---

### 3.16 Ingresses

#### API-INGRES-001 删除 api/v1/ingresses

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-001` |
| 接口名称 | 删除 api/v1/ingresses |
| 请求方式 | `DELETE` |
| URL | `/api/v1/ingresses` |
| Gin 路径 | `/api/v1/ingresses` |
| operationId | `delete_api_v1_ingresses` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-002 查询 api/v1/ingresses

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-002` |
| 接口名称 | 查询 api/v1/ingresses |
| 请求方式 | `GET` |
| URL | `/api/v1/ingresses` |
| Gin 路径 | `/api/v1/ingresses` |
| operationId | `get_api_v1_ingresses` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-003 创建/提交 v1/ingresses/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-003` |
| 接口名称 | 创建/提交 v1/ingresses/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/ingresses/apply` |
| Gin 路径 | `/api/v1/ingresses/apply` |
| operationId | `post_api_v1_ingresses_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-004 删除 v1/ingresses/classes

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-004` |
| 接口名称 | 删除 v1/ingresses/classes |
| 请求方式 | `DELETE` |
| URL | `/api/v1/ingresses/classes` |
| Gin 路径 | `/api/v1/ingresses/classes` |
| operationId | `delete_api_v1_ingresses_classes` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/classes, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-005 查询 v1/ingresses/classes

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-005` |
| 接口名称 | 查询 v1/ingresses/classes |
| 请求方式 | `GET` |
| URL | `/api/v1/ingresses/classes` |
| Gin 路径 | `/api/v1/ingresses/classes` |
| operationId | `get_api_v1_ingresses_classes` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/classes, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-006 创建/提交 ingresses/classes/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-006` |
| 接口名称 | 创建/提交 ingresses/classes/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/ingresses/classes/apply` |
| Gin 路径 | `/api/v1/ingresses/classes/apply` |
| operationId | `post_api_v1_ingresses_classes_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/classes/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-007 查询 ingresses/classes/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-007` |
| 接口名称 | 查询 ingresses/classes/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/ingresses/classes/detail` |
| Gin 路径 | `/api/v1/ingresses/classes/detail` |
| operationId | `get_api_v1_ingresses_classes_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/classes/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-008 查询 v1/ingresses/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-008` |
| 接口名称 | 查询 v1/ingresses/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/ingresses/detail` |
| Gin 路径 | `/api/v1/ingresses/detail` |
| operationId | `get_api_v1_ingresses_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-009 查询 v1/ingresses/diagnose

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-009` |
| 接口名称 | 查询 v1/ingresses/diagnose |
| 请求方式 | `GET` |
| URL | `/api/v1/ingresses/diagnose` |
| Gin 路径 | `/api/v1/ingresses/diagnose` |
| operationId | `get_api_v1_ingresses_diagnose` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/diagnose, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-INGRES-010 创建/提交 ingresses/nginx/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-INGRES-010` |
| 接口名称 | 创建/提交 ingresses/nginx/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/ingresses/nginx/restart` |
| Gin 路径 | `/api/v1/ingresses/nginx/restart` |
| operationId | `post_api_v1_ingresses_nginx_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/ingresses/nginx/restart, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.17 Jobs

#### API-JOBS-001 删除 api/v1/jobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-001` |
| 接口名称 | 删除 api/v1/jobs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/jobs` |
| Gin 路径 | `/api/v1/jobs` |
| operationId | `delete_api_v1_jobs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-002 查询 api/v1/jobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-002` |
| 接口名称 | 查询 api/v1/jobs |
| 请求方式 | `GET` |
| URL | `/api/v1/jobs` |
| Gin 路径 | `/api/v1/jobs` |
| operationId | `get_api_v1_jobs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-003 创建/提交 v1/jobs/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-003` |
| 接口名称 | 创建/提交 v1/jobs/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/jobs/apply` |
| Gin 路径 | `/api/v1/jobs/apply` |
| operationId | `post_api_v1_jobs_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-004 创建/提交 v1/jobs/container-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-004` |
| 接口名称 | 创建/提交 v1/jobs/container-resources |
| 请求方式 | `POST` |
| URL | `/api/v1/jobs/container-resources` |
| Gin 路径 | `/api/v1/jobs/container-resources` |
| operationId | `post_api_v1_jobs_container-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs/container-resources, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-005 查询 v1/jobs/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-005` |
| 接口名称 | 查询 v1/jobs/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/jobs/detail` |
| Gin 路径 | `/api/v1/jobs/detail` |
| operationId | `get_api_v1_jobs_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-006 查询 v1/jobs/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-006` |
| 接口名称 | 查询 v1/jobs/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/jobs/pods` |
| Gin 路径 | `/api/v1/jobs/pods` |
| operationId | `get_api_v1_jobs_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-JOBS-007 创建/提交 v1/jobs/rerun

| 项 | 内容 |
|----|------|
| 接口编号 | `API-JOBS-007` |
| 接口名称 | 创建/提交 v1/jobs/rerun |
| 请求方式 | `POST` |
| URL | `/api/v1/jobs/rerun` |
| Gin 路径 | `/api/v1/jobs/rerun` |
| operationId | `post_api_v1_jobs_rerun` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/jobs/rerun, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.18 K8s

#### API-K8S-001 查询 k8s/resource-watch/stream

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8S-001` |
| 接口名称 | 查询 k8s/resource-watch/stream |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/resource-watch/stream` |
| Gin 路径 | `/api/v1/k8s/resource-watch/stream` |
| operationId | `get_api_v1_k8s_resource-watch_stream` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/resource-watch/stream, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8S-002 查询 v1/k8s/search

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8S-002` |
| 接口名称 | 查询 v1/k8s/search |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/search` |
| Gin 路径 | `/api/v1/k8s/search` |
| operationId | `get_api_v1_k8s_search` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/search, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8S-003 查询 v1/k8s/topology

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8S-003` |
| 接口名称 | 查询 v1/k8s/topology |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/topology` |
| Gin 路径 | `/api/v1/k8s/topology` |
| operationId | `get_api_v1_k8s_topology` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/topology, GET)`；写操作：`OperationAudit` 审计

---

### 3.19 K8s-namespace-allow-rules

#### API-K8SNAM-001 查询 api/v1/k8s-namespace-allow-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-001` |
| 接口名称 | 查询 api/v1/k8s-namespace-allow-rules |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-namespace-allow-rules` |
| Gin 路径 | `/api/v1/k8s-namespace-allow-rules` |
| operationId | `get_api_v1_k8s-namespace-allow-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-allow-rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SNAM-002 创建/提交 api/v1/k8s-namespace-allow-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-002` |
| 接口名称 | 创建/提交 api/v1/k8s-namespace-allow-rules |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s-namespace-allow-rules` |
| Gin 路径 | `/api/v1/k8s-namespace-allow-rules` |
| operationId | `post_api_v1_k8s-namespace-allow-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-allow-rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SNAM-003 删除 api/v1/k8s-namespace-allow-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-003` |
| 接口名称 | 删除 api/v1/k8s-namespace-allow-rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/k8s-namespace-allow-rules/{id}` |
| Gin 路径 | `/api/v1/k8s-namespace-allow-rules/:id` |
| operationId | `delete_api_v1_k8s-namespace-allow-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-allow-rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

### 3.20 K8s-namespace-deny-rules

#### API-K8SNAM-001 查询 api/v1/k8s-namespace-deny-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-001` |
| 接口名称 | 查询 api/v1/k8s-namespace-deny-rules |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-namespace-deny-rules` |
| Gin 路径 | `/api/v1/k8s-namespace-deny-rules` |
| operationId | `get_api_v1_k8s-namespace-deny-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-deny-rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SNAM-002 创建/提交 api/v1/k8s-namespace-deny-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-002` |
| 接口名称 | 创建/提交 api/v1/k8s-namespace-deny-rules |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s-namespace-deny-rules` |
| Gin 路径 | `/api/v1/k8s-namespace-deny-rules` |
| operationId | `post_api_v1_k8s-namespace-deny-rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-deny-rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SNAM-003 删除 api/v1/k8s-namespace-deny-rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SNAM-003` |
| 接口名称 | 删除 api/v1/k8s-namespace-deny-rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/k8s-namespace-deny-rules/{id}` |
| Gin 路径 | `/api/v1/k8s-namespace-deny-rules/:id` |
| operationId | `delete_api_v1_k8s-namespace-deny-rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-namespace-deny-rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

### 3.21 K8s-services

#### API-K8SSER-001 删除 api/v1/k8s-services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SSER-001` |
| 接口名称 | 删除 api/v1/k8s-services |
| 请求方式 | `DELETE` |
| URL | `/api/v1/k8s-services` |
| Gin 路径 | `/api/v1/k8s-services` |
| operationId | `delete_api_v1_k8s-services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-services, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-K8SSER-002 查询 api/v1/k8s-services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SSER-002` |
| 接口名称 | 查询 api/v1/k8s-services |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-services` |
| Gin 路径 | `/api/v1/k8s-services` |
| operationId | `get_api_v1_k8s-services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-services, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SSER-003 创建/提交 v1/k8s-services/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SSER-003` |
| 接口名称 | 创建/提交 v1/k8s-services/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s-services/apply` |
| Gin 路径 | `/api/v1/k8s-services/apply` |
| operationId | `post_api_v1_k8s-services_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-services/apply, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SSER-004 查询 v1/k8s-services/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SSER-004` |
| 接口名称 | 查询 v1/k8s-services/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-services/detail` |
| Gin 路径 | `/api/v1/k8s-services/detail` |
| operationId | `get_api_v1_k8s-services_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-services/detail, GET)`；写操作：`OperationAudit` 审计

---

### 3.22 K8sEventForward

#### API-K8SEF-001 查询 k8s/event-forward/rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-001` |
| 接口名称 | 查询 k8s/event-forward/rules |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/event-forward/rules` |
| Gin 路径 | `/api/v1/k8s/event-forward/rules` |
| operationId | `get_api_v1_k8s_event-forward_rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/rules, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-002 创建/提交 k8s/event-forward/rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-002` |
| 接口名称 | 创建/提交 k8s/event-forward/rules |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s/event-forward/rules` |
| Gin 路径 | `/api/v1/k8s/event-forward/rules` |
| operationId | `post_api_v1_k8s_event-forward_rules` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/rules, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-003 删除 k8s/event-forward/rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-003` |
| 接口名称 | 删除 k8s/event-forward/rules |
| 请求方式 | `DELETE` |
| URL | `/api/v1/k8s/event-forward/rules/{id}` |
| Gin 路径 | `/api/v1/k8s/event-forward/rules/:id` |
| operationId | `delete_api_v1_k8s_event-forward_rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/rules/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-004 查询 k8s/event-forward/rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-004` |
| 接口名称 | 查询 k8s/event-forward/rules |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/event-forward/rules/{id}` |
| Gin 路径 | `/api/v1/k8s/event-forward/rules/:id` |
| operationId | `get_api_v1_k8s_event-forward_rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/rules/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-005 更新 k8s/event-forward/rules

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-005` |
| 接口名称 | 更新 k8s/event-forward/rules |
| 请求方式 | `PUT` |
| URL | `/api/v1/k8s/event-forward/rules/{id}` |
| Gin 路径 | `/api/v1/k8s/event-forward/rules/:id` |
| operationId | `put_api_v1_k8s_event-forward_rules_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/rules/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-006 查询 k8s/event-forward/settings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-006` |
| 接口名称 | 查询 k8s/event-forward/settings |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s/event-forward/settings` |
| Gin 路径 | `/api/v1/k8s/event-forward/settings` |
| operationId | `get_api_v1_k8s_event-forward_settings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/settings, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SEF-007 更新 k8s/event-forward/settings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SEF-007` |
| 接口名称 | 更新 k8s/event-forward/settings |
| 请求方式 | `PUT` |
| URL | `/api/v1/k8s/event-forward/settings` |
| Gin 路径 | `/api/v1/k8s/event-forward/settings` |
| operationId | `put_api_v1_k8s_event-forward_settings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s/event-forward/settings, PUT)`；写操作：`OperationAudit` 审计

---

### 3.23 K8sScopedPolicy

#### API-K8SP-001 查询 api/v1/k8s-policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-001` |
| 接口名称 | 查询 api/v1/k8s-policies |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies` |
| Gin 路径 | `/api/v1/k8s-policies` |
| operationId | `get_api_v1_k8s-policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-002 查询 v1/k8s-policies/actions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-002` |
| 接口名称 | 查询 v1/k8s-policies/actions |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies/actions` |
| Gin 路径 | `/api/v1/k8s-policies/actions` |
| operationId | `get_api_v1_k8s-policies_actions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/actions, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-003 查询 v1/k8s-policies/cluster-auth-matrix

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-003` |
| 接口名称 | 查询 v1/k8s-policies/cluster-auth-matrix |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies/cluster-auth-matrix` |
| Gin 路径 | `/api/v1/k8s-policies/cluster-auth-matrix` |
| operationId | `get_api_v1_k8s-policies_cluster-auth-matrix` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/cluster-auth-matrix, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-004 创建/提交 k8s-policies/cluster-grants/batch-delete

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-004` |
| 接口名称 | 创建/提交 k8s-policies/cluster-grants/batch-delete |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s-policies/cluster-grants/batch-delete` |
| Gin 路径 | `/api/v1/k8s-policies/cluster-grants/batch-delete` |
| operationId | `post_api_v1_k8s-policies_cluster-grants_batch-delete` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/cluster-grants/batch-delete, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-005 删除 v1/k8s-policies/cluster-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-005` |
| 接口名称 | 删除 v1/k8s-policies/cluster-grants |
| 请求方式 | `DELETE` |
| URL | `/api/v1/k8s-policies/cluster-grants/{id}` |
| Gin 路径 | `/api/v1/k8s-policies/cluster-grants/:id` |
| operationId | `delete_api_v1_k8s-policies_cluster-grants_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/cluster-grants/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-006 创建/提交 v1/k8s-policies/grant-preset

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-006` |
| 接口名称 | 创建/提交 v1/k8s-policies/grant-preset |
| 请求方式 | `POST` |
| URL | `/api/v1/k8s-policies/grant-preset` |
| Gin 路径 | `/api/v1/k8s-policies/grant-preset` |
| operationId | `post_api_v1_k8s-policies_grant-preset` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/grant-preset, POST)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-007 查询 v1/k8s-policies/my-access

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-007` |
| 接口名称 | 查询 v1/k8s-policies/my-access |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies/my-access` |
| Gin 路径 | `/api/v1/k8s-policies/my-access` |
| operationId | `get_api_v1_k8s-policies_my-access` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/my-access, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-008 查询 v1/k8s-policies/paths

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-008` |
| 接口名称 | 查询 v1/k8s-policies/paths |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies/paths` |
| Gin 路径 | `/api/v1/k8s-policies/paths` |
| operationId | `get_api_v1_k8s-policies_paths` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/paths, GET)`；写操作：`OperationAudit` 审计

---

#### API-K8SP-009 查询 v1/k8s-policies/user-cluster-auth

| 项 | 内容 |
|----|------|
| 接口编号 | `API-K8SP-009` |
| 接口名称 | 查询 v1/k8s-policies/user-cluster-auth |
| 请求方式 | `GET` |
| URL | `/api/v1/k8s-policies/user-cluster-auth` |
| Gin 路径 | `/api/v1/k8s-policies/user-cluster-auth` |
| operationId | `get_api_v1_k8s-policies_user-cluster-auth` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/k8s-policies/user-cluster-auth, GET)`；写操作：`OperationAudit` 审计

---

### 3.24 LogPlatform

#### API-LOG-001 查询 v1/log-platform/es-config

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-001` |
| 接口名称 | 查询 v1/log-platform/es-config |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/es-config` |
| Gin 路径 | `/api/v1/log-platform/es-config` |
| operationId | `get_api_v1_log-platform_es-config` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/es-config, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOG-002 删除 v1/log-platform/es-indices

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-002` |
| 接口名称 | 删除 v1/log-platform/es-indices |
| 请求方式 | `DELETE` |
| URL | `/api/v1/log-platform/es-indices/{index}` |
| Gin 路径 | `/api/v1/log-platform/es-indices/:index` |
| operationId | `delete_api_v1_log-platform_es-indices_index_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `index` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/es-indices/:index, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-LOG-003 查询 v1/log-platform/es-storage

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-003` |
| 接口名称 | 查询 v1/log-platform/es-storage |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/es-storage` |
| Gin 路径 | `/api/v1/log-platform/es-storage` |
| operationId | `get_api_v1_log-platform_es-storage` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/es-storage, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOG-004 查询 v1/log-platform/kafka-config

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-004` |
| 接口名称 | 查询 v1/log-platform/kafka-config |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/kafka-config` |
| Gin 路径 | `/api/v1/log-platform/kafka-config` |
| operationId | `get_api_v1_log-platform_kafka-config` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/kafka-config, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOG-005 查询 v1/log-platform/kafka-stats

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-005` |
| 接口名称 | 查询 v1/log-platform/kafka-stats |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/kafka-stats` |
| Gin 路径 | `/api/v1/log-platform/kafka-stats` |
| operationId | `get_api_v1_log-platform_kafka-stats` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/kafka-stats, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOG-006 删除 v1/log-platform/kafka-topics

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-006` |
| 接口名称 | 删除 v1/log-platform/kafka-topics |
| 请求方式 | `DELETE` |
| URL | `/api/v1/log-platform/kafka-topics/{topic}` |
| Gin 路径 | `/api/v1/log-platform/kafka-topics/:topic` |
| operationId | `delete_api_v1_log-platform_kafka-topics_topic_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `topic` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/kafka-topics/:topic, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-LOG-007 查询 v1/log-platform/retention

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-007` |
| 接口名称 | 查询 v1/log-platform/retention |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/retention` |
| Gin 路径 | `/api/v1/log-platform/retention` |
| operationId | `get_api_v1_log-platform_retention` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/retention, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOG-008 更新 v1/log-platform/retention

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-008` |
| 接口名称 | 更新 v1/log-platform/retention |
| 请求方式 | `PUT` |
| URL | `/api/v1/log-platform/retention` |
| Gin 路径 | `/api/v1/log-platform/retention` |
| operationId | `put_api_v1_log-platform_retention` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/retention, PUT)`；写操作：`OperationAudit` 审计

---

#### API-LOG-009 创建/提交 log-platform/retention/cleanup

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-009` |
| 接口名称 | 创建/提交 log-platform/retention/cleanup |
| 请求方式 | `POST` |
| URL | `/api/v1/log-platform/retention/cleanup` |
| Gin 路径 | `/api/v1/log-platform/retention/cleanup` |
| operationId | `post_api_v1_log-platform_retention_cleanup` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/retention/cleanup, POST)`；写操作：`OperationAudit` 审计

---

#### API-LOG-010 查询 log-platform/retention/list

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOG-010` |
| 接口名称 | 查询 log-platform/retention/list |
| 请求方式 | `GET` |
| URL | `/api/v1/log-platform/retention/list` |
| Gin 路径 | `/api/v1/log-platform/retention/list` |
| operationId | `get_api_v1_log-platform_retention_list` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/log-platform/retention/list, GET)`；写操作：`OperationAudit` 审计

---

### 3.25 Loggie

#### API-LGG-001 创建/提交 loggie/heartbeat/report

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LGG-001` |
| 接口名称 | 创建/提交 loggie/heartbeat/report |
| 请求方式 | `POST` |
| URL | `/api/v1/loggie/heartbeat/report` |
| Gin 路径 | `/api/v1/loggie/heartbeat/report` |
| operationId | `post_api_v1_loggie_heartbeat_report` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

Agent 心跳上报（公开接口，按部署网络隔离）

---

### 3.26 Login-logs

#### API-LOGINL-001 查询 api/v1/login-logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOGINL-001` |
| 接口名称 | 查询 api/v1/login-logs |
| 请求方式 | `GET` |
| URL | `/api/v1/login-logs` |
| Gin 路径 | `/api/v1/login-logs` |
| operationId | `get_api_v1_login-logs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/login-logs, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOGINL-002 创建/提交 v1/login-logs/delete

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOGINL-002` |
| 接口名称 | 创建/提交 v1/login-logs/delete |
| 请求方式 | `POST` |
| URL | `/api/v1/login-logs/delete` |
| Gin 路径 | `/api/v1/login-logs/delete` |
| operationId | `post_api_v1_login-logs_delete` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/login-logs/delete, POST)`；写操作：`OperationAudit` 审计

---

#### API-LOGINL-003 查询 v1/login-logs/export

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOGINL-003` |
| 接口名称 | 查询 v1/login-logs/export |
| 请求方式 | `GET` |
| URL | `/api/v1/login-logs/export` |
| Gin 路径 | `/api/v1/login-logs/export` |
| operationId | `get_api_v1_login-logs_export` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/login-logs/export, GET)`；写操作：`OperationAudit` 审计

---

#### API-LOGINL-004 删除 api/v1/login-logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-LOGINL-004` |
| 接口名称 | 删除 api/v1/login-logs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/login-logs/{id}` |
| Gin 路径 | `/api/v1/login-logs/:id` |
| operationId | `delete_api_v1_login-logs_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/login-logs/:id, DELETE)`；写操作：`OperationAudit` 审计

---

### 3.27 Menus

#### API-MENU-001 创建/提交 api/v1/menus

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-001` |
| 接口名称 | 创建/提交 api/v1/menus |
| 请求方式 | `POST` |
| URL | `/api/v1/menus` |
| Gin 路径 | `/api/v1/menus` |
| operationId | `post_api_v1_menus` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus, POST)`；写操作：`OperationAudit` 审计

---

#### API-MENU-002 更新 v1/menus/status

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-002` |
| 接口名称 | 更新 v1/menus/status |
| 请求方式 | `PUT` |
| URL | `/api/v1/menus/status` |
| Gin 路径 | `/api/v1/menus/status` |
| operationId | `put_api_v1_menus_status` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/status, PUT)`；写操作：`OperationAudit` 审计

---

#### API-MENU-003 查询 v1/menus/tree

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-003` |
| 接口名称 | 查询 v1/menus/tree |
| 请求方式 | `GET` |
| URL | `/api/v1/menus/tree` |
| Gin 路径 | `/api/v1/menus/tree` |
| operationId | `get_api_v1_menus_tree` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/tree, GET)`；写操作：`OperationAudit` 审计

---

#### API-MENU-004 删除 api/v1/menus

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-004` |
| 接口名称 | 删除 api/v1/menus |
| 请求方式 | `DELETE` |
| URL | `/api/v1/menus/{id}` |
| Gin 路径 | `/api/v1/menus/:id` |
| operationId | `delete_api_v1_menus_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-MENU-005 更新 api/v1/menus

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-005` |
| 接口名称 | 更新 api/v1/menus |
| 请求方式 | `PUT` |
| URL | `/api/v1/menus/{id}` |
| Gin 路径 | `/api/v1/menus/:id` |
| operationId | `put_api_v1_menus_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-MENU-006 查询 v1/menus/bindings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-006` |
| 接口名称 | 查询 v1/menus/bindings |
| 请求方式 | `GET` |
| URL | `/api/v1/menus/{id}/bindings` |
| Gin 路径 | `/api/v1/menus/:id/bindings` |
| operationId | `get_api_v1_menus_id__bindings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/:id/bindings, GET)`；写操作：`OperationAudit` 审计

---

#### API-MENU-007 更新 v1/menus/bindings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-MENU-007` |
| 接口名称 | 更新 v1/menus/bindings |
| 请求方式 | `PUT` |
| URL | `/api/v1/menus/{id}/bindings` |
| Gin 路径 | `/api/v1/menus/:id/bindings` |
| operationId | `put_api_v1_menus_id__bindings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/menus/:id/bindings, PUT)`；写操作：`OperationAudit` 审计

---

### 3.28 Namespaces

#### API-NS-001 删除 api/v1/namespaces

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NS-001` |
| 接口名称 | 删除 api/v1/namespaces |
| 请求方式 | `DELETE` |
| URL | `/api/v1/namespaces` |
| Gin 路径 | `/api/v1/namespaces` |
| operationId | `delete_api_v1_namespaces` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/namespaces, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NS-002 查询 api/v1/namespaces

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NS-002` |
| 接口名称 | 查询 api/v1/namespaces |
| 请求方式 | `GET` |
| URL | `/api/v1/namespaces` |
| Gin 路径 | `/api/v1/namespaces` |
| operationId | `get_api_v1_namespaces` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/namespaces, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NS-003 创建/提交 v1/namespaces/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NS-003` |
| 接口名称 | 创建/提交 v1/namespaces/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/namespaces/apply` |
| Gin 路径 | `/api/v1/namespaces/apply` |
| operationId | `post_api_v1_namespaces_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/namespaces/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NS-004 查询 v1/namespaces/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NS-004` |
| 接口名称 | 查询 v1/namespaces/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/namespaces/detail` |
| Gin 路径 | `/api/v1/namespaces/detail` |
| operationId | `get_api_v1_namespaces_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/namespaces/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.29 Network-policies

#### API-NETWOR-001 删除 api/v1/network-policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NETWOR-001` |
| 接口名称 | 删除 api/v1/network-policies |
| 请求方式 | `DELETE` |
| URL | `/api/v1/network-policies` |
| Gin 路径 | `/api/v1/network-policies` |
| operationId | `delete_api_v1_network-policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/network-policies, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-NETWOR-002 查询 api/v1/network-policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NETWOR-002` |
| 接口名称 | 查询 api/v1/network-policies |
| 请求方式 | `GET` |
| URL | `/api/v1/network-policies` |
| Gin 路径 | `/api/v1/network-policies` |
| operationId | `get_api_v1_network-policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/network-policies, GET)`；写操作：`OperationAudit` 审计

---

#### API-NETWOR-003 创建/提交 v1/network-policies/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NETWOR-003` |
| 接口名称 | 创建/提交 v1/network-policies/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/network-policies/apply` |
| Gin 路径 | `/api/v1/network-policies/apply` |
| operationId | `post_api_v1_network-policies_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/network-policies/apply, POST)`；写操作：`OperationAudit` 审计

---

#### API-NETWOR-004 查询 v1/network-policies/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NETWOR-004` |
| 接口名称 | 查询 v1/network-policies/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/network-policies/detail` |
| Gin 路径 | `/api/v1/network-policies/detail` |
| operationId | `get_api_v1_network-policies_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/network-policies/detail, GET)`；写操作：`OperationAudit` 审计

---

### 3.30 Nodes

#### API-NODE-001 查询 api/v1/nodes

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NODE-001` |
| 接口名称 | 查询 api/v1/nodes |
| 请求方式 | `GET` |
| URL | `/api/v1/nodes` |
| Gin 路径 | `/api/v1/nodes` |
| operationId | `get_api_v1_nodes` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/nodes, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NODE-002 查询 v1/nodes/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NODE-002` |
| 接口名称 | 查询 v1/nodes/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/nodes/detail` |
| Gin 路径 | `/api/v1/nodes/detail` |
| operationId | `get_api_v1_nodes_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/nodes/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NODE-003 创建/提交 v1/nodes/schedulability

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NODE-003` |
| 接口名称 | 创建/提交 v1/nodes/schedulability |
| 请求方式 | `POST` |
| URL | `/api/v1/nodes/schedulability` |
| Gin 路径 | `/api/v1/nodes/schedulability` |
| operationId | `post_api_v1_nodes_schedulability` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/nodes/schedulability, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-NODE-004 更新 v1/nodes/taints

| 项 | 内容 |
|----|------|
| 接口编号 | `API-NODE-004` |
| 接口名称 | 更新 v1/nodes/taints |
| 请求方式 | `PUT` |
| URL | `/api/v1/nodes/taints` |
| Gin 路径 | `/api/v1/nodes/taints` |
| operationId | `put_api_v1_nodes_taints` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/nodes/taints, PUT)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.31 Operation-logs

#### API-OPERAT-001 查询 api/v1/operation-logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OPERAT-001` |
| 接口名称 | 查询 api/v1/operation-logs |
| 请求方式 | `GET` |
| URL | `/api/v1/operation-logs` |
| Gin 路径 | `/api/v1/operation-logs` |
| operationId | `get_api_v1_operation-logs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/operation-logs, GET)`；写操作：`OperationAudit` 审计

---

#### API-OPERAT-002 创建/提交 v1/operation-logs/delete

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OPERAT-002` |
| 接口名称 | 创建/提交 v1/operation-logs/delete |
| 请求方式 | `POST` |
| URL | `/api/v1/operation-logs/delete` |
| Gin 路径 | `/api/v1/operation-logs/delete` |
| operationId | `post_api_v1_operation-logs_delete` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/operation-logs/delete, POST)`；写操作：`OperationAudit` 审计

---

#### API-OPERAT-003 查询 v1/operation-logs/export

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OPERAT-003` |
| 接口名称 | 查询 v1/operation-logs/export |
| 请求方式 | `GET` |
| URL | `/api/v1/operation-logs/export` |
| Gin 路径 | `/api/v1/operation-logs/export` |
| operationId | `get_api_v1_operation-logs_export` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/operation-logs/export, GET)`；写操作：`OperationAudit` 审计

---

#### API-OPERAT-004 删除 api/v1/operation-logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OPERAT-004` |
| 接口名称 | 删除 api/v1/operation-logs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/operation-logs/{id}` |
| Gin 路径 | `/api/v1/operation-logs/:id` |
| operationId | `delete_api_v1_operation-logs_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/operation-logs/:id, DELETE)`；写操作：`OperationAudit` 审计

---

### 3.32 Overview

#### API-OVW-001 查询 api/v1/overview

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OVW-001` |
| 接口名称 | 查询 api/v1/overview |
| 请求方式 | `GET` |
| URL | `/api/v1/overview` |
| Gin 路径 | `/api/v1/overview` |
| operationId | `get_api_v1_overview` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/overview, GET)`；写操作：`OperationAudit` 审计

---

#### API-OVW-002 查询 v1/overview/project-launches

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OVW-002` |
| 接口名称 | 查询 v1/overview/project-launches |
| 请求方式 | `GET` |
| URL | `/api/v1/overview/project-launches` |
| Gin 路径 | `/api/v1/overview/project-launches` |
| operationId | `get_api_v1_overview_project-launches` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/overview/project-launches, GET)`；写操作：`OperationAudit` 审计

---

#### API-OVW-003 查询 v1/overview/release-by-person

| 项 | 内容 |
|----|------|
| 接口编号 | `API-OVW-003` |
| 接口名称 | 查询 v1/overview/release-by-person |
| 请求方式 | `GET` |
| URL | `/api/v1/overview/release-by-person` |
| Gin 路径 | `/api/v1/overview/release-by-person` |
| operationId | `get_api_v1_overview_release-by-person` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/overview/release-by-person, GET)`；写操作：`OperationAudit` 审计

---

### 3.33 Permissions

#### API-PERM-001 查询 api/v1/permissions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-001` |
| 接口名称 | 查询 api/v1/permissions |
| 请求方式 | `GET` |
| URL | `/api/v1/permissions` |
| Gin 路径 | `/api/v1/permissions` |
| operationId | `get_api_v1_permissions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions, GET)`；写操作：`OperationAudit` 审计

---

#### API-PERM-002 创建/提交 api/v1/permissions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-002` |
| 接口名称 | 创建/提交 api/v1/permissions |
| 请求方式 | `POST` |
| URL | `/api/v1/permissions` |
| Gin 路径 | `/api/v1/permissions` |
| operationId | `post_api_v1_permissions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions, POST)`；写操作：`OperationAudit` 审计

---

#### API-PERM-003 创建/提交 permissions/k8s-scope/batch

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-003` |
| 接口名称 | 创建/提交 permissions/k8s-scope/batch |
| 请求方式 | `POST` |
| URL | `/api/v1/permissions/k8s-scope/batch` |
| Gin 路径 | `/api/v1/permissions/k8s-scope/batch` |
| operationId | `post_api_v1_permissions_k8s-scope_batch` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions/k8s-scope/batch, POST)`；写操作：`OperationAudit` 审计

---

#### API-PERM-004 删除 api/v1/permissions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-004` |
| 接口名称 | 删除 api/v1/permissions |
| 请求方式 | `DELETE` |
| URL | `/api/v1/permissions/{id}` |
| Gin 路径 | `/api/v1/permissions/:id` |
| operationId | `delete_api_v1_permissions_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-PERM-005 查询 api/v1/permissions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-005` |
| 接口名称 | 查询 api/v1/permissions |
| 请求方式 | `GET` |
| URL | `/api/v1/permissions/{id}` |
| Gin 路径 | `/api/v1/permissions/:id` |
| operationId | `get_api_v1_permissions_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-PERM-006 更新 api/v1/permissions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERM-006` |
| 接口名称 | 更新 api/v1/permissions |
| 请求方式 | `PUT` |
| URL | `/api/v1/permissions/{id}` |
| Gin 路径 | `/api/v1/permissions/:id` |
| operationId | `put_api_v1_permissions_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/permissions/:id, PUT)`；写操作：`OperationAudit` 审计

---

### 3.34 Persistentvolumeclaims

#### API-PERSIS-001 删除 api/v1/persistentvolumeclaims

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-001` |
| 接口名称 | 删除 api/v1/persistentvolumeclaims |
| 请求方式 | `DELETE` |
| URL | `/api/v1/persistentvolumeclaims` |
| Gin 路径 | `/api/v1/persistentvolumeclaims` |
| operationId | `delete_api_v1_persistentvolumeclaims` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumeclaims, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-PERSIS-002 查询 api/v1/persistentvolumeclaims

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-002` |
| 接口名称 | 查询 api/v1/persistentvolumeclaims |
| 请求方式 | `GET` |
| URL | `/api/v1/persistentvolumeclaims` |
| Gin 路径 | `/api/v1/persistentvolumeclaims` |
| operationId | `get_api_v1_persistentvolumeclaims` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumeclaims, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-PERSIS-003 创建/提交 v1/persistentvolumeclaims/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-003` |
| 接口名称 | 创建/提交 v1/persistentvolumeclaims/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/persistentvolumeclaims/apply` |
| Gin 路径 | `/api/v1/persistentvolumeclaims/apply` |
| operationId | `post_api_v1_persistentvolumeclaims_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumeclaims/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-PERSIS-004 查询 v1/persistentvolumeclaims/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-004` |
| 接口名称 | 查询 v1/persistentvolumeclaims/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/persistentvolumeclaims/detail` |
| Gin 路径 | `/api/v1/persistentvolumeclaims/detail` |
| operationId | `get_api_v1_persistentvolumeclaims_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumeclaims/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.35 Persistentvolumes

#### API-PERSIS-001 删除 api/v1/persistentvolumes

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-001` |
| 接口名称 | 删除 api/v1/persistentvolumes |
| 请求方式 | `DELETE` |
| URL | `/api/v1/persistentvolumes` |
| Gin 路径 | `/api/v1/persistentvolumes` |
| operationId | `delete_api_v1_persistentvolumes` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumes, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-PERSIS-002 查询 api/v1/persistentvolumes

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-002` |
| 接口名称 | 查询 api/v1/persistentvolumes |
| 请求方式 | `GET` |
| URL | `/api/v1/persistentvolumes` |
| Gin 路径 | `/api/v1/persistentvolumes` |
| operationId | `get_api_v1_persistentvolumes` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumes, GET)`；写操作：`OperationAudit` 审计

---

#### API-PERSIS-003 创建/提交 v1/persistentvolumes/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-003` |
| 接口名称 | 创建/提交 v1/persistentvolumes/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/persistentvolumes/apply` |
| Gin 路径 | `/api/v1/persistentvolumes/apply` |
| operationId | `post_api_v1_persistentvolumes_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumes/apply, POST)`；写操作：`OperationAudit` 审计

---

#### API-PERSIS-004 查询 v1/persistentvolumes/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PERSIS-004` |
| 接口名称 | 查询 v1/persistentvolumes/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/persistentvolumes/detail` |
| Gin 路径 | `/api/v1/persistentvolumes/detail` |
| operationId | `get_api_v1_persistentvolumes_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/persistentvolumes/detail, GET)`；写操作：`OperationAudit` 审计

---

### 3.36 Plugins

#### API-PLG-001 查询 api/v1/plugins

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PLG-001` |
| 接口名称 | 查询 api/v1/plugins |
| 请求方式 | `GET` |
| URL | `/api/v1/plugins` |
| Gin 路径 | `/api/v1/plugins` |
| operationId | `get_api_v1_plugins` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/plugins, GET)`；写操作：`OperationAudit` 审计

---

### 3.37 Pods

#### API-POD-001 删除 api/v1/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-001` |
| 接口名称 | 删除 api/v1/pods |
| 请求方式 | `DELETE` |
| URL | `/api/v1/pods` |
| Gin 路径 | `/api/v1/pods` |
| operationId | `delete_api_v1_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-002 查询 api/v1/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-002` |
| 接口名称 | 查询 api/v1/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/pods` |
| Gin 路径 | `/api/v1/pods` |
| operationId | `get_api_v1_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-003 创建/提交 pods/create/simple

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-003` |
| 接口名称 | 创建/提交 pods/create/simple |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/create/simple` |
| Gin 路径 | `/api/v1/pods/create/simple` |
| operationId | `post_api_v1_pods_create_simple` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/create/simple, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-004 创建/提交 pods/create/yaml

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-004` |
| 接口名称 | 创建/提交 pods/create/yaml |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/create/yaml` |
| Gin 路径 | `/api/v1/pods/create/yaml` |
| operationId | `post_api_v1_pods_create_yaml` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/create/yaml, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-005 查询 v1/pods/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-005` |
| 接口名称 | 查询 v1/pods/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/detail` |
| Gin 路径 | `/api/v1/pods/detail` |
| operationId | `get_api_v1_pods_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-006 查询 v1/pods/diagnose

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-006` |
| 接口名称 | 查询 v1/pods/diagnose |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/diagnose` |
| Gin 路径 | `/api/v1/pods/diagnose` |
| operationId | `get_api_v1_pods_diagnose` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/diagnose, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-007 查询 v1/pods/events

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-007` |
| 接口名称 | 查询 v1/pods/events |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/events` |
| Gin 路径 | `/api/v1/pods/events` |
| operationId | `get_api_v1_pods_events` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/events, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-008 创建/提交 v1/pods/exec

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-008` |
| 接口名称 | 创建/提交 v1/pods/exec |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/exec` |
| Gin 路径 | `/api/v1/pods/exec` |
| operationId | `post_api_v1_pods_exec` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/exec, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-009 查询 pods/exec/ws

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-009` |
| 接口名称 | 查询 pods/exec/ws |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/exec/ws` |
| Gin 路径 | `/api/v1/pods/exec/ws` |
| operationId | `get_api_v1_pods_exec_ws` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/exec/ws, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；WebSocket：先 `POST /api/v1/auth/ws-ticket`，连接 URL 携带一次性 `ticket`；写操作：`OperationAudit` 审计

---

#### API-POD-010 查询 v1/pods/file

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-010` |
| 接口名称 | 查询 v1/pods/file |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/file` |
| Gin 路径 | `/api/v1/pods/file` |
| operationId | `get_api_v1_pods_file` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/file, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-011 创建/提交 pods/file/delete

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-011` |
| 接口名称 | 创建/提交 pods/file/delete |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/file/delete` |
| Gin 路径 | `/api/v1/pods/file/delete` |
| operationId | `post_api_v1_pods_file_delete` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/file/delete, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-012 查询 pods/file/download

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-012` |
| 接口名称 | 查询 pods/file/download |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/file/download` |
| Gin 路径 | `/api/v1/pods/file/download` |
| operationId | `get_api_v1_pods_file_download` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/file/download, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-013 创建/提交 pods/file/upload

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-013` |
| 接口名称 | 创建/提交 pods/file/upload |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/file/upload` |
| Gin 路径 | `/api/v1/pods/file/upload` |
| operationId | `post_api_v1_pods_file_upload` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/file/upload, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-014 查询 v1/pods/files

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-014` |
| 接口名称 | 查询 v1/pods/files |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/files` |
| Gin 路径 | `/api/v1/pods/files` |
| operationId | `get_api_v1_pods_files` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/files, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-015 查询 v1/pods/logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-015` |
| 接口名称 | 查询 v1/pods/logs |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/logs` |
| Gin 路径 | `/api/v1/pods/logs` |
| operationId | `get_api_v1_pods_logs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/logs, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-016 查询 pods/logs/download

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-016` |
| 接口名称 | 查询 pods/logs/download |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/logs/download` |
| Gin 路径 | `/api/v1/pods/logs/download` |
| operationId | `get_api_v1_pods_logs_download` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/logs/download, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-017 查询 pods/logs/stream

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-017` |
| 接口名称 | 查询 pods/logs/stream |
| 请求方式 | `GET` |
| URL | `/api/v1/pods/logs/stream` |
| Gin 路径 | `/api/v1/pods/logs/stream` |
| operationId | `get_api_v1_pods_logs_stream` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/logs/stream, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-018 创建/提交 v1/pods/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-018` |
| 接口名称 | 创建/提交 v1/pods/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/restart` |
| Gin 路径 | `/api/v1/pods/restart` |
| operationId | `post_api_v1_pods_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/restart, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-POD-019 创建/提交 pods/update/simple

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POD-019` |
| 接口名称 | 创建/提交 pods/update/simple |
| 请求方式 | `POST` |
| URL | `/api/v1/pods/update/simple` |
| Gin 路径 | `/api/v1/pods/update/simple` |
| operationId | `post_api_v1_pods_update_simple` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/pods/update/simple, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.38 Policies

#### API-POL-001 删除 api/v1/policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-001` |
| 接口名称 | 删除 api/v1/policies |
| 请求方式 | `DELETE` |
| URL | `/api/v1/policies` |
| Gin 路径 | `/api/v1/policies` |
| operationId | `delete_api_v1_policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-POL-002 查询 api/v1/policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-002` |
| 接口名称 | 查询 api/v1/policies |
| 请求方式 | `GET` |
| URL | `/api/v1/policies` |
| Gin 路径 | `/api/v1/policies` |
| operationId | `get_api_v1_policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies, GET)`；写操作：`OperationAudit` 审计

---

#### API-POL-003 创建/提交 api/v1/policies

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-003` |
| 接口名称 | 创建/提交 api/v1/policies |
| 请求方式 | `POST` |
| URL | `/api/v1/policies` |
| Gin 路径 | `/api/v1/policies` |
| operationId | `post_api_v1_policies` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies, POST)`；写操作：`OperationAudit` 审计

---

#### API-POL-004 查询 v1/policies/conflicts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-004` |
| 接口名称 | 查询 v1/policies/conflicts |
| 请求方式 | `GET` |
| URL | `/api/v1/policies/conflicts` |
| Gin 路径 | `/api/v1/policies/conflicts` |
| operationId | `get_api_v1_policies_conflicts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/conflicts, GET)`；写操作：`OperationAudit` 审计

---

#### API-POL-005 创建/提交 policies/conflicts/fix-disabled-plugin

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-005` |
| 接口名称 | 创建/提交 policies/conflicts/fix-disabled-plugin |
| 请求方式 | `POST` |
| URL | `/api/v1/policies/conflicts/fix-disabled-plugin` |
| Gin 路径 | `/api/v1/policies/conflicts/fix-disabled-plugin` |
| operationId | `post_api_v1_policies_conflicts_fix-disabled-plugin` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/conflicts/fix-disabled-plugin, POST)`；写操作：`OperationAudit` 审计

---

#### API-POL-006 创建/提交 policies/conflicts/fix-menu-entry

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-006` |
| 接口名称 | 创建/提交 policies/conflicts/fix-menu-entry |
| 请求方式 | `POST` |
| URL | `/api/v1/policies/conflicts/fix-menu-entry` |
| Gin 路径 | `/api/v1/policies/conflicts/fix-menu-entry` |
| operationId | `post_api_v1_policies_conflicts_fix-menu-entry` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/conflicts/fix-menu-entry, POST)`；写操作：`OperationAudit` 审计

---

#### API-POL-007 查询 v1/policies/menu-links

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-007` |
| 接口名称 | 查询 v1/policies/menu-links |
| 请求方式 | `GET` |
| URL | `/api/v1/policies/menu-links` |
| Gin 路径 | `/api/v1/policies/menu-links` |
| operationId | `get_api_v1_policies_menu-links` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/menu-links, GET)`；写操作：`OperationAudit` 审计

---

#### API-POL-008 查询 v1/policies/permission-tree

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-008` |
| 接口名称 | 查询 v1/policies/permission-tree |
| 请求方式 | `GET` |
| URL | `/api/v1/policies/permission-tree` |
| Gin 路径 | `/api/v1/policies/permission-tree` |
| operationId | `get_api_v1_policies_permission-tree` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/permission-tree, GET)`；写操作：`OperationAudit` 审计

---

#### API-POL-009 创建/提交 v1/policies/simulate

| 项 | 内容 |
|----|------|
| 接口编号 | `API-POL-009` |
| 接口名称 | 创建/提交 v1/policies/simulate |
| 请求方式 | `POST` |
| URL | `/api/v1/policies/simulate` |
| Gin 路径 | `/api/v1/policies/simulate` |
| operationId | `post_api_v1_policies_simulate` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/policies/simulate, POST)`；写操作：`OperationAudit` 审计

---

### 3.39 Projects

#### API-PRJ-001 查询 api/v1/projects

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-001` |
| 接口名称 | 查询 api/v1/projects |
| 请求方式 | `GET` |
| URL | `/api/v1/projects` |
| Gin 路径 | `/api/v1/projects` |
| operationId | `get_api_v1_projects` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects, GET)`；写操作：`OperationAudit` 审计

---

#### API-PRJ-002 创建/提交 api/v1/projects

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-002` |
| 接口名称 | 创建/提交 api/v1/projects |
| 请求方式 | `POST` |
| URL | `/api/v1/projects` |
| Gin 路径 | `/api/v1/projects` |
| operationId | `post_api_v1_projects` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects, POST)`；写操作：`OperationAudit` 审计

---

#### API-PRJ-003 删除 api/v1/projects

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-003` |
| 接口名称 | 删除 api/v1/projects |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}` |
| Gin 路径 | `/api/v1/projects/:id` |
| operationId | `delete_api_v1_projects_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-004 更新 api/v1/projects

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-004` |
| 接口名称 | 更新 api/v1/projects |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}` |
| Gin 路径 | `/api/v1/projects/:id` |
| operationId | `put_api_v1_projects_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-005 查询 v1/projects/cicd-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-005` |
| 接口名称 | 查询 v1/projects/cicd-access-grants |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd-access-grants` |
| Gin 路径 | `/api/v1/projects/:id/cicd-access-grants` |
| operationId | `get_api_v1_projects_id__cicd-access-grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd-access-grants, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-006 创建/提交 v1/projects/cicd-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-006` |
| 接口名称 | 创建/提交 v1/projects/cicd-access-grants |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd-access-grants` |
| Gin 路径 | `/api/v1/projects/:id/cicd-access-grants` |
| operationId | `post_api_v1_projects_id__cicd-access-grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd-access-grants, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-007 创建/提交 projects/cicd-access-grants/bootstrap

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-007` |
| 接口名称 | 创建/提交 projects/cicd-access-grants/bootstrap |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd-access-grants/bootstrap` |
| Gin 路径 | `/api/v1/projects/:id/cicd-access-grants/bootstrap` |
| operationId | `post_api_v1_projects_id__cicd-access-grants_bootstrap` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd-access-grants/bootstrap, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-008 创建/提交 projects/cicd-access-grants/bulk

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-008` |
| 接口名称 | 创建/提交 projects/cicd-access-grants/bulk |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd-access-grants/bulk` |
| Gin 路径 | `/api/v1/projects/:id/cicd-access-grants/bulk` |
| operationId | `post_api_v1_projects_id__cicd-access-grants_bulk` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd-access-grants/bulk, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-009 删除 v1/projects/cicd-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-009` |
| 接口名称 | 删除 v1/projects/cicd-access-grants |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cicd-access-grants/{grantId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd-access-grants/:grantId` |
| operationId | `delete_api_v1_projects_id__cicd-access-grants_grantId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `grantId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd-access-grants/:grantId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-010 查询 projects/cicd/approval-flow

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-010` |
| 接口名称 | 查询 projects/cicd/approval-flow |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/approval-flow` |
| Gin 路径 | `/api/v1/projects/:id/cicd/approval-flow` |
| operationId | `get_api_v1_projects_id__cicd_approval-flow` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/approval-flow, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-011 更新 projects/cicd/approval-flow

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-011` |
| 接口名称 | 更新 projects/cicd/approval-flow |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cicd/approval-flow` |
| Gin 路径 | `/api/v1/projects/:id/cicd/approval-flow` |
| operationId | `put_api_v1_projects_id__cicd_approval-flow` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/approval-flow, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-012 查询 projects/cicd/build-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-012` |
| 接口名称 | 查询 projects/cicd/build-runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/build-runs` |
| Gin 路径 | `/api/v1/projects/:id/cicd/build-runs` |
| operationId | `get_api_v1_projects_id__cicd_build-runs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/build-runs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-013 删除 projects/cicd/build-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-013` |
| 接口名称 | 删除 projects/cicd/build-runs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cicd/build-runs/{runId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/build-runs/:runId` |
| operationId | `delete_api_v1_projects_id__cicd_build-runs_runId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/build-runs/:runId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-014 查询 projects/cicd/build-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-014` |
| 接口名称 | 查询 projects/cicd/build-runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/build-runs/{runId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/build-runs/:runId` |
| operationId | `get_api_v1_projects_id__cicd_build-runs_runId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/build-runs/:runId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-015 查询 cicd/build-runs/log

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-015` |
| 接口名称 | 查询 cicd/build-runs/log |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/build-runs/{runId}/log` |
| Gin 路径 | `/api/v1/projects/:id/cicd/build-runs/:runId/log` |
| operationId | `get_api_v1_projects_id__cicd_build-runs_runId__log` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/build-runs/:runId/log, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-016 查询 projects/cicd/helm-scaffold

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-016` |
| 接口名称 | 查询 projects/cicd/helm-scaffold |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/helm-scaffold` |
| Gin 路径 | `/api/v1/projects/:id/cicd/helm-scaffold` |
| operationId | `get_api_v1_projects_id__cicd_helm-scaffold` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/helm-scaffold, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-017 查询 projects/cicd/release-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-017` |
| 接口名称 | 查询 projects/cicd/release-runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/release-runs` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs` |
| operationId | `get_api_v1_projects_id__cicd_release-runs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-018 创建/提交 cicd/release-runs/batch-approve

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-018` |
| 接口名称 | 创建/提交 cicd/release-runs/batch-approve |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/batch-approve` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/batch-approve` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_batch-approve` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/batch-approve, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-019 创建/提交 cicd/release-runs/batch-execute

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-019` |
| 接口名称 | 创建/提交 cicd/release-runs/batch-execute |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/batch-execute` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/batch-execute` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_batch-execute` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/batch-execute, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-020 创建/提交 cicd/release-runs/batch-reject

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-020` |
| 接口名称 | 创建/提交 cicd/release-runs/batch-reject |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/batch-reject` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/batch-reject` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_batch-reject` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/batch-reject, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-021 创建/提交 cicd/release-runs/batch-terminate

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-021` |
| 接口名称 | 创建/提交 cicd/release-runs/batch-terminate |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/batch-terminate` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/batch-terminate` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_batch-terminate` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/batch-terminate, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-022 删除 projects/cicd/release-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-022` |
| 接口名称 | 删除 projects/cicd/release-runs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId` |
| operationId | `delete_api_v1_projects_id__cicd_release-runs_runId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-023 查询 projects/cicd/release-runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-023` |
| 接口名称 | 查询 projects/cicd/release-runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId` |
| operationId | `get_api_v1_projects_id__cicd_release-runs_runId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-024 查询 cicd/release-runs/approval-steps

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-024` |
| 接口名称 | 查询 cicd/release-runs/approval-steps |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/approval-steps` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/approval-steps` |
| operationId | `get_api_v1_projects_id__cicd_release-runs_runId__approval-steps` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/approval-steps, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-025 创建/提交 cicd/release-runs/approve

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-025` |
| 接口名称 | 创建/提交 cicd/release-runs/approve |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/approve` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/approve` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_runId__approve` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/approve, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-026 创建/提交 cicd/release-runs/execute

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-026` |
| 接口名称 | 创建/提交 cicd/release-runs/execute |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/execute` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/execute` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_runId__execute` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/execute, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-027 查询 cicd/release-runs/log

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-027` |
| 接口名称 | 查询 cicd/release-runs/log |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/log` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/log` |
| operationId | `get_api_v1_projects_id__cicd_release-runs_runId__log` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/log, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-028 创建/提交 cicd/release-runs/reject

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-028` |
| 接口名称 | 创建/提交 cicd/release-runs/reject |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/reject` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/reject` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_runId__reject` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/reject, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-029 创建/提交 cicd/release-runs/terminate

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-029` |
| 接口名称 | 创建/提交 cicd/release-runs/terminate |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/terminate` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/terminate` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_runId__terminate` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/terminate, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-030 创建/提交 cicd/release-runs/verify

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-030` |
| 接口名称 | 创建/提交 cicd/release-runs/verify |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/release-runs/{runId}/verify` |
| Gin 路径 | `/api/v1/projects/:id/cicd/release-runs/:runId/verify` |
| operationId | `post_api_v1_projects_id__cicd_release-runs_runId__verify` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/release-runs/:runId/verify, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）；写操作：`OperationAudit` 审计

---

#### API-PRJ-031 查询 projects/cicd/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-031` |
| 接口名称 | 查询 projects/cicd/services |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services` |
| operationId | `get_api_v1_projects_id__cicd_services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-032 创建/提交 projects/cicd/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-032` |
| 接口名称 | 创建/提交 projects/cicd/services |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/services` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services` |
| operationId | `post_api_v1_projects_id__cicd_services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-033 删除 projects/cicd/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-033` |
| 接口名称 | 删除 projects/cicd/services |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId` |
| operationId | `delete_api_v1_projects_id__cicd_services_serviceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-034 查询 projects/cicd/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-034` |
| 接口名称 | 查询 projects/cicd/services |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId` |
| operationId | `get_api_v1_projects_id__cicd_services_serviceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-035 更新 projects/cicd/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-035` |
| 接口名称 | 更新 projects/cicd/services |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId` |
| operationId | `put_api_v1_projects_id__cicd_services_serviceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-036 查询 cicd/services/artifacts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-036` |
| 接口名称 | 查询 cicd/services/artifacts |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/artifacts` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/artifacts` |
| operationId | `get_api_v1_projects_id__cicd_services_serviceId__artifacts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/artifacts, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-037 创建/提交 cicd/services/builds

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-037` |
| 接口名称 | 创建/提交 cicd/services/builds |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/builds` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/builds` |
| operationId | `post_api_v1_projects_id__cicd_services_serviceId__builds` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/builds, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-038 查询 cicd/services/ci-config

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-038` |
| 接口名称 | 查询 cicd/services/ci-config |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/ci-config` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/ci-config` |
| operationId | `get_api_v1_projects_id__cicd_services_serviceId__ci-config` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/ci-config, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-039 更新 cicd/services/ci-config

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-039` |
| 接口名称 | 更新 cicd/services/ci-config |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/ci-config` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/ci-config` |
| operationId | `put_api_v1_projects_id__cicd_services_serviceId__ci-config` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/ci-config, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-040 查询 cicd/services/deploy-configs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-040` |
| 接口名称 | 查询 cicd/services/deploy-configs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/deploy-configs` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs` |
| operationId | `get_api_v1_projects_id__cicd_services_serviceId__deploy-configs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-041 创建/提交 cicd/services/deploy-configs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-041` |
| 接口名称 | 创建/提交 cicd/services/deploy-configs |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/deploy-configs` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs` |
| operationId | `post_api_v1_projects_id__cicd_services_serviceId__deploy-configs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-042 删除 cicd/services/deploy-configs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-042` |
| 接口名称 | 删除 cicd/services/deploy-configs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/deploy-configs/{configId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId` |
| operationId | `delete_api_v1_projects_id__cicd_services_serviceId__deploy-configs_configId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| path | `configId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-043 更新 cicd/services/deploy-configs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-043` |
| 接口名称 | 更新 cicd/services/deploy-configs |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/deploy-configs/{configId}` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId` |
| operationId | `put_api_v1_projects_id__cicd_services_serviceId__deploy-configs_configId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| path | `configId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-044 查询 cicd/services/helm-scaffold

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-044` |
| 接口名称 | 查询 cicd/services/helm-scaffold |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/helm-scaffold` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/helm-scaffold` |
| operationId | `get_api_v1_projects_id__cicd_services_serviceId__helm-scaffold` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/helm-scaffold, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-045 创建/提交 cicd/services/releases

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-045` |
| 接口名称 | 创建/提交 cicd/services/releases |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cicd/services/{serviceId}/releases` |
| Gin 路径 | `/api/v1/projects/:id/cicd/services/:serviceId/releases` |
| operationId | `post_api_v1_projects_id__cicd_services_serviceId__releases` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cicd/services/:serviceId/releases, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-046 查询 v1/projects/cloud-accounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-046` |
| 接口名称 | 查询 v1/projects/cloud-accounts |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/cloud-accounts` |
| Gin 路径 | `/api/v1/projects/:id/cloud-accounts` |
| operationId | `get_api_v1_projects_id__cloud-accounts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cloud-accounts, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-047 创建/提交 v1/projects/cloud-accounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-047` |
| 接口名称 | 创建/提交 v1/projects/cloud-accounts |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/cloud-accounts` |
| Gin 路径 | `/api/v1/projects/:id/cloud-accounts` |
| operationId | `post_api_v1_projects_id__cloud-accounts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cloud-accounts, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-048 删除 v1/projects/cloud-accounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-048` |
| 接口名称 | 删除 v1/projects/cloud-accounts |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/cloud-accounts/{accountId}` |
| Gin 路径 | `/api/v1/projects/:id/cloud-accounts/:accountId` |
| operationId | `delete_api_v1_projects_id__cloud-accounts_accountId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `accountId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cloud-accounts/:accountId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-049 更新 v1/projects/cloud-accounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-049` |
| 接口名称 | 更新 v1/projects/cloud-accounts |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cloud-accounts/{accountId}` |
| Gin 路径 | `/api/v1/projects/:id/cloud-accounts/:accountId` |
| operationId | `put_api_v1_projects_id__cloud-accounts_accountId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `accountId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cloud-accounts/:accountId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-050 更新 projects/cloud-accounts/sync

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-050` |
| 接口名称 | 更新 projects/cloud-accounts/sync |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/cloud-accounts/{accountId}/sync` |
| Gin 路径 | `/api/v1/projects/:id/cloud-accounts/:accountId/sync` |
| operationId | `put_api_v1_projects_id__cloud-accounts_accountId__sync` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `accountId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/cloud-accounts/:accountId/sync, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-051 查询 projects/dbmgmt/access-requests

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-051` |
| 接口名称 | 查询 projects/dbmgmt/access-requests |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/access-requests` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/access-requests` |
| operationId | `get_api_v1_projects_id__dbmgmt_access-requests` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/access-requests, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-052 创建/提交 projects/dbmgmt/access-requests

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-052` |
| 接口名称 | 创建/提交 projects/dbmgmt/access-requests |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/access-requests` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/access-requests` |
| operationId | `post_api_v1_projects_id__dbmgmt_access-requests` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/access-requests, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-053 创建/提交 dbmgmt/access-requests/approve

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-053` |
| 接口名称 | 创建/提交 dbmgmt/access-requests/approve |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/access-requests/{requestId}/approve` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/access-requests/:requestId/approve` |
| operationId | `post_api_v1_projects_id__dbmgmt_access-requests_requestId__approve` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `requestId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/access-requests/:requestId/approve, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-054 创建/提交 dbmgmt/access-requests/reject

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-054` |
| 接口名称 | 创建/提交 dbmgmt/access-requests/reject |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/access-requests/{requestId}/reject` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/access-requests/:requestId/reject` |
| operationId | `post_api_v1_projects_id__dbmgmt_access-requests_requestId__reject` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `requestId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/access-requests/:requestId/reject, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-055 查询 projects/dbmgmt/app-user-requests

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-055` |
| 接口名称 | 查询 projects/dbmgmt/app-user-requests |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/app-user-requests` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/app-user-requests` |
| operationId | `get_api_v1_projects_id__dbmgmt_app-user-requests` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/app-user-requests, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-056 创建/提交 projects/dbmgmt/app-user-requests

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-056` |
| 接口名称 | 创建/提交 projects/dbmgmt/app-user-requests |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/app-user-requests` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/app-user-requests` |
| operationId | `post_api_v1_projects_id__dbmgmt_app-user-requests` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/app-user-requests, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-057 创建/提交 dbmgmt/app-user-requests/approve

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-057` |
| 接口名称 | 创建/提交 dbmgmt/app-user-requests/approve |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/app-user-requests/{requestId}/approve` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/approve` |
| operationId | `post_api_v1_projects_id__dbmgmt_app-user-requests_requestId__approve` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `requestId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/approve, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-058 创建/提交 dbmgmt/app-user-requests/reject

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-058` |
| 接口名称 | 创建/提交 dbmgmt/app-user-requests/reject |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/app-user-requests/{requestId}/reject` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/reject` |
| operationId | `post_api_v1_projects_id__dbmgmt_app-user-requests_requestId__reject` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `requestId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/reject, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-059 查询 projects/dbmgmt/approval-flow

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-059` |
| 接口名称 | 查询 projects/dbmgmt/approval-flow |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/approval-flow` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/approval-flow` |
| operationId | `get_api_v1_projects_id__dbmgmt_approval-flow` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/approval-flow, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-060 更新 projects/dbmgmt/approval-flow

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-060` |
| 接口名称 | 更新 projects/dbmgmt/approval-flow |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/dbmgmt/approval-flow` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/approval-flow` |
| operationId | `put_api_v1_projects_id__dbmgmt_approval-flow` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/approval-flow, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-061 查询 projects/dbmgmt/audit-logs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-061` |
| 接口名称 | 查询 projects/dbmgmt/audit-logs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/audit-logs` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/audit-logs` |
| operationId | `get_api_v1_projects_id__dbmgmt_audit-logs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/audit-logs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-062 查询 projects/dbmgmt/executions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-062` |
| 接口名称 | 查询 projects/dbmgmt/executions |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/executions` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/executions` |
| operationId | `get_api_v1_projects_id__dbmgmt_executions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/executions, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-063 查询 projects/dbmgmt/grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-063` |
| 接口名称 | 查询 projects/dbmgmt/grants |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/grants` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/grants` |
| operationId | `get_api_v1_projects_id__dbmgmt_grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/grants, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-064 创建/提交 projects/dbmgmt/grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-064` |
| 接口名称 | 创建/提交 projects/dbmgmt/grants |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/grants` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/grants` |
| operationId | `post_api_v1_projects_id__dbmgmt_grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/grants, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-065 查询 dbmgmt/grants/effective

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-065` |
| 接口名称 | 查询 dbmgmt/grants/effective |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/grants/effective` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/grants/effective` |
| operationId | `get_api_v1_projects_id__dbmgmt_grants_effective` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/grants/effective, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-066 删除 projects/dbmgmt/grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-066` |
| 接口名称 | 删除 projects/dbmgmt/grants |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/dbmgmt/grants/{grantId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/grants/:grantId` |
| operationId | `delete_api_v1_projects_id__dbmgmt_grants_grantId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `grantId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/grants/:grantId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-067 更新 projects/dbmgmt/grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-067` |
| 接口名称 | 更新 projects/dbmgmt/grants |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/dbmgmt/grants/{grantId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/grants/:grantId` |
| operationId | `put_api_v1_projects_id__dbmgmt_grants_grantId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `grantId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/grants/:grantId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-068 查询 projects/dbmgmt/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-068` |
| 接口名称 | 查询 projects/dbmgmt/instances |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-069 创建/提交 projects/dbmgmt/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-069` |
| 接口名称 | 创建/提交 projects/dbmgmt/instances |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-070 删除 projects/dbmgmt/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-070` |
| 接口名称 | 删除 projects/dbmgmt/instances |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId` |
| operationId | `delete_api_v1_projects_id__dbmgmt_instances_instanceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-071 查询 projects/dbmgmt/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-071` |
| 接口名称 | 查询 projects/dbmgmt/instances |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-072 更新 projects/dbmgmt/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-072` |
| 接口名称 | 更新 projects/dbmgmt/instances |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId` |
| operationId | `put_api_v1_projects_id__dbmgmt_instances_instanceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-073 查询 instances/accounts/password

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-073` |
| 接口名称 | 查询 instances/accounts/password |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/accounts/{accountId}/password` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/accounts/:accountId/password` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__accounts_accountId__password` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| path | `accountId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/accounts/:accountId/password, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-074 创建/提交 dbmgmt/instances/check

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-074` |
| 接口名称 | 创建/提交 dbmgmt/instances/check |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/check` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/check` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances_instanceId__check` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/check, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-075 创建/提交 dbmgmt/instances/execute

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-075` |
| 接口名称 | 创建/提交 dbmgmt/instances/execute |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/execute` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/execute` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances_instanceId__execute` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/execute, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-076 创建/提交 dbmgmt/instances/import

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-076` |
| 接口名称 | 创建/提交 dbmgmt/instances/import |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/import` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/import` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances_instanceId__import` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/import, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-077 查询 instances/metadata/columns

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-077` |
| 接口名称 | 查询 instances/metadata/columns |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/metadata/columns` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/columns` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__metadata_columns` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/columns, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-078 查询 instances/metadata/databases

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-078` |
| 接口名称 | 查询 instances/metadata/databases |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/metadata/databases` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/databases` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__metadata_databases` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/databases, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-079 查询 instances/metadata/tables

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-079` |
| 接口名称 | 查询 instances/metadata/tables |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/metadata/tables` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/tables` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__metadata_tables` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/tables, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-080 查询 dbmgmt/instances/mysql-user-privileges

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-080` |
| 接口名称 | 查询 dbmgmt/instances/mysql-user-privileges |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/mysql-user-privileges` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-user-privileges` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__mysql-user-privileges` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-user-privileges, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-081 查询 dbmgmt/instances/mysql-users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-081` |
| 接口名称 | 查询 dbmgmt/instances/mysql-users |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/mysql-users` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-users` |
| operationId | `get_api_v1_projects_id__dbmgmt_instances_instanceId__mysql-users` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-users, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-082 创建/提交 dbmgmt/instances/ping

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-082` |
| 接口名称 | 创建/提交 dbmgmt/instances/ping |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/ping` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/ping` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances_instanceId__ping` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/ping, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-083 创建/提交 dbmgmt/instances/query

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-083` |
| 接口名称 | 创建/提交 dbmgmt/instances/query |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/instances/{instanceId}/query` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/instances/:instanceId/query` |
| operationId | `post_api_v1_projects_id__dbmgmt_instances_instanceId__query` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/instances/:instanceId/query, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-084 查询 projects/dbmgmt/tickets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-084` |
| 接口名称 | 查询 projects/dbmgmt/tickets |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-085 查询 projects/dbmgmt/tickets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-085` |
| 接口名称 | 查询 projects/dbmgmt/tickets |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-086 创建/提交 dbmgmt/tickets/approve

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-086` |
| 接口名称 | 创建/提交 dbmgmt/tickets/approve |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/approve` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/approve` |
| operationId | `post_api_v1_projects_id__dbmgmt_tickets_ticketId__approve` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/approve, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-087 创建/提交 dbmgmt/tickets/execute

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-087` |
| 接口名称 | 创建/提交 dbmgmt/tickets/execute |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/execute` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/execute` |
| operationId | `post_api_v1_projects_id__dbmgmt_tickets_ticketId__execute` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/execute, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-088 查询 dbmgmt/tickets/osc

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-088` |
| 接口名称 | 查询 dbmgmt/tickets/osc |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/osc` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId__osc` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-089 查询 dbmgmt/tickets/osc

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-089` |
| 接口名称 | 查询 dbmgmt/tickets/osc |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/osc/{sqlsha1}` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId__osc_sqlsha1_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| path | `sqlsha1` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-090 创建/提交 tickets/osc/control

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-090` |
| 接口名称 | 创建/提交 tickets/osc/control |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/osc/{sqlsha1}/control` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1/control` |
| operationId | `post_api_v1_projects_id__dbmgmt_tickets_ticketId__osc_sqlsha1__control` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| path | `sqlsha1` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1/control, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-091 创建/提交 dbmgmt/tickets/reject

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-091` |
| 接口名称 | 创建/提交 dbmgmt/tickets/reject |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/reject` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/reject` |
| operationId | `post_api_v1_projects_id__dbmgmt_tickets_ticketId__reject` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/reject, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-092 查询 dbmgmt/tickets/rollback

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-092` |
| 接口名称 | 查询 dbmgmt/tickets/rollback |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/rollback` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId__rollback` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-093 查询 tickets/rollback/preview

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-093` |
| 接口名称 | 查询 tickets/rollback/preview |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/rollback/preview` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/preview` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId__rollback_preview` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/preview, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-094 创建/提交 tickets/rollback/submit

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-094` |
| 接口名称 | 创建/提交 tickets/rollback/submit |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/rollback/submit` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/submit` |
| operationId | `post_api_v1_projects_id__dbmgmt_tickets_ticketId__rollback_submit` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/submit, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-095 查询 dbmgmt/tickets/steps

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-095` |
| 接口名称 | 查询 dbmgmt/tickets/steps |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/dbmgmt/tickets/{ticketId}/steps` |
| Gin 路径 | `/api/v1/projects/:id/dbmgmt/tickets/:ticketId/steps` |
| operationId | `get_api_v1_projects_id__dbmgmt_tickets_ticketId__steps` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `ticketId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/dbmgmt/tickets/:ticketId/steps, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）；写操作：`OperationAudit` 审计

---

#### API-PRJ-096 查询 projects/inspect/items

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-096` |
| 接口名称 | 查询 projects/inspect/items |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/items` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items` |
| operationId | `get_api_v1_projects_id__inspect_items` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-097 创建/提交 projects/inspect/items

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-097` |
| 接口名称 | 创建/提交 projects/inspect/items |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/items` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items` |
| operationId | `post_api_v1_projects_id__inspect_items` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-098 创建/提交 inspect/items/reset-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-098` |
| 接口名称 | 创建/提交 inspect/items/reset-template |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/items/reset-template` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items/reset-template` |
| operationId | `post_api_v1_projects_id__inspect_items_reset-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items/reset-template, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-099 创建/提交 inspect/items/sync-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-099` |
| 接口名称 | 创建/提交 inspect/items/sync-template |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/items/sync-template` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items/sync-template` |
| operationId | `post_api_v1_projects_id__inspect_items_sync-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items/sync-template, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-100 删除 projects/inspect/items

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-100` |
| 接口名称 | 删除 projects/inspect/items |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/inspect/items/{itemId}` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items/:itemId` |
| operationId | `delete_api_v1_projects_id__inspect_items_itemId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `itemId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items/:itemId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-101 更新 projects/inspect/items

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-101` |
| 接口名称 | 更新 projects/inspect/items |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/inspect/items/{itemId}` |
| Gin 路径 | `/api/v1/projects/:id/inspect/items/:itemId` |
| operationId | `put_api_v1_projects_id__inspect_items_itemId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `itemId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/items/:itemId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-102 查询 projects/inspect/plan

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-102` |
| 接口名称 | 查询 projects/inspect/plan |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/plan` |
| Gin 路径 | `/api/v1/projects/:id/inspect/plan` |
| operationId | `get_api_v1_projects_id__inspect_plan` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/plan, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-103 更新 projects/inspect/plan

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-103` |
| 接口名称 | 更新 projects/inspect/plan |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/inspect/plan` |
| Gin 路径 | `/api/v1/projects/:id/inspect/plan` |
| operationId | `put_api_v1_projects_id__inspect_plan` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/plan, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-104 查询 projects/inspect/report-templates

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-104` |
| 接口名称 | 查询 projects/inspect/report-templates |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/report-templates` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates` |
| operationId | `get_api_v1_projects_id__inspect_report-templates` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-105 创建/提交 projects/inspect/report-templates

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-105` |
| 接口名称 | 创建/提交 projects/inspect/report-templates |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/report-templates` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates` |
| operationId | `post_api_v1_projects_id__inspect_report-templates` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-106 创建/提交 inspect/report-templates/copy

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-106` |
| 接口名称 | 创建/提交 inspect/report-templates/copy |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/report-templates/copy` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates/copy` |
| operationId | `post_api_v1_projects_id__inspect_report-templates_copy` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates/copy, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-107 创建/提交 inspect/report-templates/preview

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-107` |
| 接口名称 | 创建/提交 inspect/report-templates/preview |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/report-templates/preview` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates/preview` |
| operationId | `post_api_v1_projects_id__inspect_report-templates_preview` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates/preview, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-108 删除 projects/inspect/report-templates

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-108` |
| 接口名称 | 删除 projects/inspect/report-templates |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/inspect/report-templates/{templateId}` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates/:templateId` |
| operationId | `delete_api_v1_projects_id__inspect_report-templates_templateId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `templateId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates/:templateId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-109 更新 projects/inspect/report-templates

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-109` |
| 接口名称 | 更新 projects/inspect/report-templates |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/inspect/report-templates/{templateId}` |
| Gin 路径 | `/api/v1/projects/:id/inspect/report-templates/:templateId` |
| operationId | `put_api_v1_projects_id__inspect_report-templates_templateId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `templateId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/report-templates/:templateId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-110 查询 projects/inspect/runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-110` |
| 接口名称 | 查询 projects/inspect/runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs` |
| operationId | `get_api_v1_projects_id__inspect_runs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-111 创建/提交 projects/inspect/runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-111` |
| 接口名称 | 创建/提交 projects/inspect/runs |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/runs` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs` |
| operationId | `post_api_v1_projects_id__inspect_runs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-112 查询 projects/inspect/runs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-112` |
| 接口名称 | 查询 projects/inspect/runs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId` |
| operationId | `get_api_v1_projects_id__inspect_runs_runId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-113 查询 inspect/runs/report.html

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-113` |
| 接口名称 | 查询 inspect/runs/report.html |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}/report.html` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId/report.html` |
| operationId | `get_api_v1_projects_id__inspect_runs_runId__report.html` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId/report.html, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-114 查询 inspect/runs/report.pdf

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-114` |
| 接口名称 | 查询 inspect/runs/report.pdf |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}/report.pdf` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId/report.pdf` |
| operationId | `get_api_v1_projects_id__inspect_runs_runId__report.pdf` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId/report.pdf, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-115 查询 inspect/runs/report.print.html

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-115` |
| 接口名称 | 查询 inspect/runs/report.print.html |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}/report.print.html` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId/report.print.html` |
| operationId | `get_api_v1_projects_id__inspect_runs_runId__report.print.html` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId/report.print.html, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-116 查询 inspect/runs/report.xlsx

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-116` |
| 接口名称 | 查询 inspect/runs/report.xlsx |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}/report.xlsx` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId/report.xlsx` |
| operationId | `get_api_v1_projects_id__inspect_runs_runId__report.xlsx` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId/report.xlsx, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-117 创建/提交 inspect/runs/resend-email

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-117` |
| 接口名称 | 创建/提交 inspect/runs/resend-email |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/inspect/runs/{runId}/resend-email` |
| Gin 路径 | `/api/v1/projects/:id/inspect/runs/:runId/resend-email` |
| operationId | `post_api_v1_projects_id__inspect_runs_runId__resend-email` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `runId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/inspect/runs/:runId/resend-email, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-118 删除 v1/projects/log-retention

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-118` |
| 接口名称 | 删除 v1/projects/log-retention |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/log-retention` |
| Gin 路径 | `/api/v1/projects/:id/log-retention` |
| operationId | `delete_api_v1_projects_id__log-retention` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-retention, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-119 查询 v1/projects/log-retention

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-119` |
| 接口名称 | 查询 v1/projects/log-retention |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/log-retention` |
| Gin 路径 | `/api/v1/projects/:id/log-retention` |
| operationId | `get_api_v1_projects_id__log-retention` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-retention, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-120 更新 v1/projects/log-retention

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-120` |
| 接口名称 | 更新 v1/projects/log-retention |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/log-retention` |
| Gin 路径 | `/api/v1/projects/:id/log-retention` |
| operationId | `put_api_v1_projects_id__log-retention` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-retention, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-121 查询 v1/projects/log-sources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-121` |
| 接口名称 | 查询 v1/projects/log-sources |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/log-sources` |
| Gin 路径 | `/api/v1/projects/:id/log-sources` |
| operationId | `get_api_v1_projects_id__log-sources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-sources, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-122 创建/提交 v1/projects/log-sources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-122` |
| 接口名称 | 创建/提交 v1/projects/log-sources |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/log-sources` |
| Gin 路径 | `/api/v1/projects/:id/log-sources` |
| operationId | `post_api_v1_projects_id__log-sources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-sources, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-123 删除 v1/projects/log-sources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-123` |
| 接口名称 | 删除 v1/projects/log-sources |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/log-sources/{logSourceId}` |
| Gin 路径 | `/api/v1/projects/:id/log-sources/:logSourceId` |
| operationId | `delete_api_v1_projects_id__log-sources_logSourceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `logSourceId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/log-sources/:logSourceId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-124 创建/提交 projects/loggie/bootstrap

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-124` |
| 接口名称 | 创建/提交 projects/loggie/bootstrap |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/bootstrap` |
| Gin 路径 | `/api/v1/projects/:id/loggie/bootstrap` |
| operationId | `post_api_v1_projects_id__loggie_bootstrap` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/bootstrap, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-125 查询 projects/loggie/bootstrap-sources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-125` |
| 接口名称 | 查询 projects/loggie/bootstrap-sources |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/loggie/bootstrap-sources` |
| Gin 路径 | `/api/v1/projects/:id/loggie/bootstrap-sources` |
| operationId | `get_api_v1_projects_id__loggie_bootstrap-sources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/bootstrap-sources, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-126 创建/提交 projects/loggie/deploy

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-126` |
| 接口名称 | 创建/提交 projects/loggie/deploy |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/deploy` |
| Gin 路径 | `/api/v1/projects/:id/loggie/deploy` |
| operationId | `post_api_v1_projects_id__loggie_deploy` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/deploy, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-127 创建/提交 projects/loggie/install

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-127` |
| 接口名称 | 创建/提交 projects/loggie/install |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/install` |
| Gin 路径 | `/api/v1/projects/:id/loggie/install` |
| operationId | `post_api_v1_projects_id__loggie_install` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/install, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-128 查询 loggie/pipeline/download

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-128` |
| 接口名称 | 查询 loggie/pipeline/download |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/loggie/pipeline/download` |
| Gin 路径 | `/api/v1/projects/:id/loggie/pipeline/download` |
| operationId | `get_api_v1_projects_id__loggie_pipeline_download` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/pipeline/download, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-129 创建/提交 projects/loggie/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-129` |
| 接口名称 | 创建/提交 projects/loggie/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/restart` |
| Gin 路径 | `/api/v1/projects/:id/loggie/restart` |
| operationId | `post_api_v1_projects_id__loggie_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/restart, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-130 创建/提交 projects/loggie/start

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-130` |
| 接口名称 | 创建/提交 projects/loggie/start |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/start` |
| Gin 路径 | `/api/v1/projects/:id/loggie/start` |
| operationId | `post_api_v1_projects_id__loggie_start` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/start, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-131 查询 projects/loggie/status

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-131` |
| 接口名称 | 查询 projects/loggie/status |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/loggie/status` |
| Gin 路径 | `/api/v1/projects/:id/loggie/status` |
| operationId | `get_api_v1_projects_id__loggie_status` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/status, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-132 创建/提交 projects/loggie/stop

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-132` |
| 接口名称 | 创建/提交 projects/loggie/stop |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/stop` |
| Gin 路径 | `/api/v1/projects/:id/loggie/stop` |
| operationId | `post_api_v1_projects_id__loggie_stop` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/stop, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-133 创建/提交 projects/loggie/sync

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-133` |
| 接口名称 | 创建/提交 projects/loggie/sync |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/sync` |
| Gin 路径 | `/api/v1/projects/:id/loggie/sync` |
| operationId | `post_api_v1_projects_id__loggie_sync` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/sync, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-134 创建/提交 projects/loggie/uninstall

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-134` |
| 接口名称 | 创建/提交 projects/loggie/uninstall |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/loggie/uninstall` |
| Gin 路径 | `/api/v1/projects/:id/loggie/uninstall` |
| operationId | `post_api_v1_projects_id__loggie_uninstall` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/loggie/uninstall, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-135 查询 projects/logs/export

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-135` |
| 接口名称 | 查询 projects/logs/export |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/logs/export` |
| Gin 路径 | `/api/v1/projects/:id/logs/export` |
| operationId | `get_api_v1_projects_id__logs_export` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/logs/export, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-136 查询 projects/logs/search

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-136` |
| 接口名称 | 查询 projects/logs/search |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/logs/search` |
| Gin 路径 | `/api/v1/projects/:id/logs/search` |
| operationId | `get_api_v1_projects_id__logs_search` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/logs/search, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-137 查询 v1/projects/members

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-137` |
| 接口名称 | 查询 v1/projects/members |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/members` |
| Gin 路径 | `/api/v1/projects/:id/members` |
| operationId | `get_api_v1_projects_id__members` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/members, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-138 创建/提交 v1/projects/members

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-138` |
| 接口名称 | 创建/提交 v1/projects/members |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/members` |
| Gin 路径 | `/api/v1/projects/:id/members` |
| operationId | `post_api_v1_projects_id__members` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/members, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-139 删除 v1/projects/members

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-139` |
| 接口名称 | 删除 v1/projects/members |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/members/{memberId}` |
| Gin 路径 | `/api/v1/projects/:id/members/:memberId` |
| operationId | `delete_api_v1_projects_id__members_memberId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `memberId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/members/:memberId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-140 更新 v1/projects/members

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-140` |
| 接口名称 | 更新 v1/projects/members |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/members/{memberId}` |
| Gin 路径 | `/api/v1/projects/:id/members/:memberId` |
| operationId | `put_api_v1_projects_id__members_memberId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `memberId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/members/:memberId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-141 查询 projects/mysql-backup/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-141` |
| 接口名称 | 查询 projects/mysql-backup/instances |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances` |
| operationId | `get_api_v1_projects_id__mysql-backup_instances` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-142 创建/提交 projects/mysql-backup/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-142` |
| 接口名称 | 创建/提交 projects/mysql-backup/instances |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances` |
| operationId | `post_api_v1_projects_id__mysql-backup_instances` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-143 删除 projects/mysql-backup/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-143` |
| 接口名称 | 删除 projects/mysql-backup/instances |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances/{instanceId}` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances/:instanceId` |
| operationId | `delete_api_v1_projects_id__mysql-backup_instances_instanceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances/:instanceId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-144 更新 projects/mysql-backup/instances

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-144` |
| 接口名称 | 更新 projects/mysql-backup/instances |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances/{instanceId}` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances/:instanceId` |
| operationId | `put_api_v1_projects_id__mysql-backup_instances_instanceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances/:instanceId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-145 创建/提交 mysql-backup/instances/check-remote

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-145` |
| 接口名称 | 创建/提交 mysql-backup/instances/check-remote |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances/{instanceId}/check-remote` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances/:instanceId/check-remote` |
| operationId | `post_api_v1_projects_id__mysql-backup_instances_instanceId__check-remote` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances/:instanceId/check-remote, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-146 创建/提交 mysql-backup/instances/ping

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-146` |
| 接口名称 | 创建/提交 mysql-backup/instances/ping |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances/{instanceId}/ping` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances/:instanceId/ping` |
| operationId | `post_api_v1_projects_id__mysql-backup_instances_instanceId__ping` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances/:instanceId/ping, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-147 创建/提交 mysql-backup/instances/run

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-147` |
| 接口名称 | 创建/提交 mysql-backup/instances/run |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/mysql-backup/instances/{instanceId}/run` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/instances/:instanceId/run` |
| operationId | `post_api_v1_projects_id__mysql-backup_instances_instanceId__run` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `instanceId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/instances/:instanceId/run, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-148 查询 projects/mysql-backup/jobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-148` |
| 接口名称 | 查询 projects/mysql-backup/jobs |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/mysql-backup/jobs` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/jobs` |
| operationId | `get_api_v1_projects_id__mysql-backup_jobs` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/jobs, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-149 删除 projects/mysql-backup/jobs

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-149` |
| 接口名称 | 删除 projects/mysql-backup/jobs |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/mysql-backup/jobs/{jobId}` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/jobs/:jobId` |
| operationId | `delete_api_v1_projects_id__mysql-backup_jobs_jobId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `jobId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/jobs/:jobId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-150 查询 mysql-backup/jobs/presign

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-150` |
| 接口名称 | 查询 mysql-backup/jobs/presign |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/mysql-backup/jobs/{jobId}/presign` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/jobs/:jobId/presign` |
| operationId | `get_api_v1_projects_id__mysql-backup_jobs_jobId__presign` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `jobId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/jobs/:jobId/presign, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-151 创建/提交 mysql-backup/jobs/stop

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-151` |
| 接口名称 | 创建/提交 mysql-backup/jobs/stop |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/mysql-backup/jobs/{jobId}/stop` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/jobs/:jobId/stop` |
| operationId | `post_api_v1_projects_id__mysql-backup_jobs_jobId__stop` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `jobId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/jobs/:jobId/stop, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-152 查询 projects/mysql-backup/mysqldump-options

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-152` |
| 接口名称 | 查询 projects/mysql-backup/mysqldump-options |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/mysql-backup/mysqldump-options` |
| Gin 路径 | `/api/v1/projects/:id/mysql-backup/mysqldump-options` |
| operationId | `get_api_v1_projects_id__mysql-backup_mysqldump-options` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/mysql-backup/mysqldump-options, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-153 查询 v1/projects/server-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-153` |
| 接口名称 | 查询 v1/projects/server-access-grants |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/server-access-grants` |
| Gin 路径 | `/api/v1/projects/:id/server-access-grants` |
| operationId | `get_api_v1_projects_id__server-access-grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-access-grants, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-154 创建/提交 v1/projects/server-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-154` |
| 接口名称 | 创建/提交 v1/projects/server-access-grants |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/server-access-grants` |
| Gin 路径 | `/api/v1/projects/:id/server-access-grants` |
| operationId | `post_api_v1_projects_id__server-access-grants` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-access-grants, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-155 创建/提交 projects/server-access-grants/bootstrap

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-155` |
| 接口名称 | 创建/提交 projects/server-access-grants/bootstrap |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/server-access-grants/bootstrap` |
| Gin 路径 | `/api/v1/projects/:id/server-access-grants/bootstrap` |
| operationId | `post_api_v1_projects_id__server-access-grants_bootstrap` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-access-grants/bootstrap, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-156 创建/提交 projects/server-access-grants/bulk

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-156` |
| 接口名称 | 创建/提交 projects/server-access-grants/bulk |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/server-access-grants/bulk` |
| Gin 路径 | `/api/v1/projects/:id/server-access-grants/bulk` |
| operationId | `post_api_v1_projects_id__server-access-grants_bulk` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-access-grants/bulk, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-157 删除 v1/projects/server-access-grants

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-157` |
| 接口名称 | 删除 v1/projects/server-access-grants |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/server-access-grants/{grantId}` |
| Gin 路径 | `/api/v1/projects/:id/server-access-grants/:grantId` |
| operationId | `delete_api_v1_projects_id__server-access-grants_grantId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `grantId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-access-grants/:grantId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-158 创建/提交 v1/projects/server-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-158` |
| 接口名称 | 创建/提交 v1/projects/server-groups |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/server-groups` |
| Gin 路径 | `/api/v1/projects/:id/server-groups` |
| operationId | `post_api_v1_projects_id__server-groups` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-groups, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-159 查询 projects/server-groups/tree

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-159` |
| 接口名称 | 查询 projects/server-groups/tree |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/server-groups/tree` |
| Gin 路径 | `/api/v1/projects/:id/server-groups/tree` |
| operationId | `get_api_v1_projects_id__server-groups_tree` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-groups/tree, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-160 删除 v1/projects/server-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-160` |
| 接口名称 | 删除 v1/projects/server-groups |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/server-groups/{groupId}` |
| Gin 路径 | `/api/v1/projects/:id/server-groups/:groupId` |
| operationId | `delete_api_v1_projects_id__server-groups_groupId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `groupId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-groups/:groupId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-161 更新 v1/projects/server-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-161` |
| 接口名称 | 更新 v1/projects/server-groups |
| 请求方式 | `PUT` |
| URL | `/api/v1/projects/{id}/server-groups/{groupId}` |
| Gin 路径 | `/api/v1/projects/:id/server-groups/:groupId` |
| operationId | `put_api_v1_projects_id__server-groups_groupId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `groupId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/server-groups/:groupId, PUT)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-162 查询 v1/projects/servers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-162` |
| 接口名称 | 查询 v1/projects/servers |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers` |
| Gin 路径 | `/api/v1/projects/:id/servers` |
| operationId | `get_api_v1_projects_id__servers` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-163 创建/提交 v1/projects/servers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-163` |
| 接口名称 | 创建/提交 v1/projects/servers |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers` |
| Gin 路径 | `/api/v1/projects/:id/servers` |
| operationId | `post_api_v1_projects_id__servers` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-164 查询 projects/servers/export

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-164` |
| 接口名称 | 查询 projects/servers/export |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers/export` |
| Gin 路径 | `/api/v1/projects/:id/servers/export` |
| operationId | `get_api_v1_projects_id__servers_export` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/export, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-165 创建/提交 projects/servers/import

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-165` |
| 接口名称 | 创建/提交 projects/servers/import |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/import` |
| Gin 路径 | `/api/v1/projects/:id/servers/import` |
| operationId | `post_api_v1_projects_id__servers_import` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/import, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-166 查询 projects/servers/import-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-166` |
| 接口名称 | 查询 projects/servers/import-template |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers/import-template` |
| Gin 路径 | `/api/v1/projects/:id/servers/import-template` |
| operationId | `get_api_v1_projects_id__servers_import-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/import-template, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-167 创建/提交 projects/servers/sync

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-167` |
| 接口名称 | 创建/提交 projects/servers/sync |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/sync` |
| Gin 路径 | `/api/v1/projects/:id/servers/sync` |
| operationId | `post_api_v1_projects_id__servers_sync` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/sync, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-168 创建/提交 projects/servers/test

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-168` |
| 接口名称 | 创建/提交 projects/servers/test |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/test` |
| Gin 路径 | `/api/v1/projects/:id/servers/test` |
| operationId | `post_api_v1_projects_id__servers_test` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/test, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-169 创建/提交 servers/test/batch

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-169` |
| 接口名称 | 创建/提交 servers/test/batch |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/test/batch` |
| Gin 路径 | `/api/v1/projects/:id/servers/test/batch` |
| operationId | `post_api_v1_projects_id__servers_test_batch` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/test/batch, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-170 删除 v1/projects/servers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-170` |
| 接口名称 | 删除 v1/projects/servers |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/servers/{serverId}` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId` |
| operationId | `delete_api_v1_projects_id__servers_serverId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-171 查询 v1/projects/servers

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-171` |
| 接口名称 | 查询 v1/projects/servers |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers/{serverId}` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId` |
| operationId | `get_api_v1_projects_id__servers_serverId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-172 创建/提交 projects/servers/cloud-actions

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-172` |
| 接口名称 | 创建/提交 projects/servers/cloud-actions |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/{serverId}/cloud-actions` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId/cloud-actions` |
| operationId | `post_api_v1_projects_id__servers_serverId__cloud-actions` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId/cloud-actions, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-173 创建/提交 projects/servers/exec

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-173` |
| 接口名称 | 创建/提交 projects/servers/exec |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/servers/{serverId}/exec` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId/exec` |
| operationId | `post_api_v1_projects_id__servers_serverId__exec` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId/exec, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；资源 ACL：服务器 `can_exec`（`server_access_grants`）；写操作：`OperationAudit` 审计

---

#### API-PRJ-174 查询 projects/servers/my-access

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-174` |
| 接口名称 | 查询 projects/servers/my-access |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers/{serverId}/my-access` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId/my-access` |
| operationId | `get_api_v1_projects_id__servers_serverId__my-access` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId/my-access, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-175 查询 servers/terminal/ws

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-175` |
| 接口名称 | 查询 servers/terminal/ws |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/servers/{serverId}/terminal/ws` |
| Gin 路径 | `/api/v1/projects/:id/servers/:serverId/terminal/ws` |
| operationId | `get_api_v1_projects_id__servers_serverId__terminal_ws` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serverId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/servers/:serverId/terminal/ws, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；WebSocket：先 `POST /api/v1/auth/ws-ticket`，连接 URL 携带一次性 `ticket`；资源 ACL：服务器 `can_exec`（`server_access_grants`）；写操作：`OperationAudit` 审计

---

#### API-PRJ-176 查询 v1/projects/service-catalog

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-176` |
| 接口名称 | 查询 v1/projects/service-catalog |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/service-catalog` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog` |
| operationId | `get_api_v1_projects_id__service-catalog` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-177 创建/提交 v1/projects/service-catalog

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-177` |
| 接口名称 | 创建/提交 v1/projects/service-catalog |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/service-catalog` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog` |
| operationId | `post_api_v1_projects_id__service-catalog` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-178 删除 v1/projects/service-catalog

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-178` |
| 接口名称 | 删除 v1/projects/service-catalog |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/service-catalog/{catalogId}` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog/:catalogId` |
| operationId | `delete_api_v1_projects_id__service-catalog_catalogId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `catalogId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog/:catalogId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-179 查询 v1/projects/service-catalog

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-179` |
| 接口名称 | 查询 v1/projects/service-catalog |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/service-catalog/{catalogId}` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog/:catalogId` |
| operationId | `get_api_v1_projects_id__service-catalog_catalogId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `catalogId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog/:catalogId, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-180 创建/提交 projects/service-catalog/links

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-180` |
| 接口名称 | 创建/提交 projects/service-catalog/links |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/service-catalog/{catalogId}/links` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog/:catalogId/links` |
| operationId | `post_api_v1_projects_id__service-catalog_catalogId__links` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `catalogId` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog/:catalogId/links, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-181 删除 projects/service-catalog/links

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-181` |
| 接口名称 | 删除 projects/service-catalog/links |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/service-catalog/{catalogId}/links/{linkId}` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog/:catalogId/links/:linkId` |
| operationId | `delete_api_v1_projects_id__service-catalog_catalogId__links_linkId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `catalogId` | string | 是 | 路径参数 |
| path | `linkId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog/:catalogId/links/:linkId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-182 查询 projects/service-catalog/portrait

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-182` |
| 接口名称 | 查询 projects/service-catalog/portrait |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/service-catalog/{catalogId}/portrait` |
| Gin 路径 | `/api/v1/projects/:id/service-catalog/:catalogId/portrait` |
| operationId | `get_api_v1_projects_id__service-catalog_catalogId__portrait` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `catalogId` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/service-catalog/:catalogId/portrait, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-183 查询 v1/projects/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-183` |
| 接口名称 | 查询 v1/projects/services |
| 请求方式 | `GET` |
| URL | `/api/v1/projects/{id}/services` |
| Gin 路径 | `/api/v1/projects/:id/services` |
| operationId | `get_api_v1_projects_id__services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/services, GET)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-184 创建/提交 v1/projects/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-184` |
| 接口名称 | 创建/提交 v1/projects/services |
| 请求方式 | `POST` |
| URL | `/api/v1/projects/{id}/services` |
| Gin 路径 | `/api/v1/projects/:id/services` |
| operationId | `post_api_v1_projects_id__services` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/services, POST)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

#### API-PRJ-185 删除 v1/projects/services

| 项 | 内容 |
|----|------|
| 接口编号 | `API-PRJ-185` |
| 接口名称 | 删除 v1/projects/services |
| 请求方式 | `DELETE` |
| URL | `/api/v1/projects/{id}/services/{serviceId}` |
| Gin 路径 | `/api/v1/projects/:id/services/:serviceId` |
| operationId | `delete_api_v1_projects_id__services_serviceId_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| path | `serviceId` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/projects/:id/services/:serviceId, DELETE)`；项目成员：`RequireProjectMemberAccess`（`:id` 项目）；写操作：`OperationAudit` 审计

---

### 3.40 Rbac

#### API-RBAC-001 删除 api/v1/rbac

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-001` |
| 接口名称 | 删除 api/v1/rbac |
| 请求方式 | `DELETE` |
| URL | `/api/v1/rbac` |
| Gin 路径 | `/api/v1/rbac` |
| operationId | `delete_api_v1_rbac` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-RBAC-002 创建/提交 v1/rbac/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-002` |
| 接口名称 | 创建/提交 v1/rbac/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/rbac/apply` |
| Gin 路径 | `/api/v1/rbac/apply` |
| operationId | `post_api_v1_rbac_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/apply, POST)`；写操作：`OperationAudit` 审计

---

#### API-RBAC-003 查询 v1/rbac/clusterrolebindings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-003` |
| 接口名称 | 查询 v1/rbac/clusterrolebindings |
| 请求方式 | `GET` |
| URL | `/api/v1/rbac/clusterrolebindings` |
| Gin 路径 | `/api/v1/rbac/clusterrolebindings` |
| operationId | `get_api_v1_rbac_clusterrolebindings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/clusterrolebindings, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-RBAC-004 查询 v1/rbac/clusterroles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-004` |
| 接口名称 | 查询 v1/rbac/clusterroles |
| 请求方式 | `GET` |
| URL | `/api/v1/rbac/clusterroles` |
| Gin 路径 | `/api/v1/rbac/clusterroles` |
| operationId | `get_api_v1_rbac_clusterroles` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/clusterroles, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-RBAC-005 查询 v1/rbac/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-005` |
| 接口名称 | 查询 v1/rbac/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/rbac/detail` |
| Gin 路径 | `/api/v1/rbac/detail` |
| operationId | `get_api_v1_rbac_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/detail, GET)`；写操作：`OperationAudit` 审计

---

#### API-RBAC-006 查询 v1/rbac/rolebindings

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-006` |
| 接口名称 | 查询 v1/rbac/rolebindings |
| 请求方式 | `GET` |
| URL | `/api/v1/rbac/rolebindings` |
| Gin 路径 | `/api/v1/rbac/rolebindings` |
| operationId | `get_api_v1_rbac_rolebindings` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/rolebindings, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-RBAC-007 查询 v1/rbac/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-RBAC-007` |
| 接口名称 | 查询 v1/rbac/roles |
| 请求方式 | `GET` |
| URL | `/api/v1/rbac/roles` |
| Gin 路径 | `/api/v1/rbac/roles` |
| operationId | `get_api_v1_rbac_roles` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/rbac/roles, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.41 Registrations

#### API-REGIST-001 查询 api/v1/registrations

| 项 | 内容 |
|----|------|
| 接口编号 | `API-REGIST-001` |
| 接口名称 | 查询 api/v1/registrations |
| 请求方式 | `GET` |
| URL | `/api/v1/registrations` |
| Gin 路径 | `/api/v1/registrations` |
| operationId | `get_api_v1_registrations` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/registrations, GET)`；写操作：`OperationAudit` 审计

---

#### API-REGIST-002 创建/提交 v1/registrations/review

| 项 | 内容 |
|----|------|
| 接口编号 | `API-REGIST-002` |
| 接口名称 | 创建/提交 v1/registrations/review |
| 请求方式 | `POST` |
| URL | `/api/v1/registrations/{id}/review` |
| Gin 路径 | `/api/v1/registrations/:id/review` |
| operationId | `post_api_v1_registrations_id__review` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/registrations/:id/review, POST)`；写操作：`OperationAudit` 审计

---

### 3.42 Roles

#### API-ROLE-001 查询 api/v1/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ROLE-001` |
| 接口名称 | 查询 api/v1/roles |
| 请求方式 | `GET` |
| URL | `/api/v1/roles` |
| Gin 路径 | `/api/v1/roles` |
| operationId | `get_api_v1_roles` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/roles, GET)`；写操作：`OperationAudit` 审计

---

#### API-ROLE-002 创建/提交 api/v1/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ROLE-002` |
| 接口名称 | 创建/提交 api/v1/roles |
| 请求方式 | `POST` |
| URL | `/api/v1/roles` |
| Gin 路径 | `/api/v1/roles` |
| operationId | `post_api_v1_roles` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/roles, POST)`；写操作：`OperationAudit` 审计

---

#### API-ROLE-003 删除 api/v1/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ROLE-003` |
| 接口名称 | 删除 api/v1/roles |
| 请求方式 | `DELETE` |
| URL | `/api/v1/roles/{id}` |
| Gin 路径 | `/api/v1/roles/:id` |
| operationId | `delete_api_v1_roles_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/roles/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-ROLE-004 查询 api/v1/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ROLE-004` |
| 接口名称 | 查询 api/v1/roles |
| 请求方式 | `GET` |
| URL | `/api/v1/roles/{id}` |
| Gin 路径 | `/api/v1/roles/:id` |
| operationId | `get_api_v1_roles_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/roles/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-ROLE-005 更新 api/v1/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-ROLE-005` |
| 接口名称 | 更新 api/v1/roles |
| 请求方式 | `PUT` |
| URL | `/api/v1/roles/{id}` |
| Gin 路径 | `/api/v1/roles/:id` |
| operationId | `put_api_v1_roles_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/roles/:id, PUT)`；写操作：`OperationAudit` 审计

---

### 3.43 Secrets

#### API-SECRET-001 删除 api/v1/secrets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECRET-001` |
| 接口名称 | 删除 api/v1/secrets |
| 请求方式 | `DELETE` |
| URL | `/api/v1/secrets` |
| Gin 路径 | `/api/v1/secrets` |
| operationId | `delete_api_v1_secrets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/secrets, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SECRET-002 查询 api/v1/secrets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECRET-002` |
| 接口名称 | 查询 api/v1/secrets |
| 请求方式 | `GET` |
| URL | `/api/v1/secrets` |
| Gin 路径 | `/api/v1/secrets` |
| operationId | `get_api_v1_secrets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/secrets, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SECRET-003 创建/提交 v1/secrets/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECRET-003` |
| 接口名称 | 创建/提交 v1/secrets/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/secrets/apply` |
| Gin 路径 | `/api/v1/secrets/apply` |
| operationId | `post_api_v1_secrets_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/secrets/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SECRET-004 查询 v1/secrets/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECRET-004` |
| 接口名称 | 查询 v1/secrets/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/secrets/detail` |
| Gin 路径 | `/api/v1/secrets/detail` |
| operationId | `get_api_v1_secrets_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/secrets/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.44 Security

#### API-SECURI-001 查询 v1/security/banned-ips

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECURI-001` |
| 接口名称 | 查询 v1/security/banned-ips |
| 请求方式 | `GET` |
| URL | `/api/v1/security/banned-ips` |
| Gin 路径 | `/api/v1/security/banned-ips` |
| operationId | `get_api_v1_security_banned-ips` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/security/banned-ips, GET)`；写操作：`OperationAudit` 审计

---

#### API-SECURI-002 创建/提交 security/banned-ips/unban

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SECURI-002` |
| 接口名称 | 创建/提交 security/banned-ips/unban |
| 请求方式 | `POST` |
| URL | `/api/v1/security/banned-ips/unban` |
| Gin 路径 | `/api/v1/security/banned-ips/unban` |
| operationId | `post_api_v1_security_banned-ips_unban` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/security/banned-ips/unban, POST)`；写操作：`OperationAudit` 审计

---

### 3.45 Serviceaccounts

#### API-SERVIC-001 删除 api/v1/serviceaccounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SERVIC-001` |
| 接口名称 | 删除 api/v1/serviceaccounts |
| 请求方式 | `DELETE` |
| URL | `/api/v1/serviceaccounts` |
| Gin 路径 | `/api/v1/serviceaccounts` |
| operationId | `delete_api_v1_serviceaccounts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/serviceaccounts, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SERVIC-002 查询 api/v1/serviceaccounts

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SERVIC-002` |
| 接口名称 | 查询 api/v1/serviceaccounts |
| 请求方式 | `GET` |
| URL | `/api/v1/serviceaccounts` |
| Gin 路径 | `/api/v1/serviceaccounts` |
| operationId | `get_api_v1_serviceaccounts` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/serviceaccounts, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SERVIC-003 创建/提交 v1/serviceaccounts/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SERVIC-003` |
| 接口名称 | 创建/提交 v1/serviceaccounts/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/serviceaccounts/apply` |
| Gin 路径 | `/api/v1/serviceaccounts/apply` |
| operationId | `post_api_v1_serviceaccounts_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/serviceaccounts/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-SERVIC-004 查询 v1/serviceaccounts/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SERVIC-004` |
| 接口名称 | 查询 v1/serviceaccounts/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/serviceaccounts/detail` |
| Gin 路径 | `/api/v1/serviceaccounts/detail` |
| operationId | `get_api_v1_serviceaccounts_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/serviceaccounts/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.46 Statefulsets

#### API-STATEF-001 删除 api/v1/statefulsets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-001` |
| 接口名称 | 删除 api/v1/statefulsets |
| 请求方式 | `DELETE` |
| URL | `/api/v1/statefulsets` |
| Gin 路径 | `/api/v1/statefulsets` |
| operationId | `delete_api_v1_statefulsets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-002 查询 api/v1/statefulsets

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-002` |
| 接口名称 | 查询 api/v1/statefulsets |
| 请求方式 | `GET` |
| URL | `/api/v1/statefulsets` |
| Gin 路径 | `/api/v1/statefulsets` |
| operationId | `get_api_v1_statefulsets` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-003 创建/提交 v1/statefulsets/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-003` |
| 接口名称 | 创建/提交 v1/statefulsets/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/statefulsets/apply` |
| Gin 路径 | `/api/v1/statefulsets/apply` |
| operationId | `post_api_v1_statefulsets_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-004 创建/提交 v1/statefulsets/container-resources

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-004` |
| 接口名称 | 创建/提交 v1/statefulsets/container-resources |
| 请求方式 | `POST` |
| URL | `/api/v1/statefulsets/container-resources` |
| Gin 路径 | `/api/v1/statefulsets/container-resources` |
| operationId | `post_api_v1_statefulsets_container-resources` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/container-resources, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-005 查询 v1/statefulsets/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-005` |
| 接口名称 | 查询 v1/statefulsets/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/statefulsets/detail` |
| Gin 路径 | `/api/v1/statefulsets/detail` |
| operationId | `get_api_v1_statefulsets_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-006 查询 v1/statefulsets/pods

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-006` |
| 接口名称 | 查询 v1/statefulsets/pods |
| 请求方式 | `GET` |
| URL | `/api/v1/statefulsets/pods` |
| Gin 路径 | `/api/v1/statefulsets/pods` |
| operationId | `get_api_v1_statefulsets_pods` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/pods, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-007 创建/提交 v1/statefulsets/preview-apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-007` |
| 接口名称 | 创建/提交 v1/statefulsets/preview-apply |
| 请求方式 | `POST` |
| URL | `/api/v1/statefulsets/preview-apply` |
| Gin 路径 | `/api/v1/statefulsets/preview-apply` |
| operationId | `post_api_v1_statefulsets_preview-apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/preview-apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-008 创建/提交 v1/statefulsets/restart

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-008` |
| 接口名称 | 创建/提交 v1/statefulsets/restart |
| 请求方式 | `POST` |
| URL | `/api/v1/statefulsets/restart` |
| Gin 路径 | `/api/v1/statefulsets/restart` |
| operationId | `post_api_v1_statefulsets_restart` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/restart, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STATEF-009 创建/提交 v1/statefulsets/scale

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STATEF-009` |
| 接口名称 | 创建/提交 v1/statefulsets/scale |
| 请求方式 | `POST` |
| URL | `/api/v1/statefulsets/scale` |
| Gin 路径 | `/api/v1/statefulsets/scale` |
| operationId | `post_api_v1_statefulsets_scale` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/statefulsets/scale, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.47 Storageclasses

#### API-STORAG-001 删除 api/v1/storageclasses

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STORAG-001` |
| 接口名称 | 删除 api/v1/storageclasses |
| 请求方式 | `DELETE` |
| URL | `/api/v1/storageclasses` |
| Gin 路径 | `/api/v1/storageclasses` |
| operationId | `delete_api_v1_storageclasses` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/storageclasses, DELETE)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STORAG-002 查询 api/v1/storageclasses

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STORAG-002` |
| 接口名称 | 查询 api/v1/storageclasses |
| 请求方式 | `GET` |
| URL | `/api/v1/storageclasses` |
| Gin 路径 | `/api/v1/storageclasses` |
| operationId | `get_api_v1_storageclasses` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/storageclasses, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STORAG-003 创建/提交 v1/storageclasses/apply

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STORAG-003` |
| 接口名称 | 创建/提交 v1/storageclasses/apply |
| 请求方式 | `POST` |
| URL | `/api/v1/storageclasses/apply` |
| Gin 路径 | `/api/v1/storageclasses/apply` |
| operationId | `post_api_v1_storageclasses_apply` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/storageclasses/apply, POST)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

#### API-STORAG-004 查询 v1/storageclasses/detail

| 项 | 内容 |
|----|------|
| 接口编号 | `API-STORAG-004` |
| 接口名称 | 查询 v1/storageclasses/detail |
| 请求方式 | `GET` |
| URL | `/api/v1/storageclasses/detail` |
| Gin 路径 | `/api/v1/storageclasses/detail` |
| operationId | `get_api_v1_storageclasses_detail` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/storageclasses/detail, GET)`；K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）；写操作：`OperationAudit` 审计

---

### 3.48 System

#### API-SYS-001 查询 api/v1/health

| 项 | 内容 |
|----|------|
| 接口编号 | `API-SYS-001` |
| 接口名称 | 查询 api/v1/health |
| 请求方式 | `GET` |
| URL | `/api/v1/health` |
| Gin 路径 | `/api/v1/health` |
| operationId | `get_api_v1_health` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

公开接口（无需登录）

---

### 3.49 User-groups

#### API-USERGR-001 查询 api/v1/user-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-001` |
| 接口名称 | 查询 api/v1/user-groups |
| 请求方式 | `GET` |
| URL | `/api/v1/user-groups` |
| Gin 路径 | `/api/v1/user-groups` |
| operationId | `get_api_v1_user-groups` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups, GET)`；写操作：`OperationAudit` 审计

---

#### API-USERGR-002 创建/提交 api/v1/user-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-002` |
| 接口名称 | 创建/提交 api/v1/user-groups |
| 请求方式 | `POST` |
| URL | `/api/v1/user-groups` |
| Gin 路径 | `/api/v1/user-groups` |
| operationId | `post_api_v1_user-groups` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups, POST)`；写操作：`OperationAudit` 审计

---

#### API-USERGR-003 删除 api/v1/user-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-003` |
| 接口名称 | 删除 api/v1/user-groups |
| 请求方式 | `DELETE` |
| URL | `/api/v1/user-groups/{id}` |
| Gin 路径 | `/api/v1/user-groups/:id` |
| operationId | `delete_api_v1_user-groups_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-USERGR-004 查询 api/v1/user-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-004` |
| 接口名称 | 查询 api/v1/user-groups |
| 请求方式 | `GET` |
| URL | `/api/v1/user-groups/{id}` |
| Gin 路径 | `/api/v1/user-groups/:id` |
| operationId | `get_api_v1_user-groups_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-USERGR-005 更新 api/v1/user-groups

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-005` |
| 接口名称 | 更新 api/v1/user-groups |
| 请求方式 | `PUT` |
| URL | `/api/v1/user-groups/{id}` |
| Gin 路径 | `/api/v1/user-groups/:id` |
| operationId | `put_api_v1_user-groups_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-USERGR-006 更新 v1/user-groups/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USERGR-006` |
| 接口名称 | 更新 v1/user-groups/users |
| 请求方式 | `PUT` |
| URL | `/api/v1/user-groups/{id}/users` |
| Gin 路径 | `/api/v1/user-groups/:id/users` |
| operationId | `put_api_v1_user-groups_id__users` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/user-groups/:id/users, PUT)`；写操作：`OperationAudit` 审计

---

### 3.50 Users

#### API-USR-001 查询 api/v1/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-001` |
| 接口名称 | 查询 api/v1/users |
| 请求方式 | `GET` |
| URL | `/api/v1/users` |
| Gin 路径 | `/api/v1/users` |
| operationId | `get_api_v1_users` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users, GET)`；写操作：`OperationAudit` 审计

---

#### API-USR-002 创建/提交 api/v1/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-002` |
| 接口名称 | 创建/提交 api/v1/users |
| 请求方式 | `POST` |
| URL | `/api/v1/users` |
| Gin 路径 | `/api/v1/users` |
| operationId | `post_api_v1_users` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users, POST)`；写操作：`OperationAudit` 审计

---

#### API-USR-003 查询 v1/users/export

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-003` |
| 接口名称 | 查询 v1/users/export |
| 请求方式 | `GET` |
| URL | `/api/v1/users/export` |
| Gin 路径 | `/api/v1/users/export` |
| operationId | `get_api_v1_users_export` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/export, GET)`；写操作：`OperationAudit` 审计

---

#### API-USR-004 创建/提交 v1/users/import

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-004` |
| 接口名称 | 创建/提交 v1/users/import |
| 请求方式 | `POST` |
| URL | `/api/v1/users/import` |
| Gin 路径 | `/api/v1/users/import` |
| operationId | `post_api_v1_users_import` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/import, POST)`；写操作：`OperationAudit` 审计

---

#### API-USR-005 查询 v1/users/import-template

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-005` |
| 接口名称 | 查询 v1/users/import-template |
| 请求方式 | `GET` |
| URL | `/api/v1/users/import-template` |
| Gin 路径 | `/api/v1/users/import-template` |
| operationId | `get_api_v1_users_import-template` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/import-template, GET)`；写操作：`OperationAudit` 审计

---

#### API-USR-006 删除 api/v1/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-006` |
| 接口名称 | 删除 api/v1/users |
| 请求方式 | `DELETE` |
| URL | `/api/v1/users/{id}` |
| Gin 路径 | `/api/v1/users/:id` |
| operationId | `delete_api_v1_users_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/:id, DELETE)`；写操作：`OperationAudit` 审计

---

#### API-USR-007 查询 api/v1/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-007` |
| 接口名称 | 查询 api/v1/users |
| 请求方式 | `GET` |
| URL | `/api/v1/users/{id}` |
| Gin 路径 | `/api/v1/users/:id` |
| operationId | `get_api_v1_users_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/:id, GET)`；写操作：`OperationAudit` 审计

---

#### API-USR-008 更新 api/v1/users

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-008` |
| 接口名称 | 更新 api/v1/users |
| 请求方式 | `PUT` |
| URL | `/api/v1/users/{id}` |
| Gin 路径 | `/api/v1/users/:id` |
| operationId | `put_api_v1_users_id_` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/:id, PUT)`；写操作：`OperationAudit` 审计

---

#### API-USR-009 更新 v1/users/roles

| 项 | 内容 |
|----|------|
| 接口编号 | `API-USR-009` |
| 接口名称 | 更新 v1/users/roles |
| 请求方式 | `PUT` |
| URL | `/api/v1/users/{id}/roles` |
| Gin 路径 | `/api/v1/users/:id/roles` |
| operationId | `put_api_v1_users_id__roles` |

**请求参数**

| 位置 | 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| path | `id` | string | 是 | 路径参数 |
| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |

**响应结构**

**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ "list": [], "total": 0, "page": 1, "page_size": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```

**错误码**

| HTTP | error_code | 说明 |
|------|------------|------|
| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |
| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |
| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |
| 404 | 10004 / 11021 | 资源不存在 |
| 409 | 10005 / 11024 | 状态冲突 |
| 429 | 10007 / 10902 | 限流 |
| 500 | 10006 / 10901 | 内部错误 |

**权限要求**

JWT Bearer（`Authorization: Bearer <access_token>`）；Casbin：`Enforce(user:<id>, /api/v1/users/:id/roles, PUT)`；写操作：`OperationAudit` 审计

---

## 4. 维护说明

1. 路由变更后必须重跑 genopenapi + 本脚本，保持 OpenAPI 与本说明书同步。
2. Body 字段级契约以 Handler 结构体为准；可在对应 Handler 补充 swag 注解后用 `swag init` 增强细节。
3. Casbin 权限资源为 **完整 API 路径**（含 `/api/v1`）与 **HTTP 方法**，经 seed/权限管理写入。
