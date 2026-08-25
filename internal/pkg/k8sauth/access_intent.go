package k8sauth

import "context"

// AccessIntent 控制台访问集群凭证的意图：决定选用只读/可写 kubeconfig 以及是否允许变更。
type AccessIntent string

const (
	AccessIntentRead  AccessIntent = "read"
	AccessIntentWrite AccessIntent = "write"
	AccessIntentExec  AccessIntent = "exec"
)

type destructiveConfirmKey struct{}

// WithDestructiveConfirm 标记请求已携带高危确认（confirm=true）。
func WithDestructiveConfirm(ctx context.Context, ok bool) context.Context {
	return context.WithValue(ctx, destructiveConfirmKey{}, ok)
}

// DestructiveConfirmFromContext 是否已确认高危操作。
func DestructiveConfirmFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(destructiveConfirmKey{}).(bool)
	return ok && v
}

// AccessIntentFromScope 从 RequestScope 取意图；未设置时默认 write（兼容后台任务）。
func AccessIntentFromScope(scope RequestScope) AccessIntent {
	switch scope.Intent {
	case AccessIntentRead, AccessIntentWrite, AccessIntentExec:
		return scope.Intent
	default:
		return AccessIntentWrite
	}
}
