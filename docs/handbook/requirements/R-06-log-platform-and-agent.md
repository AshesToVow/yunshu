# 需求说明：日志平台与 Agent

> **演进说明（2026-07）**：原「自研 gRPC log-agent」方案已废弃。现行方案为 **Loggie 二进制 Agent + Elasticsearch**。  
> **标准模块开发需求请以 [modules/M-04-log-platform.md](./modules/M-04-log-platform.md) 为准。**

## 1. 目标

在项目服务器上部署 **Loggie**，采集日志写入 ES；Yunshu 提供服务/日志源配置、Agent 生命周期、检索导出与保留策略。

## 2. 子功能

| 子功能 | 说明 |
|--------|------|
| 服务与日志源 | 同页 Tab；绑定服务器路径 |
| Agent 引导/安装 | Token + pipeline；离线包 `deploy/loggie/binary/loggie` |
| 心跳 | `POST /api/v1/loggie/heartbeat/report` + systemd timer |
| 检索/导出 | ES `yunshu-agent-*` |
| 保留与 ES 统计 | 日索引清理；全量索引列表 |

## 3. 注意事项

- `yunshu_url` 填**后端**地址。  
- 热更/重启须同步心跳 timer。  
- 官方/离线包架构需与目标机 `uname -m` 一致。

## 4. 相关文档

- [M-04-log-platform.md](./modules/M-04-log-platform.md)
- [docs/log-platform-es.md](../../log-platform-es.md)
- [deploy/loggie/binary/README.md](../../../deploy/loggie/binary/README.md)
