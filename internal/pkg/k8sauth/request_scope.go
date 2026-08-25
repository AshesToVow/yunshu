package k8sauth

import "context"

type scopeCtxKey struct{}

// RequestScope 挂在 HTTP 请求上下文上，供服务层做命名空间策略过滤与凭证意图选择。
type RequestScope struct {
	ClusterID  uint
	Namespace  string
	Pack       PrincipalPack
	Intent     AccessIntent
	AccessRank int
}

// WithRequestScope 写入请求范围（中间件在解析 cluster_id / namespace 后调用）。
func WithRequestScope(ctx context.Context, scope RequestScope) context.Context {
	return context.WithValue(ctx, scopeCtxKey{}, scope)
}

// RequestScopeFromContext 读取请求范围；未设置时 ok=false。
func RequestScopeFromContext(ctx context.Context) (RequestScope, bool) {
	if ctx == nil {
		return RequestScope{}, false
	}
	v, ok := ctx.Value(scopeCtxKey{}).(RequestScope)
	return v, ok
}

// ClusterIDFromContext 读取中间件写入的 cluster_id。
func ClusterIDFromContext(ctx context.Context) uint {
	if s, ok := RequestScopeFromContext(ctx); ok {
		return s.ClusterID
	}
	return 0
}

type skipNSPolicyKey struct{}

// WithSkipNamespacePolicy 平台托管下发（如 yunshu-logging DaemonSet）跳过 NS 白名单校验。
func WithSkipNamespacePolicy(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipNSPolicyKey{}, true)
}

// SkipNamespacePolicy 是否跳过命名空间策略。
func SkipNamespacePolicy(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(skipNSPolicyKey{}).(bool)
	return ok && v
}
