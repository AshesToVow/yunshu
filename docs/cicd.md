# CI/CD 插件说明

**最后更新**: 2026-08-06  
**插件名**: `cicd`（须与 `project` 同时启用）

---

## 1. 能力概览

| 能力 | 说明 |
|------|------|
| 应用服务 | 项目内 CI/CD 服务定义（标识符、Jenkins Job 名、制品类型等） |
| CI 配置 | Jenkinsfile 仓库/分支、构建参数 |
| 发布配置 | SSH / K8s / MinIO 等多目标发布模板 |
| CI 打包 | 触发 Jenkins Job，同步构建状态与控制台日志 |
| CD 发布 | 多级审批流 + 执行发布（SSH 脚本 / K8s apply / MinIO 制品） |
| 金丝雀 / 蓝绿 | 容器发布配置 `deploy_strategy`；Jenkins 参数 + 平台晋级/中止 API |
| 待办与批量 | 待审批/待执行工单列表，支持批量通过/驳回/执行 |
| 总览图表 | 首页「项目上线趋势」「按人发布统计」读 `cicd_release_runs` |

**代码入口**：

- 插件：`internal/plugins/cicd/plugin.go`（`StartWorkers` → `RunSyncWorker` 同步 Jenkins 状态）
- 服务：`internal/service/cicd/service.go`
- 路由：`internal/router/register_cicd_routes.go`（前缀 `/api/v1/projects/:id/cicd/...`）
- 前端：`web/src/modules/cicd/routes.tsx`

---

## 2. 启用步骤

1. `configs/config.yaml`：

```yaml
plugins:
  enabled:
    - core
    - project
    - cicd
    # ...
```

2. `go run . migrate && go run . server`（创建 `cicd_*` 表）
3. **系统管理 → 数据字典** 配置（优先级高于 YAML 默认值，见 `internal/dictconfig/cicd.go`）：

| dict_type（示例） | 含义 |
|-------------------|------|
| `cicd_enabled` | 是否启用 CI/CD |
| `cicd_jenkins_base_url` | Jenkins 地址 |
| `cicd_jenkins_username` / `cicd_jenkins_api_token` | Jenkins 凭据 |
| `cicd_jenkins_job_folder` | Job 目录（可选） |
| `cicd_jenkinsfile_*` | Jenkinsfile 仓库与分支 |
| `cicd_minio_*` | 制品 MinIO（与备份 `minio_*` **独立**） |
| `cicd_sonar_enabled` | **SonarQube 门禁总开关**（`true`/`false`，默认关闭） |
| `cicd_sonar_url` / `cicd_sonar_token` | Sonar 地址与 Token（启用后门禁写入构建参数） |
| `cicd_sonar_gate_block` | 门禁失败是否拦截进入审批/发布（默认 `true`） |
| `cicd_jenkins_callback_url` | Jenkins → Yunshu 回调完整 URL（注入 `YUNSHU_CALLBACK_URL`） |
| `cicd_jenkins_callback_hmac_secret` | 回调 HMAC 密钥（同时注入 Jenkins 参数 `YUNSHU_CALLBACK_HMAC_SECRET`） |

4. 重启后端使字典与插件 Worker 生效。

---

## 3. API 前缀

所有接口在项目作用域下，需 **项目成员** 权限：

```text
GET/POST   /api/v1/projects/:id/cicd/services
GET/PUT    /api/v1/projects/:id/cicd/services/:serviceId/ci-config
GET/POST   /api/v1/projects/:id/cicd/services/:serviceId/deploy-configs
POST       /api/v1/projects/:id/cicd/services/:serviceId/builds
POST       /api/v1/projects/:id/cicd/services/:serviceId/releases
GET        /api/v1/projects/:id/cicd/build-runs
GET        /api/v1/projects/:id/cicd/build-runs/:runId/stages
GET        /api/v1/projects/:id/cicd/build-runs/:runId/artifacts-meta
GET        /api/v1/projects/:id/cicd/release-runs
GET/PUT    /api/v1/projects/:id/cicd/approval-flow
POST       /api/v1/projects/:id/cicd/release-runs/:runId/approve|reject|execute|terminate
```

Jenkins 回调（**无登录**，HMAC 鉴权）：

```text
POST /api/v1/cicd/jenkins/callback
Header: X-Yunshu-Signature: sha256=<hmac-sha256-hex(body)>
```

完整列表见 `internal/router/register_cicd_routes.go` 或 OpenAPI。

---

## 3.1 SonarQube 质量门禁（字典开关）

