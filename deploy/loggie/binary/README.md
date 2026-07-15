# 服务器二进制部署：Elasticsearch + Loggie Agent

适用于物理机/虚拟机，不依赖 Kubernetes。

## 1. 组件分工

| 组件 | 职责 |
|------|------|
| **Elasticsearch** | 按 Agent 分索引存储：`yunshu-agent-{server_id}-YYYY.MM.DD` |
| **Loggie** | 采集、解析、上报（systemd） |
| **Yunshu** | 项目侧一键安装/启停/热更、代理检索、保留策略 |

控制台：**日志平台 → Agent 管理**。

## 2. 架构限制（重要）

官方发布的裸二进制  
`https://github.com/loggie-io/loggie/releases/download/v1.5.0/loggie`  
**仅 linux/amd64**。在 arm64 节点上会出现：

- `cannot execute binary file: Exec format error`
- systemd `status=203/EXEC`

安装前请确认：`uname -m` 为 `x86_64`。arm64 需自行编译 Loggie 并配置 `loggie.binary_url`。

## 3. Elasticsearch

版本 **7.x / 8.x**。验证：`curl -s http://ES:9200 | grep number`

## 4. 通过 Yunshu 安装（推荐）

1. CMDB 配置项目服务器、服务、日志源路径  
2. **Agent 管理** → 选择项目 → **引导**（生成 Token / pipeline）  
3. 配置 `binary_url`（默认即官方 amd64 直链）→ **安装**  
4. 状态列查看：在线、采集、ES Sink  

安装会下发配置、上传二进制，并启用：

- `loggie.service` — 采集进程  
- `loggie-heartbeat.timer` — 每 60s 跑一次 `heartbeat.sh`（读取 `loggie-heartbeat.env`）

### 目标机目录结构

```
/export/loggie/
  loggie                      # 二进制（amd64 ELF）
  pipeline.yml                # ★ 系统配置（-config.system）：reload/monitor/defaults
  pipelines.yml               # ★ 采集管道（-config.pipeline）：sources/sink
  start.sh                    # 手工调试启动（生产用 systemd）
  heartbeat.sh                # 心跳脚本
  loggie-heartbeat.env        # 心跳环境变量（YUNSHU_URL / TOKEN）
  loggie.service              # 拷贝到 /etc/systemd/system/
  loggie-heartbeat.service    # oneshot
  loggie-heartbeat.timer      # 定时触发 heartbeat
```

**两个 yml 都要保留，缺一不可：**

| 文件 | 用途 |
|------|------|
| `pipeline.yml` | `-config.system` |
| `pipelines.yml` | `-config.pipeline` |

**`start.sh` vs `loggie.service`：** 生产用 `systemctl`；`start.sh` 仅前台调试。

热更：改日志源后点 **热更**（下发 yml/脚本 + `systemctl reload-or-restart`）。

## 5. 手动安装（备选）

```bash
sudo cp loggie.service /etc/systemd/system/loggie.service
sudo cp loggie-heartbeat.service /etc/systemd/system/
sudo cp loggie-heartbeat.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now loggie loggie-heartbeat.timer
sudo systemctl start loggie-heartbeat.service   # 立即打一次心跳
```

## 6. Yunshu 配置

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

## 7. 排查 Exec format error

```bash
uname -m
file /export/loggie/loggie
ls -la /export/loggie/loggie
# 期望类似: ELF 64-bit LSB executable, x86-64
```

若 `file` 显示 ASCII/HTML，说明下载到了错误页面，需重装。  
节点上若已有 K8s DaemonSet 的 `/opt/loggie` 进程，与 Yunshu 的 `/export/loggie` 是两套，互不替代。

## 8. 验证

```bash
systemctl status loggie
systemctl status loggie-heartbeat.timer
curl -s http://127.0.0.1:9196/api/v1/help/log | head
curl 'http://ES:9200/yunshu-agent-*/_count'
```

元信息字段说明见 [docs/log-platform-es.md](../../../docs/log-platform-es.md) 与 [Loggie 元信息文档](https://loggie-io.github.io/docs/user-guide/best-practice/log-enrich/)。
