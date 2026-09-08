package plugin

// As 从 Runtime 的 any 槽位取出具体类型（worker 启动用）。
// Deps 仍为 any 以打破 plugin↔router 循环；业务路由依赖请用 router 包内窄接口。
func As[T any](v any) (T, bool) {
	t, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return t, true
}
