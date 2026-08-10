# PendingUnschedulable 排障剧本

## 目标
定位 Pod 无法调度的原因（资源、污点、亲和性、PVC）。

## 检查步骤
1. 确认 Phase=Pending，查看 Conditions（PodScheduled）。
2. Events：FailedScheduling 消息（insufficient cpu/memory、taints、node affinity、PVC）。
3. 核对 requests 是否过大；节点可分配资源。
4. 核对 tolerations 与 nodeSelector/affinity。
5. 若依赖 PVC：StorageClass、PV 是否 Bound。

## 输出要求
- 明确调度失败主因
- 建议只读验证命令与配置检查项
