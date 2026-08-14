# CrashLoopBackOff 排障剧本

## 目标
定位容器反复崩溃原因并给出可验证修复建议（只分析，不自动变更）。

## 检查步骤
1. 查看 Pod 阶段、Ready、重启次数与最近终止原因（OOMKilled / Error / Completed）。
2. 拉取当前与 previous 容器日志尾部，定位 panic、配置错误、依赖连不上。
3. 检查 Events：BackOff、Failed、Unhealthy、Killing。
4. 核对探针（liveness/readiness）是否过严导致自杀循环。
5. 核对资源 requests/limits，是否 OOM。
6. 核对配置/密钥挂载是否缺失或权限错误。

## 输出要求
- 根因按置信度排序
- 给出最小验证步骤（日志关键字、kubectl 只读命令建议）
- 不要编造不存在的容器名
