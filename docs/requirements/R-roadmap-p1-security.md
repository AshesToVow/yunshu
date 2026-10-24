# P1 安全与成熟度路线立项台账

> 本文为立项台账：只做立项、方案要点与验收标准，**不包含编码**。
> 每个条目在编码前须先产出 `R-*` 详细设计文档（接口、表结构、边界与回滚），评审通过后独立分支、独立提交。
> 承接 [`../yunshu-audit-report.md`](../yunshu-audit-report.md) 的 P1 遗留项与产品成熟度差距分析。
> 状态核实日期：2026-10-24。

## 1. 背景

审计报告（2026-08-05）登记的中危项多数已修复，但仍有几类缺口：

1. **授权纵深不足**：敏感数据明文接口（dict RevealValue）只有路由级 Casbin 门禁，无二次角色校验。
2. **前端权限形同虚设**：深链 + `PATH_COMPONENT_FALLBACK` 绕过菜单控制，路由仅校验登录。
3. **账号安全能力空白**：无 SSO/OAuth2、无 MFA、无异常登录风控（异地/IP/失败锁定）。
4. **可靠性与覆盖**：告警 Redis 去重降级路径缺可观测；关键路径（auth/Casbin/WS ticket/审批流/工单 FSM）单测近 0，全仓语句覆盖率 10.6%。

## 2. 状态核实（2026-10-24 实测，覆盖审计报告第 9 节）

| 原 ID | 结论 | 证据 |
| --- | --- | --- |
| B-H1/B-H2/B-H3、F-H1/F-H2、T-H1 | ✅ 已修 | 审计报告第 9 节 + 本会话回归 |
| B-M2/B-M4/B-M6/B-M7/B-M8、F-M3 | ✅ 已修 | 审计报告第 9 节 |
| F-M1（JWT localStorage） | ✅ 已修 | `web/src/services/storage.ts:130` 清除 JWT 残留，会话已迁 HttpOnly Cookie |
| F-M4（platform-provider 2500 行） | ✅ 已修 | `web/src/pages/alert-monitor/platform-provider.tsx` 现 30 行，已拆分 12 文件 |
| T-M2（前端测试框架） | ✅ 已落地 Vitest | `web/package.json` test 脚本；仅剩 Playwright E2E 未做 |
| B-M1（自审自批） | 保留 | 产品明确接受，列为可配置 SoD 远期项 |
| B-M3 / B-M5 / F-M2 / T-M1 | ❌ 未闭合 | 见下文 SEC-01/02/03、TST-01 |

## 3. 立项清单

| 编号 | 域 | 项 | 优先级 | 工作量 | 状态 |
| --- | --- | --- | --- | --- | --- |
| SEC-01 | 授权 | Dict 敏感值明文接口二次门禁（原 B-M5） | P1 | 0.5d | 待立项 |
| SEC-02 | 可靠 | 告警 Redis 去重降级复核与可观测（原 B-M3） | P1 | 1d | 待立项 |
| SEC-03 | 授权 UX | 前端路由权限元数据，废除深链 fallback（原 F-M2） | P1 | 3d | 待立项 |
| SEC-04 | 账号 | SSO / OAuth2 企业身份源对接 | P1 | 5d | 待立项 |
| SEC-05 | 账号 | MFA（TOTP）+ 敏感操作二次验证 | P1 | 3d | 待立项 |
| SEC-06 | 账号 | 异常登录风控（异地/IP/失败锁定/登录日志告警） | P1 | 3d | 待立项 |
| TST-01 | 测试 | 关键路径单测补齐（auth/Casbin/WS ticket/审批流/工单 FSM） | P1 | 5d | 待立项 |
| TST-02 | 测试 | 一条 Playwright E2E 冒烟（登录→项目→列表） | P2 | 2d | 待立项 |
| TST-03 | 质量 | golangci-lint 存量债清零（errcheck 50 / staticcheck 50） | P2 | 3d | 待立项 |

### SEC-01 Dict 敏感值明文接口二次门禁

- **现状**：`RevealValue`（`internal/handler/dict_entry_handler.go:47` → `internal/service/system/dict_entry_service.go:132`）服务层仅校验「敏感类型 + 路由 Casbin 权限」，无角色级二次校验；路由权限一旦配宽，任何持有该权限者可见全部敏感值明文。
- **方案要点**：服务层注入 `PermissionService`，按字典类型映射「可查看明文」的角色/用户组；`RevealValue` 内先查角色再放行；审计日志记录谁在何时看了哪条。
- **验收标准**：非授权角色请求返回 403 且不落审计外泄；授权角色正常返回；单测覆盖两种路径。

### SEC-02 告警 Redis 去重降级复核与可观测

