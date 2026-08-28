---
name: use-modern-go
description: >-
  Use Modern Go Guidelines whenever writing, modifying, fixing, or refactoring
  Go code. Prefer Go 1.26 idioms available in this repo (go.mod: go 1.26).
  Source: https://github.com/JetBrains/go-modern-guidelines
---

# Modern Go Guidelines (Yunshu)

本项目 `go.mod` 为 **Go 1.26**。编写/修改 Go 代码时必须优先使用现代惯用法。

权威来源：[JetBrains/go-modern-guidelines](https://github.com/JetBrains/go-modern-guidelines)

## 改 Go 代码前

1. 阅读 `references/guidelines.md`（按版本适用）。
2. 可用时运行 modernize：

```powershell
modernize -fix ./path/to/pkg/...
# 或
go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...
```

3. 可选：本地 CLI（需能访问 GitHub 安装）：

```powershell
.agents\skills\use-modern-go\scripts\run-tool.ps1 list --go-version 1.26
.agents\skills\use-modern-go\scripts\run-tool.ps1 explain rangeint slicescontains cmpor newliteral
```

## 本仓库优先应用（高影响）

| ID | 做法 |
|----|------|
| `rangeint` | `for i := range n` 替代 `for i := 0; i < n; i++` |
| `slicescontains` | `slices.Contains` |
| `minmax` | 内建 `min`/`max` |
| `efaceany` | `any` 替代 `interface{}` |
| `cmpor` | `cmp.Or(a, b)` 取默认值 |
| `waitgroup` | `wg.Go(func(){...})`（Go 1.25+） |
| `newliteral` | `new(42)` 取字面量指针（Go 1.26） |
| `errors_as_type` | `errors.AsType[T](err)`（Go 1.26，适用时） |
| `bloop` | benchmark 用 `for b.Loop()` |
| `stringscutprefix` | `strings.CutPrefix` |
| `stringsseq` | 仅遍历时用 `SplitSeq`/`FieldsSeq` |
| `omitzero` | JSON 零值用 `omitzero`（确认 API 兼容后） |

## 禁止

- 为“现代化”做无关大面积重构
- 改动会破坏 GORM/Casbin/第三方接口签名的 `interface{}`
- 在行为会变时强行替换（先 `explain` 或对照 guidelines）
