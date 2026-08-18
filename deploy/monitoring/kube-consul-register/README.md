# K8s Pod → Consul：kube-consul-register

用 [kube-consul-register](https://github.com/tczekajlo/kube-consul-register) 监听 K8s 事件，把 Pod 注册到机房 Consul。参考实践：[CSDN 指南](https://blog.csdn.net/weixin_50902636/article/details/147460063)。

**不要与 `consul_k8s_pods_sync.py` cron 同时跑**，否则同一 Pod 会注册两份。选定本方案后停掉 cron。

Yunshu 侧约定不变：

| Consul Name | Tag | 用途 |
|-------------|-----|------|
| `k8s-pod` | `k8s` + `yunshu-metrics` | 仅目录（Yunshu 看 PodIP） |
| `k8s-pod-metrics` | `k8s` + `yunshu-metrics` | Prom job `k8s-pod` 才刮；Meta 必须有 `metrics_path` |

机房 Consul 在监控机（如 `10.10.10.5:8500`），**`register_mode=single`**。

**`k8s_tag` 必须与 telegraf/拨测区分开。** 控制器每隔 `clean-interval`（以及**启动立刻**）会：列出 Consul 里所有带 `k8s_tag` 的服务 → 不在当前 K8s Pod 列表里的一律反注册。若 `k8s_tag=yunshu-metrics`，`consul-targets-ctl.sh sync` 出来的 telegraf/icmp/http/tcp **会被清掉**，k8s-pod 仍保留。ConfigMap 请用 `k8s_tag: "k8s"`，业务 Pod 再加 label `yunshu-metrics: "tag"`（值必须是 `tag`，才会变成裸 tag `yunshu-metrics`）。

---

## 1. 构建镜像

先看 `go.mod` 的 `module` 行，两种源码 **不要混用 Go 版本**：

| 源码 | 特征 | 构建镜像 |
|------|------|----------|
| **重写版（你现在这份）** | `module kube-consul-register`，`consul/api v1.32`，`client-go v0.28`，无 2015 ugorji | **Go ≥1.22.12**（推荐 1.23.6） |
| 上游 0.1.9 | `module github.com/tczekajlo/kube-consul-register`，`client-go v2`，ugorji 2015 | **只能 Go ≤1.21**，否则 `encoding alphabet includes duplicate symbols` |

当前目录是重写版时：

```bash
cd /path/to/kube-consul-register
IMG=harbor.deploy.local/registry/kube-consul-register:go123
docker build -t "$IMG" .
# 若拉不到 golang:1.23.6：
# docker build --build-arg GO_IMAGE=swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.22.12 -t "$IMG" .
docker push "$IMG"

kubectl -n yunshu-monitor set image deploy/kube-consul-register \
  kube-consul-register="$IMG"
# imagePullPolicy: Always；不要 :latest + IfNotPresent
kubectl -n yunshu-monitor delete pod -l app=kube-consul-register
```

不要用 Go 1.21 编重写版（`consul/api@v1.32.0 requires go >= 1.22.12`）。
---

## 2. Consul ACL

注册 Token 用 **metrics-register**（不要用 Management / Agent）。Policy 已含 `k8s-pod` / `k8s-pod-metrics`，见 [`metrics-register.hcl`](../metrics-register.hcl)。

若业务忘记写 `consul.register/service.name`，服务名会变成 Deployment 名（如 `nginx`），ACL deny 下会 403。Yunshu 场景**必须**写服务名。

---

## 3. 集群内部署（推荐）

```bash
cd deploy/monitoring/kube-consul-register
# 1) Token
cp 03-secret.example.yaml 03-secret.yaml
# 编辑 consul_token = metrics-register SecretID
kubectl apply -f 03-secret.yaml

# 2) 改 02-configmap.yaml：consul_address / consul_datacenter
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-rbac.yaml
kubectl apply -f 02-configmap.yaml
kubectl apply -f 04-deployment.yaml

kubectl -n yunshu-monitor logs -f deploy/kube-consul-register
```

控制器只注册带注解 `consul.register/enabled: "true"` 的 Pod。

---

## 4. 集群外运行（监控机，可选）

与现网 Python 脚本同一台机、同一 kubeconfig：

```bash
# ConfigMap / Secret 仍先 apply 到集群，或本地起一份 ConfigMap
./kube-consul-register \
  -kubeconfig=/root/.kube/config \
  -in-cluster=false \
  -configmap=yunshu-monitor/kube-consul-register-config \
  -consul-secret=yunshu-monitor/consul-token \
  -logtostderr=true -v=2
```

---

## 5. 业务 Pod 注解（写在 Pod 模板）

本工具**不读** `prometheus.io/path`。目录 vs 采集靠 **服务名 + Meta**。

### 5.0 上游能力边界（改不了就只能换方案）

未改二进制时，**只有注解**，下列行为是写死的：

| 你想要的 | 上游实际 | 注解能否改 |
|----------|----------|------------|
| Service **ID** = `命名空间-Pod名` | 固定为 **`Pod名-容器名`** | **不能** |
| Meta 自动含 `pod_ip` / `node_ip` | Meta **只来自** `service.meta.*` 静态注解 | **不能**（动态字段写不进去） |
| Consul **Address** = PodIP | 已是 `status.podIP` | 已满足，无需注解 |
| Node 信息 | Tags 里有 `node:<NodeName>`（主机名，不是 HostIP） | 不能改成 IP |
| `container.name` 自动 | 省略 = **该 Pod 每个容器各注册一份** | 单容器可省略；有 sidecar **必须写** |

若必须 `ns-pod` 这种 ID、或 Meta 自动带 `pod_ip`/`host_ip`，需要 **fork 改代码**，或改回 [`consul_k8s_pods_sync.py`](../consul_k8s_pods_sync.py)（ID≈`k8s-<集群>-<ns>-<pod>-<port>`，Meta 含 `pod_ip`）。

Yunshu 看地址时用的是 **Address（PodIP）**，不是 Service ID；ID 主要给人眼辨认。

### 5.1 只登记（nginx）

见 [`pod-catalog.example.yaml`](./pod-catalog.example.yaml)：

- `consul.register/service.name: "k8s-pod"`
- **不要**写 `metrics_path`
- 无真实探针时关闭 liveness/readiness check，避免 Consul critical
- **单容器**可省略 `pod.container.name`；多容器必须写主容器名

### 5.2 要 Prom 采集

见 [`pod-metrics.example.yaml`](./pod-metrics.example.yaml)：

- `consul.register/service.name: "k8s-pod-metrics"`
- `consul.register/service.meta.metrics_path: "/metrics"`（必填，否则 scrape 会 drop）

| 注解 | 作用 |
|------|------|
| `consul.register/enabled` | **必填** `true` 才注册 |
| `consul.register/service.name` | Yunshu：**必须** `k8s-pod` 或 `k8s-pod-metrics` |
| `consul.register/pod.container.name` | 只注册列出的容器；省略=全部容器各一份 |
| `consul.register/pod.container.probe.liveness` | 默认 true；无探针的静态站建议 `false` |
| `consul.register/service.meta.<key>` | **静态** Meta（`cluster`/`app`/`namespace`/`yunshu_project`/`metrics_path`）；**不能**写运行时 PodIP |

官方注解说明：[kube-consul-register README](https://github.com/tczekajlo/kube-consul-register#annotations)。

---

## 6. Prometheus / Yunshu

无需改 scrape job：仍只发现 `k8s-pod-metrics` + tag `yunshu-metrics`（[`prometheus-scrape-acl.yml`](../prometheus-scrape-acl.yml)）。业务 Pod 必须带 label `yunshu-metrics: "tag"`，否则改完 `k8s_tag=k8s` 后 Yunshu/Prom 会看不到。

Yunshu 监控对象用 **prometheus-sd 只读 Token** 浏览 Consul，Address 应为 **PodIP**。

---

## 7. 验收

- [ ] `kubectl -n yunshu-monitor get pod` Running  
- [ ] 日志出现 `Adding service` / `Service's been registered`  
- [ ] Consul：nginx → Name=`k8s-pod`；有 metrics 的 → `k8s-pod-metrics` 且 Meta 含 `metrics_path`  
- [ ] Prom Targets：**没有**纯目录 nginx；有 path 的为 UP  
- [ ] 删 Deployment 后 Consul 实例被反注册  

---

## 8. 排障

| 现象 | 处理 |
|------|------|
| 无任何注册 | 注解是否在 **Pod 模板**；`enabled=true` |
| 启动日志 `ConsulToken:""` | **正常现象**：打印配置在读 Secret **之前**；以是否出现 `Permission denied` / 能否注册为准 |
| `ACL not found` | Secret 空/键名不是 `consul_token` / `stringData` 里误填了 base64 |
| `Missing service:write on k8s-pod` | ① Deployment 未加 `-consul-secret=yunshu-monitor/consul-token`（匿名无写权限）② 线上 Policy 未含 `k8s-pod`（仓库 `metrics-register.hcl` 已有，需 `acl policy update`） |
| 服务名变成 nginx / default | 未写 `consul.register/service.name` |
| Prom 刮到 nginx UNKNOWN | 误写成 `k8s-pod-metrics`，或 job 发现了 `k8s-pod` |
| 有 metrics 却被 drop | 未写 `service.meta.metrics_path` |
| `consul-targets-ctl.sh sync` 后 telegraf/icmp/http/tcp 过一会全没了，k8s-pod 还在 | 控制器 Clean 把带 `k8s_tag` 的非 Pod 服务删了。把 ConfigMap `k8s_tag` 改成 `k8s`（不要用 `yunshu-metrics`），apply 后重启控制器 |
| `encoding alphabet includes duplicate symbols` | 用 Go 1.22+ 编的二进制；按 §1 用 Go 1.21 Dockerfile 重建 |
