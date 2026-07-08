# Jenkins 前端多 Node 版本

Yunshu 将前端 CI 的 `node_version` 映射为 Jenkins Job 参数 **`nodeToolName`**。共享库与 Jenkinsfile 已实现按该参数切换 Node 安装。

**实现位置（本仓库）：**

| 文件 | 改动 |
|------|------|
| `jenkinslib/src/org/devops/tools.groovy` | `resolveNodeHome()`、`resolveFrontendBuildOpts()` 增加 `nodeToolName` |
| `jenkinslib/src/org/devops/build.groovy` | `compileFrontend()` 使用 `tool(nodeToolName)` 下的 node/npm |
| `jenkinsfile-new/front.jenkinsfile` | 编译阶段打印所选 Node 工具 |

## 1. Jenkins 全局工具（一次性）

**Manage Jenkins → Global Tool Configuration → NodeJS**，新增：

| Name（工具名） | 版本示例 |
|----------------|----------|
| `node18`         | 18.20.x  |
| `node20`         | 20.11.x  |
| `node22`         | 22.x     |

名称必须与 Yunshu CI 配置下拉选项一致。保留原有 `npm`/`yarn` 工具不影响；yarn 若不在 Node 安装目录内会回退到 Global Tool `yarn`。

## 2. 部署到 Jenkins

将 `jenkinslib` 推送到 Gitee 共享库（如 `jenkins_share_libraries_yunshu`），将 `jenkinsfile-new` 推送到流水线仓库。Jenkins **Global Pipeline Libraries** 名称须与 `front.jenkinsfile` 中 `@Library("...")` 一致。

## 3. 验证

1. 在 Yunshu 保存前端 CI 配置（会 sync Jenkins Job，出现 `nodeToolName` 参数）。
2. 选手动构建，确认日志出现所选工具名。
3. 构建日志中应出现 `nodeToolName=node20` 及对应 `node -v`。

## 4. 已有 Job 未出现新参数

在 Yunshu 中重新保存该服务的 **CI 配置**，触发 Jenkins Job XML 更新；或于 Jenkins 中删除 Job 后由 Yunshu 再次 sync 创建。
