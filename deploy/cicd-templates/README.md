# CI/CD 模板参考（Apollo 多 Meta + 多语言流水线）

Jenkins 共享库（正式源）：

- https://gitee.com/wxd_ops/jenkins_share_libraries_yunshu.git

Jenkinsfile 仓库：

- https://gitee.com/wxd_ops/jenkinsfile_yunshu.git

## 多语言 Script Path（Yunshu `language_type`）

Yunshu CI 配置中的「流水线语言模板」会写入 `cicd_ci_configs.language_type`，同步 Job 时优先解析为下列 Script Path（`custom` 仍按服务类型选 front/backend/k8s）：

| language_type | 推荐 Script Path | 说明 |
|---------------|------------------|------|
| `go` | `backend.jenkinsfile` | Go 后端（复用 backend 流水线） |
| `java` | `backend.jenkinsfile` | Maven/Gradle |
| `frontend` | `front.jenkinsfile` | npm/yarn |
| `python` | `backend.jenkinsfile` | Python 后端 |
| `custom` | （按 service_type） | 兼容旧行为 |

`build_type` 仍表示编译器/打包参数模板（npm/mvn 等），与 `language_type` 并存。

参数约定（与共享库一致）：`enableSonar`、`SONAR_*`、`YUNSHU_CALLBACK_*`、`PROJECT_GROUP`、`APOLLO_*` 等由 Yunshu 触发构建时注入。

## Apollo

对应文件：

- `resources/backend-launch-template.sh`
- `resources/k8s-skywalking-*.yaml` / `resources/k8s-basic-*.yaml`
- `src/org/devops/deploy.groovy`（SSH 占位符替换）
- `src/org/devops/template.groovy`（生成 `APOLLO_OPTS`）

`APOLLO_META` 本身就是逗号分隔字符串，例如：

```text
http://10.241.243.21:8080,http://10.241.243.20:8080,http://10.241.243.19:8080
```

占位符仍用单个 `{{APOLLO_META}}`（不要拆成多个参数）。SSH 启动须给 `-Dapollo.meta=...` 加引号。

本目录片段便于对照；改共享库以仓库内真实文件为准。

> **Yunshu 模板中心（P1）**：平台菜单「系统管理 → 模板中心」已纳管下列引用键（MySQL 权威 + 可选 MinIO 镜像）。业务解析优先已发布版本，未发布回退内置种子；**不改变** Jenkins 共享库 SCM 权威源，可在此编辑后人工同步回 Gitee。
>
> | template_key | 对应本目录文件 |
> |--------------|----------------|
> | `cicd.apollo.backend-launch` | `resources/backend-launch-apollo.snippet.sh` |
> | `cicd.apollo.k8s-env` | `resources/k8s-apollo.env.snippet.yaml` |
> | `cicd.consul.register` | `resources/k8s-consul-register.snippet.yaml` |

| 文件 | 用途 |
|------|------|
| `resources/backend-launch-apollo.snippet.sh` | SSH 启动脚本中 Apollo 启动参数片段 |
| `resources/k8s-apollo.env.snippet.yaml` | K8s Deployment env 片段 |
| `resources/k8s-consul-register.snippet.yaml` | Pod 模板 Consul 必填标签/注解（同步到 k8s-basic / k8s-skywalking） |
| `resources/k8s-consul-register.md` | 必填项说明（kube-consul-register） |

## Consul 注册（kube-consul-register）

业务 Pod **必填**（写在 `spec.template.metadata`）：

1. 注解 `consul.register/enabled: "true"`
2. 注解 `consul.register/service.name: "k8s-pod"`（目录）或 `"k8s-pod-metrics"`（采集）
3. 标签 `yunshu-metrics: "tag"`（值必须是 `tag`）

Helm 脚手架默认已写入 `deployment-base.consulRegister`。kubectl 模板把 [`resources/k8s-consul-register.snippet.yaml`](./resources/k8s-consul-register.snippet.yaml) 合并进共享库 YAML。详情见同目录 `k8s-consul-register.md`。
