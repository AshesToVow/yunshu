---
name: web 前端工具链在本机不可用
description: >-
  本机 node/npm 不在 PATH 且 web/node_modules 缺失，前端 lint/build/test 命令无法执行（2026-08-29
  实测）
type: project
---

`node` 与 `npm` 均不在 PATH，`d:\gocode\yunshu\web\node_modules` 不存在（2026-08-29 实测）。因此 `npm install`、`npm run lint`、`tsc -b`、`vite build`、`vitest run` 全部无法执行。

**Why:** 用户要求每阶段以「可编译 + 测试全绿」收尾。前端工程化（ESLint/Prettier/Vitest）、构建体积优化（React.lazy/manualChunks）这类改动如果无法运行验证，就只是不可验证的静态文本，与该约束冲突。

**How to apply:** 承接 `web/` 相关改造前先复测 `node -v`；若仍不可用，只推进「不依赖工具链即可人工复核」的纯源码修复（例如已完成的 `admin-layout.tsx` 精确清理 localStorage），并就「装 Node / 接受不可验证改动 / 只做源码级修复」三条路径请用户决策，不要静默产出成套配置文件。