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
3. 填写 `loggie.binary_url`（或引导表单 URL，支持 `{arch}`）→ **安装**  
4. 状态列查看：在线、采集（近 5 分钟 ES 文档）、ES Sink  

热更采集配置：改日志源后点 **热更**（仅下发 `pipelines.yml` + reload-or-restart）。

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
  binary_url: "https://.../loggie_linux_{arch}.tar.gz"
  deploy_dir: "/export/loggie"
```

## 6. 验证

```bash
curl -s http://127.0.0.1:9196/api/v1/help/log | head
curl 'http://ES:9200/yunshu-agent-*/_count'
```

元信息字段说明见 [docs/log-platform-es.md](../../../docs/log-platform-es.md) 与 [Loggie 元信息文档](https://loggie-io.github.io/docs/user-guide/best-practice/log-enrich/)。
