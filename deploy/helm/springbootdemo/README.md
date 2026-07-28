# springbootdemo Helm Chart

> 推荐：在 Yunshu CI/CD 容器化发布里选 `deploy_method=helm`，点「下载 Helm 脚手架」，比手工复制本目录更省事。

配合 Yunshu + Jenkins K8s 流水线（`deployMethod=helm`）使用。

## 放入业务仓库

将本目录 **整体复制** 到 Git 仓库根目录，命名为 `helm/`：

```text
springboot-demo/          # 你的 Gitee 仓库
├── Dockerfile
├── pom.xml
├── src/
└── helm/                 # ← 复制本目录内容到这里
    ├── Chart.yaml
    ├── values.yaml
    ├── .helmignore
    └── templates/
```

Jenkins 共享库要求路径：**仓库根目录 `helm/Chart.yaml`**。

## 与 Jenkins 参数对应

| Jenkins 参数 | Chart 字段 |
|--------------|------------|
| `imageName` | Chart.name / Release 前缀 |
| `FULL_IMAGE_NAME` | 解析为 `image.repository` + `image.tag` |
| `replicas` | `replicaCount` |
| `ContainerPort` | `containerPort` |
| `k8s_ns` | `helm upgrade -n` 命名空间 |
| `Tenv` | Release 名：`{imageName}-{Tenv}` |

## 前置条件

1. 目标命名空间（如 `cityos`）已创建
2. 已创建拉取镜像 Secret：`registry-secret`
3. Harbor Chart 仓库已启用（CI 会 `helm package` 并 push）

## 本地校验

```bash
cd helm
helm lint .
helm template springbootdemo-prod . \
  --set image.repository=harbor.deploy.local/registry/springbootdemo \
  --set image.tag=latest_prod_20260705_002619 \
  --set replicaCount=2 \
  --set containerPort=8080
```

## CD「制品发布」注意

Yunshu CD 会跳过 CheckOut，Helm 部署前工作区没有 `helm/` 目录会报错。
除本 Chart 外，还需在 Jenkinsfile 增加「制品发布 + helm 时拉取代码」阶段，或 CD 时改用 `deployMethod=kubectl`。
