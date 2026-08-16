# 监控采集与发现配置样例（Yunshu 告警 P0）

配套方案：[`docs/requirements/R-alert-platform-remediation-nightingale-consul.md`](../../docs/requirements/R-alert-platform-remediation-nightingale-consul.md)

## 链路

```text
Telegraf / Pushgateway / blackbox（及目标）
  → 注册到 Consul（推荐）或静态 scrape
  → Prometheus / VictoriaMetrics
  → Yunshu 数据源 + 规则中心（唯一告警引擎）
  → 通知渠道
```

**不再使用 Alertmanager。** Prometheus 只负责存指标与 scrape；告警规则在 Yunshu 配置。

## 文件一览

| 文件 | 用途 |
|------|------|
| `prometheus-scrape.yml` | Prometheus scrape 片段：Consul SD + Pushgateway + blackbox |
| `telegraf.conf` | 主机 Telegraf：采集 + prometheus_client 输出 + Consul 服务注册 |
| `blackbox.yml` | blackbox_exporter 模块配置 |
| `consul-service-telegraf.json` | 手工注册 Telegraf 到 Consul 的示例 |
| `consul-service-blackbox-target.json` | 拨测目标注册示例（由 Prom blackbox job 发现） |
| `yunshu-alert.snippet.yaml` | Yunshu `configs/config.yaml` 告警段建议注释/取值 |

## 推荐标签（写入时序，供 Yunshu 规则/订阅匹配）

| label | 来源建议 |
|-------|----------|
| `yunshu_project` | Consul meta → Prom relabel |
| `env` | Consul meta |
| `exporter_role` | `telegraf` / `blackbox` / `pushgateway` / `app` |
| `host` / `instance` | 机器标识 |

## 快速验收

1. Prometheus targets 中能看到 Consul 发现的 telegraf / blackbox。
2. Yunshu「数据源」指向该 Prom，`Ping` 成功。
3. 「规则中心」从模板创建 Telegraf/Blackbox 规则并启用。
4. 人为打高负载或关闭探测目标，事件台出现通知（钉钉/企微/邮件）。
