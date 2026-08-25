package k8s

import (
	"context"
	"strings"

	"yunshu/internal/pkg/constants"
)

// NamespacePresetSplit 按命名空间拆分档位授权（同一主体多行 grant，每行不同 allow_namespaces + preset）。
type NamespacePresetSplit struct {
	Namespace string `json:"namespace" binding:"required"`
	Preset    string `json:"preset" binding:"required"`
}

type SplitRoleByNamespacesRequest struct {
	PrincipalKind string                 `json:"principal_kind"`
	RoleID        uint                   `json:"role_id"`
	UserID        uint                   `json:"user_id"`
	GroupID       uint                   `json:"group_id"`
	ClusterIDs    []uint                 `json:"cluster_ids" binding:"required,min=1"`
	Splits        []NamespacePresetSplit `json:"splits" binding:"required,min=1"`
}

// SplitByNamespaces 为同一主体按命名空间批量下发不同档位（封装多次 GrantPreset）。
func (s *K8sScopedPolicyService) SplitByNamespaces(ctx context.Context, req SplitRoleByNamespacesRequest) (*K8sScopedPolicyGrantPresetResponse, error) {
	if len(req.Splits) == 0 {
		return nil, constants.ErrBadRequestWithMsg("splits 不能为空")
	}
	total := &K8sScopedPolicyGrantPresetResponse{}
	for _, sp := range req.Splits {
		ns := strings.TrimSpace(sp.Namespace)
		if ns == "" {
			continue
		}
		grantReq := K8sScopedPolicyGrantPresetRequest{
			PrincipalKind:   req.PrincipalKind,
			RoleID:          req.RoleID,
			UserID:          req.UserID,
			GroupID:         req.GroupID,
			ClusterIDs:      req.ClusterIDs,
			Preset:          strings.TrimSpace(sp.Preset),
			AllowNamespaces: []string{ns},
		}
		resp, err := s.GrantPreset(ctx, grantReq)
		if err != nil {
			return total, err
		}
		total.Added += resp.Added
		total.Skipped += resp.Skipped
		total.AllowRulesAdded += resp.AllowRulesAdded
		total.AllowRulesSkipped += resp.AllowRulesSkipped
	}
	return total, nil
}
