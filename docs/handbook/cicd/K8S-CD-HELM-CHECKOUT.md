# K8s CD 制品发布 + Helm 必须拉取 Chart

## 现象

仓库已有 `helm/Chart.yaml`，CI 自动发布成功，但 Yunshu **CD 制品发布**仍报错：

```text
ERROR: Helm Chart目录不存在或Chart.yaml文件缺失
```

日志特征：

```text
[CD] 制品发布，使用镜像: harbor.../springbootdemo:...
Stage "CheckOut" skipped due to when conditional
Stage "发布到K8s" → fileExists → error
```

## 原因

| 构建类型 | publishMode | CheckOut | 工作区是否有 helm/ |
|----------|-------------|----------|-------------------|
| CI 自动发布 | 自动发布 | ✅ 执行 | ✅ 有 |
| Yunshu CD | **制品发布** | ❌ 跳过 | ❌ **空** |

CD 与 CI 是**两次独立的 Jenkins 构建**（如 #24 CI、#25 CD）。CI 推送到 Harbor 的 Chart **不会**出现在 CD 构建的工作区。

Helm 部署脚本在本地检查 `helm/Chart.yaml`，制品发布未拉代码则必然失败。

## 修复（jenkinsfile_yunshu）

在 `cigroovy.jenkinsfile` 中，于 **「发布到K8s」之前** 增加阶段：

```groovy
// 文件顶部已有：def flow = new org.devops.pipeline()
// String publishMode / deployMethod 已在 pipeline 块外解析好

stage('拉取Helm Chart') {
    when {
        allOf {
            expression { publishMode == 'artifact-deploy' }
            expression { deployMethod.toString().trim().toLowerCase() == 'helm' }
        }
    }
    steps {
        script {
            tools.PrintMsg(this, "制品发布+Helm：拉取 Chart 代码", "checkout")
            flow.checkoutCode(this, [branch: branch, srcURL: srcURL, recordCommit: false])
        }
    }
}
```

**注意**：`when { expression { ... } }` 内不能写 `org.devops.tools.xxx`，Jenkins 沙箱会把 `org` 当成 Binding 变量，报 `MissingPropertyException: No such property: org`。

提交到 `git@gitee.com:wxd_ops/jenkinsfile_yunshu.git` 后重新触发 CD 发布。

## 可选：共享库 k8sdeploy.groovy

在 `PackageAndPushChart` 或 `HelmDeploy` 开头：

```groovy
def chartFile = 'helm/Chart.yaml'
if (!fileExists(chartFile)) {
    echo '制品发布+Helm：工作区无 Chart，拉取代码...'
    // 调用与 pipeline.checkoutCode 等价的 checkout
}
```

## 临时绕过

CD 仅更新镜像、不改 Chart 时，部署配置改用 `deploy_method=kubectl`（走「生成部署模板」+ FULL_IMAGE_NAME）。

## Harbor Chart 仓库地址 / 证书路径（构建 #27 类错误）

若日志出现：

```text
helm repo add ... https://harbor.jdicity.local/chartrepo/myrepo --ca-file /usr/local/harbor.jdicity.local.crt
Error: can't read CA file: /usr/local/harbor.jdicity.local.crt
```

说明共享库 `k8sdeploy.groovy` 使用了 **写死的旧 Harbor 配置**，与 Pod 挂载的 `harbor.deploy.local.crt` 不一致。

修复：更新 `jenkins_share_libraries_yunshu` 中 `config.groovy` + `k8sdeploy.groovy`，改为读取 Job 参数：

| 参数 | 用途 |
|------|------|
| `HARBOR_URL` | 如 `harbor.deploy.local` |
| `PROJECT_GROUP` | Chart 项目，如 `registry`（不是 `myrepo`） |

Chart 仓库 URL：`https://${HARBOR_URL}/chartrepo/${PROJECT_GROUP}`  
CA 证书路径：`/usr/local/${HARBOR_URL}.crt`（与 `k8s.groovy` helm 容器卷挂载一致）

本地参考实现见 `jenkinslib/src/org/devops/config.groovy`、`k8sdeploy.groovy`。


CD 构建日志应出现：

1. `拉取Helm Chart` 阶段执行（非 skipped）
2. `发布到K8s` 不再报 Chart.yaml 缺失
3. `helm upgrade --install` 成功
