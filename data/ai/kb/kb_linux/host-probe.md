# Linux / 主机探测知识（种子）

## 本机脚本工具（只读，AI 运行环境）

| 工具 | 用途 |
|------|------|
| `linux.disk.check` | 路径磁盘用量（statvfs）— **仅 AI 后端/容器本机** |
| `linux.mem.check` | 内存概况 — 同上 |
| `linux.load.check` | 负载与 CPU — 同上 |

输入 JSON 经 stdin；输出 JSON。默认探测**工具运行环境**，不是业务远端主机。

## 远端 CMDB 主机（推荐）

| 工具 / 入口 | 用途 |
|------|------|
| `probe_server_metrics` | 经 SSH 只读探测磁盘/内存/负载（需 `project_id` + `server_id` + exec 权限） |
| API `POST /projects/:id/servers/:serverId/probe` | 同上，前端/集成可直接调用 |
| 服务器管理 → 操作台 | 交互式命令 / SFTP |

查业务机 `/export` 等路径时，必须用 `probe_server_metrics`，不要用 `linux.disk.check`。

## 磁盘打满建议

1. `list_servers` 定位主机 → `probe_server_metrics`（kind=disk, path=/export）
2. 对照 `list_change_events` 看近期发布/变更
3. 清理与扩容需人工在主机上执行并二次确认路径
