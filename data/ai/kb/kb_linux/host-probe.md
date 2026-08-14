# Linux / 主机探测知识（种子）

## 脚本工具（只读）

| 工具 | 用途 |
|------|------|
| `linux.disk.check` | 路径磁盘用量（statvfs） |
| `linux.mem.check` | 内存概况 |
| `linux.load.check` | 负载与 CPU 核数 |

输入 JSON 经 stdin；输出 JSON。默认探测**工具运行环境**（AI 沙箱机），不是任意远端主机。

## 远端主机

要对 CMDB 纳管服务器做文件/命令操作：使用「服务器管理 → 连接 → 操作台」（SSH/SFTP）。AI 脚本工具当前不直连任意主机。

## 磁盘打满建议

先 `linux.disk.check` 看 used_ratio；清理与扩容需人工在主机上执行并二次确认路径。
