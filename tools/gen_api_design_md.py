# gen_api_design_md.py — 从 OpenAPI 生成《API接口设计说明书》
# 用法（仓库根目录）:
#   go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml
#   python tools/gen_api_design_md.py
from __future__ import annotations

import re
import sys
from collections import defaultdict
from pathlib import Path

try:
    import yaml
except ImportError:
    print("需要 PyYAML: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "apipost" / "permission-system.openapi.yaml"
OUT_MD = ROOT / "docs" / "API接口设计说明书.md"

PUBLIC = {
    ("GET", "/api/v1/health"),
    ("POST", "/api/v1/auth/verification-code"),
    ("POST", "/api/v1/auth/login-code"),
    ("POST", "/api/v1/auth/password-login-code"),
    ("POST", "/api/v1/auth/login"),
    ("POST", "/api/v1/auth/email-login"),
    ("POST", "/api/v1/auth/register"),
    ("POST", "/api/v1/alerts/webhook/alertmanager"),
    ("POST", "/api/v1/loggie/heartbeat/report"),
}

METHOD_CN = {
    "get": "查询",
    "post": "创建/提交",
    "put": "更新",
    "delete": "删除",
    "patch": "部分更新",
}

TAG_PREFIX = {
    "Auth": "AUTH",
    "System": "SYS",
    "Plugins": "PLG",
    "Alerts": "ALT",
    "AlertsWebhook": "ALTWH",
    "Loggie": "LGG",
    "LogPlatform": "LOG",
    "Dict": "DICT",
    "K8sScopedPolicy": "K8SP",
    "K8sEventForward": "K8SEF",
    "K8s": "K8S",
    "Projects": "PRJ",
    "Users": "USR",
    "Roles": "ROLE",
    "Permissions": "PERM",
    "Policies": "POL",
    "Menus": "MENU",
    "Departments": "DEPT",
    "Overview": "OVW",
    "Clusters": "CLS",
    "Pods": "POD",
    "Deployments": "DEP",
    "Namespaces": "NS",
    "Nodes": "NODE",
    "Helm": "HELM",
}


def gin_path(openapi_path: str) -> str:
    return re.sub(r"\{([a-zA-Z0-9_]+)\}", r":\1", openapi_path)


def path_params(openapi_path: str) -> list[str]:
    return re.findall(r"\{([a-zA-Z0-9_]+)\}", openapi_path)


def human_name(method: str, path: str) -> str:
    segs = [s for s in path.strip("/").split("/") if s and not s.startswith("{") and not s.startswith(":")]
    tail = "/".join(segs[-3:]) if segs else path
    action = METHOD_CN.get(method.lower(), method.upper())
    return f"{action} {tail}"


def permission_req(method: str, openapi_path: str) -> str:
    gin = gin_path(openapi_path)
    m = method.upper()
    if (m, gin) in PUBLIC:
        if "webhook/alertmanager" in gin:
            return "Webhook Token（请求头 `X-Alert-Token` 或 `Authorization: Bearer <token>`）；无需 JWT"
        if "loggie/heartbeat" in gin:
            return "Agent 心跳上报（公开接口，按部署网络隔离）"
        return "公开接口（无需登录）"
    parts = [
        "JWT Bearer（`Authorization: Bearer <access_token>`）",
        f"Casbin：`Enforce(user:<id>, {gin}, {m})`",
    ]
    if "/projects/{id}" in openapi_path or "/projects/:id" in gin:
        parts.append("项目成员：`RequireProjectMemberAccess`（`:id` 项目）")
    if any(
        x in openapi_path
        for x in (
            "/pods",
            "/deployments",
            "/statefulsets",
            "/daemonsets",
            "/jobs",
            "/cronjobs",
            "/namespaces",
            "/nodes",
            "/services",
            "/ingresses",
            "/configmaps",
            "/secrets",
            "/persistentvolumeclaims",
            "/storageclasses",
            "/serviceaccounts",
            "/roles",
            "/rolebindings",
            "/clusterroles",
            "/clusterrolebindings",
            "/networkpolicies",
            "/horizontalpodautoscalers",
            "/crds",
            "/crs",
            "/events",
        )
    ) or openapi_path.startswith("/api/v1/helm"):
        # K8s 资源路由通常挂 K8sScopeAuthorize；精确以 register_k8s_routes 为准
        if openapi_path.startswith("/api/v1/") and not openapi_path.startswith("/api/v1/projects"):
            if not any(
                openapi_path.startswith(p)
                for p in (
                    "/api/v1/auth",
                    "/api/v1/users",
                    "/api/v1/alerts",
                    "/api/v1/dict",
                    "/api/v1/menus",
                    "/api/v1/roles",
                    "/api/v1/permissions",
                    "/api/v1/policies",
                    "/api/v1/departments",
                    "/api/v1/overview",
                    "/api/v1/plugins",
                    "/api/v1/log-platform",
                    "/api/v1/loggie",
                    "/api/v1/health",
                    "/api/v1/operation-logs",
                    "/api/v1/login-logs",
                    "/api/v1/registrations",
                    "/api/v1/banned",
                    "/api/v1/k8s-policies",
                    "/api/v1/clusters",
                )
            ):
                parts.append("K8s 作用域：`K8sScopeAuthorize`（集群档位 + NS 规则）")
    if "clusters" in openapi_path and openapi_path.startswith("/api/v1/clusters"):
        parts.append("K8s 集群权限按接口与 Casbin 策略；部分读操作可回退集群授权档位")
    if "terminal" in openapi_path or "exec/ws" in openapi_path or openapi_path.endswith("/ws"):
        parts.append("WebSocket：先 `POST /api/v1/auth/ws-ticket`，连接 URL 携带一次性 `ticket`")
    if "cicd" in openapi_path and ("build-runs" in openapi_path or "release-runs" in openapi_path):
        parts.append("资源 ACL：CICD 服务级 `AssertCicdAccess`（详情/日志需 view）")
    if "servers" in openapi_path and ("exec" in openapi_path or "terminal" in openapi_path):
        parts.append("资源 ACL：服务器 `can_exec`（`server_access_grants`）")
    if "dbmgmt" in openapi_path:
        parts.append("资源 ACL：库表授权 / 有效权限（`db_access_grants` 等）")
    parts.append("写操作：`OperationAudit` 审计")
    return "；".join(parts)


def request_params_md(method: str, openapi_path: str) -> str:
    lines = []
    for p in path_params(openapi_path):
        lines.append(f"| path | `{p}` | string | 是 | 路径参数 |")
    m = method.lower()
    if m == "get":
        lines.append("| query | `page` / `page_size` 等 | — | 否 | 列表类接口常见分页；以 Handler `form`/`binding` 为准 |")
    if m in ("post", "put", "patch"):
        lines.append("| body | JSON | object | 视接口 | `Content-Type: application/json`；字段以 Handler 请求 DTO / `binding` 为准 |")
    if m == "delete":
        lines.append("| body | JSON（可选） | object | 否 | 部分删除接口可带 JSON；以 Handler 为准 |")
    if "webhook/alertmanager" in openapi_path:
        lines.append("| header | `X-Alert-Token` | string | 是* | 与 `Authorization: Bearer` 二选一 |")
        lines.append("| body | Alertmanager webhook payload | object | 是 | 标准 AM JSON |")
    if not lines:
        lines.append("| — | — | — | — | 无额外参数 |")
    header = "| 位置 | 名称 | 类型 | 必填 | 说明 |\n|------|------|------|------|------|\n"
    return header + "\n".join(lines)


def response_md() -> str:
    return """**成功（HTTP 200/201）**

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

`data` 结构因接口而异（列表多为 `{ \"list\": [], \"total\": 0, \"page\": 1, \"page_size\": 20 }` 或领域 DTO）。

**失败（HTTP 4xx/5xx，经 ErrorHandler）**

```json
{
  "code": 401,
  "reason": "...",
  "message": "产品话术",
  "error_code": "10002",
  "metadata": {}
}
```
"""


def common_errors_for(method: str, openapi_path: str) -> str:
    rows = [
        "| HTTP | error_code | 说明 |",
        "|------|------------|------|",
        "| 400 | 10001 / 11003 / 11020 | 参数无效 / 绑定失败 |",
        "| 401 | 10002 / 10008–10015 | 未登录、Token/Session/WS ticket 无效 |",
        "| 403 | 10003 / 10012 / 11022 | Casbin/项目/资源拒绝；账号禁用 |",
        "| 404 | 10004 / 11021 | 资源不存在 |",
        "| 409 | 10005 / 11024 | 状态冲突 |",
        "| 429 | 10007 / 10902 | 限流 |",
        "| 500 | 10006 / 10901 | 内部错误 |",
    ]
    gin = gin_path(openapi_path)
    if (method.upper(), gin) in PUBLIC and "login" in gin:
        rows.append("| 401 | 20002 | 账号或密码错误 |")
        rows.append("| 429 | 20014 | 登录失败次数过多 |")
    return "\n".join(rows)


def tag_code(tag: str) -> str:
    return TAG_PREFIX.get(tag, re.sub(r"[^A-Z0-9]", "", tag.upper())[:6] or "API")


def main() -> None:
    if not OPENAPI.is_file():
        print(f"缺少 {OPENAPI}，请先运行 genopenapi", file=sys.stderr)
        sys.exit(1)
    doc = yaml.safe_load(OPENAPI.read_text(encoding="utf-8"))
    paths = doc.get("paths") or {}

    by_tag: dict[str, list[tuple[str, str, dict]]] = defaultdict(list)
    for path, methods in paths.items():
        for method, op in methods.items():
            if method.startswith("x-") or not isinstance(op, dict):
                continue
            tags = op.get("tags") or ["Other"]
            tag = tags[0]
            by_tag[tag].append((path, method.upper(), op))

    for tag in by_tag:
        by_tag[tag].sort(key=lambda x: (x[0], x[1]))

    lines: list[str] = []
    lines.append("# Yunshu API 接口设计说明书")
    lines.append("")
    lines.append("| 项 | 内容 |")
    lines.append("|----|------|")
    lines.append("| 文档编号 | YUNSHU-API-2026-002 |")
    lines.append("| 版本 | V2.0.0 |")
    lines.append("| Base Path | `/api/v1` |")
    lines.append("| OpenAPI | 3.0.3（[`docs/apipost/permission-system.openapi.yaml`](apipost/permission-system.openapi.yaml)） |")
    lines.append("| 生成方式 | `go run ./tools/genopenapi` + `python tools/gen_api_design_md.py` |")
    lines.append("| 权威来源 | `internal/router/register_*.go` |")
    lines.append("| 日期 | 2026-08-05 |")
    lines.append("")
    lines.append("> 本文档接口清单由路由自动生成，与 OpenAPI 对齐；请求 Body 字段细节以实现 Handler DTO 为准。")
    lines.append("")
    lines.append("---")
    lines.append("")
    lines.append("## 1. API 规范")
    lines.append("")
    lines.append("每个接口条目包含：")
    lines.append("")
    lines.append("| 要素 | 说明 |")
    lines.append("|------|------|")
    lines.append("| 接口编号 | `API-<域前缀>-<序号>` |")
    lines.append("| 接口名称 | 中文简述 |")
    lines.append("| 请求方式 | GET/POST/PUT/PATCH/DELETE |")
    lines.append("| URL | OpenAPI 路径（`{param}` 对应 Gin `:param`） |")
    lines.append("| 请求参数 | Path / Query / Header / Body |")
    lines.append("| 响应结构 | 统一 `StandardResponse` / `ErrorBody` |")
    lines.append("| 错误码 | HTTP + 业务 `error_code`（见 `internal/pkg/constants`） |")
    lines.append("| 权限要求 | JWT / Casbin / 项目成员 / K8sScope / 资源 ACL / Webhook |")
    lines.append("")
    lines.append("### 1.1 鉴权")
    lines.append("")
    lines.append("```http")
    lines.append("Authorization: Bearer <JWT>")
    lines.append("```")
    lines.append("")
    lines.append("- Session：JWT Claims.`TokenID` 须在 Redis 白名单（失败关闭）。")
    lines.append("- WebSocket：`POST /api/v1/auth/ws-ticket` → 连接带 `?ticket=`。")
    lines.append("- Alertmanager：`X-Alert-Token` 或 Bearer webhook token。")
    lines.append("")
    lines.append("### 1.2 统一响应（兼容 OpenAPI `StandardResponse`）")
    lines.append("")
    lines.append(response_md())
    lines.append("### 1.3 通用错误码（节选）")
    lines.append("")
    lines.append("| error_code | HTTP | 说明 |")
    lines.append("|------------|------|------|")
    lines.append("| 10001 | 400 | 请求参数无效 |")
    lines.append("| 10002 | 401 | 登录失效/凭证无效 |")
    lines.append("| 10003 | 403 | 无权执行 |")
    lines.append("| 10004 | 404 | 资源不存在 |")
    lines.append("| 10005 | 409 | 状态冲突 |")
    lines.append("| 10006 | 500 | 服务异常 |")
    lines.append("| 10007 | 429 | 过于频繁 |")
    lines.append("| 10008–10015 | 401 | 鉴权头/Token/Session/WS ticket |")
    lines.append("| 11020–11024 | 4xx | WithMsg 可变文案（参数/未找到/禁止/未授权/冲突） |")
    lines.append("| 20001–20014 | 4xx | 认证与账号域 |")
    lines.append("| 其他 21xxx+ | — | 按功能域，见 `internal/pkg/constants/constant.go` |")
    lines.append("")
    lines.append("### 1.4 Swagger / OpenAPI 兼容")
    lines.append("")
    lines.append("| 产物 | 路径 |")
    lines.append("|------|------|")
    lines.append("| **推荐（全量路由）** | [`docs/apipost/permission-system.openapi.yaml`](apipost/permission-system.openapi.yaml) |")
    lines.append("| Swagger UI | 配置 `swagger.enabled=true` → `/swagger/index.html` |")
    lines.append("| swag 注解集 | `docs/swagger/`（覆盖不全，仅作补充） |")
    lines.append("")
    lines.append("重新生成：")
    lines.append("")
    lines.append("```bash")
    lines.append("go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml")
    lines.append("python tools/gen_api_design_md.py")
    lines.append("```")
    lines.append("")
    lines.append("---")
    lines.append("")
    lines.append("## 2. 接口目录总览")
    lines.append("")
    total = sum(len(v) for v in by_tag.values())
    lines.append(f"合计 **{len(paths)}** 个路径、**{total}** 个操作，按 Tag 分组如下。")
    lines.append("")
    lines.append("| Tag | 接口数 | 编号前缀 |")
    lines.append("|-----|--------|----------|")
    for tag in sorted(by_tag.keys()):
        lines.append(f"| {tag} | {len(by_tag[tag])} | API-{tag_code(tag)}-* |")
    lines.append("")
    lines.append("---")
    lines.append("")
    lines.append("## 3. 接口明细")
    lines.append("")

    for tag in sorted(by_tag.keys()):
        prefix = tag_code(tag)
        lines.append(f"### 3.{sorted(by_tag.keys()).index(tag)+1} {tag}")
        lines.append("")
        for i, (path, method, _op) in enumerate(by_tag[tag], 1):
            api_id = f"API-{prefix}-{i:03d}"
            name = human_name(method, path)
            lines.append(f"#### {api_id} {name}")
            lines.append("")
            lines.append("| 项 | 内容 |")
            lines.append("|----|------|")
            lines.append(f"| 接口编号 | `{api_id}` |")
            lines.append(f"| 接口名称 | {name} |")
            lines.append(f"| 请求方式 | `{method}` |")
            lines.append(f"| URL | `{path}` |")
            lines.append(f"| Gin 路径 | `{gin_path(path)}` |")
            lines.append(f"| operationId | `{_op.get('operationId', '')}` |")
            lines.append("")
            lines.append("**请求参数**")
            lines.append("")
            lines.append(request_params_md(method, path))
            lines.append("")
            lines.append("**响应结构**")
            lines.append("")
            lines.append(response_md())
            lines.append("**错误码**")
            lines.append("")
            lines.append(common_errors_for(method, path))
            lines.append("")
            lines.append("**权限要求**")
            lines.append("")
            lines.append(permission_req(method, path))
            lines.append("")
            lines.append("---")
            lines.append("")

    lines.append("## 4. 维护说明")
    lines.append("")
    lines.append("1. 路由变更后必须重跑 genopenapi + 本脚本，保持 OpenAPI 与本说明书同步。")
    lines.append("2. Body 字段级契约以 Handler 结构体为准；可在对应 Handler 补充 swag 注解后用 `swag init` 增强细节。")
    lines.append("3. Casbin 权限资源为 **完整 API 路径**（含 `/api/v1`）与 **HTTP 方法**，经 seed/权限管理写入。")
    lines.append("")

    OUT_MD.write_text("\n".join(lines), encoding="utf-8")
    print(f"wrote {OUT_MD} ({total} operations, {OUT_MD.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
