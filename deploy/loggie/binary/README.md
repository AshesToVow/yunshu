# 服务器二进制部署：Elasticsearch + Loggie

适用于在物理机/虚拟机直接安装，不依赖 Kubernetes。

## 1. 组件分工

| 组件 | 职责 | 运维入口 |
|------|------|----------|
| **Elasticsearch** | 持久化存储、索引检索 | `curl localhost:9200/_cluster/health` |
| **Loggie** | 采集、解析、上报 | `loggie` 进程 / `pipeline.yml` |
| **Yunshu** | 权限代理查询、保留策略 | 控制台「日志检索」「保留策略」 |

Loggie 文档中的能力：

- **状态检查**：Agent 进程健康、与 ES Sink 连通（服务器本地排查）
- **实时上报**：持续 tail 并 bulk 写入 ES
- **pipeline.yml**：采集源、多行、正则、字段注入
- **默认正则解析**：`interceptors` / `transformer` 配置

Yunshu 管理 **ES 保留天数** 与 **查询权限**，不替代 Loggie 采集配置。

## 2. Elasticsearch

**版本要求：Elasticsearch 7.x 或 8.x**（推荐 7.17+）。Loggie v1.5 内置 `go-elasticsearch/v7` 客户端，**不支持 ES 6.x** bulk 写入；采集侧 `successEvent` 可能上涨但索引始终为空。

单节点示例 `config/elasticsearch.yml`：

```
cluster.name: yunshu-logs
discovery.type: single-node
network.host: 0.0.0.0
http.port: 9200
```

建议按日滚动索引：`yunshu-logs-2026.07.13`（Loggie sink 写法：`yunshu-logs-${+YYYY.MM.DD}`，不是 `%{+yyyy.MM.dd}`）

验证 ES 版本：`curl -s http://ES_IP:9200 | grep number`（须 ≥ 7.0.0）。

## 3. Loggie 安装

https://github.com/loggie-io/loggie/releases

> **v1.5 注意**：官方 release 二进制在部分 glibc 环境可能 CGO 崩溃，建议在目标机 `CGO_ENABLED=0 go build` 自编译，或 systemd 设置 `DBUS_SESSION_BUS_ADDRESS=disabled:`。

```bash
sudo cp pipeline.yml /etc/loggie/pipeline.yml
loggie -config.system=/etc/loggie/loggie.yml -config.pipeline=/etc/loggie/pipelines.yml
```

### transformer 正则（v1.5 语法）

```yaml
interceptors:
  - type: transformer
    actions:
      - action: regex(body)    # 不是 action: regex + source
        pattern: '...'         # 不是 regex 字段
        ignoreError: true
      - action: timestamp(ts)  # 把日志内时间写回 @timestamp
        fromLayout: "2006-01-02T15:04:05,000"
        fromLocation: Local
        toLayout: "2006-01-02T15:04:05.000Z07:00"
        toLocation: UTC
        ignoreError: true
      - action: move(ts, @timestamp)
        ignoreError: true
      - action: move(message, body)
        ignoreError: true
```

Yunshu 按日志源路径自动选择解析模板（elasticsearch / spring / syslog），引导/同步下发后生效。

`/var/log/messages`（syslog）示例：

```yaml
- action: regex(body)
  pattern: '^(?P<ts>\w+\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<message>.*)$'
  ignoreError: true
```

若 pipeline 校验失败，日志会出现 `configs invalid: expression regex is invalid`，且 `pipelines: []` 为空——进程在跑但不会采集。

**分体配置**（`loggie.yml` + `pipelines.yml`）：pipeline 文件只保留 `pipelines:` 段，系统配置写在 `loggie.yml`。syslog 完整示例见 [pipeline-syslog.yml](./pipeline-syslog.yml)。

## 4. Yunshu 配置

```yaml
elasticsearch:
  enabled: true
  addresses: ["http://ES_IP:9200"]
  index_pattern: "yunshu-logs-*"
  default_retention_days: 30
  cleanup_cron_spec: "0 3 * * *"
```

## 5. 保留策略

- 默认 **30 天**，前台「保留策略」可改
- 有日期后缀的索引：删除整索引
- 单索引：按 `@timestamp` delete_by_query

## 6. 验证

```bash
# Loggie 监控（pipeline 加载成功后应有 pipeline 名）
curl -s http://127.0.0.1:9196/api/v1/help/log | head
systemctl restart loggie && journalctl -u loggie -n 20 --no-pager

# ES
curl http://127.0.0.1:9200/_cluster/health
curl 'http://127.0.0.1:9200/yunshu-logs-*/_count'
```
