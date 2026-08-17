# Blackbox 自定义 Module + Consul / Prometheus

## 分工

| 放哪里 | 放什么 |
|--------|--------|
| `blackbox.yml` → `modules` | **怎么探**：超时、状态码、body 正则、POST body、Header（含鉴权） |
| `consul-targets.json` → `http.probes` | **探谁**：`url` + 用哪个 `module` |
| Prometheus `blackbox-http` job | 发现 Consul `http` 服务，用 Meta 填 `__param_target` / `__param_module` |

不要为每个 module 新建一个 Prometheus job；**一个 job + Meta.probe_module** 即可。

## 扩展步骤

1. 在 `blackbox.yml` 增加 module（你贴的 `http_2xx_xihe` 等），重启/reload blackbox。  
2. 在 `consul-targets.json` 增加 probe：

```json
{
  "type": "http",
  "enabled": true,
  "service": "http",
  "tags": ["probe-http", "yunshu-metrics"],
  "probes": [
    {"id": "xihe", "url": "https://你的地址/...", "module": "http_2xx_xihe"}
  ]
}
```

3. `export CONSUL_TOKEN=... && ./consul-targets-ctl.sh sync --type http`  
4. Prometheus 使用 `prometheus-scrape-acl.yml` 里带 `probe_module` relabel 的 `blackbox-http` job，reload。  
5. 查：`probe_success{job="blackbox-http", module="http_2xx_xihe"}`

## 安全注意

- Module 里的 **JWT / clientId / token** 属于密钥：文件权限收紧（`chmod 600`），**不要提交到 Git**。  
- Token 会过期（你配置里部分 JWT `exp` 已过期），过期后 `probe_success=0`，需轮换。  
- 更稳妥：鉴权类探测用短时 token，或改为不依赖长期 JWT 的健康检查接口。

## Yunshu

规则中心对 `probe_success{module="http_2xx_xihe"} == 0` 建规则即可；标签里会有 `module`、`instance`（url）。

## 按目标加业务标签（TCP / HTTP / ICMP 通用）

Consul Meta → Prometheus `__meta_consul_service_metadata_*` → scrape `relabel_configs`。

TCP 示例（`consul-targets.json`，在**监控机**维护）：

```json
{
  "type": "tcp",
  "enabled": true,
  "service": "tcp",
  "tags": ["probe-tcp", "yunshu-metrics"],
  "endpoints": [
    {
      "id": "mysql",
      "host": "10.10.10.4:3306",
      "meta": { "service": "mysql", "app": "order", "component": "db" }
    },
    {
      "id": "redis",
      "host": "10.10.10.4:6379",
      "meta": { "service": "redis", "app": "order", "component": "cache" }
    }
  ]
}
```

同步后查询示例：`probe_success{job="blackbox-tcp", service="mysql"}`。

约定常用 Meta key（需在 `prometheus-scrape-acl.yml` 有对应 relabel）：`service` / `app` / `component` / `team` / `env` / `yunshu_project`。值必须是字符串；新增 key 时同步加一条 `source_labels: [__meta_consul_service_metadata_<key>]`。
