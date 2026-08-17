# Consul 1.10.x + ACL 运维手册（对接 Yunshu）

适用：机房单机/小集群 Consul `1.10.4`，Prometheus Consul SD，Telegraf 启停注册/反注册。

> **CLI 注意（1.10.4）**：创建 policy 时 **不要** 用 `-rules @- <<EOF`（会报 `open -: no such file`）。  
> 一律先 `cat > /path/xxx.hcl`，再用 `-rules @/path/xxx.hcl`。

## 总链路

```text
① Consul（ACL）就绪 + Agent Token 写入 hcl 并重启
② 建各类业务 Token（见下方「Token 一览」）
③ Telegraf / Pushgateway / blackbox-target 用「注册 Token」注册
④ Prometheus 用「只读 Token」做 consul_sd（多 job）
⑤ Yunshu 数据源指向 Prom；规则在 Yunshu；监控对象可选同步 Consul
```

---

## Token 一览（推荐只建 3 个）

| 名称（建议） | 谁持有 | 权限要点 | 用途 | 能否写进业务脚本 |
|--------------|--------|----------|------|------------------|
| **Management（Bootstrap）** | 运维 / 密码库 | 全能 | UI 登录、建 policy/token | **禁止**写入脚本 |
| **Agent** | 仅 `consul.hcl` → `tokens.agent` | node 写、service 读 | Consul 进程自身 | 只写 hcl |
| **prometheus-sd** | Prometheus / Yunshu 监控对象 | 全部 service/node **只读** | 服务发现、同步目录 | 可写 Prom 配置 |
| **metrics-register**（统一） | `consul_targets_sync.py`、Telegraf 启停 | telegraf/icmp/http/tcp/pushgateway 等 **写** | 注册 / 反注册所有采集与拨测目标 | **脚本只用这一个** |

不再为 icmp、telegraf、http 各建 Token；一个 `metrics-register` 即可。

**记错会怎样：**

- 脚本用 **Agent Token** → 注册会 **403**
- 脚本用 **Management** → 能注册，泄露风险大
- Prom 用 **metrics-register** → 权限过大，应继续用 **prometheus-sd** 只读 Token
- 未配 **Agent Token** → 日志刷 `Coordinate update blocked by ACLs`

---

## 1. Consul 开启 ACL（1.10.4）

### 1.1 配置文件示例

路径：`/export/server/monitor/consul/bin/consul.hcl`

```hcl
datacenter = "prod"
data_dir   = "/export/server/monitor/consul/data"
server           = true
bootstrap_expect = 1

# 多网卡时禁止 bind 0.0.0.0，必须写真实 IP（例：10.10.10.5）
bind_addr      = "10.10.10.5"
advertise_addr = "10.10.10.5"

# HTTP API / UI 可监听所有网卡
client_addr = "0.0.0.0"
ui_config { enabled = true }

acl {
  enabled                  = true
  default_policy           = "deny"
  enable_token_persistence = true
  # tokens.agent 在创建 Agent Token 后再填，见 1.3
}
```

启动：

```bash
cd /export/server/monitor/consul/bin
./consul agent -config-file=/export/server/monitor/consul/bin/consul.hcl &
```

### 1.2 Bootstrap 管理 Token（只做一次）

```bash
cd /export/server/monitor/consul/bin
./consul acl bootstrap
# 记下 SecretID = MANAGEMENT_TOKEN，立刻存入密码库
# 日志里 acl-bootstrap-reset 的 WARN 可忽略

export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<MANAGEMENT_TOKEN>
```

**作用：** 全局管理。Consul UI 登录用这个。后续所有 `acl policy/token create` 都带此 Token。

### 1.3 Agent Token

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<MANAGEMENT_TOKEN>

cat > /export/server/monitor/consul/agent-policy.hcl <<'EOF'
node_prefix "" {
  policy = "write"
}
service_prefix "" {
  policy = "read"
}
EOF

./consul acl policy create -name agent-policy -rules @/export/server/monitor/consul/agent-policy.hcl

