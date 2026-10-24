## 变更说明

<!-- 一句话说明本 PR 做了什么、为什么做 -->

## 变更类型

- [ ] 新功能
- [ ] Bug 修复
- [ ] 重构（不改变行为）
- [ ] 文档
- [ ] CI / 工程化
- [ ] 其它

## 影响面

<!-- 涉及的插件/模块（如 k8s、dbmgmt、alert、前端页面等）与 2 跳上游调用方 -->

## 验证

<!-- 必须给出证据：命令+退出码、测试用例数、接口 Status Code+Body、截图+步骤等 -->

- [ ] `go build ./...` 通过
- [ ] `go test ./... -short -count=1` 通过
- [ ] `cd web && npm run build && npm test` 通过
- [ ] 涉及接口变更时已同步重新生成 docs/apipost/permission-system.openapi.yaml 与 docs/API接口设计说明书.md
- [ ] 涉及 UI 时已完成真机/浏览器功能验证

## 风险与回滚

<!-- 有无数据变更/迁移、权限变更、不可回滚操作；如何回滚 -->

## 相关 Issue / 文档
