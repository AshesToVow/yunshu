# 日志平台（方案 B）：Loggie + Elasticsearch

## 概述

平台进程**仅 HTTP**（默认 `:8080`）。日志采集已完全迁移为：

1. **Loggie** 采集
2. **Elasticsearch** 存储
3. **Yunshu** 代理查询

已移除旧 `log-agent` gRPC（`:18080`）、SSE 日志流与 Agent 列表。

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:id/logs/search` | 分页检索 |
| GET | `/api/v1/projects/:id/loggie/status` | Loggie 状态（含 HTTP 监控探测） |
| POST | `/api/v1/projects/:id/loggie/bootstrap` | 引导登记 + 生成 pipeline |
| GET | `/api/v1/projects/:id/loggie/pipeline/download?server_id=` | 下载 pipeline/env/heartbeat |
| POST | `/api/v1/loggie/heartbeat/report` | Agent 心跳（含监控端口快照） |

## 配置

```yaml
elasticsearch:
  enabled: true
  addresses: ["http://es:9200"]
  index_pattern: "yunshu-logs-*"
```

**Elasticsearch 须为 7.x / 8.x**（Loggie v1.5 使用 v7 API 客户端，ES 6.x 无法写入；表现为 Loggie `successEvent` 上涨但 `yunshu-logs-*` 索引为空）。

## 保留策略

控制台 **日志平台 → 保留策略** 可配置：

- 全局默认保留天数（默认 30 天）
- 项目级覆盖
- ES 存储占用概览
- 手动触发清理

后台按 `cleanup_cron_spec`（默认每天 03:00）删除过期索引或文档。推荐 Loggie 使用按日索引 `yunshu-logs-yyyy.MM.dd`。

服务器二进制部署见 [deploy/loggie/binary/README.md](../deploy/loggie/binary/README.md)。

## 已废弃

- `cmd/logagent`、`/api/v1/agents/*`、`/logs/stream`
- 菜单「Agent 列表」
- Loggie K8s DaemonSet / ClusterLogConfig 引导（仅保留 SSH 二进制部署）
