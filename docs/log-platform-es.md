# 日志平台：Loggie Agent + Elasticsearch

## 概述

1. **Loggie**（目标机 Agent）：采集文件日志  
   - `kafka.enabled=false`：bulk 直写 ES  
   - `kafka.enabled=true`：写入 Kafka，由 Yunshu 消费后 bulk 写入 ES  
2. **Elasticsearch**：按 **Agent（服务器）分索引** 持久化  
3. **Yunshu**：项目/CMDB 管 Agent 生命周期，代理检索与保留策略；可选 Kafka→ES 消费与积压观测  

不依赖 Grafana / Loki；K8s ClusterLogConfig 引导已移除。若日志路径为 `/var/log/pods/...`，pipeline 会按 [typePodFields](https://loggie-io.github.io/docs/reference/global/discovery/#typepodfields) 风格从路径抽取 `namespace` / `podname` / `containername`。

## Kafka 中转（可选）

数据字典（`kafka_*`，优先于 `configs/config.yaml` 的 `kafka:`）：

| dict_type | 说明 |
|-----------|------|
| `kafka_enabled` | `true` 启用中转；`false` 直写 ES |
| `kafka_brokers` | 单节点或集群（JSON 数组 / 逗号分隔） |
| `kafka_topic_prefix` | Topic 前缀，默认 `yunshu-agent`；实际 Topic=`{prefix}-{server_id}` |
| `kafka_consumer_group` | **必填**消费组，默认 `yunshu-log-es` |
| `kafka_username` / `kafka_password` / `kafka_sasl_mechanism` | 可选 SASL |

开启后需 **重新热更/下发** Agent pipeline（引导/热更时自动创建该 Agent 的 Topic）。观测：**日志平台 → 保留策略 → Kafka 队列** Tab。

## 索引约定

| 写入（Loggie sink） | 检索 / 保留 |
|---------------------|-------------|
| `yunshu-agent-{server_id}-${+YYYY.MM.DD}` | 单机：`yunshu-agent-{server_id}-*`；项目多机：拼接各 server 前缀；全局：`yunshu-agent-*` |

`fields`（CMDB 标签，推荐 string）：

- `project_id` / `project_code` / `project_name`
- `server_id` / `server_host` / `server_name`
- `service_id` / `service_name`
- `log_source_id`
- 采集态：`file_path`、`host`（来自 addonMeta）

历史 `yunshu-logs-*` **不会自动迁移**；默认同检索只查 `yunshu-agent-*`。

## Agent 生命周期（项目作用域）

| API | 说明 |
|-----|------|
| `POST .../loggie/bootstrap` | 登记 Token + 生成 pipeline |
| `POST .../loggie/install` | SSH 拉二进制、装 systemd、首启 |
| `POST .../loggie/deploy` / `sync` | 热更 `pipelines.yml`（reload-or-restart） |
| `POST .../loggie/start\|stop\|restart` | systemd 启停 |
| `GET .../loggie/status` | 在线 / 采集（近窗 ES）/ 监控探测 |
| `POST /api/v1/loggie/heartbeat/report` | Agent 心跳（公开） |

配置：

```yaml
elasticsearch:
  enabled: true
  addresses: ["http://es:9200"]
  index_pattern: "yunshu-agent-*"

loggie:
  offline_binary_path: "deploy/loggie/binary/loggie"
  unit_name: "loggie.service"
  deploy_dir: "/export/loggie"
```

## 检索

`GET /api/v1/projects/:id/logs/search`  
筛选：服务器、服务、日志源、级别、文件名、关键字；结果含服务名、主机、K8s 元信息。

## 保留策略

默认清理 `yunshu-agent-*` 过期日索引；可为单服务器策略指定 `yunshu-agent-{id}-*`。

## 与 Loggie 手册对照

- [元信息与 fields / addonMeta](https://loggie-io.github.io/docs/user-guide/best-practice/log-enrich/)
- [discovery typePodFields](https://loggie-io.github.io/docs/reference/global/discovery/#typepodfields)
- 系统配置：`reload` + `defaults.interceptors.schema`（`@timestamp`）写入完整 `pipeline.yml`

二进制部署细节见 [deploy/loggie/binary/README.md](../deploy/loggie/binary/README.md)。
