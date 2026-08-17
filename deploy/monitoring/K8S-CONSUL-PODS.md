# K8s Pod → Consul 注册（Yunshu 监控对象 / Prometheus Consul SD）

## 做什么

用监控机上的 `kubectl` 拉取 Pod，注册到 Consul：

| Consul 字段 | 来源 |
|-------------|------|
| Service.ID | `k8s-<集群>-<ns>-<pod名>-<port>` |
| Name | 默认 `k8s-pod` |
| Address / Port | PodIP + 指标端口 |
| Meta | `cluster` / `namespace` / `pod` / `app` / `metrics_path` / `managed_by=yunshu-k8s-pods` |

`sync` 会删除本脚本托管但已不存在的旧实例（按 `Meta.managed_by`）。

## 不做什么

- 不替代 kube-state-metrics / cAdvisor；那是集群指标，用 Prom `kubernetes_sd` 更合适。  
- 默认**不加** Consul HTTP/TCP check（跨网段常失败）；连通性用 blackbox 或应用自身 `/metrics`。

## 准备

1. 监控机可 `kubectl get pods`（kubeconfig / context）。  
2. Consul ACL：`metrics-register` 增加 `service "k8s-pod" { policy = "write" }`（见 `metrics-register.hcl`），再 `acl policy update`。  
3. 拷贝：

```bash
cp consul_k8s_pods_sync.py consul-k8s-pods-ctl.sh consul-k8s-pods.example.json \
  /export/server/monitor/consul/
cd /export/server/monitor/consul
cp consul-k8s-pods.example.json consul-k8s-pods.json
# 编辑 namespaces / label_selector / kubeconfig
chmod 700 consul-k8s-pods-ctl.sh consul_k8s_pods_sync.py
```

## Pod 怎么被选中

配置里 `label_selector` 示例：`yunshu.io/consul=true`。

Deployment 示例：

```yaml
metadata:
  labels:
    yunshu.io/consul: "true"
    app: myapp
  annotations:
    prometheus.io/scrape: "true"          # 可选；require_scrape_annotation=true 时必填
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

端口优先级：注解 `yunshu.io/metrics-port` / `prometheus.io/port` → 名为 `metrics` 的 containerPort → 第一个 containerPort → `default_port`。

## 命令

```bash
export CONSUL_TOKEN=<metrics-register>
export KUBECONFIG=/root/.kube/config

./consul-k8s-pods-ctl.sh list          # 只看将注册什么
./consul-k8s-pods-ctl.sh sync          # 注册 + 清理 stale
./consul-k8s-pods-ctl.sh deregister    # 清掉全部托管项
```

cron：

```cron
*/2 * * * * /export/server/monitor/consul/consul-k8s-pods-ctl.sh sync >>/var/log/consul-k8s-pods.log 2>&1
```

## Prometheus

合并 `prometheus-scrape-acl.yml` 中 `job_name: k8s-pod`，reload。  
查询：`up{job="k8s-pod", namespace="yunshu"}`。

## Yunshu

监控对象同步 Consul 后，应看到 Address=PodIP、Port=指标端口。  
规则中心对 `up{job="k8s-pod"}==0` 或业务指标告警即可。

## 多集群

`consul-k8s-pods.json` 里配置多个 `clusters[]`，每个可不同 `kubectl.context`（把 context 放到各 cluster 段需扩展时：当前全局 `kubectl.context`；多集群可跑多份配置 / 多次 cron）。

多集群简易做法：每个集群一份 json + 一条 cron，`kubectl.context` 不同，`clusters[0].name` 不同。
