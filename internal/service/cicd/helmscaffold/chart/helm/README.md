# Yunshu Application Chart 脚手架

对齐文档《不是推标准，而是降门槛：一套研发愿用的 Helm 模板体系》目录架构。

## 解压后仓库根目录

```text
your-repo/
├── setup/                      # 全局固化配置 Chart（可选）
└── helm/                       # Application Chart（Jenkins 要求此路径）
    ├── Chart.yaml
    ├── values.yaml             # 研发主要改这里
    ├── values-dev|test|prod.yaml
    ├── config-files/
    ├── charts/
    │   ├── deployment-base/    # Deployment + SkyWalking/策略/DNS/lifecycle…
    │   ├── service-base/
    │   ├── config-base/
    │   ├── hpa-base/
    │   └── pvc-base/
    └── templates/
```

## 研发动作

1. 改 `helm/values.yaml`（镜像、端口、副本、探活、env）
2. 配置文件放 `helm/config-files/`
3. 环境叠加：`helm upgrade -f values.yaml -f values-prod.yaml ...`
4. 需要时打开模块开关（见下）

## 常用开关（deployment-base）

| 能力 | values 路径 | 默认 |
|------|-------------|------|
| SkyWalking Agent | `deployment-base.skywalking.enabled` | false |
| 多端口 | `deployment-base.ports` / `service-base.ports` | 空则用 containerPort/port |
| 更新策略 | `deployment-base.strategy` | RollingUpdate |
| 重启策略 | `deployment-base.restartPolicy` | Always |
| lifecycle | `deployment-base.lifecycle` | {} |
| DNS | `deployment-base.dnsPolicy` / `dnsConfig` | ClusterFirst |
| Consul 注册 | `deployment-base.consulRegister.enabled` | **true**（必填注解 + `yunshu-metrics: "tag"`） |
| 探活 | `deployment-base.probes.enabled` | false |
| HPA / PVC | `hpa-base.enabled` / `pvc-base.enabled` | false |

### 多端口示例

```yaml
deployment-base:
  ports:
    - name: web
      containerPort: 8913
      protocol: TCP
  probes:
    enabled: true
    port: web   # 探针打到名为 web 的端口

service-base:
  ports:
    - name: web
      port: 8913
      targetPort: web
      protocol: TCP
```

未配置 `ports` 时仍可用单字段 `containerPort` / `service-base.port`（兼容 Jenkins `--set`）。

### 开启 SkyWalking 示例

```yaml
deployment-base:
  skywalking:
    enabled: true
    image: harbor.xxx/registry/skywalking-java-agent:9.1.0
    collectorBackendServices: "skywalking-oap.observability:11800"
    # agentName 为空则用 Chart/Release 名
```

会增加 initContainer 拷贝 Agent，并给主容器注入 `JAVA_TOOL_OPTIONS`、`SW_AGENT_*`。

### Consul 注册（kube-consul-register）

默认开启。控制器只认 **Pod 模板**上的注解/标签（[说明](https://github.com/AshesToVow/kube-consul-register)）：

```yaml
deployment-base:
  consulRegister:
    enabled: true
    mode: catalog          # catalog=k8s-pod；metrics=k8s-pod-metrics + metrics_path
    # containerName: app   # 有 sidecar 时必填主容器名
    yunshuProject: "1"
```

生成的必填项：`consul.register/enabled=true`、`consul.register/service.name`、标签 `yunshu-metrics: "tag"`。

### 优雅下线示例

```yaml
deployment-base:
  terminationGracePeriodSeconds: 45
  lifecycle:
    preStop:
      exec:
        command: ["/bin/sh", "-c", "sleep 15"]
```

### 自定义 DNS

```yaml
deployment-base:
  dnsPolicy: ClusterFirst
  dnsConfig:
    options:
      - name: ndots
        value: "2"
```

## 本地校验

```bash
cd helm
helm lint .
helm template release-name . -f values.yaml
# 预览开启 SkyWalking：
helm template release-name . -f values.yaml \
  --set deployment-base.skywalking.enabled=true
```

## Jenkins

`--set` 建议：

- `deployment-base.image.repository` / `deployment-base.image.tag`
- `deployment-base.replicaCount` / `deployment-base.containerPort`
- `service-base.port`
- 可选：`deployment-base.skywalking.enabled=true`
