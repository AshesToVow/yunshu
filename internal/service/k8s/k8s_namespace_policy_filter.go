package k8s

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8srole"
)

// FilterNamespaceNamesByPolicy 优先使用请求上下文中的 k8m 授权行；无 Grants 时回退独立 NS 表。
func FilterNamespaceNamesByPolicy(
	ctx context.Context,
	deny interfaces.K8sNamespaceDenyRepository,
	allow interfaces.K8sNamespaceAllowRepository,
	pack k8sauth.PrincipalPack,
	clusterID uint,
	names []string,
) ([]string, error) {
	if clusterID == 0 || len(names) == 0 {
		return names, nil
	}
	if k8sauth.SkipNamespacePolicy(ctx) {
		return names, nil
	}
	if scope, ok := k8sauth.RequestScopeFromContext(ctx); ok {
		if scope.BypassAuth {
			return names, nil
		}
		if len(scope.Grants) > 0 {
			user := scope.Username
			if user == "" {
				user = k8sauth.UserRefString(scope.Pack.UserID)
			}
			return k8srole.FilterNamespaces(scope.Grants, strconv.FormatUint(uint64(clusterID), 10), user, names), nil
		}
	}
	if len(pack.PrincipalRows()) == 0 {
		return names, nil
	}

	out := append([]string(nil), names...)

	if deny != nil {
		denied, err := deny.DeniedNamespaceNames(ctx, pack, clusterID)
		if err != nil {
			return nil, err
		}
		if len(denied) > 0 {
			rm := make(map[string]struct{}, len(denied))
			for _, n := range denied {
				rm[n] = struct{}{}
			}
			filtered := out[:0]
			for _, n := range out {
				if _, bad := rm[n]; !bad {
					filtered = append(filtered, n)
				}
			}
			out = filtered
		}
	}

	if allow != nil {
		active, err := allow.WhitelistActiveForCluster(ctx, pack, clusterID)
		if err != nil {
			return nil, err
		}
		if active {
			allowed, err := allow.WhitelistUnionNamespaces(ctx, pack, clusterID)
			if err != nil {
				return nil, err
			}
			ok := make(map[string]struct{}, len(allowed))
			for _, n := range allowed {
				ok[n] = struct{}{}
			}
			filtered := out[:0]
			for _, n := range out {
				if _, yes := ok[n]; yes {
					filtered = append(filtered, n)
				}
			}
			out = filtered
		}
	}

	return out, nil
}

// NamespaceAllowedByPolicy 判断当前主体能否访问某集群下的命名空间，与 K8sScopeAuthorize 语义对齐。
func NamespaceAllowedByPolicy(
	ctx context.Context,
	deny interfaces.K8sNamespaceDenyRepository,
	allow interfaces.K8sNamespaceAllowRepository,
	pack k8sauth.PrincipalPack,
	clusterID uint,
	namespace string,
) (bool, error) {
	ns := strings.TrimSpace(namespace)
	if clusterID == 0 || ns == "" {
		return true, nil
	}
	filtered, err := FilterNamespaceNamesByPolicy(ctx, deny, allow, pack, clusterID, []string{ns})
	if err != nil {
		return false, err
	}
	return len(filtered) > 0, nil
}
