// Package k8srole 对齐 k8m 的集群角色与鉴权：cluster_admin / cluster_readonly / cluster_pod_exec，
// 命名空间黑白名单写在授权行上（Namespaces / BlacklistNamespaces）。
package k8srole

import (
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8scaps"
)

const (
	ClusterAdmin    = "cluster_admin"
	ClusterReadonly = "cluster_readonly"
	ClusterPodExec  = "cluster_pod_exec"
)

// Normalize 将历史档位别名归一为 k8m 角色码。readonly_exec 不是存储角色，原样返回以便 Expand。
func Normalize(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case ClusterAdmin, k8scaps.PresetAdmin:
		return ClusterAdmin
	case ClusterReadonly, k8scaps.PresetReadonly:
		return ClusterReadonly
	case ClusterPodExec, "pod_exec":
		return ClusterPodExec
	case k8scaps.PresetReadonlyExec:
		return k8scaps.PresetReadonlyExec
	default:
		return strings.ToLower(strings.TrimSpace(preset))
	}
}

// ExpandToStoredRoles 一次下发要写入的角色行（只读+Exec 拆成两行，与 k8m 一致）。
func ExpandToStoredRoles(preset string, caps []string) []string {
	if n := k8scaps.Normalize(caps); len(n) > 0 {
		switch k8scaps.Rank(k8scaps.EnsureRead(n)) {
		case k8scaps.RankAdmin:
			return []string{ClusterAdmin}
		case k8scaps.RankReadonlyExec:
			return []string{ClusterReadonly, ClusterPodExec}
		default:
			return []string{ClusterReadonly}
		}
	}
	switch Normalize(preset) {
	case ClusterAdmin:
		return []string{ClusterAdmin}
	case ClusterPodExec:
		return []string{ClusterPodExec}
	case k8scaps.PresetReadonlyExec:
		return []string{ClusterReadonly, ClusterPodExec}
	case ClusterReadonly:
		return []string{ClusterReadonly}
	default:
		return nil
	}
}

// CapsForRole 单行角色对应的展示用能力包（兼容旧 UI / my-access）。
func CapsForRole(role string) []string {
	switch Normalize(role) {
	case ClusterAdmin:
		return k8scaps.All()
	case ClusterPodExec:
		return []string{k8scaps.Exec}
	case ClusterReadonly:
		return []string{k8scaps.Read}
	case k8scaps.PresetReadonlyExec:
		return []string{k8scaps.Read, k8scaps.Exec}
	default:
		return k8scaps.ForPreset(role)
	}
}

// CapsFromGrants 合并多行角色。
func CapsFromGrants(grants []model.K8sClusterAccessGrant) []string {
	var merged []string
	for _, g := range grants {
		merged = append(merged, CapsForRole(g.Preset)...)
	}
	return k8scaps.Normalize(merged)
}

// RankFromGrants 与旧 AccessRank 对齐：admin=3，只读+exec=2，只读=1。
func RankFromGrants(grants []model.K8sClusterAccessGrant) int {
	return k8scaps.Rank(CapsFromGrants(grants))
}

// InferPresetFromGrants 用于 my-access 展示。
func InferPresetFromGrants(grants []model.K8sClusterAccessGrant) string {
	hasAdmin, hasRO, hasExec := false, false, false
	for _, g := range grants {
		switch Normalize(g.Preset) {
		case ClusterAdmin:
			hasAdmin = true
		case ClusterReadonly:
			hasRO = true
		case ClusterPodExec:
			hasExec = true
		case k8scaps.PresetReadonlyExec:
			hasRO, hasExec = true, true
		}
	}
	switch {
	case hasAdmin:
		return ClusterAdmin
	case hasRO && hasExec:
		return k8scaps.PresetReadonlyExec
	case hasExec:
		return ClusterPodExec
	case hasRO:
		return ClusterReadonly
	default:
		return ""
	}
}

// LabelCN 角色中文名。
func LabelCN(preset string) string {
	switch Normalize(preset) {
	case ClusterAdmin:
		return "集群管理员"
	case ClusterReadonly:
		return "集群只读"
	case ClusterPodExec:
		return "Pod Exec"
	case k8scaps.PresetReadonlyExec:
		return "只读+Exec"
	default:
		return k8scaps.PresetLabelCN(preset)
	}
}

// JoinNamespaces 逗号拼接（去空、去重保序）。
func JoinNamespaces(list []string) string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range list {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return strings.Join(out, ",")
}

// SplitNamespaces 解析授权行上的逗号名单。
func SplitNamespaces(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(csv, ",") {
		s := strings.TrimSpace(part)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
