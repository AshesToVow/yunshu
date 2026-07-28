# Yunshu 与「研发愿用的 Helm 模板体系」对照整理

> 源文档：[不是推标准，而是降门槛：一套研发愿用的 Helm 模板体系.md](./不是推标准，而是降门槛：一套研发愿用的%20Helm%20模板体系.md)

---

## 目录架构（已与源文档对齐）

一键下载脚手架解压到业务仓库根目录后：

```text
your-repo/
├── setup/                         # §四 全局固化配置 Chart（默认 setup.enabled=false）
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
└── helm/                          # §3.2 Application Chart（Jenkins 要求此路径）
    ├── Chart.yaml                 # dependencies → deployment/service/config/hpa/pvc-base + setup
    ├── values.yaml                # 研发主要改这里（文档示例风格）
    ├── values-dev.yaml            # §3.4 多环境
    ├── values-test.yaml
    ├── values-prod.yaml
    ├── config-files/              # §3.3 方法一：目录注入
    ├── charts/                    # §3.1 公共模块（base charts，本地 vendored）
    │   ├── deployment-base/       # Deployment + startup/liveness/readiness
    │   ├── service-base/          # Service
    │   ├── config-base/           # ConfigMap/Secret（fileConfigs）
    │   ├── hpa-base/
    │   └── pvc-base/
    └── templates/                 # Application 薄层：config-files → ConfigMap
```

研发日常：**只改 `helm/values.yaml`（或 `values-<env>.yaml`）+ 往 `config-files/` 丢文件**，不必写 Helm 模板语法。

---

## 怎么用

1. CI/CD → 容器化发布 → 部署方式选 **helm** → **下载 Helm 脚手架**
2. 解压，提交 `helm/` 与可选 `setup/`
3. 改 `values.yaml`：镜像、端口、副本、探活、env、模块开关（`*.enabled`）
4. 照常 CI/CD；`--set` 建议打在 `deployment-base.*` / `service-base.*`

接口：`GET .../cicd/services/:serviceId/helm-scaffold`

---

## 与源文档能力对照

| 源文档 | 脚手架 |
|--------|--------|
| deployment / service / config / hpa / pvc base | `helm/charts/*-base` |
| SkyWalking / 更新策略 / lifecycle / DNS | `deployment-base.skywalking`、`strategy`、`lifecycle`、`dnsPolicy`/`dnsConfig` |
| Application 只维护 values + dependencies | `helm/Chart.yaml` + `values.yaml` |
| config-files 目录注入 | `helm/config-files/` + App 层 CM 模板 |
| values 内 fileConfigs | `config-base.fileConfigs` / 顶层 `fileConfigs` 说明 |
| 多环境 | `values-dev/test/prod.yaml` |
| setup 全局固化 | 仓库根 `setup/` + `setup.enabled` |

后续可将 `charts/*-base` 换成 Harbor OCI 上的公共库，Application 侧目录与用法可保持不变。
