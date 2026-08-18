# kube-consul-register：业务 Pod 必填注解 / 标签

控制器：[AshesToVow/kube-consul-register](https://github.com/AshesToVow/kube-consul-register)  
清单：`deploy/monitoring/kube-consul-register/`  
YAML 片段：[`k8s-consul-register.snippet.yaml`](./k8s-consul-register.snippet.yaml)（同步到 Jenkins `k8s-basic-*.yaml` / `k8s-skywalking-*.yaml` 的 **Pod 模板**）

Helm 脚手架已默认注入，见 `deployment-base.consulRegister`。

## 必填（不写则注册失败或 Yunshu/Prom 看不见）

| 位置 | Key | 值 | 不写会怎样 |
|------|-----|----|------------|
| **注解** | `consul.register/enabled` | `"true"` | 控制器跳过，Consul 无记录 |
| **注解** | `consul.register/service.name` | `k8s-pod` 或 `k8s-pod-metrics` | 服务名变成 Deployment 名，metrics-register ACL 常 403 |
| **标签** | `yunshu-metrics` | `"tag"`（必须是这三个字母） | Prom / Yunshu 按 tag 过滤时看不到 |

写在 `spec.template.metadata`，不要写在 Deployment 顶层 `metadata`。

## 按场景

| 场景 | `service.name` | 额外必填 |
|------|----------------|----------|
| 只进 Consul 目录 | `k8s-pod` | 无；建议关掉 check |
| Prometheus 刮指标 | `k8s-pod-metrics` | `consul.register/service.meta.metrics_path`（如 `/metrics`） |
| 多容器 / SkyWalking sidecar | 同上 | `consul.register/pod.container.name` = 主容器名 |

## 可选注解（重写版）

| 注解 | 默认 | 说明 |
|------|------|------|
| `consul.register/pod.container.name` | 全部 `spec.containers` 各注册一份 | 逗号分隔多个名字 |
| `consul.register/liveness.check.enabled` | 不启用 | 有真实 livenessProbe 再开 |
| `consul.register/readiness.check.enabled` | 不启用 | 有真实 readinessProbe 再开 |
| `consul.register/service.meta.<key>` | 无 | 静态 Meta；不能写运行时 PodIP |

## 标签 → Consul tag

值为字面量 `tag` 的 Pod label，**key** 会成为 Consul tag：

```yaml
labels:
  yunshu-metrics: "tag"   # Consul tag = yunshu-metrics
  # production: "tag"     # 额外 tag = production
```

控制器 ConfigMap 的 `k8s_tag`（现网应为 `k8s`）会自动打到服务上，业务 YAML **不必**再写 `k8s: "tag"`。不要把 `k8s_tag` 配成 `yunshu-metrics`，Clean 会误删 telegraf。

## Helm

```yaml
deployment-base:
  consulRegister:
    enabled: true
    mode: catalog          # 或 metrics
    # serviceName: ""      # 空则 catalog→k8s-pod，metrics→k8s-pod-metrics
    containerName: ""      # sidecar 时填主容器名
    yunshuProject: "1"
```

关闭：`--set deployment-base.consulRegister.enabled=false`
