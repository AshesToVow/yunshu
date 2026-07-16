# 需求说明文档索引

## 模块级标准开发需求（推荐）

按业务模块输出接口、入参、结果、请求方式、数据库等，便于研发与联调：

**[modules/_INDEX.md](./modules/_INDEX.md)**（M-00～M-10）

模板：[modules/_TEMPLATE.md](./modules/_TEMPLATE.md)

## 业务域摘要（旧编号，仍保留）

每个文件对应一个业务域摘要；细节以 **modules/** 与 **menus/** 为准。

| 文件 | 域 | 对应模块 |
|------|-----|----------|
| [R-01-auth-and-identity.md](./R-01-auth-and-identity.md) | 登录、注册、个人设置 | M-01 |
| [R-02-project-management.md](./R-02-project-management.md) | 项目、成员、服务器 | M-03 |
| [R-03-alert-and-monitor.md](./R-03-alert-and-monitor.md) | 告警 | M-05 |
| [R-04-kubernetes-console.md](./R-04-kubernetes-console.md) | K8s 控制台 | M-06 |
| [R-05-system-administration.md](./R-05-system-administration.md) | 系统管理 | M-02 |
| [R-06-log-platform-and-agent.md](./R-06-log-platform-and-agent.md) | 日志平台（已指向 Loggie+ES） | **M-04** |

## 按菜单（前端路由）细分

每个可见菜单对应一篇说明（路由、组件、API、表、权限、注意点）：**[menus/_INDEX.md](./menus/_INDEX.md)**。  
Kubernetes 多页共用模式见 **[menus/menu-k8s-resource-pattern.md](./menus/menu-k8s-resource-pattern.md)**。
