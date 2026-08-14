# CI/CD 构建排障知识（种子）

## 必备参数

`project_id` 必填。缺省时请用户在助手页选择项目，或先澄清。

## 工具链

1. `list_cicd_builds` 找失败记录
2. `get_cicd_build` 看阶段/状态
3. `get_cicd_build_log` 定位首个 ERROR

## 常见根因

- 编译/依赖失败
- 单元测试失败
- Harbor 推送鉴权失败
- 回调超时

禁止编造 Jenkins 控制台内容；必须以工具返回日志为准。
