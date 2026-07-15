# 服务器二进制部署：Elasticsearch + Loggie Agent

适用于物理机/虚拟机，不依赖 Kubernetes。

## 1. 组件分工

| 组件 | 职责 |
|------|------|
| **Elasticsearch** | 按 Agent 分索引存储：`yunshu-agent-{server_id}-YYYY.MM.DD` |
| **Loggie** | 采集、解析、上报（systemd） |
| **Yunshu** | 项目侧一键安装/启停/热更、代理检索、保留策略 |

控制台：**日志平台 → Agent 管理**。

## 2. Elasticsearch

版本 **7.x / 8.x**。验证：`curl -s http://ES:9200 | grep number`

## 3. 通过 Yunshu 安装（推荐）

1. CMDB 配置项目服务器、服务、日志源路径  
2. **Agent 管理** → 选择项目 → **引导**（生成 Token / pipeline）  
3. 二进制直链默认  
   `https://github.com/loggie-io/loggie/releases/download/v1.5.0/loggie`  
   （裸二进制，安装到目录下的 `loggie` 文件）→ **安装**  
4. 状态列查看：在线、采集、ES Sink  

### 目标机目录结构

```
/export/loggie/
  loggie                 # 二进制
  pipeline.yml           # 系统配置（含 reload/monitor）
  pipelines.yml          # 采集管道
  start.sh               # 前台启动脚本
  heartbeat.sh           # 心跳
  loggie-heartbeat.env
  loggie.service         # systemd 单元（安装时拷到 /etc/systemd/system）
```

热更：改日志源后点 **热更**（下发 yml/脚本 + `systemctl reload-or-restart`）。

## 4. 手动安装（备选）

```bash
# 将 Yunshu 下载的 pipeline.yml / pipelines.yml / heartbeat 放到部署目录
sudo cp loggie.service /etc/systemd/system/loggie.service
sudo systemctl daemon-reload
sudo systemctl enable --now loggie
```

完整 `pipeline.yml` 含：

- `reload.enabled: true`
- `defaults.interceptors` schema `@timestamp`
- 每日志源一条 pipeline，`fields` 含项目/服务器/服务名
- sink.index：`yunshu-agent-{server_id}-${+YYYY.MM.DD}`

## 5. Yunshu 配置

```yaml
elasticsearch:
  enabled: true
  addresses: ["http://ES_IP:9200"]
  index_pattern: "yunshu-agent-*"
  default_retention_days: 30

loggie:
  binary_url: "https://github.com/loggie-io/loggie/releases/download/v1.5.0/loggie"
  deploy_dir: "/export/loggie"
```

前台启动：`/export/loggie/start.sh`  
systemd：`ExecStart=/export/loggie/loggie -config.system=... -config.pipeline=...`


## 6. 验证

```bash
curl -s http://127.0.0.1:9196/api/v1/help/log | head
curl 'http://ES:9200/yunshu-agent-*/_count'
```

元信息字段说明见 [docs/log-platform-es.md](../../../docs/log-platform-es.md) 与 [Loggie 元信息文档](https://loggie-io.github.io/docs/user-guide/best-practice/log-enrich/)。
