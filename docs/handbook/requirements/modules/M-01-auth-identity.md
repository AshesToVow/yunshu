# M-01 认证与身份

| 项 | 内容 |
|----|------|
| 文档编号 | M-01 |
| 模块名称 | 认证与身份 |
| 前端 | `/login`、`/personal-settings` |
| 后端 | `internal/service/system`（Auth）、`handler` auth 相关 |
| 状态 | 已对齐源码 |

## 1. 目标

提供登录（密码/邮箱验证码）、注册申请、登出、当前用户、改密、WebSocket 一次性 ticket。

## 2. 功能需求

| ID | 功能 | 优先级 |
|----|------|--------|
| F-01 | 用户名密码登录（可配合图形验证码） | P0 |
| F-02 | 邮箱验证码登录 | P0 |
| F-03 | 注册申请 → 管理员审核（见 M-02） | P0 |
| F-04 | 登出、拉取/更新个人资料、改密 | P0 |
| F-05 | WS ticket（终端 / Pod Exec） | P0 |

## 3. 接口规格

**Base**：`/api/v1/auth`

| 方法 | 路径 | 鉴权 | 入参 | 结果 `data` |
|------|------|------|------|-------------|
| POST | `/verification-code` | 否 | `{ email, scene }` scene=`login`/`register` | 发送结果/冷却提示 |
| POST | `/password-login-code` | 否 | 图形验证码相关 | captcha 信息 |
| POST | `/login` | 否 | `{ username, password, captcha? }` | `{ access_token, user, ... }` |
| POST | `/email-login` | 否 | `{ email, code }` | 同登录 |
| POST | `/register` | 否 | 用户名/邮箱/密码等 | 申请受理结果 |
| POST | `/logout` | 是 | — | ok |
| GET | `/me` | 是 | — | 当前用户与角色 |
| PUT | `/me` | 是 | 昵称/邮箱等 | 更新后资料 |
| PUT | `/password` | 是 | `{ old_password, new_password }` | ok |
| POST | `/ws-ticket` | 是 | — | 一次性 ticket |

### 登录成功示例

```json
{
  "code": 200,
  "data": {
    "access_token": "<jwt>",
    "user": { "id": 1, "username": "admin", "roles": ["super-admin"] }
  }
}
```

后续请求头：`Authorization: Bearer <access_token>`。

## 4. 数据模型

| 表 | 说明 |
|----|------|
| `users` | 账号，软删 |
| `user_roles` | 用户-角色 |
| `roles` | 角色编码 |
| `registration_requests` | 注册申请 |
| `login_logs` | 登录审计 |

## 5. 依赖

| 依赖 | 用途 |
|------|------|
| JWT / 安全配置 | `configs` → `auth` / `security` |
| 邮件 | 验证码（`mail` 配置） |
| Redis | 验证码存储（若启用） |

## 6. 验收

- [ ] 错误密码不泄露用户是否存在的敏感信息策略符合产品约定  
- [ ] Token 失效返回 401，前端清会话  
- [ ] 注册需审核后方可登录  

## 7. 相关文档

- [R-01-auth-and-identity.md](../R-01-auth-and-identity.md)
- [menu-login.md](../menus/menu-login.md)
- [menu-personal-settings.md](../menus/menu-personal-settings.md)
