# 贡献与二次开发指南

面向外部开发者：如何在 Yunshu 单仓库中快速上手、扩展插件、提交变更。

---

## 1. 5 分钟本地跑通

```bash
git clone <repo-url> && cd yunshu

# 后端
cp configs/config.yaml configs/config.local.yaml   # 按需改 MySQL/Redis
go run . migrate
go run . seed
go run . server

# 前端（另开终端，正式版 web-pro）
cd web-pro && npm install && npm run dev
```

默认管理员：`admin` / `Admin@123`（仅首次 seed 写入）。

**必读文档（按顺序）**：

| 顺序 | 文档 | 内容 |
|------|------|------|
| 1 | [docs/CODEBASE-MAP.md](docs/CODEBASE-MAP.md) | 后端分层、Service 目录、阅读路径 |
| 2 | [docs/plugins.md](docs/plugins.md) | 插件机制、内置插件、新增步骤 |
| 3 | [docs/handbook/api/http-api-conventions.md](docs/handbook/api/http-api-conventions.md) | HTTP 错误格式、鉴权 |
| 4 | [web-pro/package.json](web-pro/package.json) | 前端 npm scripts（正式版） |
| 5 | [docs/web-pro-migration.md](docs/web-pro-migration.md) | Pro 前端架构与 sync:legacy |

---

## 2. 仓库结构（心智模型）

```text
cmd/              CLI 入口（server / migrate / seed）
configs/          config.yaml + plugins.enabled
internal/
  handler/        HTTP 薄层（参数绑定 → Service → response）
  service/<域>/   业务逻辑（alert / k8s / cicd / cmdb …）
  repository/     GORM 数据访问 + interfaces
  plugin/         插件注册、路径过滤、Runtime
  plugins/<名>/   各插件 init() 注册 + Models 迁移
  router/         register_*_routes.go + Wire 装配
  menu/           内置菜单 catalog + DB 同步
web-pro/src/
  pages/          页面组件（*-page.tsx）
  components/     业务组件
  services/       API 客户端（axios 封装）
  modules/        插件路由、plugin-path
  modules/        按插件拆分的前端路由
  modules/plugin-path.ts   菜单/API → 插件映射（须与后端同步）
```

**请求链路**：`HTTP → middleware → handler → service → repository → model`

**Handler 统一用** `response.Error(c, err)` 返回业务错误（告警模块部分仍用 `abortService`，新代码勿再增加）。

**业务错误**：`bizerrors.Pass(ctx, "domain", "Method", err)`  
**日志**：`logutil.HTTP("http.xxx")` / `logutil.Worker("worker.xxx")`

---

## 3. 新增 API 检查清单

1. **Service** — `internal/service/<domain>/`
2. **Handler** — `internal/handler/<domain>_handler.go`
3. **路由** — `internal/router/register_<plugin>_routes.go`
4. **权限种子** — `cmd/seed.go` 增加 `Permission` 行
5. **OpenAPI** — `go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml`
6. **前端** — `web-pro/src/services/` + 页面；若新菜单则改 `internal/menu/catalog.go` 并 `go run . seed`

---

## 4. 新增插件检查清单

后端：

1. `internal/plugins/<name>/plugin.go` — `init()` 里 `plugin.Register`
2. `internal/plugins/all/all.go` — blank import
3. `internal/router/register_<name>_routes.go` + `plugin_bind.go` 分支
4. `configs/config.yaml` — `plugins.enabled` 加入插件名
5. `cmd/seed.go` — 权限项

前端：

1. `web-pro/src/modules/<name>/routes.tsx`（或动态菜单 + page loader）
2. **`web-pro/src/modules/plugin-path.ts`** — 菜单 path 与 API resource 规则
3. **`internal/plugin/path_filter.go`** — 与上条保持同步（见 §5）
4. `internal/menu/catalog.go` — 侧栏菜单

Worker（可选）：实现 `Module.StartWorkers`，在 `plugin.Runtime` 注入依赖。

---

## 5. 插件路径规则（重要：双端同步）

菜单/API 归属哪个插件，由两处共同决定：

| 文件 | 用途 |
|------|------|
| `internal/plugin/path_filter.go` | 后端菜单过滤、策略授权 UI |
| `web-pro/src/modules/plugin-path.ts` | 前端侧栏、权限列表过滤 |

**修改任一侧时必须同步另一侧。** 特殊规则：

- **CMDB**：`/project-servers` 需同时启用 `cmdb` + `project`
- **CI/CD**：`/cicd` 与 `/api/v1/projects/*/cicd/*` 需同时启用 `cicd` + `project`
- **dbmgmt**：`/dbmgmt` 与 `/api/v1/projects/*/dbmgmt/*` 需同时启用 `dbmgmt` + `project`
- **总览 CI 图表 API**：`/api/v1/overview/project-launches` 等归属 `cicd` 插件；未启用时返回空数据而非 SQL 错误

运行测试：`go test ./internal/plugin/...`

---

## 6. 配置与数据字典

- 静态默认：`configs/config.yaml`
- 运行时覆盖：`dict_entries` 表（见 `internal/dictconfig/`）
- 统一读取：`dictconfig.FetchEnabledDictValue(ctx, db, dictType)`

MinIO 有两套键：`minio_*`（备份）与 `cicd_minio_*`（CI/CD 制品），见 `internal/dictconfig/minio.go`、`cicd_minio.go`。

---

## 7. 测试与构建

```bash
# 后端单元测试
go test ./internal/... -count=1

# 指定包
go test ./internal/plugin/... ./internal/pkg/mysqlbackup/...

# 编译
go build -o yunshu .

# 前端
cd web-pro && npm run build && npm run lint
```

---

## 8. 提交前自检

- [ ] 新 API 已写入 `cmd/seed.go` 权限
- [ ] 已运行 `genopenapi` 更新 OpenAPI（若改路由）
- [ ] 插件 path 规则前后端已同步
- [ ] 未引入 `svcerr` / `svclog` / `apperror`（已废弃）
- [ ] Service 层无新增裸 SQL（走 repository）
- [ ] 敏感信息未写入日志或提交物

---

## 9. 常见问题

**Q: 禁用 cicd 插件后总览图表空白？**  
A: 预期行为。图表依赖 `cicd_release_runs` 表；启用 `cicd` 插件并 migrate 后即有数据。

**Q: 菜单/API 在禁用插件后仍可见？**  
A: 检查 `path_filter.go` 与 `plugin-path.ts` 是否包含该 path 前缀。

**Q: Handler 里 import 哪个 service 包？**  
A: 统一 `import "yunshu/internal/service"`，类型见 `internal/service/exports.go`。

**Q: 如何调试 Casbin 权限？**  
A: 见 [docs/handbook/permissions/casbin-and-k8s-triple-policy.md](docs/handbook/permissions/casbin-and-k8s-triple-policy.md)。

---

## 10. 相关链接

- [README.md](README.md) — 运维部署、功能说明
- [docs/cicd.md](docs/cicd.md) — CI/CD 插件
- [docs/dbmgmt.md](docs/dbmgmt.md) — 数据库管理插件
- [docs/refactoring-report.md](docs/refactoring-report.md) — 重构进度
- [docs/plugins.md](docs/plugins.md) — 插件详解
