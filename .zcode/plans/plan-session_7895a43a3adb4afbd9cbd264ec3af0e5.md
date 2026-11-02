# 修复 AI 运维助手输入框悬空(应吸附面板底部)

## 根因
`.ai-assistant-layout`(web/src/styles/global.css:5678)没有视口高度约束,列高由内容(消息区 minHeight:420)决定,消息少时输入框悬在页面中部,下方大片空白。

## 修改点

### 1. web/src/styles/global.css — AI 助手布局块(5677-5710)
- `.ai-assistant-layout` 增加:
  - `height: calc(100vh - 88px)`(header 56px + content 上下 padding 32px)
  - `min-height: 480px`(矮窗口兜底,页面可滚动)
- 增加 compact 模式覆盖:`.admin-shell.layout-compact .ai-assistant-layout { height: calc(100vh - 76px) }`
- 会话侧栏卡改为 flex 纵向:`display:flex; flex-direction:column; min-height:0`,body 改 `flex:1; min-height:0; overflow:auto`,删除旧的 `max-height: calc(100vh - 190px)`
- `.ai-assistant-main-card` 补 `min-height: 0`
- 768px 媒体查询:布局改 `height:auto`(纵向堆叠不适用定高),会话 body 恢复 `flex:none; max-height:260px`

### 2. web/src/pages/ai-assistant-page.tsx:705
消息区 `minHeight: 420` → `minHeight: 160`。原 420 在矮视口(如 768px 高)会把输入框顶出可视区,改 160 后由 `flex:1` 主导,移动端也不塌陷。

### 3. 重新构建前端
`cd web && npm run build`(web/dist 构建产物入库,须同步更新;build 含 tsc 类型检查)

## 验证
- `npm run build` 通过
- 打开 /ai/assistant:输入框贴住面板/视口底部;消息增多时仅消息区内部滚动;缩短窗口高度输入框不出视口;拖窄到 <768px 时会话列表与主区正常纵向堆叠