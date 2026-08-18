# 监控采集与发现配置样例（Yunshu 告警）

配套方案：[`docs/requirements/R-alert-platform-remediation-nightingale-consul.md`](../../docs/requirements/R-alert-platform-remediation-nightingale-consul.md)

## 告警路径（先读）

**完整说明：[`ALERT-PATH.md`](./ALERT-PATH.md)**

```text
采集 → Prometheus（只存指标）
        → Yunshu 规则中心（唯一告警引擎）→ 事件台 / 钉钉企微邮件
```

- 平台规则结果看 **Yunshu 事件台**，不看 Prom Alerts。  
- 投递等待看 **`configs/config.yaml` → `alert.group_*` + 规则评估间隔**，**不看** `alertmanager.yml`。  
- Prom `rules/*.yml` + Alertmanager 为旧路径，与平台互不同步。

## 采集链路

```text
Telegraf / Pushgateway / blackbox / K8s Pod(可选)
  → 注册到 Consul（推荐）或静态 scrape
  → Prometheus / VictoriaMetrics
  → Yunshu 数据源 + 规则中心
  → 通知渠道
```

## 文件一览

| 文件 | 用途 |
|------|------|
| **`ALERT-PATH.md`** | **主/旧告警路径、等待时间来源、验收（必读）** |
| **`CONSUL-ACL-RUNBOOK.md`** | Consul 1.10 + ACL、SD、Token、Telegraf/拨测注册 |
| **`K8S-CONSUL-PODS.md`** | **K8s Pod → Consul（总览）；推荐 kube-consul-register** |
| **`kube-consul-register/`** | 事件驱动注册：RBAC/ConfigMap/Deployment + 业务注解 |
| `BLACKBOX-MODULES.md` | 自定义 blackbox module + Consul Meta.probe_module |
| `prometheus-scrape.yml` | Prometheus scrape（无 ACL） |
| `prometheus-scrape-acl.yml` | 带 token 的多类型 Consul SD（含 `k8s-pod-metrics`） |
| `telegraf.conf` | 主机 Telegraf 样例 |
| `blackbox.yml` | blackbox 模块样例（生产自定义 module 放机房，勿提交密钥） |
| `consul_targets_sync.py` + `consul-targets-*.sh/json` | 统一注册（Py2.7+）；telegraf 每台，拨测仅监控机 |
| `consul_k8s_pods_sync.py` + `consul-k8s-pods-ctl.sh` | K8s Pod → Consul 备选（kubectl + cron，勿与控制器同开） |
| `tcp-endpoints.example.list` / `http-probes.example.list` | TCP/HTTP 批量清单样例 |
| `metrics-register.hcl` | 统一注册 ACL policy（含 k8s-pod / k8s-pod-metrics） |
| `yunshu-alert.snippet.yaml` | Yunshu `alert` 段建议 |

## 脚本跑在哪

| 类型 | 执行位置 |
|------|----------|
| Telegraf 注册 | **每台** Telegraf 主机（`--type telegraf`） |
| ICMP/HTTP/TCP 目标 | **仅** Prom/Consul 监控机 |
| K8s Pod → Consul | **推荐** 集群内 `kube-consul-register`；或监控机 cron（二选一） |
| 规则与通知 | Yunshu |

## 快速验收

1. Prom Targets 中 telegraf/blackbox 为 UP；`k8s-pod` job 仅含带 `prometheus.io/path` 的实例。  
2. Yunshu 数据源 Ping 成功；监控对象可见 PodIP（`k8s-pod` / `k8s-pod-metrics`）。  
3. 规则中心启用规则 → **事件台**出现当前告警与投递流水。  
4. 不要求 Prometheus Alerts 页有对应项。

## 相关专题

- 拨测批量与标签：[`BLACKBOX-MODULES.md`](./BLACKBOX-MODULES.md)、`consul-targets.example.json`  
- K8s Pod 登记与采集分流：[`K8S-CONSUL-PODS.md`](./K8S-CONSUL-PODS.md)、[`kube-consul-register/README.md`](./kube-consul-register/README.md)  
- Consul ACL：[`CONSUL-ACL-RUNBOOK.md`](./CONSUL-ACL-RUNBOOK.md)  
