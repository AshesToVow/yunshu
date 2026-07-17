# M-04 日志平台（Loggie Agent + Elasticsearch）

| 项 | 内容 |
|----|------|
| 文档编号 | M-04 |
| 模块名称 | 日志平台 |
| 插件 | `project`（日志相关路由随项目域） |
| 前端 | `/log-platform/*`：服务与日志源、日志检索、保留策略、Agent 管理 |
| 后端 | `internal/service/logplatform`、`handler/loggie_handler.go`、`log_platform_handler.go` |
| 状态 | **已对齐 Loggie+ES（非旧 gRPC Agent）** |

## 1. 背景与目标

1. 在目标机安装 **Loggie**，采集文件日志并写入 **Elasticsearch**。  
2. 索引策略：`yunshu-agent-{server_id}-YYYY.MM.DD`。  
3. Yunshu 负责：CMDB 服务/日志源、Agent 引导/离线安装/启停/热更、心跳、检索、导出、保留清理。

**非目标**：不再使用平台自研 gRPC log-agent；不依赖 Grafana/Loki。

## 2. 功能需求

| ID | 功能 | 菜单 |
|----|------|------|
| F-01 | 服务与日志源配置（同页 Tab） | `/project-services` |
| F-02 | ES 日志检索与导出 | `/project-logs` |
| F-03 | 保留策略与 ES 全量索引统计 | `/log-retention` |
| F-04 | Agent 添加/引导/离线安装/热更/启停/卸载 | `/loggie-status` |
| F-05 | Agent 心跳上报 | 公开 API |

## 3. 接口规格

### 3.1 服务与日志源

| 方法 | 路径 | 入参 | 结果 |
|------|------|------|------|
| GET | `/api/v1/projects/:id/services` | `page,server_id?` | `{ list }` |
| POST | `/api/v1/projects/:id/services` | Upsert：`server_id,name,env,status...` | 服务 |
| DELETE | `/api/v1/projects/:id/services/:serviceId` | — | ok |
| GET | `/api/v1/projects/:id/log-sources` | `service_id?` | `{ list }` |
| POST | `/api/v1/projects/:id/log-sources` | Upsert：`service_id,log_type,path,include/exclude...` | 日志源 |
| DELETE | `/api/v1/projects/:id/log-sources/:logSourceId` | — | ok |

### 3.2 日志检索 / 导出

| 方法 | 路径 | Query | 结果 |
|------|------|-------|------|
| GET | `/api/v1/projects/:id/logs/search` | `server_id,service_id,log_source_id,keyword,level,file_path,from,to,page,page_size` | `{ list,total,page,page_size }` |
| GET | `/api/v1/projects/:id/logs/export` | 同上（`page_size` 默认取 ES `max_size`） | **纯文本** `text/plain` 附件（非 JSON 包装） |

检索 `list` 项字段示例：`timestamp,level,message,service_name,host,file_path,namespace,podname,containername`。

### 3.3 保留策略与 ES 统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/PUT | `/api/v1/log-platform/retention` | 全局保留策略 |
| GET | `/api/v1/log-platform/retention/list` | 策略列表 |
| POST | `/api/v1/log-platform/retention/cleanup` | 立即清理 |
| GET | `/api/v1/log-platform/es-storage` | **全量索引统计**（见下） |
| GET | `/api/v1/log-platform/es-config` | ES 配置预览 |
| GET/PUT | `/api/v1/projects/:id/log-retention` | 项目覆盖 |
| DELETE | `/api/v1/projects/:id/log-retention` | 删除项目覆盖 |

#### `GET /es-storage` 成功 `data`

| 字段 | 类型 | 说明 |
|------|------|------|
| `index_pattern` | string | 配置模式，如 `yunshu-agent-*` |
| `index_count` | int | **全部**非系统索引数 |
| `document_count` / `store_bytes` / `store_human` | | 全量汇总 |
| `pattern_index_count` 等 | | 匹配 pattern 的子集汇总 |
| `indices[]` | array | `{ name, docs_count, store_bytes, store_human, matched_pattern }` |

### 3.4 Loggie Agent 生命周期

| 方法 | 路径 | Body 要点 | 结果 |
|------|------|-----------|------|
| POST | `/api/v1/loggie/heartbeat/report` | `token` + 健康/监控字段 | `{ message: ok }`（**无 JWT**） |
| GET | `/api/v1/projects/:id/loggie/status` | — | `{ list: LoggieStatusItem[] }` |
| GET | `.../loggie/bootstrap-sources?server_id=` | — | 日志源预览 |
| POST | `.../loggie/bootstrap` | `server_id,yunshu_url,deploy_dir,monitor_port,auto_from_log_sources,deploy_after_bootstrap` | Token + pipeline YAML |
| GET | `.../loggie/pipeline/download?server_id=&file=` | `file=pipeline\|pipelines\|env\|heartbeat\|start` | 文件下载 |
| POST | `.../loggie/install` | `server_id,deploy_dir,yunshu_url,monitor_port` | 离线包上传 + systemd；超时建议 ≥300s |
| POST | `.../loggie/uninstall` | `server_id,skip_remote?,force_local?,keep_files?` | 卸载 + 清登记 |
| POST | `.../loggie/deploy` / `sync` / `start` / `stop` / `restart` | `{ server_id }` | `LoggieDeployResult` |

#### `yunshu_url` 约定

填 **后端 API 根地址**（如 `http://IP:8080`），供心跳脚本调用 `/api/v1/loggie/heartbeat/report`；**不是**前端 Vite 端口。

#### 安装二进制

平台从本机 **`deploy/loggie/binary/loggie`**（配置 `loggie.offline_binary_path`）SFTP 上传，**不在线下载**。

启停/热更须同步重启 `loggie-heartbeat.timer`，否则平台显示离线。

## 4. 数据模型

| 表 / 存储 | 说明 |
|-----------|------|
| `services` | 项目服务 |
| `service_log_sources` | 日志源路径/规则 |
| `loggie_agents` | Token、心跳、监控快照、`bootstrap_config` |
| `log_retention_policies` | 全局/项目/服务器保留策略 |
| Elasticsearch | 日志正文；索引 `yunshu-agent-{server_id}-*` |

### `loggie_agents` 关键字段

`project_id, server_id, token, health_status, pipeline_status, es_sink_ok, last_seen_at, monitor_port, bootstrap_config`。

## 5. 配置

```yaml
elasticsearch:
  enabled: true
  addresses: ["http://ES:9200"]
  index_pattern: "yunshu-agent-*"
  default_retention_days: 30

loggie:
  offline_binary_path: "deploy/loggie/binary/loggie"
  unit_name: "loggie.service"
  deploy_dir: "/export/loggie"
```

## 6. 验收标准

- [ ] 离线包缺失时安装返回明确错误  
- [ ] 引导后热更/重启会刷新心跳 timer  
- [ ] 检索与导出筛选条件一致；导出为可下载 txt  
- [ ] ES 统计展示全量索引表 + pattern 子集汇总  
- [ ] 卸载可清远端与登记（或仅清登记）  

## 7. 相关文档

- [log-platform-es.md](../../../log-platform-es.md)
- [deploy/loggie/binary/README.md](../../../../deploy/loggie/binary/README.md)
- [R-06-log-platform-and-agent.md](../R-06-log-platform-and-agent.md)（已指向本模块）
