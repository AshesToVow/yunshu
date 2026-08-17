# K8s Pod → Consul 注册

供 **Yunshu 监控对象** 展示 PodIP，以及（可选）**Prometheus** 按注解采集指标。

相关文件：

| 文件 | 说明 |
|------|------|
| `consul_k8s_pods_sync.py` | 同步脚本（Py2.7+，标准库 + kubectl） |
| `consul-k8s-pods-ctl.sh` | 包装脚本 |
| `consul-k8s-pods.example.json` | 配置样例 → 机房拷为 `consul-k8s-pods.json` |
| `metrics-register.hcl` | ACL：需含 `k8s-pod`、`k8s-pod-metrics` |
| `prometheus-scrape-acl.yml` | job `k8s-pod` 只发现 **`k8s-pod-metrics`** |

---

## 1. 行为概览

```text
kubectl get pods（按 ns + label）
        │
        ▼
   有 prometheus.io/path ?
    /              \
  否                是
   │                │
   ▼                ▼
Consul Name:     Consul Name:
k8s-pod          k8s-pod-metrics
（仅目录）        （目录 + Prom 采集）
Address=PodIP    Address=PodIP
```

| Consul 字段 | 规则 |
|-------------|------|
| **Address** | **始终 = PodIP**（Yunshu / UI 直接显示） |
| **Port** | `prometheus.io/port` → 否则第一个 `containerPort` → 否则 `0` |
| **Name** | 无 path → `k8s-pod`；有 `prometheus.io/path` → `k8s-pod-metrics` |
| **ID** | `k8s-<集群>-<ns>-<pod名>-<port>` |
| **Meta** | `cluster` / `namespace` / `pod` / `pod_ip` / `app` / `managed_by=yunshu-k8s-pods`；有 path 时才有 `metrics_path` |

`sync` 会删除本脚本托管、但当前已不存在的实例（按 `Meta.managed_by`）。

**不会**默认使用 `/metrics`。没有 path 注解的 Pod（如 nginx）**只登记、不采集**，避免 Targets 里 `UNKNOWN`。

---

## 2. Deployment / Pod 要配什么

标签、注解必须写在 **`spec.template.metadata`**（Pod 模板），只写在 Deployment 顶层无效。

### 2.1 仅登记（不采集）—— nginx / 静态站等

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dodo-web
  namespace: dodo-test
spec:
  selector:
    matchLabels:
      app: dodo-web
  template:
    metadata:
      labels:
        app: dodo-web
        yunshu.io/consul: "true"    # 必填：匹配配置里 label_selector
      annotations:
        prometheus.io/port: "80"    # 可选：Consul 显示端口
        # 不要写 prometheus.io/path
        # prometheus.io/scrape 可省略；没有 path 不会进采集
    spec:
      containers:
        - name: nginx
          image: nginx:alpine
          ports:
            - containerPort: 80
```

结果：Consul 服务名 **`k8s-pod`**，Address=PodIP，Prom Targets **无此项**。

### 2.2 有指标端点、要 Prometheus 采集

```yaml
template:
  metadata:
    labels:
      app: myapp
      yunshu.io/consul: "true"
    annotations:
      prometheus.io/port: "8080"
      prometheus.io/path: "/metrics"   # 关键：才会注册为 k8s-pod-metrics
```

结果：Consul 服务名 **`k8s-pod-metrics`**，Prom 按 path 刮取。

### 2.3 字段对照

| 字段 | 位置 | 必填？ | 作用 |
|------|------|--------|------|
| `yunshu.io/consul: "true"` | labels | **是** | 被 `label_selector` 选中 |
| `app` 或 `app.kubernetes.io/name` | labels | 建议 | Meta `app` / `service` |
| `prometheus.io/port` | annotations | 否 | Consul Port；无则用 containerPort |
| `prometheus.io/path` | annotations | 采集时必填 | **有才采集**；决定服务名是否为 `k8s-pod-metrics` |
| `prometheus.io/scrape` | annotations | 否 | 可省略；`false` 时即使有 path 也不当采集目标 |
| `yunshu.io/metrics-port` | annotations | 否 | 与 `prometheus.io/port` 等价（优先列表可配） |
| `yunshu.io/metrics-path` | annotations | 否 | 与 `prometheus.io/path` 等价 |

等价地，配置文件里默认：

- `label_selector`: `yunshu.io/consul=true`
- `port_annotations`: `yunshu.io/metrics-port`, `prometheus.io/port`
- `path_annotations`: `yunshu.io/metrics-path`, `prometheus.io/path`

---

## 3. 机房安装

### 3.1 拷贝

```bash
mkdir -p /export/server/monitor/consul
cp consul_k8s_pods_sync.py consul-k8s-pods-ctl.sh consul-k8s-pods.example.json \
  /export/server/monitor/consul/
