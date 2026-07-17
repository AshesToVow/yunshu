# 服务器二进制部署：Elasticsearch + Loggie Agent

适用于物理机/虚拟机，不依赖 Kubernetes。

## 1. 组件分工

| 组件 | 职责 |
|------|------|
| **Elasticsearch** | 按 Agent 分索引存储：`yunshu-agent-{server_id}-YYYY.MM.DD` |
| **Loggie** | 采集、解析、上报（systemd） |
| **Yunshu** | 离线包安装 / 启停 / 热更、代理检索、保留策略 |

控制台：**日志平台 → Agent 管理**。

## 2. 离线包（重要）

将可执行文件放到本目录并命名为 **`loggie`**（无后缀）：

```
deploy/loggie/binary/loggie
```

Yunshu 安装 Agent 时从此文件 **SFTP 上传**到目标机，**不再在线下载**。

配置（`configs/config.yaml`）：

```yaml
loggie:
  offline_binary_path: "deploy/loggie/binary/loggie"
  unit_name: "loggie.service"
  deploy_dir: "/export/loggie"
```

Docker 镜像构建会把该文件打进 `/app/deploy/loggie/binary/loggie`。

官方 Linux amd64 可用 [Loggie Releases](https://github.com/loggie-io/loggie/releases) 中的裸二进制。arm64 需自行编译同名文件覆盖。

安装前请确认目标机：`uname -m` 与二进制架构一致。

## 3. Elasticsearch

版本 **7.x / 8.x**。验证：`curl -s http://ES:9200 | grep number`

## 4. 通过 Yunshu 安装（推荐）

1. CMDB：服务器 → **服务与日志源**（同一页两个 Tab）  
2. **Agent 管理** → **添加 Agent**（或行内「引导」）→ 填写 **后端 API 地址**（如 `http://IP:8080`，不是前端端口）  
3. **一键安装** / **安装**（上传离线包 + 启用 systemd）  
4. 状态列查看：在线、采集、ES Sink  

安装会启用：

- `loggie.service` — 采集进程  
- `loggie-heartbeat.timer` — 每 60s 心跳（`heartbeat.sh` + `loggie-heartbeat.env`）

热更 / 启动 / 重启会 **同时重启心跳 timer**，避免 Token 更新后平台显示离线。

### 目标机目录结构

```
/export/loggie/
  loggie                      # 二进制
  pipeline.yml                # 系统配置（-config.system）
  pipelines.yml               # 采集管道（-config.pipeline）
  start.sh                    # 手工调试
  heartbeat.sh
  loggie-heartbeat.env
  loggie.service
  loggie-heartbeat.service
  loggie-heartbeat.timer
```

**两个 yml 都要保留。** 生产用 `systemctl`；`start.sh` 仅调试。

## 5. 手动安装（备选）

```bash
sudo cp loggie.service /etc/systemd/system/loggie.service
sudo cp loggie-heartbeat.service /etc/systemd/system/
sudo cp loggie-heartbeat.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now loggie loggie-heartbeat.timer
sudo systemctl start loggie-heartbeat.service
```

## 6. 验证

```bash
systemctl status loggie
systemctl status loggie-heartbeat.timer
curl -s http://127.0.0.1:9196/api/v1/help/log | head
curl 'http://ES:9200/yunshu-agent-*/_count'
```

元信息说明见 [docs/log-platform-es.md](../../../docs/log-platform-es.md)。
