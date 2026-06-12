# 菜单需求：组件状态（`/component-status`）

## 1. 定位

- **路由**：`/component-status`，`component-status-page`。  
- **目标**：查看集群 **节点 Ready 状态** 与 **kube-system 控制平面 Pod**（apiserver / controller-manager / scheduler / etcd）运行健康。

> 自 Kubernetes 1.19+ 起 `ComponentStatus` API 已废弃；后端改为 Node conditions + 控制平面 Pod 聚合，条目 `name` 形如 `node/<nodeName>`、`pod/kube-system/<podName>`。

## 2. API

- `GET /api/v1/clusters/:id/component-statuses`（需先选集群 ID，与页面参数一致）。

## 3. 注意事项

- 仅作运维参考；异常需结合集群真实日志与云厂商面板。
- 托管集群（EKS/ACK 等）可能看不到全部控制平面 Pod，属正常现象。
