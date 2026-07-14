# Loggie → Elasticsearch 日志采集

Yunshu 日志平台已切换为 **Loggie 采集 + Elasticsearch 存储 + Yunshu 代理查询**。
请停用原 `log-agent` DaemonSet，改用本目录清单。

## 架构

```
Pod/Node 日志 → Loggie DaemonSet → ES Sink → Elasticsearch
                                              ↓
                                    Yunshu GET /logs/search
```

## 前置条件

1. 已部署 Elasticsearch（建议 7.x/8.x，索引前缀 `yunshu-logs`）
2. 已安装 Loggie CRD 与 Controller（参考 [Loggie 安装文档](https://loggie-io.github.io/docs/user-guide/quick-start/quick-start/)）
3. Yunshu `config.yaml` 中 `elasticsearch.enabled: true` 并填写 `addresses`

## 字段约定

| 字段 | 说明 |
|------|------|
| `project_id` | 项目 ID |
| `server_id` | 可选，CMDB 服务器 ID |
| `service_id` | 可选，服务 ID |
| `log_source_id` | 可选，日志源 ID |
| `message` / `body` | 日志正文 |
| `@timestamp` | 时间戳 |
| `file_path` | 文件路径 |

清单里会固定写入 ES 字段 `project_id`（与项目一致）。  

- **默认（联调）**：`labelSelector: pod-template-hash: "*"`（匹配多数 Deployment 创建的 Pod）。Loggie **必须**有 labelSelector，省略≠采全部。  
- **生产**：勾选「仅采集带 yunshu.project_id 的 Pod」。注意 `kubectl label deploy` **不会**写到 Pod，需：  
  `kubectl -n <ns> label pod --all yunshu.project_id=<ID> --overwrite`  
  或 `patch deploy ... spec.template.metadata.labels`。  
- `kubectl apply` 缺 last-applied 注解时，省略字段**删不掉**旧 `labelSelector`——应先 `kubectl delete clusterlogconfig ...` 再 apply。  

注意：项目里的「日志源配置」（节点+容器路径）只服务**二进制** Loggie；K8s DaemonSet 只认 ClusterLogConfig/Sink。

引用独立 Sink CR 时用 **`sinkRef: yunshu-es`**。误写成 `sink: yunshu-es` 会触发  
`cannot unmarshal !!str yunshu-es into sink.Config`，采集链路不会生效。

`labelSelector` 必须是简单 map（例如 `"yunshu.project_id": "1"`），**不要**写 Kubernetes 的 `matchExpressions`。

## 部署

### 方式 A：Yunshu 控制台（推荐）

1. 先 Helm 安装 Loggie CRD / Controller / DaemonSet（见 `01-install-note.yaml`）
2. 打开 **Loggie 状态** → **K8s 引导**
3. 选择已接入的集群、Namespace（默认 `loggie`）、DaemonSet 名（默认 `loggie`）
4. 勾选「引导后立即 apply」：Yunshu 会通过集群 kubeconfig apply Namespace + Sink + ClusterLogConfig，并滚动重启 DaemonSet
5. 给业务 Pod 打标签：`yunshu.project_id=<项目ID>`（可选 `yunshu.server_id` / `yunshu.service_id` / `yunshu.log_source_id`）

二进制主机采集仍走同一页面的服务器行「引导」→ SSH 下发。

### 方式 B：kubectl 手工

```bash
kubectl apply -f deploy/k8s/loggie/00-namespace.yaml
# 先 helm install loggie（见 01-install-note.yaml）
kubectl apply -f deploy/k8s/loggie/02-sink-elasticsearch.yaml
kubectl apply -f deploy/k8s/loggie/03-clusterlogconfig.yaml
```