1. 在数据字典将 `cicd_sonar_enabled` 设为 `true`，并配置 `cicd_sonar_url` / `cicd_sonar_token`。
2. CI 触发时注入 Jenkins 参数：`enableSonar=true`、`SONAR_HOST_URL`、`YUNSHU_CALLBACK_URL`、`YUNSHU_BUILD_RUN_ID`。
3. 流水线在 `sonar` / `quality_gate` 阶段通过 HMAC 回调回写 `cicd_run_stages` 与构建上的 `quality_gate_status`。
4. 当 `cicd_sonar_gate_block=true` 且门禁为 `ERROR`/`NONE`（或成功构建无门禁结果）时，**容器发布选择该构建会被拒绝**（先于审批与执行）。
5. CD「制品发布」默认强制 `enableSonar=false`，避免重复扫描。
6. Jenkins 侧实现：共享库 `org.devops.yunshu`（[jenkins_share_libraries_yunshu](https://gitee.com/wxd_ops/jenkins_share_libraries_yunshu)）+ [jenkinsfile_yunshu](https://gitee.com/wxd_ops/jenkinsfile_yunshu)。

---

## 3.2 阶段回调与制品元数据

- 表：`cicd_run_stages`（阶段）、`cicd_artifacts`（包/镜像/Helm + digest）。
- 回调 `event`：`stage` | `sonar` | `artifact` | `run`；可用 `run_id` 或 `jenkins_job`+`build_number` 定位记录。

---

## 4. 审批与通知

- 审批阶段定义：`cicd_approval_flow_stages`（项目级 `GET/PUT .../approval-flow`）
- **空审批流禁止发布**：未配置阶段时 `configured=false`，发布申请被拒绝
- 工单步骤：`cicd_release_approval_steps`，按阶段顺序推进
- **职责分离**：`forbid_self_approve`（配置/字典）禁止审批人审批自己提交的工单
- **生产强制审计**：`prod_force_audit` 要求生产环境发布须走审批留痕
- 批量审批/驳回/执行：全部失败时返回**首个错误**原因（非静默 `count=0`）
- Jenkins 回调 `event=run`：状态机校验（如 running → success/failure），拒绝乱序回写
- 邮件通知：`internal/service/cicd/notify_email.go` 通过 **`UserRepository`** 解析用户邮箱（不直查 DB）
- 前端「应用服务」：新建应用须项目 owner/admin；操作按钮按 `access.can_build/can_release/can_manage` fail-closed

配置项见 `configs/config.yaml` → `cicd.forbid_self_approve` / `cicd.prod_force_audit`，以及字典 `cicd_*`。
---

## 5. 后台 Worker

`cicd` 插件 `StartWorkers` 启动 `Service.RunSyncWorker`：

- 间隔：`config.cicd.run_sync_interval_seconds`（默认 15s，可被字典覆盖）
- 作用：轮询 Jenkins，更新 `cicd_build_runs` 状态与日志片段（与 HMAC 回调互补）

---

## 6. 与其它模块关系

| 模块 | 关系 |
|------|------|
| `project` | 路由挂载、成员鉴权、服务器列表（发布目标） |
| `overview` | 读 `cicd_release_runs` 做图表；**cicd 未启用时返回空数据** |
| `dictconfig` | Jenkins / MinIO / Sonar / 回调运行期配置 |
| `backup` | MinIO 键名空间独立（`minio_*` vs `cicd_minio_*`） |

---

## 7. 前端菜单

内置 catalog（`internal/menu/catalog.go`）：

| 路径 | 页面 |
|------|------|
| `/cicd/services` | 应用服务 |
| `/cicd/todo` | 待办列表 |
| `/cicd/approval-flow` | 审批管理 |
| `/cicd/build-records` | CI 打包记录 |
| `/cicd/release-records` | CD 历史工单 |

路径规则：`web/src/modules/plugin-path.ts` 与 `internal/plugin/path_filter.go` 须同步（`/cicd` 与 `/api/v1/projects/*/cicd/*` 归属 `cicd` + `project`）。

---

## 8. 金丝雀 / 蓝绿发布

容器发布配置字段（`cicd_deploy_configs`）：

| 字段 | 说明 |
|------|------|
| `deploy_strategy` | `rolling`（默认）/ `canary` / `blue_green` |
| `canary_replicas` / `canary_percent` / `canary_steps_json` | 金丝雀初始副本、占比提示、晋级步骤（如 `10,50,100`） |
| `blue_green_service` | 蓝绿切换的 Service 名（空则用工作负载名） |

Jenkins 透传参数：`deployStrategy`、`canaryReplicas`、`canaryPercent`、`canarySteps`、`blueGreenService`。

平台晋级 API（发布成功后可在「CD 历史工单」详情操作）：

- `POST .../release-runs/:runId/progressive/promote` — 金丝雀按步骤扩缩 / 蓝绿切到 green
- `POST .../release-runs/:runId/progressive/abort` — 中止（金丝雀缩 0 / 切回 blue）

K8s 约定标签：`yunshu.io/track=canary`、`yunshu.io/color=blue|green`。

---

## 9. 相关文档

- [plugins.md](./plugins.md) — 插件机制
- [CONTRIBUTING.md](../CONTRIBUTING.md) §5 — 路径规则
- [handbook/requirements/menus/menu-cicd.md](./handbook/requirements/menus/menu-cicd.md) — 菜单级需求
- [handbook/cicd/K8S-CD-HELM-CHECKOUT.md](./handbook/cicd/K8S-CD-HELM-CHECKOUT.md) — K8s + Helm CD 实践
- [《企业级K8s容器化部署方案：Jenkins+Harbor+Helm+Gitee+Jenkinsfile自动化实践》.md](./《企业级K8s容器化部署方案：Jenkins+Harbor+Helm+Gitee+Jenkinsfile自动化实践》.md) — Harbor + Jenkins 流水线参考
- [《企业级前端部署方案：Jenkins+MinIO+SSH+Gitee+Jenkinsfile自动化实践》.md](./《企业级前端部署方案：Jenkins+MinIO+SSH+Gitee+Jenkinsfile自动化实践》.md) — 前端 MinIO/SSH 发布参考
- [《企业级后端部署方案：Jenkins+MinIO+SSH+Gitee+Jenkinsfile自动化实践》.md](./《企业级后端部署方案：Jenkins+MinIO+SSH+Gitee+Jenkinsfile自动化实践》.md) — 后端 MinIO/SSH 发布参考
