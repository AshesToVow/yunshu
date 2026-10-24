package model

import "time"

// K8sClusterAccessGrant 对齐 k8m ClusterUserRole：主体 + 集群 + 三角色，NS 黑白名单写在同一行。
// ClusterID=0 表示适用于全部已纳管集群。同一主体同一集群可有多行（只读 + Pod Exec）。
type K8sClusterAccessGrant struct {
	ID            uint   `json:"id" gorm:"primaryKey;comment:主键"`
	PrincipalKind string `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_k8s_cluster_grant;comment:role|user|group"`
	PrincipalRef  string `json:"principal_ref" gorm:"size:128;not null;uniqueIndex:uk_k8s_cluster_grant;comment:角色码/用户ID字符串/组编码"`
	ClusterID     uint   `json:"cluster_id" gorm:"not null;default:0;uniqueIndex:uk_k8s_cluster_grant;comment:平台集群ID，0=全部集群"`
	// Preset 存储 k8m 角色：cluster_admin / cluster_readonly / cluster_pod_exec（兼容旧值 readonly|admin）。
	Preset string `json:"preset" gorm:"size:32;not null;uniqueIndex:uk_k8s_cluster_grant;comment:cluster_admin|cluster_readonly|cluster_pod_exec"`
	// Namespaces 白名单，逗号分隔；空表示该角色不限制命名空间（黑名单仍生效）。
	Namespaces string `json:"namespaces" gorm:"type:text;comment:命名空间白名单逗号分隔，空=不限制"`
	// BlacklistNamespaces 黑名单，逗号分隔，优先于白名单。
	BlacklistNamespaces string `json:"blacklist_namespaces" gorm:"type:text;comment:命名空间黑名单逗号分隔"`
	// Capabilities 由角色展开写入，仅兼容旧接口展示。
	Capabilities string    `json:"capabilities" gorm:"type:text;comment:能力包JSON数组（由角色展开）"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (K8sClusterAccessGrant) TableName() string { return "k8s_cluster_access_grants" }
