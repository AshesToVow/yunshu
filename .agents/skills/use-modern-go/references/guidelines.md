# Modern Go Guidelines（摘录 · Go 1.26）

来源：https://github.com/JetBrains/go-modern-guidelines/blob/main/FEATURES.md

按本仓库 `go 1.26` 全量适用下列惯用法（仅列高/关键影响）。

## Collections

- `slices.Contains(s, v)` 替代手动查找循环
- `slices.Sort` / `slices.SortFunc` + `cmp.Compare` 替代 `sort.Slice`
- 内建 `min` / `max`
- `maps.Clone` / `slices.Collect(maps.Keys(m))` 等（适用时）

## Loops

- `for i := range n` 替代 `for i := 0; i < n; i++`
- Go 1.22+ 循环变量按次捕获，勿再写 `v := v`

## Types / Errors

- `any` 替代 `interface{}`（第三方接口签名除外）
- `errors.Is` / `errors.As`；Go 1.26 可用 `errors.AsType[T](err)`
- `errors.Join` 聚合多错误

## Sync / Testing

- `wg.Go(func(){ ... })`（Go 1.25+）替代 Add/go/Done
- Benchmark：`for b.Loop() { ... }`

## Utilities

- `cmp.Or(a, default)` 取默认值
- `new(42)` 取字面量指针（Go 1.26）
- `strings.CutPrefix` / 仅遍历时 `SplitSeq`

## JSON

- 确认 API 兼容后，零值字段可用 `omitzero`
