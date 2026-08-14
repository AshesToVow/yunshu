# Kubernetes 排障知识（种子）

## 工具顺序（Pod 异常）

1. `list_clusters` 确认 `cluster_id`
2. `diagnose_pod` 获取状态/Reason/重启
3. `get_pod_logs` / `list_events` / `get_pod_detail` 补证据
4. `run_diagnose_runbook` 对齐典型场景

## CrashLoopBackOff

- 看退出码与上一容器日志
- 探针过严、配置错误、依赖不可达、OOM

## ImagePullBackOff

- 镜像地址/Tag、imagePullSecrets、仓库网络与 Harbor 权限

## Pending

- FailedScheduling、requests、污点/亲和、PVC、Quota

## OOMKilled

- memory limit、泄漏、滚动重启需审批

## 写操作

`scale_deployment` / `restart_deployment` / `delete_pod` 仅创建审批单，禁止声称已变更。