./consul acl token create -description "agent token" -policy-name agent-policy
# 记下输出的 SecretID = AGENT_TOKEN（不是 Policy ID）
```

写入 `consul.hcl` 后 **重启 Consul**：

```hcl
acl {
  enabled                  = true
  default_policy           = "deny"
  enable_token_persistence = true
  tokens {
    agent = "<AGENT_TOKEN>"
  }
}
```

**作用：** 仅给 Consul agent 进程；消除节点/坐标更新被 ACL 拦截。  
**不是** Telegraf 注册 Token，也不是 UI 日常管理 Token。

验证：

```bash
export CONSUL_HTTP_TOKEN=<MANAGEMENT_TOKEN>
./consul members
# 不应再持续刷 Coordinate update blocked by ACLs
```

### 1.4 Prometheus / Yunshu 只读 Token（prometheus-sd）

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<MANAGEMENT_TOKEN>

cat > /export/server/monitor/consul/prometheus-sd.hcl <<'EOF'
node_prefix "" {
  policy = "read"
}
service_prefix "" {
  policy = "read"
}
agent_prefix "" {
  policy = "read"
}
EOF

./consul acl policy create -name prometheus-sd -rules @/export/server/monitor/consul/prometheus-sd.hcl

./consul acl token create -description "prometheus consul_sd" -policy-name prometheus-sd
# 记下 SecretID = PROM_TOKEN
```

**作用：**

- Prometheus `consul_sd_configs.token`
- Yunshu「监控对象」同步 Consul（可复用同一 Token）

只有读权限，不能注册/删除服务。

### 1.5 统一注册 Token（metrics-register，推荐）

一个 Policy 覆盖 `consul_targets_sync.py` 支持的全部服务名：

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<MANAGEMENT_TOKEN>

# 规则文件见仓库 deploy/monitoring/metrics-register.hcl
cat > /export/server/monitor/consul/metrics-register.hcl <<'EOF'
service "telegraf" {
  policy = "write"
}
service "icmp" {
  policy = "write"
}
service "http" {
  policy = "write"
}
service "tcp" {
  policy = "write"
}
service "pushgateway" {
  policy = "write"
}
service "blackbox-target" {
  policy = "write"
}
node_prefix "" {
  policy = "read"
}
EOF

./consul acl policy create -name metrics-register -rules @/export/server/monitor/consul/metrics-register.hcl

./consul acl token create -description "metrics register (all types)" -policy-name metrics-register
# 记下 SecretID = METRICS_REGISTER_TOKEN
```

**作用：** Telegraf / ICMP / HTTP / TCP / Pushgateway 注册与反注册 **共用这一个 Token**。

```bash
export CONSUL_TOKEN=<METRICS_REGISTER_TOKEN>
python3 consul_targets_sync.py -c consul-targets.json sync
```

若之前已建过 `telegraf-register`、`icmp-register` 等，可继续用，但建议迁到统一 Token 后废弃旧 Token。

以后若自定义服务名（如 `service: "my-icmp"`），把对应 `service "my-icmp"` 补进此 hcl，再：

```bash
./consul acl policy update -name metrics-register -rules @/export/server/monitor/consul/metrics-register.hcl
```

---

## 2. Prometheus：Consul SD + 多监控类型

合并 [`prometheus-scrape-acl.yml`](./prometheus-scrape-acl.yml) 进 `prometheus.yml`。

| job | Consul 服务名 | Tag | 用途 |
|-----|---------------|-----|------|
| `telegraf` | `telegraf` | `yunshu-metrics` | 主机/中间件指标 |
| `pushgateway` | `pushgateway` | `yunshu-metrics` | 短任务推送 |
| `http` / blackbox-http | `http` 或 `blackbox-target` | `probe-http` | HTTP 拨测 |
| `tcp` / blackbox-tcp | `tcp` 或 `blackbox-target` | `probe-tcp` | TCP 端口 |
| `icmp` / blackbox-icmp | `icmp` 或 `blackbox-target` | `probe-icmp` | ICMP |

每个 `consul_sd_configs`：

```yaml
server: "127.0.0.1:8500"   # Prom 与 Consul 同机；跨机写 10.10.10.5:8500
token: "<PROM_TOKEN>"      # 只用 prometheus-sd，不要用管理/metrics-register
datacenter: prod
```

blackbox 的 `__address__` → **blackbox_exporter**（如 `127.0.0.1:9115`），目标在 Meta。

```bash
curl -X POST http://127.0.0.1:9090/-/reload   # 需开启 lifecycle
# 或 restart.sh
```

---

## 3. Telegraf 注册 / 反注册

### 3.1 手工注册示例

见 [`consul-service-telegraf.json`](./consul-service-telegraf.json)：

```bash
curl -sS -X PUT "http://127.0.0.1:8500/v1/agent/service/register" \
  -H "X-Consul-Token: <TELEGRAF_TOKEN>" \
  -H "Content-Type: application/json" \
  -d @consul-service-telegraf.json
