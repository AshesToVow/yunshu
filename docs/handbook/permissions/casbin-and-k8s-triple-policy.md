# 权限设计：Casbin API 与 K8s 三元策略

## 0. 控制台里三块东西分别管什么（先读这段）

运维控制台里和「谁能动 K8s」相关的配置**不是**一个开关搞定，而是**三道独立闸门**，顺序大致如下（同一请求可能依次经过多层）：

| 控制台入口 | 数据库 / Casbin 落点 | 管什么 |
|------------|----------------------|--------|
| **授权管理**（角色 + 权限树） | `casbin_rule` 里 **`/api/v1/...` + HTTP 方法** 的 `p` 策略 | **能不能调用这个 HTTP 接口**（最外层：没勾对应能力，多数情况直接 403）。 |
| **API 管理** 里每行能力的 **「K8s 范围校验」开关**（`permissions.k8s_scope_enabled`） | 不单独存一条策略，只决定该路由是否进入 **K8s 三元中间件目录** | **仅开关**：打开后，该接口在带 `cluster_id` 等条件时才会继续做「集群 + 命名空间 + 路径」校验；关闭则三元中间件**不处理**该路由（见 §3.5）。 |
| **K8s 集群访问档位** 页 + **命名空间黑/白名单** | 表 **`k8s_cluster_access_grants`**（主体 `role` / `user` / `group` + 集群 + `preset` + **`capabilities` JSON 能力包**）；黑名单 **`k8s_namespace_deny_rules`**、白名单 **`k8s_namespace_allow_rules`** | 在已通过 API 鉴权的前提下，按 **能力包**（路由映射到 `read`/`exec`/`restart`/…）与 NS 规则收紧；快捷三档会展开为能力包；**黑名单优先**（见 §3.5）。 |

**容易混淆的点**：

- **三元开关不是「给角色授权」**：它只是标记「这条 API 要不要走 K8s 范围校验逻辑」。真正控制集群侧能力的是 **K8s 集群访问档位** 页写入的 **`k8s_cluster_access_grants`**；API 能否调用仍看 **授权管理**。
- **黑名单只管「请求里出现的那个 namespace」**：列表命名空间等请求若未携带具体 `namespace`，中间件会当成集群级 `_cluster`，**不会**套用按命名空间的黑名单（详见 §3.5 与场景 A 的验证方式）。

### 0.1 API 管理里的「K8s 范围校验」开关（`k8s_scope_enabled`）是什么意思？

- **开**：该 `resource + action` 对应的 Gin 路由会进入 `K8sScopeAuthorize` 的**接口目录**；当请求能解析出 `cluster_id`（以及部分场景下的 `namespace`）时，会叠加 **命名空间黑名单、白名单（若启用）** 与 **`k8s_cluster_access_grants` 档位判定**（按路由所需最低档位与当前用户**角色 + 用户 ID + 用户组编码**对应主体在该集群上的最高档位比较）。
- **关**：该路由对三元中间件而言视为**未纳入目录**，**不做**上述 K8s 范围校验（其它层仍可能拦截，例如授权管理未勾选）。
- **误开给非 K8s 接口时**：`K8sScopeAuthorize` 仅在请求能解析出 **`cluster_id > 0`** 时才做黑/白名单与档位比较；解析不到 `cluster_id` 时**直接放行**该层（见 `k8s_scope_authorize.go`）。因此一般不会把「项目管理」等接口整块弄挂；但若某接口的请求体/查询串里**误带** `cluster_id` 且用户又无集群档位，则可能在本层 403——应对方式是**只对真实访问集群资源的 API 打开开关**。

实现入口：`internal/middleware/k8s_scope_authorize.go`（目录来自 `permissions` 表构建的映射）、`internal/model/permission.go` 字段 `K8sScopeEnabled`。

---

## 1. 模型

- 配置文件：`configs/casbin_model.conf`（随仓库提供）。  
- 适配器：GORM 表 `casbin_rule`（见 Casbin GORM Adapter）。