- **现状**：`internal/service/alert/state_redis.go` 已具备「DB 已写入时 Redis 失败不阻断、恢复判定降级读库」（`:255`）；但 Redis 故障期间去重行为、重复通知放大程度无指标、无告警，故障不可见。
- **方案要点**：核对 `IsDuplicate`/`MarkFiringDelivered` 在 Redis 故障时的 fail-open 面；为降级路径加计数器（Prometheus）与「降级持续时间」日志；必要时对高优先级规则改为 fail-closed。
- **验收标准**：Redis 宕机时重复通知放大有指标可查；降级/恢复有明确日志；设计文档给出「哪些路径可降级、哪些必须失败关闭」的决策表。

### SEC-03 前端路由权限元数据，废除深链 fallback

- **现状**：`web/src/pages/dynamic-menu-page.tsx:10` 的 `PATH_COMPONENT_FALLBACK` 允许未绑定菜单的路径渲染组件；路由仅校验登录态（`app-routes.tsx`）。「关菜单」无法阻止深链访问。
- **方案要点**：后端菜单接口下发「路径→组件/权限」元数据；前端路由守卫按元数据渲染，无绑定即 403 页；保留白名单（首页/个人中心）；`PATH_COMPONENT_FALLBACK` 全量删除。
- **验收标准**：未授权深链一律 403 页或跳首页；删除 fallback 后 `grep PATH_COMPONENT_FALLBACK` 零命中；Vitest 覆盖路由守卫。

### SEC-04 SSO / OAuth2 企业身份源

- **现状**：仅账密 + 邮箱验证码登录；无 SSO/OAuth2。
- **方案要点**：OAuth2 授权码模式对接企业 IdP（如 Casdoor/Keycloak）；`oauth_provider` 配置项 + `oauth_users` 绑定表；首次登录绑定已有账号或自动建号（需管理员开关）；与现有 Session/Casbin 体系衔接，不另起炉灶。
- **验收标准**：授权码→用户映射→Session 全链路；未绑定用户可配置「拒绝/自动创建」；登出联动 IdP。

### SEC-05 MFA（TOTP）

- **现状**：登录仅单因子。
- **方案要点**：TOTP（RFC 6238）绑定与登录二次校验；关键操作（删除项目、批量授权、RevealValue 类）可选二次验证；恢复码机制防锁死；绑定/解绑需审计。
- **验收标准**：绑定、登录、恢复码三流程可用；错误码与 `docs/错误码清单.md` 同步（generrno 漂移检查随 CI 生效）。

### SEC-06 异常登录风控

- **现状**：登录日志仅记录，无风险判定与联动。
- **方案要点**：失败次数锁定（渐进退避）；异地/IP 变更触发邮件通知；高频失败进 `login_risk_events` 表并可在告警平台建规则；预留与 SEC-05 联动（风控命中强制 MFA）。
- **验收标准**：失败锁定、异地提醒、事件表三路可测；不误伤内网同出口用户（IP 段白名单）。

### TST-01 关键路径单测补齐

- **现状**：全仓语句覆盖率 10.6%（CI `-short` 实测 2026-10）；auth 中间件、Casbin 策略、WS ticket、审批流、工单 FSM 基本无覆盖。
- **方案要点**：按域拆分立项：auth/Session 中间件、Casbin 决策、WS ticket 生命周期、CICD/dbmgmt 审批流、dbmgmt 工单 FSM；目标每域 ≥60% 行覆盖，全仓覆盖率同步上调 CI 阈值（现 10.0% 地板）。
- **验收标准**：五域测试文件落地且 CI 绿；覆盖率阈值随提交上调有据可查。

### TST-02 Playwright E2E 冒烟

- **现状**：Vitest 单测已落地；无浏览器级冒烟。
- **方案要点**：单条链路：登录 → 项目列表 → 进入项目 → 登出；`web/e2e/` 目录 + CI `frontend` job 尾部执行。
- **验收标准**：CI 绿且对登录回归敏感（改坏登录即红）。

### TST-03 golangci-lint 存量债清零

- **现状**：golangci-lint v2.7.2 基线 141 条；2026-10 已清零 unused(36)+ineffassign(5) 共 41 条并修复 1 处 SA5011 nil 接收者 panic（`KafkaToESService.Stats`）；余 errcheck 50 / staticcheck 50（含 SA1019 弃用 API、S1016 结构体转换等低风险项）。
- **方案要点**：分两批：errcheck 逐条补错误处理或显式 `_ =` 忽略并注释理由；staticcheck 中 SA5011/SA9004 类先修，SA1019 弃用 API 排期替换；CI 已开 `only-new-issues: true`，存量不阻塞合并但新代码零新增。
- **验收标准**：`golangci-lint run ./...` 零 issue，CI 移除 `only-new-issues` 放行策略。

## 4. 立项流程约束（文档优先门禁）

1. 每个条目开工前先出 `R-*` 详细设计：需求澄清 → 接口/表结构 → 边界与异常 → 测试计划 → 回滚。
2. 评审通过后独立分支、独立提交，不与其他功能变更混排。
3. 涉及错误码/接口的条目，完工时同步 `go run ./tools/generrno` 与 `tools/gen_api_design_md.py`（CI docs-drift 门禁强制）。