```

### 3.2 启停脚本

- 启动注册：改 `start.sh`（见手册历史示例 / 现场脚本），`CONSUL_TOKEN=<TELEGRAF_TOKEN>`，末尾必须调用 `start`
- 停止反注册：[`telegraf-stop.sh`](./telegraf-stop.sh) → 现场 `stop.sh`
- 重启：[`telegraf-restart.sh`](./telegraf-restart.sh)

服务 ID 约定（启停必须一致）：

```text
telegraf-<IP>-9273
```

反注册：

```bash
curl -X PUT "${CONSUL_ADDR}/v1/agent/service/deregister/telegraf-<IP>-9273" \
  -H "X-Consul-Token: <TELEGRAF_TOKEN>"
```

> 不要再打 `http://monitorserver.icity.com/service/targets/regist` —— 自定义网关，Prometheus `consul_sd` 读不到。

systemd：`Type=forking` + **`PIDFile=`**（不是 `pid=`）；`start.sh` 末尾要执行 `start`。

---

## 4. Yunshu 对接 + 验收

1. **数据源**：`http://<prometheus>:9090`，类型 Prometheus，Ping 成功。  
2. **监控对象（可选）**：Consul `http://10.10.10.5:8500`，DC=`prod`，Token=`PROM_TOKEN`。  
3. **规则中心** + 通知渠道。  
4. **自检**：

```bash
curl -s -H "X-Consul-Token: $PROM_TOKEN" \
  http://127.0.0.1:8500/v1/catalog/service/telegraf

# Prom：up{job="telegraf"}
```

Targets 为 UP 后，在 Yunshu 配规则并人为打挂，事件台应出现 firing。

---

## 服务命名约定

| Name | Tags | Meta |
|------|------|------|
| `telegraf` | `yunshu-metrics` | `yunshu_project`, `env`, `exporter_role=telegraf` |
| `pushgateway` | `yunshu-metrics` | `exporter_role=pushgateway` |
| `blackbox-target` | `probe-http` / `probe-tcp` / `probe-icmp` | `probe_url` 或 `probe_host` |

拨测示例：[`consul-service-blackbox-target.json`](./consul-service-blackbox-target.json)、[`consul-service-blackbox-tcp.json`](./consul-service-blackbox-tcp.json)（请求头带对应注册 Token）。

### ICMP 批量注册（类 Telegraf 启停）

与 Telegraf 不同：拨测目标不是进程，用列表 + 脚本维护即可。

```bash
# 1) 编辑列表
vi /export/server/monitor/consul/icmp-targets.list
# 一行一个 IP/主机，例如：
# 10.10.10.4
# 10.10.10.5

# 2) 拷贝脚本到机房（或从本仓库 deploy/monitoring/）
# icmp-register.sh / icmp-deregister.sh / icmp-sync.sh / icmp-targets.list

# 3) 注册（Token 用 icmp-register 的 SecretID）
export CONSUL_ADDR=http://127.0.0.1:8500
export CONSUL_TOKEN=<ICMP_REGISTER_TOKEN>
chmod 700 icmp-register.sh icmp-deregister.sh icmp-sync.sh
./icmp-register.sh
# 或改列表后全量同步：
./icmp-sync.sh

# 4) 剔除列表中全部目标
./icmp-deregister.sh
```

Prometheus 侧：`services: ["icmp"]` + `tags: ["probe-icmp"]`，Meta `probe_host` 由脚本写入。

---

## 修订

| 日期 | 说明 |
|------|------|
| 2026-08-17 | 初版 ACL + SD |
| 2026-08-17 | 改为 `@/path` 建 policy；补充 Token 职责表；bind_addr 真实 IP；启停 deregister |
| 2026-08-17 | 增加 ICMP 批量注册脚本（icmp-*.sh） |
