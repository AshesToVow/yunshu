package k8s

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
)

func projectIDOf(cluster *model.K8sCluster) uint {
	if cluster == nil || cluster.OwningProjectID == nil {
		return 0
	}
	return *cluster.OwningProjectID
}

// assertK8sWritable 写操作门禁：须为 write/exec 意图，且非只读档位；高危动作另走 RequireDestructiveConfirm。
// Pod 文件 upload/delete 由中间件标为 AccessIntentExec，与终端同档。
func assertK8sWritable(ctx context.Context, cluster *model.K8sCluster, action, _ string) error {
	if cluster == nil {
		return constants.ErrBadRequestWithMsg("集群无效")
	}
	intent := accessIntentFromContext(ctx)
	if intent == k8sauth.AccessIntentRead {
		return constants.ErrForbiddenWithMsg("当前请求为只读凭证意图，禁止 " + strings.TrimSpace(action))
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if ok && u != nil && auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if scope, ok := k8sauth.RequestScopeFromContext(ctx); ok && scope.AccessRank > 0 {
		if scope.AccessRank < K8sAccessRankAdmin && intent != k8sauth.AccessIntentExec {
			// exec 档位允许终端/文件；其他变更须 admin
			if action != "exec" {
				return constants.ErrForbiddenWithMsg("变更类操作须集群 admin 档位")
			}
		}
	}
	return nil
}
