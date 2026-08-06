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

| 文件 | 用途 |
|------|------|
| `resources/backend-launch-apollo.snippet.sh` | SSH 启动脚本中 Apollo 启动参数片段 |
| `resources/k8s-apollo.env.snippet.yaml` | K8s Deployment env 片段 |