## 2. API 级权限（主体：用户 ID）

中间件：`internal/middleware/casbin.go` → `Authorize`

- **Subject**：`UserSubject(user.ID)`，形如 `user:{id}`（见 `internal/service/system/casbin_sync.go`）。
- **Object**：`c.FullPath()`，例如 `/api/v1/projects/:id/members`。
- **Action**：HTTP Method，如 `GET`、`POST`。

用户与角色码的绑定：`SyncUserRoles` 为每个用户建立 `g(user:123, viewer)` 这类分组；**API 级 `p` 策略**通常挂在 **角色码**（如 `viewer`）或资源上，由「授权管理」界面维护。

### 2.1 super-admin

角色码包含 `super-admin` 时 **不做 API 级 Casbin 校验**，直接放行（便于新接口上线前未录入策略时仍可管理）。

例外：当请求命中 **K8s 三元中间件**（路径已纳入 scope 目录且能解析出 `cluster_id`）时，**命名空间黑名单与白名单**先于 super-admin 的档位旁路执行；避免平台账号误操作受保护命名空间或越过白名单收窄。

### 2.2 权限元数据

表 `permissions` 字段：

- `resource`：与 Gin `FullPath` 一致的路径模板  
- `action`：`GET` / `POST` / `PUT` / `DELETE` 等  

种子：`cmd/seed.go` 中 `defaultPermissions()`，运行 `go run . seed` 会 upsert 并给 `super-admin` 加策略。

### 2.3 K8s 集群访问档位 / 能力包挂在谁身上？

- **集群侧能力**在表 **`k8s_cluster_access_grants`**：`principal_kind`（`role` / `user` / `group`）+ `principal_ref` + `cluster_id`（`0` = 全部集群）+ `preset`（`readonly` / `readonly_exec` / `admin` / `custom`）+ **`capabilities`**（JSON 数组，如 `["read","exec","scale"]`）。
- 空 `capabilities` 时按 `preset` 展开（兼容旧数据）；有勾选时以能力包为准，并自动补 `read`。
- 鉴权：`K8sScopeAuthorize` 用 `RequiredK8sCapability` 得到路由所需能力码，再与 `EffectiveCapabilities`（合并角色/用户/组在该集群的授权）做包含判断；`access_rank` 由能力包推导供写门禁兼容。
- 目录：`GET /api/v1/k8s-policies/capabilities`；下发：`POST /api/v1/k8s-policies/grant-preset`（可只传 `capabilities`）。
- **结论**：可对角色 / 用户 / 用户组下发快捷档位或自定义能力包，并配合命名空间黑/白名单。

## 3. K8s 范围校验（能力包 + 命名空间）

通过 API 鉴权后，按 **能力包** 收紧集群操作。控制台 **K8s 集群访问授权** 可勾选能力或使用三档快捷填充。

实现：`k8s_scoped_policy_service.go`、`k8scaps`、`k8s_capabilities.go`、`k8s_scope_authorize.go`。

### 3.1 与 k8m 多集群权限模型的对照（概念）

