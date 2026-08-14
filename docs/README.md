# Yunshu 文档索引

**最后更新**: 2026-08-05

---

## 项目交付文档（甲方验收）

| 文档 | 说明 |
|------|------|
| [软件需求规格说明书.md](软件需求规格说明书.md) | SRS：功能范围、角色、验收标准（基于当前代码） |
| [软件概要设计说明书.md](软件概要设计说明书.md) | HLD：逻辑/物理架构、插件边界、鉴权 |
| [软件详细设计说明书.md](软件详细设计说明书.md) | LLD：关键模块调用链与状态机 |
| [数据库设计说明书.md](数据库设计说明书.md) | 表清单、ER、迁移约定 |
| [API接口设计说明书.md](API接口设计说明书.md) | 接口分组与代表接口；以路由代码为准 |
| [系统部署实施方案.md](系统部署实施方案.md) | Compose/配置/升级回滚 |
| [系统运维手册.md](系统运维手册.md) | 日志、Worker、排障、备份 |
| [系统测试报告.md](系统测试报告.md) | 真实测试资产与结论（不虚构覆盖率） |
| [项目验收报告.md](项目验收报告.md) | 验收对照表与签署栏 |
| [yunshu-audit-report.md](yunshu-audit-report.md) | 安全质量核查与修复记录 |

---

## 开发者必读（后端）

| 文档 | 说明 |
|------|------|
| [CODEBASE-MAP.md](CODEBASE-MAP.md) | **代码地图**：目录结构、阅读顺序、Wire/错误规范 |
| [logging.md](logging.md) | **日志与错误码**：slog 三文件、前端 `error_code` 联调 |
| [plugins.md](plugins.md) | **业务插件**：`plugins.enabled`、路由绑定、CMDB 拆分 |
| [cicd.md](cicd.md) | **CI/CD 插件**：Jenkins、审批流、字典配置 |
| [dbmgmt.md](dbmgmt.md) | **数据库管理插件**：实例纳管、SQL 查询/审核、授权工单、goInception |
| [backend-architecture-complete.md](backend-architecture-complete.md) | 后端完整技术文档（架构、模块、部署） |
| [refactoring-report.md](refactoring-report.md) | 架构重构实施状态（与代码同步） |
| [architecture-diagrams.md](architecture-diagrams.md) | 架构图集 |

---

## 产品手册

| 文档 | 说明 |
|------|------|
| [handbook/README.md](handbook/README.md) | 需求、数据库、API、权限手册入口 |
| [handbook/00-architecture-analysis-and-optimization.md](handbook/00-architecture-analysis-and-optimization.md) | 功能域与 SQL 优化 |
| [handbook/requirements/](handbook/requirements/) | 按业务域拆分的需求 |
| [handbook/database/schema-and-relationships.md](handbook/database/schema-and-relationships.md) | 表结构与 ER |
| [handbook/permissions/casbin-and-k8s-triple-policy.md](handbook/permissions/casbin-and-k8s-triple-policy.md) | Casbin + K8s 三元策略 |

---

## 告警与日志

| 文档 | 说明 |
|------|------|
| [alert-routing-and-delivery-guide.md](alert-routing-and-delivery-guide.md) | 路由、投递、值班（运维推荐） |
| [alert-notify-guide.md](alert-notify-guide.md) | 通知配置与聚合 |
| [alert-subscription-labels-chain.md](alert-subscription-labels-chain.md) | 订阅标签约定 |
| [alert-redis-degradation.md](alert-redis-degradation.md) | 告警 Redis 降级说明 |
| [requirements/R-alert-platform-detailed-design.md](requirements/R-alert-platform-detailed-design.md) | 告警平台详细设计 |
| [log-platform-api.md](log-platform-api.md) | 日志平台 API |
| [log-platform-es.md](log-platform-es.md) | Loggie + Elasticsearch 架构说明 |

---

## API 与部署

| 文档 | 说明 |
|------|------|
| [apipost/README.md](apipost/README.md) | OpenAPI 集合说明 |
| [deployment/KYLIN_V10_X86_64.md](deployment/KYLIN_V10_X86_64.md) | 麒麟部署示例 |
| [cicd-jenkins-node-tool.md](cicd-jenkins-node-tool.md) | Jenkins Node 工具说明 |
| [统一模型-服务目录与变更事件.md](统一模型-服务目录与变更事件.md) | 服务目录与变更事件模型 |

---

## 维护约定

- Service 源码路径以 `internal/service/<域>/` 为准；Handler 门面见 `internal/service/exports.go`。
- 内置菜单单一数据源：`internal/menu/catalog.go`；同步逻辑 `internal/menu/sync.go`；`seed` 调用 `menu.Sync`。
- 路由按插件注册：`internal/router/plugin_bind.go` + `register_<plugin>_routes.go`。
- 目录或装配变更时，同步更新 **CODEBASE-MAP.md**、**plugins.md**、**dbmgmt.md**（dbmgmt 域变更时）、**backend-architecture-complete.md §3.3**、**refactoring-report.md**。
- 交付文档变更时，同步更新本索引「项目交付文档」表。
- 已清理：历史 SRS/审查快照、CSDN 博文、Jenkins 实践博文、未实施设计草图、`docs/html` 导出镜像，以及根目录过时的 `README`（无扩展名）/`CODE_REVIEW_AND_FIXES.md`/`DEPLOY_ANALYSIS.md`。
