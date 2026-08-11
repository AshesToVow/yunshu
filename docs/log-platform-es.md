# 日志平台：Loggie Agent + Elasticsearch

## 概述

1. **Loggie 主机 Agent**（SSH/systemd）：采集虚机/物理机文件日志  
   - `kafka.enabled=false`：bulk 直写 ES  
   - `kafka.enabled=true`：写入 Kafka，由 Yunshu 消费后 bulk 写入 ES  
2. **Loggie 集群 DaemonSet**（与主机并列）：采 `/var/log/pods`，规则按 namespace/workload，**不绑 server_id**  
3. **Elasticsearch**：主机索引 `yunshu-agent-{ip}-*`；集群索引 `yunshu-k8s-{clusterId}-p{projectId}-*`（按项目隔离）
4. **Yunshu**：项目管主机 Agent 与集群规则/下发；代理检索与保留策略  

不依赖 Grafana / Loki。旧 K8s ClusterLogConfig 引导已移除，改由「服务与日志采集 → 集群采集」下发 DaemonSet（见 `deploy/loggie/daemonset/README.md`）。**同集群多项目**使用独立 DaemonSet 名与索引分片，互不覆盖。

## Kafka 中转（可选）

数据字典（`kafka_*`，优先于 `configs/config.yaml` 的 `kafka:`）：

| dict_type | 说明 |
|-----------|------|
| `kafka_enabled` | `true` 启用中转；`false` 直写 ES |
| `kafka_brokers` | 单节点或集群（JSON 数组 / 逗号分隔） |
| `kafka_topic` / `kafka_topic_prefix` | Agent Topic 前缀，默认 `yunshu-agent`；实际 Topic=`yunshu-agent-{ip}-YYYY.MM.DD` |
| `kafka_consumer_group` | 默认 `yunshu-log-es` |
| `kafka_username` / `kafka_password` / `kafka_sasl_mechanism` | 可选 SASL |

开启后需 **重新热更/下发** Agent pipeline，并保证 `elasticsearch.enabled=true`（消费端写 ES）。观测页：**日志平台 → 保留策略 → Kafka 队列**。

## 索引约定

| 写入（Loggie sink） | 检索 / 保留 |
|---------------------|-------------|
| `yunshu-agent-{服务器IP}-YYYY.MM.DD`（IP 中 `.`→`-`） | 单机：`yunshu-agent-{ip}-*`；全局：`yunshu-agent-*`；过渡期兼容旧 `yunshu-agent-{server_id}-*` |

`fields`（CMDB 标签，推荐 string）：

- `project_id` / `project_code` / `project_name`
- `server_id` / `server_host` / `server_name`
- `service_id` / `service_name`
- `log_source_id`
- `collector_mode`：主机 `host`，集群 `k8s`
- 采集态：`file_path`、`host`（来自 addonMeta）

历史 `yunshu-logs-*` **不会自动迁移**；默认检索主机 `yunshu-agent-*` 与集群 `yunshu-k8s-*`（可用 `collector_mode` 收窄）。

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

## 保留策略

默认清理 `yunshu-agent-*` 与 `yunshu-k8s-*` 过期日索引；服务器作用域策略仅针对对应主机索引。可为策略显式指定 `index_pattern`。

## 集群采集限流与护栏

- DaemonSet / SA / ConfigMap：`yunshu-loggie-p{projectId}`（同集群多项目互不覆盖；旧名 `yunshu-loggie` 需手工清理）
- 项目级 QPS：`cluster_log_agents.rate_limit_qps`（部署时传入，默认 2000）
- **配额拆分**：规则 `rate_limit_qps>0` 固定占用；其余启用规则均分剩余配额（下限 50）
- 列表接口返回 `allocated_qps`（生效限流；前端勿本地重算）
- 宽采路径用 `excludeFiles` 排除 `kube-system` 等系统 ns；规则可覆盖 `exclude_namespaces`
- 下发 `yunshu-logging` DaemonSet 跳过命名空间白名单（`WithSkipNamespacePolicy`）
- 主机/集群 Loggie sink YAML 共用 `renderLoggieSinkYAML`；Kafka 开关对齐 `SinkViaKafka()`

## 检索

`GET /api/v1/projects/:id/logs/search`  

筛选：`collector_mode`（`host`|`k8s`）、`cluster_id`、`namespace`、`pod`、`container`，以及服务器/服务/日志源/级别/文件名/关键字。  
主机新 pipeline 写入 `collector_mode=host`（旧文档字段缺失仍可命中）；集群为 `k8s`。  
索引：主机 `yunshu-agent-*`；集群 `yunshu-k8s-{clusterId}-p{projectId}-*`；未指定 mode 时两边一起查（仍按 `project_id` 过滤）。

## 与 Loggie 手册对照

- [元信息与 fields / addonMeta](https://loggie-io.github.io/docs/user-guide/best-practice/log-enrich/)
- [discovery typePodFields](https://loggie-io.github.io/docs/reference/global/discovery/#typepodfields)
- 系统配置：`reload` + `defaults.interceptors.schema`（`@timestamp`）写入完整 `pipeline.yml`

二进制部署细节见 [deploy/loggie/binary/README.md](../deploy/loggie/binary/README.md)。
