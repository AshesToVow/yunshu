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
Telegraf / Pushgateway / blackbox（及目标）
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
| `BLACKBOX-MODULES.md` | 自定义 blackbox module + Consul Meta.probe_module |
| `prometheus-scrape.yml` | Prometheus scrape（无 ACL） |
| `prometheus-scrape-acl.yml` | 带 token 的多类型 Consul SD |
| `telegraf.conf` | 主机 Telegraf 样例 |
| `blackbox.yml` | blackbox 模块样例（生产自定义 module 放机房，勿提交密钥） |
| `consul_targets_sync.py` + `consul-targets-*.sh/json` | 统一注册（Py2.7+）；telegraf 每台，拨测仅监控机 |
| `consul_k8s_pods_sync.py` + `K8S-CONSUL-PODS.md` | K8s Pod → Consul（kubectl + cron） |
| `metrics-register.hcl` | 统一注册 ACL policy |
| `yunshu-alert.snippet.yaml` | Yunshu `alert` 段建议 |

## 脚本跑在哪

| 类型 | 执行位置 |
|------|----------|
| Telegraf 注册 | **每台** Telegraf 主机（`--type telegraf`） |
| ICMP/HTTP/TCP 目标 | **仅** Prom/Consul 监控机 |
| K8s Pod → Consul | **仅** 监控机（`consul-k8s-pods-ctl.sh sync` + kubectl） |
| 规则与通知 | Yunshu |

## 快速验收

1. Prom Targets 中 telegraf/blackbox 为 UP。  
2. Yunshu 数据源 Ping 成功。  
3. 规则中心启用规则 → **事件台**出现当前告警与投递流水。  
4. 不要求 Prometheus Alerts 页有对应项。
