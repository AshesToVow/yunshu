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

- 索引：`yunshu-k8s-{clusterId}-p{projectId}-*`（按项目隔离，避免同集群互盖）
- DaemonSet/ConfigMap/SA：`yunshu-loggie-p{projectId}`（同 NS `yunshu-logging` 可多项目并存）
- 平台下发跳过命名空间白名单；部署时 Ensure 当日 Kafka Topic
- 部署时可传 `rate_limit_qps`；规则变更后若已部署会自动重同步
- 列表接口返回 `allocated_qps`（生效限流，前端勿本地重算）
- 旧版 DaemonSet 名 `yunshu-loggie` 需手工删除；新名为 `yunshu-loggie-p{projectId}`
- 不要在主机日志源里再配 `/var/log/pods`，避免双采
- 主机进程日志仍用「主机日志源 + SSH Agent」
