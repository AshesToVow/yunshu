# ES 管理（esmgmt）

平台菜单：ES 连接、集群概览、REST 控制台。

- 连接：维护 Elasticsearch 地址/账号；密码经 encryption_key 加密存储。
- 概览：集群健康、索引列表、备份/恢复任务。
- REST 控制台：受限代理，支持 GET/POST/PUT/DELETE/HEAD；禁止脚本执行与节点关机。

助手侧：当前无 esmgmt 专用 Tool；排障请引导用户到 ES 管理页，或结合日志/告警模块排查。
AI 知识库索引（yunshu-ai-kb-v1）使用日志平台 ES 配置，与 esmgmt 纳管连接相互独立。
