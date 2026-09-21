# DB 连接失败排查

## 症状

- 应用报 `connection refused` / `Access denied` / 超时
- AI 助手 `list_db_instances` 可见实例但业务连不上

## 检查顺序

1. 确认实例在 Yunshu「数据库管理」中已登记且项目成员有权
2. 核对地址、端口、账号与网络（安全组 / NetworkPolicy）
3. 用平台连通性探测或客户端直连验证
4. 查看实例侧最大连接数与慢查询是否打满

## 与助手配合

- 工具：`list_db_instances`（只读）
- SOP：`sop-dbmgmt-conn-fail`
- 勿在对话中索要或回显明文密码