cd /export/server/monitor/consul
cp consul-k8s-pods.example.json consul-k8s-pods.json
chmod 700 consul-k8s-pods-ctl.sh consul_k8s_pods_sync.py
```

### 3.2 编辑 `consul-k8s-pods.json`

必改项：

- `consul.addr`：Consul 地址  
- `namespaces`：**必须包含业务 ns**（如 `dodo-test`），否则选不中 Pod  
- `kubectl.kubeconfig` / `context`，或依赖环境变量 `KUBECONFIG`  
- `clusters[].name`：写入 Meta.cluster  

Token：优先 `export CONSUL_TOKEN=...` 或同目录 `.consul_token`，json 里 `"token": ""` 即可。  
**JSON 不能写 `//` 行尾注释**，否则 Py2.7 解析失败。

### 3.3 ACL（Consul 已开启 ACL 时）

`metrics-register.hcl` 需包含：

```hcl
service "k8s-pod" {
  policy = "write"
}
service "k8s-pod-metrics" {
  policy = "write"
}
```

用**管理 Token**更新（Agent Token 不够）：

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<管理Token>
./consul acl policy update -name metrics-register \
  -rules @/export/server/monitor/consul/metrics-register.hcl
```

若 `agent/self` 里 `ACLsEnabled: false`，说明运行中的进程未加载含 ACL 的配置，需按 [`CONSUL-ACL-RUNBOOK.md`](./CONSUL-ACL-RUNBOOK.md) 重启后再 update。  
ACL 关闭时可不配 Token，直接 sync。

### 3.4 命令

```bash
export CONSUL_TOKEN=<metrics-register SecretID>   # ACL 开启时
export KUBECONFIG=/root/.kube/config

./consul-k8s-pods-ctl.sh list         # 预览将注册的服务
./consul-k8s-pods-ctl.sh sync         # 注册 + 清理 stale（推荐）
./consul-k8s-pods-ctl.sh deregister   # 删除全部本脚本托管项
```

cron（建议每 1～2 分钟）：

```cron
*/2 * * * * /export/server/monitor/consul/consul-k8s-pods-ctl.sh sync >>/var/log/consul-k8s-pods.log 2>&1
```

---

## 4. Prometheus

合并 [`prometheus-scrape-acl.yml`](./prometheus-scrape-acl.yml) 中 job：

```yaml
- job_name: k8s-pod
  consul_sd_configs:
    - services: ["k8s-pod-metrics"]   # 不要写成 k8s-pod
      tags: ["yunshu-metrics"]
```

然后 `/-/reload` 或重启 Prom。

| 现象 | 原因 |
|------|------|
| Targets 出现无 `/metrics` 的 nginx 且 UNKNOWN | job 仍发现 `k8s-pod`，或未 sync 新脚本 |
| 有 path 却刮不到 | ACL / 网络策略 / path 写错；查 Consul 是否为 `k8s-pod-metrics` |

查询示例：`up{job="k8s-pod"}`。

---

## 5. Yunshu

1. 监控对象对接 Consul（Token 用 **prometheus-sd 只读** 即可浏览）。  
2. 同步后应看到 Address=**PodIP**；服务名可能是 `k8s-pod` 或 `k8s-pod-metrics`。  
3. 告警在**规则中心**对 Prom 指标写 PromQL（主路径），不要依赖 Prom 原生 Alerts 页。

---

## 6. 验收清单

- [ ] `kubectl get pods -n <ns> -l yunshu.io/consul=true` 能列出目标 Pod  
- [ ] `./consul-k8s-pods-ctl.sh list` 中 Address 为 PodIP  
- [ ] Consul UI：无 path 的实例 Name=`k8s-pod`；有 path 的为 `k8s-pod-metrics`  
- [ ] Prom Targets：**没有**纯目录类 nginx；有 path 的为 UP 或按预期失败原因可查  
- [ ] Yunshu 监控对象能看到对应 PodIP  

---

## 7. 多集群

- 简单：每个集群一份 `consul-k8s-pods-<name>.json`（不同 `kubectl.context` / `clusters[0].name`）+ 各一条 cron。  
- 或一份 json 多个 `clusters[]`，并为每个 cluster 配 `kubectl: { "context": "..." }`（脚本支持 per-cluster 覆盖）。

---

## 8. 排障

| 问题 | 处理 |
|------|------|
| JSON `Expecting ',' delimiter` | 检查 `consul-k8s-pods.json` 语法；去掉 `//` 注释；用 `python -m json.tool` 校验 |
| list 为空 | namespaces 是否包含业务 ns；label 是否在 **Pod 模板**；Pod 是否 Running |
| ACL 401 disabled | 运行中 Consul 未开 ACL，见 runbook 重启 |
| policy update 403 | 需管理 Token，不是 agent token |
| Prom 仍刮无 metrics 的 Pod | 更新 scrape：`services: ["k8s-pod-metrics"]`，并重新 `sync` |
