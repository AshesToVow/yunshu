package system

import (
	"context"

	"yunshu/internal/repository"
)

// PrincipalGrantCleaner 主体删除后的授权级联清理。
// role/group 的授权主体引用是编码（principal_ref），而编码在软删除后可复用：
// 删除主体时若不清授权，新建同编码主体会静默继承旧授权。
type PrincipalGrantCleaner interface {
	CleanupPrincipalGrants(ctx context.Context, kind, ref string) error
}

type principalGrantCleaner struct {
	k8sAccess repository.K8sClusterAccessRepo
	nsAllow   repository.K8sNamespaceAllowRepo
	nsDeny    repository.K8sNamespaceDenyRepo
	dbmgmt    repository.DbmgmtRepo
}

func NewPrincipalGrantCleaner(
	k8sAccess repository.K8sClusterAccessRepo,
	nsAllow repository.K8sNamespaceAllowRepo,
	nsDeny repository.K8sNamespaceDenyRepo,
	dbmgmt repository.DbmgmtRepo,
) PrincipalGrantCleaner {
	return &principalGrantCleaner{k8sAccess: k8sAccess, nsAllow: nsAllow, nsDeny: nsDeny, dbmgmt: dbmgmt}
}

// CleanupPrincipalGrants 依次清理 K8s 集群档位、K8s 命名空间放行/拦截、dbmgmt 实例授权。
// 任一域失败即中止并向上返回：调用方应放弃删除主体，重试可收敛。
func (c *principalGrantCleaner) CleanupPrincipalGrants(ctx context.Context, kind, ref string) error {
	if ref == "" {
		return nil
	}
	if c.k8sAccess != nil {
		if err := c.k8sAccess.DeleteByPrincipal(ctx, kind, ref); err != nil {
			return err
		}
	}
	if c.nsAllow != nil {
		if err := c.nsAllow.DeleteByPrincipal(ctx, kind, ref); err != nil {
			return err
		}
	}
	if c.nsDeny != nil {
		if err := c.nsDeny.DeleteByPrincipal(ctx, kind, ref); err != nil {
			return err
		}
	}
	if c.dbmgmt != nil {
		return c.dbmgmt.DeleteGrantsByPrincipal(ctx, kind, ref)
	}
	return nil
}
