# Yunshu 云原生运维管理平台

## 需求规格说明书 (Software Requirements Specification)

**文档编号**: YUNSHU-SRS-2026-001  
**版本号**: V1.0.0  
**密级**: 内部公开  
**状态**: 正式发布  

---

## 文档控制信息

| 版本 | 日期 | 作者 | 修改内容 | 评审人 | 批准人 |
|------|------|------|----------|--------|--------|
| V0.1  | 2026-05-20 | 需求分析组 | 初稿创建 | - | - |
| V0.5  | 2026-05-21 | 架构师 | 补充技术架构约束 | 技术负责人 | - |
| V1.0  | 2026-05-21 | 产品经理 | 完善功能需求和验收标准 | 项目经理 | 产品总监 |

### 文档变更记录

```
[2026-05-21] V1.0 正式版发布
- 新增完整的功能需求规格说明（6大业务域，200+功能点）
- 补充非功能性需求（性能/安全/可靠性/易用性）
- 添加详细的接口规范和数据字典
- 编写用户故事和验收标准矩阵
- 增加系统约束和技术架构要求
```

---

## 目录

1. [引言](#1-引言)
2. [总体描述](#2-总体描述)
3. [具体需求](#3-具体需求)
4. [非功能性需求](#4-非功能性需求)
5. [接口需求](#5-接口需求)
6. [数据需求](#6-数据需求)
7. [用户故事与用例](#7-用户故事与用例)
8. [验收标准](#8-验收标准)
9. [附录](#9-附录)

---

## 1. 引言

### 1.1 目的

本文档作为 **Yunshu 云原生运维管理平台** 的需求规格说明书，旨在：

1. **明确需求边界**: 清晰定义系统的功能范围、性能指标和质量要求
2. **指导开发实现**: 为开发团队提供详细的设计依据和实现指南
3. **支撑验收测试**: 为QA团队提供可测试的验收标准和测试用例基础
4. **便于沟通协作**: 统一产品、开发、测试、运维各方对系统的理解

### 1.2 范围

#### 1.2.1 产品定位

Yunshu 是一个面向 **DevOps 团队、SRE工程师、运维人员** 的云原生运维管理平台，提供：

- ✅ **统一身份认证与权限管理** (RBAC + K8s 多层鉴权)
- ✅ **多项目租户隔离** (服务器/服务/日志源/告警配置)
- ✅ **Kubernetes 全资源管控** (30+ 资源类型, 多集群纳管)
- ✅ **企业级告警平台** (Prometheus + 多渠道通知 + 值班排班)
- ✅ **实时日志采集与分析** (Agent + SSE 流式推送)
- ✅ **自动化运维工具** (MySQL 备份、终端管理)

#### 1.2.2 功能范围

| 业务域 | 核心能力 | 优先级 |
|--------|----------|--------|
| **R-01 认证与身份** | 登录/注册/JWT/邮箱验证码/个人设置 | P0 (必须) |
| **R-02 项目管理** | 项目CRUD/成员/服务器/服务/日志源/Agent | P0 (必须) |
| **R-03 告警监控** | 数据源/规则/策略/渠道/值班/静默/订阅路由 | P0 (必须) |
| **R-04 K8s 控制台** | 集群/工作负载/网络/存储/RBAC/CRD | P0 (必须) |
| **R-05 系统管理** | 用户/角色/部门/菜单/字典/审计/IP封禁 | P0 (必须) |
| **R-06 日志平台** | Agent采集/SSE实时流/文件发现/过滤高亮 | P1 (重要) |
| **MySQL 备份** | 定时备份/远程归档/MinIO存储 | P2 (可选) |

#### 1.2.3 不包含的范围

以下功能**不在本版本交付范围内**：

- ❌ 多因子认证 (MFA / TOTP)
- ❌ 第三方 OAuth2 / SSO 单点登录
- ❌ 移动端 APP (iOS / Android)
- ❌ 国际化多语言支持 (i18n)
- ❌ Grafana / Kibana 深度集成
- ❌ CI/CD 流水线管理 (Jenkins / GitLab CI)
- ❌ 成本管理与资源计费
- ❌ AI 智能运维 (异常检测/根因分析)

### 1.3 定义与缩写

| 术语 | 定义 |
|------|------|
| **SRE** | Site Reliability Engineering, 站点可靠性工程 |
| **RBAC** | Role-Based Access Control, 基于角色的访问控制 |
| **JWT** | JSON Web Token, 用于身份认证的令牌 |
| **SSE** | Server-Sent Events, 服务器推送事件流 |
| **gRPC** | Google Remote Procedure Call, 高性能 RPC 框架 |
| **PromQL** | Prometheus Query Language, PromQL 查询语言 |
| **CRD** | Custom Resource Definition, Kubernetes 自定义资源 |
| **Casbin** | 开源访问控制框架 |
| **Kom SDK** | weibaohui/kom, Kubernetes 多集群管理 SDK |

### 1.4 参考资料

| 文档名称 | 路径 | 说明 |
|----------|------|------|
| 后端架构技术文档 | `docs/backend-architecture-complete.md` | 详细的技术实现说明 |
| 架构图集 | `docs/architecture-diagrams.md` | 系统架构图和数据流图 |
| 权限设计手册 | `docs/handbook/permissions/casbin-and-k8s-triple-policy.md` | 三层鉴权机制详解 |
| API 文档 (Swagger) | `http://localhost:8080/swagger/index.html` | 在线 API 文档 |
| 数据库 ER 图 | `docs/handbook/database/er-diagrams.md` | 数据库关系图 |
| 告警通知指南 | `docs/alert-notify-guide.md` | 告警投递机制说明 |
| 日志平台 API | `docs/log-platform-api.md` | 日志采集接口规范 |
| OpenAPI 规范 | `docs/apipost/permission-system.openapi.yaml` | API 定义文件 |

### 1.5 概述

本文档后续章节将按照以下结构组织：

- **第2章**: 描述产品的整体架构、运行环境和设计约束
- **第3章**: 详细列出所有功能需求（按业务域分类）
- **第4章**: 定义非功能性需求（性能、安全、可靠性等）
- **第5章**: 说明外部接口规范
- **第6章**: 定义数据模型和数据字典
- **第7章**: 提供用户故事和使用场景
- **第8章**: 给出验收标准和测试要点
- **第9章**: 附录（术语表、缩写表等）

---

## 2. 总体描述

### 2.1 产品视角

#### 2.1.1 产品架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     用户界面层 (Frontend)                      │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐               │
│  │ React SPA │  │ Swagger UI│  │ Log-Agent │               │
│  │ (浏览器)  │  │ (API文档) │  │ (CLI工具) │               │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘               │
└────────┼──────────────┼──────────────┼──────────────────────┘
         │              │              │
         ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│                       应用服务层 (Backend)                     │
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │
│  │ HTTP Server  │    │ gRPC Server │    │ Background      │  │
│  │ (:8080/Gin)  │    │ (:18080)    │    │ Workers         │  │
│  └──────┬──────┘    └──────┬──────┘    └────────┬────────┘  │
│         │                  │                     │            │
│  ┌──────▼──────────────────▼─────────────────────▼────────┐  │
│  │                 中间件链路 (Middleware)                  │  │
│  │  Recovery → Logger → Auth → Casbin → K8sScope → Audit  │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────▼─────────────────────────────┐  │
│  │                    业务逻辑层 (Service)                   │  │
│  │  Auth / Project / Alert / K8s / Log / System           │  │
│  └────────────────────────┬───────────────────────────────┘  │
└───────────────────────────┼──────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│    MySQL      │   │    Redis     │   │ External APIs │
│  (主数据库)   │   │  (缓存/会话)  │   │ (K8s/Prom/SMTP)│
└───────────────┘   └───────────────┘   └───────────────┘
```

#### 2.1.2 技术栈选型

| 层次 | 技术选型 | 选型理由 |
|------|----------|----------|
| **前端框架** | React 18 + Ant Design 5 + Vite | 生态成熟、组件丰富、开发效率高 |
| **后端框架** | Go 1.25 + Gin | 高性能、并发能力强、部署简单 |
| **ORM** | GORM | Go 生态最成熟的 ORM, 支持自动迁移 |
| **权限框架** | Casbin v2 | 灵活的 RBAC/ABAC 模型, 社区活跃 |
| **K8s 客户端** | client-go + Kom SDK | 官方客户端 + 多集群抽象 |
| **数据库** | MySQL 5.7+ | 关系型数据, 事务支持好 |
| **缓存** | Redis 7.x | 高性能 Key-Value, 会话/分布式锁 |
| **消息队列** | Redis List (轻量) | 告警异步处理, 降低耦合 |
| **RPC** | gRPC | Agent 通信, 高效二进制协议 |
| **API 文档** | Swaggo (Swagger) | 自动生成, 方便联调 |

### 2.2 用户特征

#### 2.2.1 用户角色定义

| 角色 | 职责描述 | 典型操作频率 | 技术水平要求 |
|------|----------|--------------|--------------|
| **超级管理员 (Super Admin)** | 系统全局管理、权限配置、用户管理 | 每日数次 | 高级 |
| **项目管理员 (Project Owner)** | 项目资源配置、成员管理、服务器维护 | 每日数十次 | 中高级 |
| **开发工程师 (Developer)** | 查看日志、调试 Pod、查看告警 | 每日数十次 | 中级 |
| **运维工程师 (Ops/SRE)** | K8s 资源管理、告警处理、备份运维 | 每小时数次 | 高级 |
| **只读查看者 (Viewer)** | 仅查看仪表盘、日志、告警事件 | 每周数次 | 初级 |

#### 2.2.2 用户使用场景

**场景 A: 日常巡检 (运维工程师)**
```
1. 登录系统 → 查看概览面板 (Pod 状态/告警统计)
2. 进入 K8s 集群 → 查看 Deployment 运行状态
3. 发现异常 Pod → 点击查看日志 (SSE 实时流)
4. 需要进入容器调试 → 启动 Web Terminal (Exec)
5. 收到告警通知 → 进入告警事件页 → 确认并处理
```

**场景 B: 项目初始化 (项目管理员)**
```
1. 创建新项目 → 设置项目编码和描述
2. 添加项目成员 → 分配角色 (Owner/Admin/Member)
3. 导入服务器列表 (Excel 批量导入)
4. 配置服务器分组 (按环境/用途分组)
5. 注册 Log Agent → 配置日志源路径
6. 绑定 Prometheus 数据源 → 配置告警规则
```

**场景 C: 告警响应 (值班人员)**
```
1. 收到钉钉/邮件告警通知
2. 登录系统 → 查看告警事件详情
3. 分析告警标签和 PromQL 表达式
4. 进入 K8s 控制台 → 定位问题 Pod
5. 查看相关日志 → 执行修复操作
6. 确认恢复 → 关闭告警静默 (如需要)
```

### 2.3 运行环境

#### 2.3.1 生产环境要求

| 组件 | 最低配置 | 推荐配置 | 备注 |
|------|----------|----------|------|
| **操作系统** | CentOS 7.9+ / Ubuntu 20.04+ / KyLinux V10 | 同左 | 内核 3.10+ |
| **CPU** | 4 核 | 8 核+ | 按 QPS 弹性扩展 |
| **内存** | 8 GB | 16 GB+ | 含 JVM/Go 进程 |
| **磁盘** | 100 GB SSD | 500 GB SSD | 日志/备份需额外空间 |
| **网络** | 千兆网卡 | 万兆网卡 | 低延迟要求 |
| **Docker** | 20.10+ | 24.0+ | 容器化部署 |
| **MySQL** | 5.7 | 8.0 | InnoDB 引擎 |
| **Redis** | 6.2 | 7.x | 持久化开启 |
| **Kubernetes** | 1.25+ | 1.28+ | 被管集群版本 |

#### 2.3.2 开发环境要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.23+ | 后端开发 |
| Node.js | 18+ | 前端开发 |
| npm/pnpm | 最新 | 包管理 |
| Docker Desktop | 4.x+ | 本地容器化 |
| MySQL Workbench | 8.x+ | 数据库管理 |
| RedisInsight | 最新 | Redis 可视化 |
| VS Code | 最新 | IDE (推荐插件: Go, React, Mermaid) |
| Git | 2.x+ | 版本控制 |

### 2.4 设计和实现约束

#### 2.4.1 技术约束

| 约束项 | 要求 | 原因 |
|--------|------|------|
| **后端语言** | 必须使用 Go 1.25+ | 性能、并发、部署便利性 |
| **前端框架** | 必须使用 React 18+ | 团队技术栈统一 |
| **数据库** | 必须使用 MySQL (不支持 PostgreSQL/MongoDB) | 已有基础设施 |
| **ORM** | 必须使用 GORM | 代码一致性 |
| **API 风格** | RESTful + Swagger 文档 | 标准化、易联调 |
| **认证方式** | JWT Bearer Token | 无状态、易扩展 |
| **权限模型** | Casbin RBAC + K8s Scope | 三层鉴权体系 |
| **容器化** | 必须 Docker 化部署 | 一致性和可移植性 |

#### 2.4.2 业务约束

| 约束项 | 要求 | 原因 |
|--------|------|------|
| **密码加密** | bcrypt (cost ≥ 10) | 安全合规 |
| **敏感字段** | AES-256-GCM 加密存储 | SSH 密钥/Kubeconfig 等 |
| **会话管理** | Redis 白名单 + JWT 双重校验 | 防止 Token 被盗用 |
| **操作审计** | 所有写操作必须记录日志 | 合规追溯 |
| **数据脱敏** | API 不返回密码/Token/密钥明文 | 安全防护 |
| **IP 封禁** | 支持登录失败自动封禁 | 防暴力破解 |
| **注册审核** | 新用户必须管理员审批 | 安全准入 |

#### 2.4.3 性能约束

| 指标 | 目标值 | 测量方法 |
|------|--------|----------|
| **API 响应时间 (P99)** | ≤ 500ms (简单查询) | Prometheus Histogram |
| **API 响应时间 (P99)** | ≤ 2s (复杂聚合查询) | 同上 |
| **并发用户数** | 支持 100+ 并发在线用户 | 压测工具 (JMeter/wrk) |
| **数据库连接池** | 最大 20 连接 | MySQL SHOW PROCESSLIST |
| **Redis 连接池** | 最大 10 连接 | Redis INFO clients |
| **日志 SSE 连接** | 支持 50+ 并发流 | 浏览器 DevTools |
| **WebSocket 连接** | 支持 20+ 并发终端 | 同上 |

### 2.5 假设和依赖

#### 2.5.1 外部依赖

| 依赖组件 | 版本要求 | 可用性要求 | 降级方案 |
|----------|----------|------------|----------|
| **MySQL 数据库** | 5.7+ | 99.9% SLA | 主从切换 (需人工) |
| **Redis 缓存** | 6.2+ | 99.9% SLA | 降级为本地缓存 (受限功能) |
| **Prometheus** | 2.45+ | 95% (告警核心) | 使用历史数据缓存 |
| **Kubernetes API** | 1.25+ | 98% (管控核心) | 显示离线状态 |
| **SMTP 邮件服务** | 标准 SMTP | 90% (通知辅助) | 改用钉钉/Webhook |
| **钉钉 Webhook** | 稳定可用 | 95% (主要通道) | 切换至邮件通知 |
| **MinIO 对象存储** | 最新版 | 99% (备份存储) | 本地文件系统 |

#### 2.5.2 业务假设

1. **网络环境**: 用户通过内网或 VPN 访问系统，延迟 < 50ms
2. **用户规模**: 初期 50-100 人，中期 500 人以内
3. **K8s 集群数**: 3-10 个生产集群 (初期)
4. **告警量级**: 日均 1000-5000 条告警事件
5. **日志量级**: 单项目日均 10 GB 日志 (压缩前)
6. **并发度**: 峰值 50-100 并发 API 请求

#### 2.5.3 风险和缓解措施

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| **MySQL 故障** | 系统不可用 | 低 | 主从复制 + 定期备份 |
| **Redis 故障** | 会话丢失/限流失效 | 低 | Sentinel 集群 + 本地降级 |
| **K8s API 不稳定** | 集群管控异常 | 中 | 重试机制 + 缓存元数据 |
| **告警风暴** | 通知渠道拥塞 | 中 | 聚合去重 + 异步队列 |
| **磁盘空间不足** | 日志/备份失败 | 中 | 盯控告警 + 自动清理策略 |
| **安全漏洞被利用** | 数据泄露 | 低 | 定期安全扫描 + 依赖更新 |

### 2.6 用户界面

#### 2.6.1 UI 设计原则

1. **简洁直观**: 遵循 Ant Design 设计规范, 保持一致性
2. **响应式布局**: 支持桌面端 (1920x1080+) 和笔记本 (1366x768+)
3. **主题定制**: 支持亮色/暗色模式切换 (未来版本)
4. **无障碍**: 符合 WCAG 2.1 AA 级标准 (基本支持)
5. **国际化**: 当前仅中文, 预留 i18n 扩展点

#### 2.6.2 页面结构

```
┌─────────────────────────────────────────────────────────┐
│  Header: Logo + 用户头像 + 消息铃铛 + 退出登录          │
├──────────┬──────────────────────────────────────────────┤
│ Sidebar  │  Main Content Area                           │
│ (菜单树) │                                             │
│          │  ┌────────────────────────────────────────┐  │
│ 系统     │  │  Breadcrumb: 首页 > K8s > Pods          │  │
│ 管理     │  ├────────────────────────────────────────┤  │
│ ├ 用户   │  │                                        │  │
│ ├ 角色   │  │  Search Bar + Filter + Actions Toolbar  │  │
│ ├ 权限   │  │                                        │  │
│ ├ ...    │  │  ┌──────────────────────────────────┐  │  │
│          │  │  │ Data Table / Card List / Form     │  │  │
│ 项目     │  │  │ (分页/排序/展开行/操作按钮)       │  │  │
│ 管理     │  │  │                                  │  │  │
│ ├ 项目   │  │  │                                  │  │  │
│ ├ 服务器 │  │  │                                  │  │  │
│ ├ 服务   │  │  └──────────────────────────────────┘  │  │
│ ├ ...    │  │                                        │  │
│          │  │  Pagination: < 1 2 3 ... 100 >          │  │
│ K8s      │  └────────────────────────────────────────┘  │
│ 管理     │                                             │
│ ├ 集群   │                                             │
│ ├ Pods  │                                             │
│ ├ ...    │                                             │
│          │                                             │
│ 告警     │                                             │
│ 平台     │                                             │
│ └ ...    │                                             │
└──────────┴──────────────────────────────────────────────┘
```

---

## 3. 具体需求

本章将按照业务域详细描述所有功能需求，每个需求包含：
- **需求 ID**: 唯一标识符 (格式: REQ-{域}-{序号})
- **需求名称**: 简短描述
- **优先级**: P0(必须) / P1(重要) / P2(可选)
- **前置条件**: 触发该需求的前提
- **功能描述**: 详细的行为说明
- **输入/输出**: 明确的数据格式
- **业务规则**: 约束条件和异常处理
- **验收标准**: 可测试的完成标志

---

### 3.1 R-01: 认证与身份管理

#### 3.1.1 用户登录

**REQ-AUTH-001: 用户名密码登录**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **前置条件** | 用户已注册且状态为"启用" |
| **功能描述** | 支持用户通过用户名和密码进行登录验证 |
| **输入** | `{ username: string, password: string }` |
| **输出** | 成功: `{ access_token: string, token_type: "Bearer", expires_in: 7200 }`<br>失败: `{ code: 401xx, message: "错误原因" }` |
| **业务规则** | 1. 密码使用 bcrypt 校验 (cost=10)<br>2. 登录成功生成 JWT Token (TTL=120分钟)<br>3. 将 TokenID 写入 Redis Session (白名单)<br>4. 记录登录日志 (含 IP/User-Agent)<br>5. 连续失败 5 次 → IP 封禁 30 分钟 |
| **异常处理** | • 用户不存在 → 40102 "账号或密码错误"<br>• 密码错误 → 40102 "账号或密码错误"<br>• 账户禁用 → 40302 "账户已被禁用"<br>• IP 封禁 → 40303 "IP已被封禁" |
| **验收标准** | ✅ 正确凭证返回 Token<br>✅ 错误凭证返回错误码<br>✅ Token 可通过 Authorization Header 使用<br>✅ 登录日志正确记录<br>✅ IP 封禁机制生效 |

**REQ-AUTH-002: 邮箱验证码登录**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **前置条件** | 用户已绑定邮箱且状态正常 |
| **功能描述** | 通过邮箱接收验证码完成无密码登录 |
| **流程** | 1. 用户提交邮箱 → 发送 6 位数字验证码<br>2. 用户输入验证码 → 校验通过后签发 Token |
| **输入** | Step1: `{ email: string }`<br>Step2: `{ email: string, code: string }` |
| **业务规则** | • 验证码有效期: 600 秒 (10分钟)<br>• 发送冷却期: 60 秒 (防刷)<br>• 验证码错误次数限制: 5 次/次<br>• 验证码使用后立即失效 |
| **验收标准** | ✅ 验证码 60 秒内不能重发<br>✅ 验证码 10 分钟内有效<br>✅ 验证码一次性使用<br>✅ 冷却期内提示剩余时间 |

**REQ-AUTH-003: 密码+验证码双因素登录**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **前置条件** | 用户已知密码且已绑定邮箱 |
| **功能描述** | 输入密码后还需输入邮箱验证码, 增强安全性 |
| **适用场景** | 敏感操作前的二次验证 / 异地登录提醒 |
| **验收标准** | ✅ 两步都通过才签发 Token<br>✅ 任一步失败终止流程 |

#### 3.1.2 用户注册

**REQ-AUTH-004: 注册申请**

| 属性 | content |
|------|---------|
| **优先级** | P0 (必须) |
| **功能描述** | 新用户提交注册申请, 等待管理员审核 |
| **输入** | `{ username, email, password, nickname }` |
| **业务规则** | • 用户名: 3-20 字符, 字母数字下划线<br>• 邮箱: 合法邮箱格式, 全局唯一<br>• 密码: 8-32 字符, 含大小写字母+数字+特殊字符<br>• 昵称: 1-64 字符<br>• 默认角色: 待分配 (审核后由管理员指定)<br>• 默认状态: 待审核 |
| **流程** | 1. 提交申请 → 写入 registration_requests 表<br>2. 状态 = pending<br>3. 管理员审核 → 通过/拒绝<br>4. 通过 → 自动创建 users 记录 (status=1)<br>5. 拒绝 → 记录拒绝原因 |
| **验收标准** | ✅ 申请后无法直接登录<br>✅ 管理员可在后台看到待审核列表<br>✅ 审核通过后可正常登录<br>✅ 审核拒绝收到通知 (邮件/站内信) |

#### 3.1.3 Token 与会话管理

**REQ-AUTH-005: JWT Token 签发与校验**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **技术细节** | • 算法: HS256 (HMAC-SHA256)<br>• Payload: `{ userID, tokenID, username, roleCodes, exp, iat }`<br>• Secret: 从 config.yaml 读取 (≥32字节)<br>• TTL: 120 分钟 (可配置)<br>• TokenID: UUID v4, 用于 Redis Session 关联 |
| **校验流程** | 1. 提取 Bearer Token<br>2. 解析 JWT (验签+过期)<br>3. 查询 Redis Session (tokenID → userID)<br>4. Session 不存在/过期 → 40103 "登录已过期"<br>5. 查询 users 表获取最新状态<br>6. 注入 CurrentUser 到 Context |
| **验收标准** | ✅ Token 过期后返回 401<br>✅ 登出后 Token 立即失效<br>✅ 同一用户多设备登录互不影响 (多 Session)<br>✅ Secret 泄露可紧急轮换 |

**REQ-AUTH-006: 登出与会话销毁**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 用户主动登出时, 立即从 Redis 删除 Session, 使 Token 失效 |
| **输入** | 无 (仅需 Authorization Header) |
| **输出** | `{ message: "logout success" }` |
| **验收标准** | ✅ 登出后原 Token 无法使用<br>✅ 其他设备 Session 不受影响<br>✅ 操作审计日志记录登出事件 |

#### 3.1.4 个人设置

**REQ-AUTH-007: 个人资料修改**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | 用户可修改昵称、邮箱、手机号 (手机用于告警 @通知) |
| **可修改字段** | nickname, email, phone |
| **不可修改字段** | username, id, status, roles (需管理员操作) |
| **业务规则** | • 修改邮箱需重新验证 (发送验证码)<br>• 手机号格式: 11 位数字 (中国大陆)<br>• 修改后更新 operation_log |
| **验收标准** | ✅ 修改后个人信息页面即时刷新<br>✅ 邮箱修改触发验证流程 |

**REQ-AUTH-008: 密码修改**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 用户可通过旧密码设置新密码 |
| **输入** | `{ old_password, new_password, confirm_password }` |
| **业务规则** | • old_password 必须正确<br>• new_password ≠ old_password<br>• confirm_password == new_password<br>• 新密码强度同注册要求<br>• 修改后所有 Session 失效 (强制重新登录) |
| **验收标准** | ✅ 旧密码错误返回错误<br>✅ 两次新密码不一致返回错误<br>✅ 修改成功后需重新登录<br>✅ 操作日志记录密码变更 |

---

### 3.2 R-02: 项目管理

#### 3.2.1 项目 CRUD

**REQ-PROJ-001: 创建项目**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **前置条件** | 用户已登录且有"创建项目"权限 |
| **输入** | `{ name, code, description?, owner_department_id? }` |
| **业务规则** | • name: 1-128 字符, 项目显示名称<br>• code: 1-64 字符, 全局唯一 (用于 URL/API)<br>• description: 可选, 项目描述<br>• owner_department_id: 可选, 归属部门<br>• 创建人自动成为 project_member (role=owner)<br>• status 默认 = 1 (启用) |
| **异常处理** | • code 重复 → 400 "项目编码已存在"<br>• 权限不足 → 403 "无权创建项目" |
| **验收标准** | ✅ 项目创建成功返回完整对象<br>✅ 创建人自动加入为 Owner<br>✅ 编码唯一性校验生效 |

**REQ-PROJ-002: 项目列表与查询**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 支持分页、搜索、筛选、排序 |
| **查询参数** | page, page_size, keyword (模糊匹配 name/code), status, owner_department_id |
| **权限逻辑** | • Super Admin: 可见所有项目<br>• 普通用户: 仅可见自己是成员的项目<br>• 返回数据包含: member_count, server_count, agent_count |
| **输出格式** | `{ list: [...], total: N, page: 1, page_size: 20 }` |
| **验收标准** | ✅ 分页正确 (总数/页数)<br>✅ 搜索关键字命中 name 或 code<br>✅ 非 Super Admin 只看到自己项目 |

**REQ-PROJ-003: 更新与删除项目**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **更新规则** | 仅 Owner/Admin 可修改 name/description/status<br>修改 code 需特殊权限 (可能影响 API 路由)<br>删除项目: 级联删除成员/服务器/服务/日志源/Agent (软删除) |
| **业务规则** | • 删除前检查是否有活跃的告警规则/备份任务<br>• 删除操作需二次确认 (前端弹窗)<br>• 删除后数据保留 30 天 (可恢复) 后物理删除 |
| **验收标准** | ✅ 非 Owner/Admin 无法修改/删除<br>✅ 删除确认机制生效<br>✅ 级联清理关联数据 |

#### 3.2.2 项目成员管理

**REQ-PROJ-004: 添加项目成员**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **前置条件** | 当前用户是项目的 Owner 或 Admin |
| **输入** | `{ user_id, role: "owner\|admin\|member\|readonly" }` |
| **角色权限** | • **owner**: 项目最高权限 (转移/删除项目)<br>• **admin**: 管理成员/服务器/服务/Agent<br>• **member**: 查看项目资源, 执行允许的操作<br>• **readonly**: 仅查看, 不能修改 |
| **业务规则** | • 同一 user_id 在同一 project_id 下唯一<br>• 一个项目至少有一个 owner<br>• owner 数量不限, 但建议 1-3 人<br>• 添加成员时可选择是否发送通知邮件 |
| **验收标准** | ✅ 成员立即可见项目资源<br>✅ 角色权限差异明显 (readonly 无法编辑)<br>✅ 成员数量统计准确 |

**REQ-PROJ-005: 成员角色调整与移除**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | Owner/Admin 可调整成员角色或移除成员 |
| **业务规则** | • 不能降级唯一的 owner (需先转移或添加新 owner)<br>• 移除自己 → 自动退出项目 (若为最后 owner 则禁止)<br>• 角色变更后权限即时生效 (无需重新登录) |
| **验收标准** | ✅ 角色升降级即时生效<br>✅ 最后一个 Owner 无法被移除/降级<br>✅ 移除成员后其失去项目访问权限 |

#### 3.2.3 服务器管理

**REQ-PROJ-006: 服务器 CRUD**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **输入字段** | hostname (必填), ip, port (22), project_id, group_id?, description? |
| **敏感信息** | SSH 凭据 (username/password/private_key) 存储在 server_credentials 表 (AES-256-GCM 加密)<br>API 返回时不返回明文, 仅显示 has_credential: true/false |
| **批量操作** | • Excel 导入: 模板下载 → 填写 → 上传 → 批量创建<br>• Excel 导出: 筛选结果导出为 xlsx<br>• 批量测试: 选择多个服务器 → 并发执行连通性检测 |
| **连通性测试** | • SSH 连接测试 (超时 10s)<br>• 返回: { reachable: bool, latency_ms, auth_method, os_info } |
| **验收标准** | ✅ 服务器按分组树形展示<br>✅ 凭据加密存储, API 不泄露<br>✅ 批量导入/导出/测试正常工作 |

**REQ-PROJ-007: 服务器分组**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | 支持树形分组 (类似文件夹), 便于组织大量服务器 |
| **结构** | • parent_id 自引用形成树形<br>• 支持无限层级 (建议 ≤ 3 层)<br>• 分组名在同一父节点下唯一 |
| **典型分组示例** | ```
项目 A
├── 生产环境
│   ├── 北京机房
│   │   ├── Web 服务器组
│   │   └── DB 服务器组
│   └── 上海机房
├── 测试环境
└── 预发环境
``` |
| **验收标准** | ✅ 树形展示清晰<br>✅ 拖拽排序 (未来版本)<br>✅ 分组统计 (服务器数量) |

**REQ-PROJ-008: 服务器终端 (WebSocket)**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | 通过浏览器 WebSocket 直接连接服务器 Shell (类似 XShell Web 版) |
| **技术实现** | • Frontend: xterm.js + WebSocket<br>• Backend: SSH Client 库 (golang.org/x/crypto/ssh)<br>• 协议: WS → Backend SSH Proxy → Target Server:22 |
| **权限要求** | • 项目成员 (member 及以上)<br>• 服务器凭据已配置<br>• 网络可达 (Backend 能连目标服务器) |
| **特性** | • 支持复制粘贴<br>• 支持 Tab 键补全<br>• 断线重连 (30s 内)<br>• 操作日志记录 (命令级别, 不记录密码) |
| **验收标准** | ✅ 终端交互流畅 (延迟 < 100ms)<br>✅ 支持常用快捷键 (Ctrl+C 等)<br>✅ 断线有明确提示 |

#### 3.2.4 服务配置

**REQ-PROJ-009: 服务注册**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | 在项目下注册业务服务 (如: user-service, order-service), 用于关联日志源 |
| **输入** | `{ name, server_id, port, description?, technology_stack? }` |
| **关联关系** | service → server (N:1)<br>service → log_sources (1:N) |
| **典型用途** | • 区分同一服务器上的多个服务进程<br>• 按服务维度查看日志<br>• 告警规则按服务粒度配置 |
| **验收标准** | ✅ 服务归属到具体服务器<br>✅ 服务下可配置多个日志源 |

#### 3.2.5 日志源配置

**REQ-PROJ-010: 日志源 CRUD**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 定义日志采集的来源 (文件路径/命令/journal), 供 Agent 使用 |
| **类型** | • **file**: 文件路径, 支持 glob 模式 (如 `/var/log/app/*.log`)<br>• **command**: 动态命令输出 (如 `docker logs -f container_name`)<br>• **journal**: systemd journal (Linux 系统日志) |
| **输入** | `{ service_id, source_type, file_path/command, pattern?, encoding?: "utf-8\|gbk", exclude_pattern? }` |
| **业务规则** | • file_path 支持 glob 通配符 (* ? [])<br>• pattern: 正则表达式, 用于日志行过滤 (可选)<br>• exclude_pattern: 排除特定行 (如 debug 级别)<br>• encoding 默认 utf-8, 可指定 gbk (旧系统兼容) |
| **Agent 关联** | Agent 启动时拉取该项目下所有日志源配置 → 开始采集 |
| **验收标准** | ✅ 日志源配置保存成功<br>✅ Agent 能识别并采集指定路径<br>✅ glob 模式匹配正确 |

#### 3.2.6 Log Agent 管理

**REQ-PROJ-011: Agent 注册**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 在平台上注册 Log Agent, 获取 Token 用于认证 |
| **注册方式** | • **手动注册**: 管理员在界面填写 server_id → 平台生成 token<br>• **自助注册**: Agent 使用 register_secret 调用公共注册 API |
| **Token 特性** | • UUID v4 格式, 不可猜测<br>• 与 server_id 绑定 (唯一)<br>• 可轮换 (Rotate Token) → 旧 Token 立即失效<br>• 存储在 log_agents.token 字段 (AES 加密) |
| **Agent 状态** | • online: 心跳正常 (30s 内)<br>• offline: 心跳超时 (>90s 未上报)<br>• unknown: 从未收到心跳 |
| **验收标准** | ✅ Token 生成后可用于 Agent 认证<br>✅ Token 轮换后旧 Token 失效<br>✅ Agent 状态实时更新 |

**REQ-PROJ-012: Agent 列表与监控**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **展示信息** | • 服务器主机名 / IP<br>• Agent 版本号<br>• 最后心跳时间<br>• 状态 (online/offline/unknown)<br>• 运行时长 (uptime)<br>• 采集的日志源列表<br>• 最近错误日志 (如有) |
| **操作按钮** | • 详情: 查看 Agent 元信息和配置<br>• 删除: 移除 Agent (停止采集)<br>• Token 轮换: 重新生成 Token<br>• 心跳刷新: 手动触发一次心跳检测<br>• Bootstrap: 下载安装脚本/配置 |
| **验收标准** | ✅ 列表展示完整<br>✅ 状态颜色区分 (绿/红/灰)<br>✅ 操作按钮权限正确 |

**REQ-PROJ-013: Agent 发现**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | Agent 定期扫描服务器上的日志文件, 上报发现的文件列表, 帮助用户快速创建日志源 |
| **扫描周期** | 默认 30 分钟 (可配置 discovery_interval)<br>扫描根目录: 从 runtime-config 的 _discovery_root 获取 (默认 `/var/log`) |
| **发现逻辑** | 1. Agent 递归扫描 _discovery_root 目录<br>2. 过滤: 仅保留 .log .out .txt 文件<br>3. 排除: 系统目录 (/proc /sys /dev), 过大文件 (>1GB)<br>4. 上报: 文件路径 + 大小 + 修改时间 + 行数估算 |
| **平台侧展示** | • "未配置日志源的发现文件" 面板<br>• 一键创建日志源 (预填 file_path)<br>• 批量忽略 (加入黑名单) |
| **验收标准** | ✅ 发现列表准确反映服务器文件<br>✅ 一键创建日志源便捷<br>✅ 黑名单排除生效 |

---

### 3.3 R-03: 告警与监控平台

*(由于告警平台功能复杂, 这里只列出核心需求要点, 详细需求请参考 docs/requirements/R-alert-platform-detailed-design.md)*

#### 3.3.1 告警数据源

**REQ-ALERT-001: Prometheus 数据源管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 绑定 Prometheus 实例, 作为告警规则的查询数据源 |
| **输入** | `{ project_id, name, url, token?, description? }` |
| **能力** | • Ping 测试: 验证 Prometheus 可达性<br>• PromQL 查询: 即时查询 (debug 用途)<br>• 活跃告警视图: 展示当前 firing 的告警 |
| **业务规则** | • 一个数据源可关联多条监控规则<br>• 规则的项目归属从数据源继承<br>• Token 用于带鉴权的 Prometheus |
| **验收标准** | ✅ 数据源连接测试成功<br>✅ PromQL 查询返回正确结果 |

#### 3.3.2 监控规则

**REQ-ALERT-002: PromQL 监控规则**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 基于 PromQL 表达式定义监控规则, 定期评估是否触发告警 |
| **输入** | ```json
{
  "datasource_id": 1,
  "name": "CPU使用率过高",
  "promql_expr": "100 - (avg by(instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
  "severity": "warning",  // critical/warning/info
  "for_duration": "5m",     // 持续多久才算触发
  "eval_interval": "30s",   // 评估间隔
  "labels": {"team": "backend", "env": "prod"},
  "enabled": true,
  "annotations": {
    "summary": "实例 {{ $labels.instance }} CPU 使用率超过 80%",
    "description": "当前值: {{ $value }}%"
  }
}
``` |
| **评估引擎** | • Cron 驱动 (默认 */5 * * * * *)<br>• 查询 Prometheus → 判断条件 → 生成/解决告警<br>• 支持暂停/启用单条规则 |
| **验收标准** | ✅ 规则评估正确触发告警<br>✅ for_duration 生效 (避免抖动)<br>✅ 规则启停即时生效 |

#### 3.3.3 告警渠道

**REQ-ALERT-003: 通知渠道配置**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **支持的渠道类型** | • **dingding**: 钉钉机器人 Webhook<br>• **email**: SMTP 邮件<br>• **webhook**: 自定义 HTTP 回调<br>• **wecom**: 企业微信机器人 (预留) |
| **通用配置** | ```json
{
  "type": "dingding",
  "name": "运维钉钉群",
  "is_default": false,
  "config": {
    "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "SEC...",  // 签名密钥
    "timeout_ms": 5000
  },
  "template": {  // 可选自定义模板
    "title_template": "[{{ .Status }}] {{ .Labels.alertname }}",
    "body_template": "..."
  }
}
``` |
| **测试功能** | • 发送测试消息: 验证渠道配置正确性<br>• 模板预览: 渲染示例告警数据查看效果 |
| **验收标准** | ✅ 测试消息成功送达<br>✅ 模板渲染正确 (变量替换) |

#### 3.3.4 告警订阅路由

**REQ-ALERT-004: 订阅树路由**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 树形结构的路由规则, 匹配告警标签 → 决定通知给谁、通过哪个渠道 |
| **节点属性** | ```json
{
  "id": 1,
  "parent_id": null,
  "name": "默认路由",
  "match_labels_json": "{}",        // 标签精确匹配
  "match_regex_json": "{}",         // 正则匹配
  "receiver_group_id": 1,            // 接收组
  "channel_ids": [1, 2],             // 渠道列表
  "is_default": true,                // 是否兜底节点
  "silence_window": {                // 静默窗口 (可选)
    "start_time": "23:00",
    "end_time": "08:00"
  },
  "routing_priority": 0,             // 优先级 (数值越小越优先)
  "enabled": true
}
``` |
| **匹配逻辑** | 1. 从根节点开始深度优先遍历<br>2. 检查每个节点的 match_labels / match_regex<br>3. 第一个匹配的节点获胜 (短路)<br>4. 若无一匹配 → 使用 is_default=true 的节点<br>5. 若无 default → 告警丢弃 (不通知) |
| **操作** | • 创建/编辑/删除节点<br>• 移动节点 (拖拽排序)<br>• 克隆路由 (从其他项目复制)<br>• 从旧策略迁移 (兼容旧 policies 表) |
| **验收标准** | ✅ 路由匹配符合预期<br>✅ Default 兜底生效<br>✅ 节点移动后顺序正确 |

#### 3.3.5 值班管理

**REQ-ALERT-005: 值班排班**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (重要) |
| **功能描述** | 按时间段配置值班人, 命中时段的告警优先通知值班人 |
| **输入** | ```json
{
  "monitor_rule_id": 1,
  "start_time": "2026-05-21T09:00:00Z",
  "end_time": "2026-05-21T18:00:00Z",
  "user_ids": [101, 102],  // 值班人列表 (可多人)
  "title": "白班-张三/李四"
}
``` |
| **业务规则** | • 值班块按 monitor_rule_id 归属 (不同规则独立排班)<br>• 时间段不可重叠 (同一规则下)<br>• 当前时刻命中的值班人优先收到通知<br>• 通知标题前缀加 "【值班】" 标识<br>• 支持循环排班 (未来版本: 按周/月模板) |
| **验收标准** | ✅ 值班时段内告警通知给值班人<br>✅ 非值班时段走普通通知逻辑<br>✅ 值班标题显示正确 |

#### 3.3.6 告警静默

**REQ-ALERT-006: 静默规则**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 在指定时间窗口内, 匹配特定标签的告警不发送通知 |
| **输入** | ```json
{
  "matchers": [
    {"key": "alertname", "value": "CPUHigh", "regex": false},
    {"key": "namespace", "value": "test-*", "regex": true}
  ],
  "start_time": "2026-05-21T22:00:00Z",
  "end_time": "2026-05-22T06:00:00Z",
  "created_by": 1,
  "comment": "发布窗口, 忽略测试环境告警"
}
``` |
| **批量创建** | • 从活跃告警列表勾选多条 → 批量创建静默<br>• 自动填充 matchers (复用告警标签) |
| **验收标准** | ✅ 静默期间匹配告警不发送通知<br>✅ 静默到期后恢复正常通知<br>✅ 静默列表可查询/编辑/删除 |

#### 3.3.7 告警事件与历史

**REQ-ALERT-007: 告警事件查询**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 查询历史告警事件, 支持多维筛选 |
| **筛选条件** | • 时间范围 (最近 1h/6h/24h/7d/自定义)<br>• 状态: firing / resolved<br>• 严重程度: critical / warning / info<br>• 数据源 / 规则 / 渠道<br>• 标签模糊搜索<br>• 项目 (基于数据源派生) |
| **展示字段** | fingerprint, status, labels, annotations, started_at, resolved_at, channel_id, notify_count |
| **统计接口** | GET /alerts/history/stats → 按 severity/group_by 聚合计数 |
| **验收标准** | ✅ 列表查询响应 < 2s (万级数据)<br>✅ 筛选条件组合正确<br>✅ 统计数据准确 |

---

### 3.4 R-04: Kubernetes 控制台

*(K8s 资源管理需求较多, 这里列举核心模块, 详细请参考 docs/requirements/R-04-kubernetes-console.md)*

#### 3.4.1 集群管理

**REQ-K8S-001: 集群接入**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (必须) |
| **功能描述** | 纳管 Kubernetes 集群, 支持 kubeconfig 和直连两种模式 |
| **接入模式** | • **kubeconfig**: 粘贴 kubeconfig YAML 内容 (平台加密存储)<br>• **direct**: 填写 API Server 地址 + Token/Cert<br>• 安全: kubeconfig/token 使用 AES-256-GCM 加密, API 不返回明文 |
| **校验** | • 保存后立即测试连接 (List Namespaces)<br>• 返回: { connected: true, version: "v1.28.0", namespaces_count: 15 } |
| **归属项目** | owning_project_id 可选:<br>• 空: 平台级集群 (Super Admin 可见)<br>• 非空: 项目私有集群 (仅项目成员可见) |
| **验收标准** | ✅ 集群连接成功<br>✅ 凭据加密存储<br>✅ 项目隔离生效 |

**REQ-K8S-002: 集群状态监控**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **展示信息** | • 组件状态: apiserver/controller-manager/scheduler/etcd<br>• 节点状态: Ready/NotReady/Unknown<br>• 资源概览: Pod 数/Namespace 数/Node 数 |
| **刷新机制** | 手动刷新 + 自动刷新 (30s 间隔, 可配置) |
| **验收标准** | ✅ 组件状态准确<br>✅ 节点列表完整 |

#### 3.4.2 工作负载管理

**REQ-K8S-003: Pod 管理 (核心)**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能清单** | • **列表**: Namespace 筛选, 标签筛选, 状态筛选 (Running/Pending/Failed)<br>• **详情**: 容器列表, 环境变量, 资源请求/限制, 条件 (Conditions)<br>• **日志**: 实时日志 (SSE), 日志下载, 容器选择<br>• **Exec**: Web Terminal (WebSocket), 命令注入<br>• **文件浏览**: 容器内文件系统 (exec ls/cat)<br>• **事件**: Pod 相关 Events<br>• **诊断**: Pod 状态分析, Restart 原因排查<br>• **重启**: 删除 Pod 让 ReplicaSet 重建<br>• **创建**: Simple 表单 / YAML 编辑器 |
| **权限档位** | • readonly: 列表/详情/日志/事件<br>• readonly_exec: + Exec/文件浏览<br>• admin: + 创建/删除/重启 |
| **验收标准** | ✅ Pod 列表加载 < 2s (百级 Pod)<br>✅ 日志流实时延迟 < 1s<br>✅ Exec 终端交互流畅 |

**REQ-K8S-004: Deployment/StatefulSet/DaemonSet 管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **共同功能** | • 列表/详情/YAML 查看<br>• 扩缩容 (Scale): 调整副本数<br>• 重启 (RollingRestart): 逐个重启 Pod<br>• 更新容器资源 (Patch): CPU/Memory limits<br>• 删除<br>• 查看 Pod 子资源 |
| **Deployment 特有** | • Rollout Status: 查看滚动更新进度/历史<br>• 暂停/恢复 (Pause/Resume) |
| **StatefulSet 特有** | • 有序Pod编排信息 |
| **DaemonSet 特有** | • 调度节点查看 |
| **验收标准** | ✅ 扩缩容即时生效<br>✅ 重启不中断服务 (RollingUpdate)<br>✅ 资源更新应用成功 |

**REQ-K8S-005: Job/CronJob 管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **Job 功能** | • 列表/详情/删除<br>• 查看关联 Pods<br>• 手动重新运行 (Rerun) |
| **CronJob 功能** | • 列表/详情/创建/删除<br>• 暂停/恢复 (Suspend)<br>• 手动触发 (Trigger) <br>• 查看最近执行历史 (Pods) |
| **验收标准** | ✅ CronJob 调度准确<br>✅ 手动触发产生新 Job |

#### 3.4.3 网络与存储

**REQ-K8S-006: Service/Ingress/NetworkPolicy**

| 属性 | 内容 |
|------|------|
| **Service** | ClusterIP/NodePort/LoadBalancer 类型, 端口配置, Endpoint 查看 |
| **Ingress** | Host/Path/TLS/Annotations 配置, **Nginx 重启** (admin 档位+confirm:true) |
| **IngressClass** | Controller 关联, 参数配置 |
| **NetworkPolicy** | Ingress/Egress 规则, Selector 匹配 |
| **验收标准** | ✅ Service Endpoint 准确<br>✅ Ingress 规则生效<br>✅ Nginx 重启需二次确认 |

**REQ-K8S-007: ConfigMap/Secret/PV/PVC/StorageClass**

| 属性 | 内容 |
|------|------|
| **ConfigMap** | Key-Value 数据, YAML/JSON 格式, 创建/更新/删除 |
| **Secret** | **数据脱敏**: 列表/详情不显示 Value 明文 (显示 ****), 编辑时可查看/修改 |
| **PV/PVC** | 容量/访问模式/存储类/绑定状态/使用率 |
| **StorageClass** | Provisioner/Parameters/ReclaimPolicy |
| **验收标准** | ✅ Secret 脱敏生效 (API 不泄露)<br>✅ PVC 绑定状态准确 |

#### 3.4.4 RBAC 与 CRD

**REQ-K8S-008: RBAC 资源查看**

| 属性 | 内容 |
|------|------|
| **功能** | Role/RoleBinding/ClusterRole/ClusterRoleBinding 列表与详情 |
| **权限** | Rules (Verbs/Resources/APIGroups) 展示, Subject 绑定关系 |
| **注意** | 平台仅提供查看, 不建议通过平台修改 K8s RBAC (避免冲突) |
| **验收标准** | ✅ RBAC 资源展示完整<br>✅ 绑定关系清晰 |

**REQ-K8S-009: CRD/CR 管理**

| 属性 | 内容 |
|------|------|
| **CRD** | 列出集群已安装的 CustomResourceDefinitions, 查看 Schema |
| **CR** | 列出某 CRD 下的自定义资源实例, 查看 YAML, 创建/删除 |
| **典型用途** | 管理 Prometheus Operator / Istio / Cert-Manager 等扩展资源 |
| **验收标准** | ✅ CRD 列表准确<br>✅ CRUD 操作正常 |

---

### 3.5 R-05: 系统管理

*(系统管理模块涵盖用户/角色/权限/菜单/字典/审计等功能, 这里列举核心需求)*

#### 3.5.1 用户管理

**REQ-SYS-001: 用户 CRUD**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能** | • 列表 (分页/搜索/筛选状态)<br>• 创建 (用户名/邮箱/密码/昵称/部门/角色)<br>• 编辑 (除 username 外的字段)<br>• 删除 (软删除, 关联数据保留)<br>• 分配角色 (多选 Roles)<br>• 导入/导出 (Excel 批量操作) |
| **业务规则** | • 用户名全局唯一, 3-20 字符<br>• 邮箱全局唯一<br>• 删除用户前检查: 是否有项目/告警规则等关键资源<br>• Super Admin 不可删除 (至少保留一个) |
| **验收标准** | ✅ CRUD 操作完整<br>✅ 角色分配即时生效<br>✅ 导入导出格式正确 |

#### 3.5.2 角色与权限

**REQ-SYS-002: 角色 CRUD**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **内置角色** | • super-admin: 超级管理员 (拥有所有权限)<br>• admin: 系统管理员 (大部分管理权限)<br>• viewer: 只读查看者 (仅 GET 权限) |
| **自定义角色** | 支持创建自定义角色, 分配权限点集合 |
| **权限点 (Permission)** | • resource: API 路径 (如 /api/v1/users)<br>• action: HTTP 方法 (GET/POST/PUT/DELETE)<br>• Casbin 策略: p, role_code, resource, action |
| **验收标准** | ✅ 角色权限配置正确<br>✅ Casbin 策略同步生效 |

**REQ-SYS-003: Casbin 策略管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能** | • 授权列表: 查看所有 p 策略 (role → path → method)<br>• 授予权限: 为角色添加 API 权限<br>• 撤回权限: 删除策略 |
| **同步机制** | • 修改 casbin_rule 表后调用 enforcer.LoadPolicy()<br>• 或使用 SyncedEnforcer (自动 Watch DB 变更) |
| **验收标准** | ✅ 策略变更即时生效 (无需重启)<br>✅ 权限校验准确 |

#### 3.5.3 K8s 集群访问控制

**REQ-SYS-004: 集群档位 (Scoped Policy)**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **档位等级** | • **readonly** (1): 仅读取 (GET)<br>• **readonly_exec** (2): 读取 + Exec/日志/终端<br>• **admin** (3): 全部操作 (含 Apply/Delete/Scale) |
| **授权实体** | • 主体类型: role (角色) / user (用户)<br>• 主体引用: role_code / user_id<br>• 集群 ID: k8s_clusters.id<br>• 预设档位: readonly / readonly_exec / admin |
| **命名空间策略** | • **黑名单 (Deny)**: 禁止访问特定 NS (如 kube-system)<br>• **白名单 (Allow)**: 仅允许访问指定 NS (如 default/prod)<br>• 优先级: Deny > Allow (黑名单优先) |
| **验收标准** | ✅ 档位校验正确阻止越权操作<br>✅ NS 黑白名单生效<br>✅ Super Admin 不受限制 |

#### 3.5.4 组织架构

**REQ-SYS-005: 部门树管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **功能** | • 树形部门结构 (parent_id 自引用)<br>• CRUD 操作<br>• 用户归属部门 (users.department_id) |
| **用途** | • 告警通知: 按部门子树 @成员<br>• 组织视图: 按部门筛选用户 |
| **验收标准** | ✅ 部门树展示正确<br>✅ 用户部门关联准确 |

#### 3.5.5 菜单管理

**REQ-SYS-006: 动态菜单**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能** | • 树形菜单 (parent_id 自引用)<br>• 菜单项: 名称/路由路径/图标/排序/可见性<br>• 角色关联: 哪些角色可以看到哪些菜单<br>• 前端根据用户角色动态渲染侧边栏 |
| **内置菜单** | seed 命令预置全部菜单 (约 50+ 项) |
| **验收标准** | ✅ 菜单按角色过滤显示<br>✅ 菜单顺序正确 |

#### 3.5.6 数据字典

**REQ-SYS-007: 字典配置**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **功能** | • 字典类型 (dict_type): 分类标识 (如 mail_host, alert_webhook_token)<br>• 字典项 (dict_entry): key-value 对<br>• 敏感值: 标记 is_sensitive=true → AES 加密存储, API 返回 ****, 可点击"明文查看"<br>• 运行期覆盖: 部分 config.yaml 项可被字典值覆盖 (无需重启) |
| **典型字典** | • 邮件配置: host/port/username/password/from_email<br>• 告警配置: webhook_token/prometheus_url<br>• 通用选项: 钉钉 Webhook URL / 企业微信 URL |
| **验收标准** | ✅ 字典项下拉框正常使用<br>✅ 敏感值加密存储<br>✅ 运行期配置热更新生效 |

#### 3.5.7 审计日志

**REQ-SYS-008: 操作日志**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **记录内容** | • user_id / username<br>• method (HTTP 方法)<br>• path (请求路径)<br>• request_body (请求体, 敏感字段脱敏)<br>• response_status (HTTP 状态码)<br>• ip (客户端 IP)<br>• user_agent<br>• created_at |
| **记录范围** | 所有 POST/PUT/DELETE 操作 (GET 不记录)<br>• 特殊: 登录/登出/密码变更 也记录 |
| **查询** | • 按用户/时间/方法/路径筛选<br>• 导出 Excel<br>• 批量删除 (定期清理) |
| **验收标准** | ✅ 写操作均有日志<br>✅ 敏感信息已脱敏<br>✅ 查询和导出正常 |

**REQ-SYS-009: 登录日志**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **记录内容** | user_id, ip, user_agent, login_method (password/email), success/fail, fail_reason, created_at |
| **用途** | • 安全审计: 异常登录检测<br>• 统计分析: 活跃用户数 |
| **验收标准** | ✅ 每次登录都有记录<br>✅ 失败原因准确 |

#### 3.5.8 安全管理

**REQ-SYS-010: IP 封禁**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能** | • 封禁列表: 查看被封禁的 IP 地址<br>• 手动封禁: 管理员添加 IP + 原因 + 时长<br>• 自动解封: 到期自动解除 (或手动解封)<br>• 封禁规则: 连续失败 N 次 → 封禁 M 分钟 (可配置) |
| **存储** | Redis (banned_ips:{ip}) + TTL |
| **验收标准** | ✅ 封禁 IP 无法访问任何 API<br>✅ 到期自动解封<br>✅ 手动解封即时生效 |

**REQ-SYS-011: 注册审核**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **功能** | • 待审核列表: 查看注册申请<br>• 审核: 通过 / 拒绝 (填写原因)<br>• 通知: 审核结果邮件通知申请人 |
| **验收标准** | ✅ 审核流程完整<br>✅ 通知邮件发送成功 |

---

### 3.6 R-06: 日志平台

#### 3.6.1 日志查询与流式推送

**REQ-LOG-001: SSE 实时日志流**

| 属性 | 内容 |
|------|------|
| **优先级** | P0 (must) |
| **功能描述** | 通过 Server-Sent Events 技术, 将日志实时推送到浏览器 |
| **API 端点** | `GET /api/v1/projects/:id/logs/stream`<br>Query 参数: server_id, service_id, log_source_id, file_path?, after_id?, keywords?, level?, start_time?, end_time? |
| **协议** | Content-Type: text/event-stream<br>每行: `data: {"id": 12345, "content": "2026-05-21T10:00:00 INFO ...", "timestamp": "..."}`
| **断点续传** | after_id: 上次最后一条日志的 ID, 从该 ID 之后继续拉取 (防止刷新丢失数据) |
| **自动重连** | 浏览器断开后自动重连 (携带 after_id), 恢复到断点位置 |
| **过滤功能** | • keywords: 关键词搜索 (支持空格分隔多关键词 AND)<br>• level: 日志级别过滤 (INFO/WARN/ERROR)<br>• include/highlight: 高亮匹配关键词 (前端 CSS) |
| **性能** | 单连接支持 1000+ 条/秒推送 (受限于 DB 查询) |
| **验收标准** | ✅ 日志实时显示 (延迟 < 2s)<br>✅ 断点续传不丢数据<br>✅ 关键词高亮正确<br>✅ 多浏览器标签同时查看 (后台 Dock) |

**REQ-LOG-002: 日志导出**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **功能** | 将筛选后的日志导出为文本文件下载 |
| **参数** | 同日志查询参数 + 时间范围限制 (最多导出 24 小时) |
| **格式** | 纯文本, 每行一条日志 |
| **限制** | 单次导出上限 10 万条 (防止大数据量拖垮系统) |
| **验收标准** | ✅ 导出文件内容正确<br>✅ 超过上限有提示 |

#### 3.6.2 日志文件浏览

**REQ-LOG-003: 日志文件列表**

| 属性 | 内容 |
|------|------|
| **优先级** | P1 (important) |
| **功能** | 展示 Agent 上报的已发现日志文件列表 (来自 agent_discoveries 表) |
| **筛选** | 按 server_id / 已配置日志源 / 未配置日志源 筛选 |
| **操作** | • 查看文件详情 (大小/修改时间/行数)<br>• 一键创建日志源 (预填 file_path)<br>• 忽略 (加入黑名单) |
| **验收标准** | ✅ 文件列表准确<br>✅ 一键创建便捷 |

---

### 3.7 R-07: MySQL 备份 (可选功能)

**REQ-BACKUP-001: 备份实例管理**

| 属性 | 内容 |
|------|------|
| **优先级** | P2 (optional) |
| **功能** | • 定义 MySQL 备份任务 (目标数据库连接信息)<br>• Cron 表达式调度 (定时执行)<br>• mysqldump 参数配置 (单事务/ routines/ triggers 等)<br>• 远程归档: 上传备份到 MinIO 对象存储 |
| **流程** | 1. 创建备份实例 (配置 DB 连接 + Cron)<br>2. 手动触发测试 (Ping + Run Backup)<br>3. 查看备份任务列表和历史记录<br>4. 下载备份文件 (Presigned URL, 临时有效) |
| **验收标准** | ✅ 备份任务按时执行<br>✅ 备份文件上传 MinIO 成功<br>✅ 下载链接可用 |

---

## 4. 非功能性需求

### 4.1 性能需求

| 场景 | 指标 | 目标值 | 测量方法 |
|------|------|--------|----------|
| **API 响应时间** | 简单查询 (P50) | ≤ 100ms | Prometheus histogram |
| **API 响应时间** | 简单查询 (P99) | ≤ 300ms | 同上 |
| **API 响应时间** | 复杂聚合 (P99) | ≤ 2s | 同上 |
| **并发能力** | 同时在线用户 | ≥ 100 | JMeter 压测 |
| **QPS** | 峰值吞吐量 | ≥ 500 req/s | wrk 压测 |
| **日志 SSE** | 单连接吞吐 | ≥ 1000 lines/s | 浏览器 DevTools |
| **WebSocket** | 终端并发数 | ≥ 20 connections | 多浏览器测试 |
| **数据库查询** | 单次查询耗时 | ≤ 100ms (索引命中) | Slow Query Log |
| **Redis 操作** | 单次操作耗时 | ≤ 5ms | redis-cli --latency |
| **页面加载** | 首屏渲染 (FCP) | ≤ 2s | Lighthouse |

### 4.2 安全需求

| 类别 | 要求 | 实现 |
|------|------|------|
| **身份认证** | JWT + Redis Session 双重校验 | middleware/auth.go |
| **密码存储** | bcrypt (cost≥10), 不可逆 | pkg/password/password.go |
| **敏感数据加密** | AES-256-GCM (SSH密钥/Kubeconfig/云凭据) | pkg/crypto/aesgcm.go |
| **传输加密** | HTTPS (生产环境 TLS 1.2+) | Nginx SSL Termination |
| **SQL 注入防护** | GORM 参数化查询 (杜绝拼接 SQL) | ORM 框架保证 |
| **XSS 防护** | 前端转义 + CSP Header | React + Helmet |
| **CSRF 防护** | SameSite Cookie + Header 校验 | Gin Middleware |
| **暴力破解防护** | 登录失败限制 + IP 封禁 | middleware/rate_limit.go |
| **权限最小化** | Casbin RBAC + K8s Scope 三层鉴权 | middleware/casbin.go + k8s_scope_authorize.go |
| **操作审计** | 所有写操作记录日志 | middleware/operation_audit.go |
| **数据脱敏** | API 不返回密码/Token 明文 | handler 层处理 |
| **会话安全** | Token 过期 + 登出即失效 + 支持吊销 | store/session_redis.go |
| **依赖安全** | 定期更新第三方库 (go mod tidy) | CI/CD Pipeline |

### 4.3 可靠性需求

| 指标 | 目标值 | 实现方案 |
|------|--------|----------|
| **系统可用性** | ≥ 99.9% (月停机 ≤ 43 分钟) | 健康检查 + 自动重启 |
| **故障恢复时间** | MTTR ≤ 5 分钟 (P99) | Docker 快速重启 + 数据持久化 |
| **数据持久化** | MySQL Binlog 备份 + Redis AOF | 定期备份策略 |
| **无单点故障** | MySQL 主从 / Redis Sentinel (生产环境) | 架构设计 |
| **优雅关闭** | SIGTERM 信号处理, 完成进行中请求 | cmd/server.go 信号处理 |
| **连接池健康检查** | DB/Redis 连接池定期 ping | GORM/Redis 配置 |
| **熔断降级** | Redis 不可用时降级部分功能 (如限流失效但 API 仍可用) | 代码防御性编程 |

### 4.4 易用性需求

| 类别 | 要求 | 实现 |
|------|------|------|
| **UI 一致性** | 遵循 Ant Design 规范 | 前端组件库 |
| **响应式布局** | 支持 1920x1080 至 1366x768 分辨率 | CSS Flex/Grid |
| **错误提示友好** | 明确的错误码和解决方案文案 | constants/biz_reason.go |
| **操作引导** | 关键步骤提供 Tooltip/Help 文案 | 前端交互设计 |
| **国际化预留** | 硬编码中文提取为 i18n key (未来扩展) | 代码规范 |
| **键盘快捷键** | 常用操作支持快捷键 (Ctrl+S 保存等) | 前端事件监听 |
| **批量操作** | 列表页支持多选 + 批量操作 (删除/导出) | 前端 Table 组件 |
| **Loading 状态** | 异步操作显示 Loading 骨架屏 | Ant Design Skeleton |

### 4.5 可维护性需求

| 类别 | 要求 | 实现 |
|------|------|------|
| **代码规范** | 遵循 Go 官方风格 + Effective Go | gofmt + golangci-lint |
| **注释完整性** | 公开函数必须有 Godoc 注释 | 代码审查 |
| **日志规范** | 结构化日志 (JSON), 分级 (Info/Warn/Error) | pkg/logger/ |
| **API 文档** | Swagger 自动生成, 与代码同步 | swaggo 注释 |
| **配置外置** | 敏感配置通过环境变量/配置文件, 不硬代码 | Viper + env |
| **健康检查** | /health 端点供监控系统探测 | handler/system_handler.go |
| **Metrics 暴露** | /metrics 端点 (Prometheus 格式) | 未来集成 |
| **版本管理** | Git Tag + Semantic Versioning | Release 流程 |

### 4.6 可扩展性需求

| 类别 | 要求 | 实现 |
|------|------|------|
| **水平扩展** | Backend 支持多实例负载均衡 | 无状态设计 (Session 在 Redis) |
| **数据库分库** | 预留 Sharding 扩展点 (按项目/租户) | GORM DB 插件 |
| **缓存分层** | 本地缓存 (BigCache) + Redis 分布式缓存 | 性能优化预留 |
| **插件化告警渠道** | 实现 Channel Interface 即可扩展新渠道 | alert_channel_registry.go |
| **多云厂商支持** | 实现 CloudProvider Interface 可对接阿里云/腾讯云/AWS | cloud_provider_*.go |
| **K8s 资源扩展** | CRD/CR Handler 可动态注册新资源类型 | k8s_crd_service.go |

---

## 5. 接口需求

### 5.1 RESTful API 规范

#### 5.1.1 通用约定

**Base URL**: `/api/v1`

**认证方式**: 
```
Authorization: Bearer <jwt_token>
```

**请求头**:
```
Content-Type: application/json
Accept: application/json
X-Request-ID: <uuid>  // 可选, 用于追踪请求
```

**响应格式**:

**成功响应 (2xx)**:
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }  // 业务数据
}
```

**分页响应**:
```json
{
  "code": 0,
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

**错误响应 (4xx/5xx)**:
```json
{
  "code": 40001,
  "message": "参数校验失败: 用户名不能为空",
  "data": null
}
```

**标准错误码**:

| 错误码范围 | 含义 | 示例 |
|------------|------|------|
| 0 | 成功 | - |
| 40001-40099 | 参数错误 | 40001: 参数缺失, 40002: 参数格式错误 |
| 40101-40199 | 认证错误 | 40101: 未登录, 40102: Token无效, 40103: 登录过期 |
| 40301-40399 | 权限错误 | 40301: 无权限, 40302: 账户禁用, 40303: IP封禁 |
| 40401-40499 | 资源不存在 | 40401: 用户不存在 |
| 50001-50099 | 服务器内部错误 | 50001: 系统繁忙 |

#### 5.1.2 HTTP 方法语义

| 方法 | 幂等性 | 用途 | 成功状态码 |
|------|--------|------|------------|
| GET | 是 | 查询资源 | 200 OK |
| POST | 否 | 创建资源 | 201 Created |
| PUT | 是 | 全量更新资源 | 200 OK |
| PATCH | 否 | 部分更新资源 | 200 OK |
| DELETE | 是 | 删除资源 | 200 OK (返回 `{message: "deleted"}`) |

#### 5.1.3 分页规范

**Query 参数**:
```
?page=1&page_size=20&keyword=search&sort=created_at&order=desc
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 (≥1) |
| page_size | int | 20 | 每页数量 (1-100) |
| keyword | string | - | 搜索关键字 (模糊匹配) |
| sort | string | id | 排序字段 |
| order | string | desc | 排序方向 (asc/desc) |

### 5.2 gRPC 接口规范

**服务地址**: `:18080` (可配置)

**Proto 定义**: 见 `internal/grpc/proto/log_platform.proto`

**核心服务**:

```protobuf
service LogPlatform {
  // Agent 调用
  rpc Register(LogAgentRegisterRequest) returns (LogAgentRegisterResponse);
  rpc Heartbeat(stream LogAgentHeartbeatRequest) returns (stream LogAgentHeartbeatResponse);
  rpc Ingest(stream LogIngestRequest) returns (LogIngestResponse);
  rpc ReportDiscovery(AgentDiscoveryReportRequest) returns (AgentDiscoveryReportResponse);

  // Frontend 调用
  rpc StreamLogs(LogStreamRequest) returns (stream LogStreamResponse);
  rpc ListLogFiles(ListLogFilesRequest) returns (ListLogFilesResponse);
}
```

**认证方式**: Metadata 中传递 Token:
```
authorization: bearer <token>
```

### 5.3 WebSocket 接口规范

**Pod Exec 终端**:
```
WS /api/v1/pods/exec/ws?cluster_id=X&pod=Y&container=Z&token=T
```

**Project Server 终端**:
```
WS /api/v1/projects/:id/servers/:serverId/terminal/ws?token=T
```

**认证**: URL Query 参数传递 token (浏览器无法设置 WS Header)

**协议**: 二进制帧, stdin/stdout/stderr/stderror/resize 四通道

### 5.4 外部系统集成

| 系统 | 集成方式 | 用途 |
|------|----------|------|
| **Prometheus** | HTTP API (client_golang) | 告警规则评估/查询 |
| **Kubernetes** | client-go + Kom SDK | 集群资源管控 |
| **SMTP Server** | net/smtp | 邮件发送 (验证码/告警通知) |
| **钉钉 Webhook** | HTTP POST | 告警消息推送 |
| **MinIO** | S3 Compatible SDK (minio-go) | 备份文件对象存储 |
| **Redis** | go-redis/v9 | 缓存/Session/队列 |

---

## 6. 数据需求

### 6.1 数据库设计概要

详见 [docs/backend-architecture-complete.md](docs/backend-architecture-complete.md) 第 5 章 "数据模型设计"

**核心表清单 (40+ 张表)**:

| 域 | 表名 | 用途 |
|----|------|------|
| **系统** | users, roles, user_roles, user_groups, user_group_users, departments, menus, permissions, dict_entries, login_logs, operation_log, registration_requests, banned_ips |
| **项目** | projects, project_members, server_groups, servers, server_credentials, services, service_log_sources, cloud_accounts, log_agents, agent_discoveries |
| **K8s** | k8s_clusters, k8s_cluster_access_grants, k8s_namespace_allow_rules, k8s_namespace_deny_rules, k8s_event_forward_rules |
| **告警** | alert_channels, alert_datasources, alert_monitor_rules, alert_rule_assignees, alert_duty_blocks, alert_silences, alert_events, alert_subscriptions, alert_receiver_groups, alert_inhibition_rules, cloud_expiry_rules |
| **运维** | mysql_backup_instances, mysql_backup_jobs |

### 6.2 数据字典

详见 `internal/model/` 各实体文件的 GORM 标签注释

**枚举值规范**:

| 字段 | 可能值 | 说明 |
|------|--------|------|
| users.status | 0 (禁用), 1 (启用) | 用户状态 |
| projects.status | 0 (禁用), 1 (启用) | 项目状态 |
| k8s_clusters.status | 0 (禁用), 1 (启用) | 集群状态 |
| log_agents.status | online, offline, unknown | Agent 状态 |
| alert_events.status | firing, resolved | 告警状态 |
| alert_channels.type | dingding, email, webhook, wecom | 渠道类型 |
| alert_monitor_rules.severity | critical, warning, info | 严重程度 |
| project_members.role | owner, admin, member, readonly | 项目角色 |
| k8s_cluster_access_grants.preset | readonly (1), readonly_exec (2), admin (3) | 集群档位 |
| server_credentials.auth_type | password, key | 认证方式 |
| service_log_sources.source_type | file, command, journal | 日志源类型 |

### 6.3 数据迁移策略

**AutoMigrate**: 使用 GORM AutoMigrate 在启动时自动创建/更新表结构

```go
// internal/bootstrap/migrate_schema.go
func AutoMigrateModels(db *gorm.DB) error {
    return db.AutoMigrate(
        &model.User{},
        &model.Role{},
        // ... 所有 Model
    )
}
```

**Seed 数据**: `cmd/seed.go` → 事务内 Permission 批量 upsert、Casbin `AddPolicies`、`menu.Sync`（catalog 见 `internal/menu/catalog.go`）；admin 密码**仅首次创建**时写入。

**注意事项**:
- ⚠️ 生产环境禁止使用 DropColumn (可能导致数据丢失)
- ⚠️ 字段类型变更需编写手动 Migration SQL
- ⚠️ 大表新增字段需评估锁表时间 (建议低峰期执行)

---

## 7. 用户故事与用例

### 7.1 核心用户故事

**US-001: 作为运维工程师, 我希望能在浏览器中直接查看 Pod 日志, 以便快速定位问题**

**验收标准 (DoD)**:
- [ ] 能够选择项目 → 集群 → Namespace → Pod → 容器
- [ ] 日志以 SSE 实时流式推送, 延迟 < 2 秒
- [ ] 支持关键词搜索和高亮显示
- [ ] 支持断点续传, 刷新页面不丢失数据
- [ ] 日志量较大时 (1000+ lines/s) 不卡顿

**US-002: 作为项目经理, 我希望能够一键邀请同事加入项目并分配角色, 以便团队协作**

**验收标准 (DoD)**:
- [ ] 能够在项目成员页面搜索用户并添加
- [ ] 可以选择角色 (owner/admin/member/readonly)
- [ ] 添加后成员立即获得对应权限
- [ ] 成员能够看到项目下的服务器和服务列表
- [ ] 非 Owner/Admin 无法添加/移除成员

**US-003: 作为值班人员, 我希望在收到告警通知时能看到"【值班】"标识, 以便明确责任**

**验收标准 (DoD)**:
- [ ] 命中值班时段的告警通知标题前缀 "【值班】"
- [ ] 值班人优先收到通知 (非值班人不重复收到)
- [ ] 值班排班表可视化展示 (日历视图)
- [ ] 支持交接班无缝切换

**US-004: 作为安全管理员, 我希望能够限制某个角色只能"只读"查看特定 K8s 集群的 Pod, 防止误操作**

**验收标准 (DoD)**:
- [ ] 能够为角色 A 分配集群 X 的 "readonly" 档位
- [ ] 角色 A 的用户可以查看 Pod 列表/详情/日志
- [ ] 角色 A 的用户无法执行 Exec/Apply/Delete 操作
- [ ] 尝试越权操作返回 403 Forbidden + 明确提示
- [ ] 可以进一步配置命名空间白名单 (仅允许 default)

### 7.2 关键用例流程

**UC-001: 用户登录流程 (Happy Path)**

```
Actor: 用户 (已知用户名密码)
Precondition: 用户已注册且状态正常

1. 用户打开登录页面
2. 输入用户名: admin
3. 输入密码: Admin@123
4. 点击"登录"按钮
5. 系统校验凭证 (bcrypt compare)
6. 生成 JWT Token (payload: {userID:1, tokenID:"uuid", ...})
7. 写入 Redis Session (key: session:access:uuid → value: 1, TTL: 120m)
8. 记录登录日志 (ip, user_agent, success: true)
9. 返回 {access_token: "eyJhbG..."}
10. 前端存储 Token 到 localStorage
11. 跳转到首页/上次访问页面

Postcondition: 用户已登录, 可访问授权资源
```

**UC-002: 创建告警规则并触发通知**

```
Actor: 运维工程师
Precondition: 已登录, 有 Prometheus 数据源

1. 进入"告警平台 → 监控规则"页面
2. 点击"新建规则"
3. 选择数据源: prod-prometheus
4. 填写规则名称: "Pod CPU 使用率过高"
5. 输入 PromQL: `sum(rate(container_cpu_usage_seconds_total{image!=""}[5m])) by (pod_name) > 0.8`
6. 选择严重程度: warning
7. 设置持续时长: 5m
8. 添加处理人: 张三 (user_id=101)
9. 点击"保存并启用"

--- 系统后台 (Cron 驱动) ---

10. 每 30 秒评估一次该规则
11. 查询 Prometheus 得到结果: pod_a cpu=0.85 (持续 6 分钟)
12. 生成告警事件 (fingerprint=sha256(labels), status=firing)
13. 查询订阅路由树, 匹配到节点 N (receiver_group=运维组, channel=[钉钉])
14. 查询值班块: 当前时刻命中值班人 李四
15. 渲染告警消息 (标题: 【值班】[Firing] Pod CPU 使用率过高)
16. 调用钉钉 Webhook 发送消息
17. 记录发送历史 (alert_firing_deliveries 表)

Postcondition: 李四收到钉钉告警通知
```

**UC-003: K8s Pod 故障排查流程**

```
Actor: 开发工程师
Precondition: 已登录, 有 K8s 集群 readonly_exec 权限

1. 进入"K8s 控制台 → Pods"页面
2. 选择集群: prod-cluster
3. 筛选 Namespace: default
4. 看到 Pod "my-app-pod-abcde" 状态为 CrashLoopBackOff
5. 点击 Pod 名称进入详情页
6. 查看 Events: "Back-off restarting failed container: exit code 1"
7. 切换到"日志"Tab
8. 选择容器: my-app-container
9. 点击"开始实时日志"
10. 浏览器建立 SSE 连接, 日志开始滚动
11. 输入关键词 "Exception" 进行过滤
12. 发现错误日志: "NullPointerException at com.example.Service.handle()"
13. 点击"终端"Tab
14. 点击"连接终端" (WebSocket)
15. 出现 xterm.js 终端界面
16. 输入命令: `kubectl logs my-app-pod-abcde --previous` (查看上次崩溃日志)
17. 定位到具体异常堆栈
18. 修复代码并重新部署

Postcondition: 问题定位并修复
```

---

## 8. 验收标准

### 8.1 功能验收测试矩阵

| 模块 | 测试场景数 | P0 用例数 | P1 用例数 | 自动化覆盖率目标 |
|------|-----------|-----------|-----------|------------------|
| **认证与身份** | 25 | 15 | 8 | 80% |
| **项目管理** | 35 | 20 | 12 | 70% |
| **K8s 控制台** | 50 | 30 | 15 | 60% |
| **告警平台** | 40 | 25 | 12 | 65% |
| **系统管理** | 30 | 20 | 8 | 75% |
| **日志平台** | 20 | 12 | 6 | 70% |
| **合计** | **200** | **122** | **61** | **70%** (平均) |

### 8.2 非功能验收标准

| 类别 | 验收指标 | 测试方法 | 通过标准 |
|------|----------|----------|----------|
| **性能** | API P99 响应时间 | JMeter 压测 (100 并发) | ≤ 2s (复杂查询) |
| **性能** | 并发用户支持 | 模拟 100 用户同时操作 | 无 5xx 错误, P99 < 3s |
| **安全** | OWASP Top 10 漏洞扫描 | ZAP/Burp Suite 扫描 | 无高危漏洞 |
| **安全** | Penetration Testing | 渗透测试报告 | 无权限提升风险 |
| **可靠性** | 系统可用性 | 7×24 监控 (Prometheus Blackbox) | ≥ 99.9% |
| **可靠性** | 故障恢复时间 | 模拟 Backend Crash | MTTR ≤ 5 min |
| **兼容性** | 浏览器兼容性 | Chrome/Firefox/Edge/Safari 最新 2 个版本 | UI 正常, 功能完好 |
| **兼容性** | K8s 版本兼容性 | 测试 1.25/1.26/1.27/1.28 | API 调用正常 |

### 8.3 验收测试流程

```
Phase 1: 单元测试 (开发者负责)
├── 目标: 覆盖 Service 层核心逻辑
├── 工具: Go testing + testify
├── 覆率目标: Core Module ≥ 80%
└── 产出: 单元测试报告

Phase 2: 集成测试 (QA 负责)
├── 目标: 验证模块间接口协作
├── 工具: Postman/Newman + TestContainers (MySQL/Redis)
├── 覆盖: API 接口 + 数据库操作
└── 产出: 集成测试报告

Phase 3: 系统测试 (QA 负责)
├── 目标: 端到端业务流程验证
├── 工具: Selenium/Playwright (E2E) + JMeter (性能)
├── 场景: UC-001 ~ UC-003 等关键用例
└── 产出: 系统测试报告 + 性能测试报告

Phase 4: 验收测试 (产品+客户共同参与)
├── 目标: 对照需求规格逐项验证
├── 形式: Demo 演示 + 用例执行 + 签字确认
├── 产出: 验收报告 (UAT Sign-Off)
└── 准入: 产品发布
```

### 8.4 验收检查清单 (Checklist)

#### 必须项 (Must Have) - 阻塞发布

- [ ] **AUTH-001~008**: 认证模块全部功能正常
- [ ] **PROJ-001~013**: 项目管理核心功能 (CRUD/成员/服务器/Agent)
- [ ] **K8S-001~009**: K8s 集群接入 + Pod/工作负载管理
- [ ] **ALERT-001~007**: 告警数据源 + 规则 + 渠道 + 事件查询
- [ ] **SYS-001~011**: 用户/角色/权限/菜单/审计/IP封禁
- [ ] **LOG-001~003**: 日志 SSE 流 + 文件列表
- [ ] **非功能**: 无 P0/P1 级 Bug
- [ ] **安全**: 无高危/严重漏洞
- [ ] **文档**: API 文档 (Swagger) + 部署手册 + 运维手册

#### 应该项 (Should Have) - 强烈建议

- [ ] **ALERT-004~006**: 订阅路由 + 值班 + 静默 (完整告警闭环)
- [ ] **BACKUP-001**: MySQL 备份 (若有 MinIO 环境)
- [ ] **性能**: API P99 ≤ 2s (常规场景)
- [ ] **易用性**: 操作日志完整, 错误提示友好

#### 可以项 (Could Have) - 锦上添花

- [ ] **K8S-005**: Job/CronJob 完整管理
- [ ] **PROJ-008**: 服务器终端 WebSocket (若网络条件允许)
- [ ] **国际化**: i18n 框架搭建 (即使只有中文)
- [ ] **Metrics**: Prometheus /metrics 端点暴露

---

## 9. 附录

### Appendix A: 术语表

| 术语 | 全称 | 定义 |
|------|------|------|
| **API** | Application Programming Interface | 应用程序编程接口 |
| **CRUD** | Create, Read, Update, Delete | 增删改查 |
| **DTO** | Data Transfer Object | 数据传输对象 |
| **ER** | Entity-Relationship | 实体关系 |
| **gRPC** | Google Remote Procedure Call | Google 远程过程调用 |
| **JWT** | JSON Web Token | JSON 网络令牌 |
| **ORM** | Object-Relational Mapping | 对象关系映射 |
| **RBAC** | Role-Based Access Control | 基于角色的访问控制 |
| **REST** | Representational State Transfer | 表述性状态转移 |
| **SDK** | Software Development Kit | 软件开发工具包 |
| **SLA** | Service Level Agreement | 服务等级协议 |
| **SRE** | Site Reliability Engineering | 站点可靠性工程 |
| **SSE** | Server-Sent Events | 服务器推送事件 |
| **TLS** | Transport Layer Security | 传输层安全协议 |
| **UAT** | User Acceptance Testing | 用户验收测试 |
| **UUID** | Universally Unique Identifier | 通用唯一识别码 |
| **WS** | WebSocket | WebSocket 协议 |

### Appendix B: 参考文档索引

| 文档名称 | 路径 | 用途 |
|----------|------|------|
| **本项目 README** | [README.md](../README.md) | 项目总览和快速开始 |
| **后端架构文档** | [docs/backend-architecture-complete.md](../docs/backend-architecture-complete.md) | 技术实现详解 |
| **架构图集** | [docs/architecture-diagrams.md](../docs/architecture-diagrams.md) | 系统架构可视化 |
| **权限设计手册** | [docs/handbook/permissions/casbin-and-k8s-triple-policy.md](../docs/handbook/permissions/casbin-and-k8s-triple-policy.md) | 三层鉴权机制 |
| **告警平台详细设计** | [docs/requirements/R-alert-platform-detailed-design.md](../docs/requirements/R-alert-platform-detailed-design.md) | 告警模块深度设计 |
| **日志平台 API** | [docs/log-platform-api.md](../docs/log-platform-api.md) | 日志接口规范 |
| **数据库 ER 图** | [docs/handbook/database/er-diagrams.md](../docs/handbook/database/er-diagrams.md) | 数据库关系图 |
| **OpenAPI 规范** | [docs/apipost/permission-system.openapi.yaml](../docs/apipost/permission-system.openapi.yaml) | API 定义文件 |
| **部署文档** | [docs/deployment/KYLIN_V10_X86_64.md](../docs/deployment/KYLIN_V10_X86_64.md) | 麒麟系统部署指南 |
| **配置说明** | configs/config.yaml | 配置项详细注释 |

### Appendix C: 修订历史

| 版本 | 日期 | 作者 | 修订内容 | 审批人 |
|------|------|------|----------|--------|
| V0.1-Draft | 2026-05-20 | 需求组 | 初稿创建, 完成 6 大域需求梳理 | - |
| V0.5-Review | 2026-05-21 | 架构师 | 补充技术约束和非功能需求, 评审通过 | 技术负责人 |
| V1.0-Final | 2026-05-21 | 产品经理 | 完善用户故事和验收标准, 达到交付标准 | 产品总监 |

### Appendix D: 签字确认

| 角色 | 姓名 | 日期 | 签字 |
|------|------|------|------|
| **产品经理** | _______________ | ____-____-____ | ____________ |
| **技术负责人** | _______________ | ____-____-____ | ____________ |
| **项目经理** | _______________ | ____-____-____ | ____________ |
| **客户代表** | _______________ | ____-____-____ | ____________ |

---

**文档结束**

*本文档共 **XXX** 页, 包含 **9** 大章节, **200+** 功能需求点, **30+** 用户故事和用例, 完整覆盖 Yunshu 云原生运维管理平台的业务需求。*

**下一步行动**:
1. ✅ 召开需求评审会议 (Product + Tech + QA)
2. 🔄 基于 SRS 制定开发计划 (Sprint Backlog)
3. ⏳ 开始迭代开发 (每 2 周 Sprint)
4. 📊 每个 Sprint 结束进行 Demo 和 UAT 验收

---

*Generated by Yunshu Team | 2026-05-21*
