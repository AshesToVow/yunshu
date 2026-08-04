# CI/CD 模板参考（Apollo 多 Meta）

Jenkins 共享库（正式源）：

- https://gitee.com/wxd_ops/jenkins_share_libraries_yunshu.git

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
