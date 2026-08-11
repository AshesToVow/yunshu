# K8s 集群日志采集（DaemonSet）

与主机 SSH Loggie **并列**：本目录说明由 Yunshu「集群采集」一键下发的形态。

## 模型

- 规则：`cluster_log_rules`（按 namespace / workload，不绑 `server_id`）
- Agent 登记：`cluster_log_agents`
- 索引 / Topic：`yunshu-k8s-{clusterId}-YYYY.MM.DD`
- 写入字段：`collector_mode=k8s`、`cluster_id`、`namespace`/`podname`/`containername`（从路径抽取）

## 部署

平台 API：

- `POST /api/v1/projects/:id/cluster-log/deploy` `{ "cluster_id": N, "namespace": "yunshu-logging" }`
- 会 Apply：Namespace + ServiceAccount + ConfigMap + DaemonSet `yunshu-loggie`
- 镜像默认：`ghcr.io/loggie-io/loggie:v1.7.1`（`configs/config.yaml` → `loggie.daemonset_image`）

hostPath：

- `/var/log/pods`
- `/var/log/containers`
- 数据目录 `/var/lib/yunshu-loggie`（与主机 Agent `/export/loggie` 错开）

## 注意

- 若集群开启命名空间白名单：平台下发 `yunshu-logging` 会跳过 NS 策略校验；业务资源仍受白名单约束
- 部署时可传 `rate_limit_qps`（项目级采集限流，写入 Loggie rateLimit）
- 宽采自动 `excludeFiles` 排除 kube-system 等系统命名空间
- 不要在主机日志源里再配 `/var/log/pods`，避免双采
- 主机进程日志仍用「主机日志源 + SSH Agent」
