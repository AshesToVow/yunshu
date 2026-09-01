# Yunshu 前端（web-pro）

Yunshu 唯一正式前端：**Ant Design Pro v6 + Umi Max 4 + React 19 + antd 6**。

## 快速开始

```bash
# 后端
go run . server

# 前端
cd web-pro && npm install && npm run dev
# → http://localhost:8000
```

## 目录结构

```text
web-pro/src/
├── app.tsx              # ProLayout、登录态、插件过滤菜单
├── pages/
│   ├── dynamic-menu/    # 动态菜单 + 模块路由
│   ├── user/login/      # 登录
│   ├── workflow/inbox/  # 原生 Pro：我的待办
│   ├── core/            # 原生 Pro：登录/操作日志
│   └── *-page.tsx       # 业务页面（102+）
├── components/          # 业务组件
├── services/            # API 客户端（http.ts 等）
├── modules/             # 插件静态路由、plugin-path
├── hooks/ contexts/ utils/ styles/ i18n/
└── constants/
```

## 构建与部署

```bash
npm run build          # → dist/
npm run lint           # biome + tsc
docker compose up -d --build   # 仓库根目录
```

## 开发说明

- **所有业务代码直接改 `web-pro/src/`**，不再从 `web/` sync
- 新增菜单页：在 `pages/` 增加 `xxx-page.tsx`，并在 `utils/legacy-page-registry.ts` 注册（或运行 `node scripts/generate-page-registry.mjs` 若已添加）
- 插件路径规则：`src/modules/plugin-path.ts`（须与后端 `internal/plugin/path_filter.go` 同步）
- 原生 Pro 页：在 `config/routes.ts` + `utils/yunshu-menu.tsx` 的 `MIGRATED_COMPONENT_MAP` 登记

## 技术栈

| 层 | 选型 |
|---|---|
| 框架 | Umi Max 4 |
| UI | antd 6 + Pro Components |
| 请求 | `@umijs/max` request + Cookie 会话 |
| 路由 | 静态 Pro 路由 → 模块 routes → 动态菜单 |
