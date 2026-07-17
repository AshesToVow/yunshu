# M-10 总览仪表盘

| 项 | 内容 |
|----|------|
| 文档编号 | M-10 |
| 模块名称 | 总览仪表盘 |
| 插件 | 核心；图表可依赖 `cicd` / `k8s` / 日志 Agent |
| 前端 | `/` → `dashboard-page` |
| 后端 | `internal/service/overview` |
| 状态 | 已对齐源码 |

## 1. 目标

首页聚合平台关键计数与趋势：项目、集群、告警、Loggie 在线/离线、CI 上线趋势等，便于值班一览。

## 2. 功能需求

| ID | 功能 |
|----|------|
| F-01 | 概览卡片指标 |
| F-02 | 项目上线序列（依赖 cicd 时） |
| F-03 | 按人发布统计（依赖 cicd 时） |

## 3. 接口规格

| 方法 | 路径 | 鉴权 | 入参 | 结果 `data` |
|------|------|------|------|-------------|
| GET | `/api/v1/overview` | 是 | — | 各业务计数/状态（含 loggie online/offline 等） |
| GET | `/api/v1/overview/project-launches` | 是 | — | 上线时间序列 |
| GET | `/api/v1/overview/release-by-person` | 是 | — | 按人汇总 |

插件未启用时，对应字段返回空列表或 0，不报错。

## 4. 数据来源

| 指标来源 | 说明 |
|----------|------|
| MySQL 各业务表 | 项目、用户、告警等 count |
| `loggie_agents` | 心跳超时判定在线 |
| K8s Runtime | Pod 聚合（可选） |
| `cicd_release_runs` | 上线图表 |

## 5. 相关文档

- [menu-root-dashboard.md](../menus/menu-root-dashboard.md)