[k8m](https://github.com/weibaohui/k8m)（参见 [多集群 RBAC 与命名空间权限说明](https://zread.ai/weibaohui/k8m/20-multi-cluster-rbac-and-namespace-permissions)）采用 **平台管理员 / 集群角色（cluster_admin、cluster_readonly、cluster_pod_exec）+ 用户或用户组绑定 + 命名空间白名单/黑名单** 的一体化模型，并在权限检查中 **黑名单优先于白名单**。

yunshu 当前采用 **API 用 Casbin + 集群用数据库档位**：

1. **API 权限**：`permissions` + `casbin_rule`，校验 Gin `FullPath` + HTTP 方法。
2. **K8s 范围校验**：对标记了 `k8s_scope_enabled`（或描述标签）的接口，在能解析 `cluster_id` 时，用 **`k8s_cluster_access_grants`** 比较用户档位与路由所需档位；**无档位则拒绝**（不再使用「无 Casbin 三元则放行」）。

与 k8m 的差异与对应关系：

| k8m 能力 | yunshu 融合方案 |
|----------|----------------|
| 集群级粗粒度角色（只读 / 管理员 / Exec） | **能力包**：快捷档位或勾选 `capabilities`；`POST /api/v1/k8s-policies/grant-preset`；运行时按 `RequiredK8sCapability` 校验。 |
| 用户组绑定集群权限 | 表 **`k8s_cluster_access_grants`** 支持 `principal_kind=group`；组成员通过 JWT 上下文中的组编码参与匹配。 |
| 命名空间 **黑名单** 优先 | `K8sScopeAuthorize` 内 **先于** 白名单与 `super-admin` 档位旁路执行。 |
| 命名空间 **白名单** | 表 **`k8s_namespace_allow_rules`**；若某主体在某集群存在任意白名单行，则**仅允许**所列命名空间（`_cluster` 级请求不受白名单限制）。API：`/api/v1/k8s-namespace-allow-rules`。 |

预设下发可选 **`deny_namespaces`** / **`allow_namespaces`**：在已选择**具体集群 ID**（非「全部集群」）时，同步写入 `k8s_namespace_deny_rules` / `k8s_namespace_allow_rules`。

### 3.2 读接口兜底

`allowReadByK8sClusterGrant`（`internal/middleware/casbin.go`）：当用户 **无** 对应 API 的 Casbin 允许，但对 **GET** 且路径通过 `IsK8sClusterGrantReadBypassPath` 时，若其在 **`k8s_cluster_access_grants`** 上（按 **PrincipalPack** 匹配）对该请求上下文中的 `cluster_id` 具备 **至少只读档位**（`GET /api/v1/clusters` 则要求「存在任意一条档位记录」），则放行。

该兜底**排除** `secrets`、`exec`/`file`、`k8s-policies`、Harbor Chart 等敏感面，这些必须显式 Casbin。`IsK8sReadAPIPath` 仍用于 `K8sScopeAuthorize` 的 GET 档位校验，**不可**单独作为 Casbin 放行依据。

例外：`GET /api/v1/menus/tree` 登录即可（不走 Casbin；与菜单管理 `GET /api/v1/menus` 分离）。

### 3.3 与 API 权限关系

- **写操作**（POST/PUT/DELETE）仍需要 **授权管理** 中的 API 权限；集群档位再限制其在集群侧是否达到 `admin` 等。  
- **集群列表** `GET /api/v1/clusters`：当前用户任一主体存在任意 `k8s_cluster_access_grants` 记录即可通过读接口兜底（在未授予 API 级 GET 时）。

### 3.4 与 k8m「资源 Watch」的对照（无 AI/MCP）

[k8m 文档](https://zread.ai/weibaohui/k8m/13-kubernetes-resource-watchers-and-event-handling) 描述的是**常驻**多资源 ListWatch、聚合缓存；代码里多用 **kom + 具体类型**（例如 `Resource(&v1.Pod{}).Watch(...)`），并非「HTTP 传入任意 GVR + RESTMapper 校验」这一通用形态。yunshu 未引入同等规模的常驻 Informer，改为 **按需** `GET /api/v1/k8s/resource-watch/stream`：**短名模式**仅需 `resource=pods` 等内置别名；**任意 GVR 模式**在同一路径上传 `group`（核心组可空）、`version`、`resource`（API 上的**复数名**），由当前集群 Discovery + `RESTMapper.ResourceFor` / `RESTMappings` 解析作用域并校验，再走 client-go dynamic `Watch` → SSE；鉴权仍为 `K8sScopeAuthorize` + 命名空间黑/白名单。扩展点见 `internal/pkg/eventbus`、`internal/pkg/extension`，**不是** k8m 插件状态机（参见 [插件](https://zread.ai/weibaohui/k8m/28-developing-custom-plugins) / [架构](https://zread.ai/weibaohui/k8m/6-system-architecture-and-design-philosophy)）。

### 3.5 K8s 范围中间件：判定顺序

实现见 `internal/middleware/k8s_scope_authorize.go`：

1. **未纳入 K8s 范围目录**的路由 → **不拦截**，直接 `Next`。
2. 解析 **`cluster_id`** / **`namespace`**；若 **`cluster_id == 0`** → **不拦截**（兼容旧接口）。
3. **命名空间黑名单**：当 `namespace` 非空且不为 `_cluster` 时校验；命中则 **403**（**含 super-admin**）。
4. **命名空间白名单**：当 `namespace` 非空且不为 `_cluster`，且该用户任一主体在该集群配置了白名单规则时，若当前 `namespace` 不在允许列表则 **403**（**含 super-admin**）。
5. **`super-admin`**：通过黑/白名单后 **跳过档位比较**，直接放行。
6. **其他用户**：计算路由所需最低档位与用户在当前集群上的有效最高档位；**档位不足则 403**（**无记录即无集群权限**）。

**实践含义**：为业务角色显式下发 **`k8s_cluster_access_grants`**；仅靠 API 权限而无集群档位时，在带 `cluster_id` 的 K8s 接口上会 **403**。

## 4. 配置前准备（检查清单）

| 步骤 | 说明 |
|------|------|
| 确认平台集群 ID | 黑/白名单、按集群下发档位时，需使用 **平台「集群管理」中的数字 ID**；规则唯一键为 `(principal_kind, principal_ref, cluster_id, namespace)`。 |
| 选定主体 | 集群档位与黑/白名单可按 **角色码**、**用户 ID**（字符串形式）、**用户组编码** 配置；控制台「K8s 集群访问档位」页当前以角色模板为主，API 支持 `user_id` / `group_id` 下发。 |
| 绑定用户 / 组 | 在 **用户管理 → 分配角色** 或 **角色管理 → 分配用户** 中绑定角色；在 **系统管理 → 用户组管理** 中维护组与成员（`user_groups` / `user_group_users`）。服务端鉴权从数据库加载用户、角色与组（见 `internal/middleware/auth.go`），**角色或组成员变更会在该用户下一次带 Token 的请求起生效**；若前端缓存了「当前用户」资料，可刷新页面或重新拉取 `/auth/me`。 |
| API 级菜单/授权 | 控制台菜单仍受 API 级 Casbin 控制；仅配置集群档位而未在「授权管理」勾选对应 API 时，用户可能仍无法进入某页（与 `allowReadByK8sClusterGrant` 等兜底规则并存，见 3.2）。 |

## 5. 场景化配置指导

以下场景均在 **「K8s 集群访问档位」** 相关页面或等价 API 完成；命名空间黑名单亦可在同页底部 **「命名空间黑名单」** 区块维护。

### 场景 A：禁止查看与操作若干命名空间（黑名单，推荐用于「生产 / kube-system 隔离」）

**目标**：某业务角色在 **指定集群** 下完全不能访问 `kube-system`、`production` 等命名空间（列表、详情、Exec 等凡带该 `namespace` 的请求均拒绝）。

**配置步骤**：

1. 打开 **命名空间黑名单**，选择角色码（如 `viewer`）。
2. **集群** 选择 **具体集群**（平台登记 ID），或与档位一致选择 **「全部集群（ID=0）」**：`cluster_id=0` 表示通配全部集群，在任意带真实 `cluster_id` 的请求上都会参与匹配（与 `k8s_cluster_access_grants` 语义一致）。
3. 命名空间填写 `kube-system` → 添加；对 `production` 重复添加多条规则。
4. 将需要受限的用户绑定到该角色。

**验证**：以该用户登录，在控制台选中上述集群并切换到被禁命名空间，应收到类似「当前角色在此集群下禁止访问命名空间「xxx」」的 403。

**说明**：黑名单 **优先于** 集群档位放行；且对 **super-admin** 同样生效（见 2.1），用于防止平台账号误入关键命名空间。

---

### 场景 B：仅允许少数命名空间（白名单式收窄）

**目标**：用户只能访问 `app-ns-a`、`app-ns-b`，其它命名空间一律不可操作。

**配置步骤**（当前实现为 **集群级档位**，不按命名空间拆分 preset）：

1. 为该角色在目标集群上下发所需 **档位**（如 `readonly`）。
2. 对其余**不允许**的命名空间，在 **命名空间黑名单** 中逐条添加（维护成本随 NS 数量上升）。
3. 更推荐：为「受限可见」与「全集群只读」等场景 **拆分不同角色**，减少黑名单行数。

**验证**：切换到已加入黑名单的命名空间应 403。

**说明**：若需与 k8m 一样「按 NS 白名单」且黑名单过长，属于产品演进方向（可新增 allow-list 表）；当前以 **档位 + 黑名单** 为主。

---

### 场景 C：能看工作负载与 Pod，但不能 Exec、不能删 Pod

**目标**：保留控制台只读浏览，禁止终端 / Exec、删除 Pod 等高危操作。

**配置步骤**：

1. 在 **K8s 集群访问授权** 用快捷档位 **只读**，或自定义勾选仅 `只读浏览`（不要勾 Pod 终端 / 删除 / 高危等）。
2. API：`POST /api/v1/k8s-policies/grant-preset`，传 `capabilities: ["read"]`，或 `preset: "readonly"`。
3. 若历史上误发了 `readonly_exec` / `admin`，删除对应授权行或改勾能力包后重新保存。
4. 在 **授权管理** 中避免为该角色勾选 Pod 删除、Exec 等写类 API（与 3.3 一致）。

**验证**：只读列表与详情可用；打开终端或删除 Pod 应失败（缺能力 `exec` / `delete`）。

---

### 场景 D：只读浏览 + 仅对部分命名空间开放 Exec

**目标**：在 `dev` 命名空间允许 Exec，在 `staging` 只读，生产命名空间禁止。

**限制**：**同一角色在同一集群仅有一个档位**（`readonly` / `readonly_exec` / `admin`），档位不随命名空间变化。

**可行做法**：

1. **拆分角色**：例如 `dev-operator` 绑定 `readonly_exec` + 黑名单禁止生产 NS；`staging-viewer` 绑定 `readonly`。
2. 或 **同一用户绑定多角色**：Casbin/API 权限合并；集群档位取 **各角色在该集群上的最高档位**（见 `EffectiveTier`）。
3. 对 **`production`**：仍可用 **场景 A 黑名单** 禁止整个 NS。

**说明**：若坚持单角色单集群内「按 NS 不同能力」，需要后续产品扩展（按 NS 的 allow/deny 与 Exec 标志等）。

---

### 场景 E：档位一键下发并同步写入多条黑名单

**目标**：给 `viewer` 批量下发只读档位，同时在集群 `5` 上禁止 `kube-system`、`monitoring`。

**配置步骤**（与 `K8sScopedPolicyGrantPresetRequest` 字段一致）：

1. 调用 **`POST /api/v1/k8s-policies/grant-preset`** 或在页面使用 **按档位下发**。
2. `preset` 选 `readonly`；**`cluster_ids` 必须为具体 ID 列表**（不可依赖「全部集群」占位来写黑名单）。
3. `deny_namespaces` 填 `["kube-system","monitoring"]`（或页面等价多选/逐条输入）。

**说明**：`deny_namespaces` 仅在选中 **明确集群 ID** 时才会写入 `k8s_namespace_deny_rules`；与在黑名单 UI 手工添加等价。

---

### 场景 F：多集群只读 + 每集群不同黑名单

**目标**：同一 `viewer` 在集群 A 禁止 `default`，在集群 B 禁止 `kube-system`。

**配置步骤**：在黑名单中 **按集群分别添加** 规则（每条规则绑定一个 `cluster_id`）；同一角色码可有多行。

---

### 场景 G：验证「读接口兜底」与集群列表

**目标**：理解为何「仅有集群档位、未配全 API GET」时仍可能拉通部分 K8s 读接口。

**要点**（见 3.2）：对符合 `IsK8sClusterGrantReadBypassPath` 的 **GET**，若用户在 `k8s_cluster_access_grants` 中有档位，可能通过 `allowReadByK8sClusterGrant` 获得读取能力；**集群列表** 在「存在任意集群档位记录」时亦可列出。secrets / exec / policies 不走该兜底。

**运维建议**：若希望 **严格禁止** 某读接口，应在 **授权管理** 中取消对应 API GET，而不要仅依赖「未勾选菜单」的单一手段。

## 6. 能力边界与常见误区

| 话题 | 说明 |
|------|------|
| **能否按 Pod 名称单独授权？** | **不能。** 鉴权维度为 **集群 + 命名空间 + 平台 API 路径 + 动作码**；请求体中的 Pod 名不参与三元串。若需「同一 NS 内不同 Pod 不同权限」，应在集群侧使用 K8s RBAC、Admission、OPA 等方案。 |
| **黑名单是否区分 HTTP 动作？** | 否。命中黑名单后该命名空间下相关请求一律拒绝。 |
| **`namespace` 为空时** | 中间件会规范为 `_cluster`；黑名单 **不** 对 `_cluster` 生效（见 `IsDenied` 实现）。 |
| **只配黑名单不配三元** | 黑名单仍可拦截；但若用户无任何 API 与菜单权限，可能仍无法进入控制台，需结合「授权管理」配置。 |
| **未下发集群档位** | 见 3.5：带 `cluster_id` 的 K8s 范围接口会 **403**；需为角色在 **K8s 集群访问档位** 页配置档位。 |

## 7. 运维建议

- 新增 REST 路由后：更新 `defaultPermissions` + 运行 seed + 在「授权管理」给业务角色勾选。  
- 最小权限原则：生产环境避免给普通用户 `super-admin`；K8s 生产命名空间优先用 **黑名单** 或 **窄命名空间三元** 双重保险。
- 变更集群档位或 Casbin 规则后，若用户仍报权限异常，可让其 **刷新页面** 或 **重新登录** 以排除前端状态缓存；服务端中间件每次请求会重载用户角色（见 `auth` 中间件）。

## 8. 长期凭证与第四层门禁（AccessIntent）

平台侧 Casbin/档位/NS 策略解决的是「Yunshu 用户能否点这个按钮」；真正打到 apiserver 的仍是入库的集群凭证。为缩小 blast radius，运行时再叠一层：

| 机制 | 实现要点 |
|------|----------|
| AccessIntent | 中间件按路由写入 `read` / `write` / `exec`；`GetClusterKubectl` / `GetClusterRestConfig` 按意图选凭证 |
| 只读 kubeconfig | `k8s_clusters.kubeconfig_readonly`；只读意图优先使用 |
| Impersonation | **已下线**（字段保留兼容，运行时忽略）；权限仅平台集群授权 |
| 写门禁 | `assertK8sWritable`：只读意图禁止变更 |
| 高危确认 | `require_destructive_confirm` + 请求 `confirm=true`（Drain / Helm 卸载 / RBAC Apply 等） |
| Secret | `/secrets/detail` 脱敏；`/secrets/reveal` 需 admin，审计 redact |
| 加密 | 无 `security.encryption_key` 时拒绝密封新凭证（fail-closed） |

实现：`internal/service/k8s/k8s_runtime_credential.go`、`gate.go`、`k8sauth/access_intent.go`。

**生产建议**：

- 配置 kubeconfig 或直连凭证后，仅在 Yunshu「集群授权」给用户档位与 NS 策略即可；详见 [`deploy/k8s/yunshu-cluster-connect.md`](../../../deploy/k8s/yunshu-cluster-connect.md)。
- **Impersonation 已下线**（不再向 apiserver 伪装用户，避免集群侧手工 YAML）。库表字段保留兼容，运行时忽略。
- 可选：拆分只读/可写 kubeconfig，缩小 blast radius。