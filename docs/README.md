# Yunshu 文档索引

**最后更新**: 2026-05-22

---

## 开发者必读（后端）

| 文档 | 说明 |
|------|------|
| [CODEBASE-MAP.md](CODEBASE-MAP.md) | **代码地图**：目录结构、阅读顺序、Wire/错误规范 |
| [backend-architecture-complete.md](backend-architecture-complete.md) | 后端完整技术文档（架构、模块、部署） |
| [refactoring-report.md](refactoring-report.md) | 架构重构实施状态（与代码同步） |
| [architecture-diagrams.md](architecture-diagrams.md) | 架构图集 |
| [code-review-report.md](code-review-report.md) | 2026-05-21 审查快照（文首标注已解决项） |

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
| [requirements/R-alert-platform-detailed-design.md](requirements/R-alert-platform-detailed-design.md) | 告警平台详细设计 |
| [log-platform-api.md](log-platform-api.md) | 日志平台 API |

---

## API 与部署

| 文档 | 说明 |
|------|------|
| [apipost/README.md](apipost/README.md) | OpenAPI 集合说明 |
| [deployment/KYLIN_V10_X86_64.md](deployment/KYLIN_V10_X86_64.md) | 麒麟部署示例 |
| [SRS-Yunshu-Requirements-Specification.md](SRS-Yunshu-Requirements-Specification.md) | 需求规格说明 |

---

## 维护约定

- Service 源码路径以 `internal/service/<域>/` 为准；Handler 门面见 `internal/service/exports.go`。
- 目录或装配变更时，同步更新 **CODEBASE-MAP.md**、**backend-architecture-complete.md §3.3**、**refactoring-report.md**。
