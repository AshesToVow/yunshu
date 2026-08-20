package model

import (
	"time"

	"gorm.io/gorm"
)

// K8sCluster 已接入的 Kubernetes 集群：名称与 kubeconfig（仅服务端存储，不落 JSON）。
// OwningProjectID 非空时：仅该项目成员可在控制台查看/操作该集群（超级管理员除外）；为空表示平台级共享集群。
//
// 长期凭证策略：
//   - Kubeconfig：可写/高权限凭证（变更类操作）
//   - KubeconfigReadonly：可选只读凭证；配置后只读 API 优先使用，实现最小权限
//   - ImpersonateEnabled：开启后以「网关 SA + Impersonate 用户」访问 apiserver，集群侧按用户/组做 RBAC
type K8sCluster struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name string `json:"name" gorm:"size:128;not null;index;comment:集群名称"`

	OwningProjectID *uint `json:"owning_project_id,omitempty" gorm:"index;comment:可选归属项目，非空则租户隔离"`

	// ConnectionMode 连接模式: kubeconfig 或 direct
	ConnectionMode string `json:"-" gorm:"size:32;default:'kubeconfig';comment:连接模式 kubeconfig/direct"`

	// Kubeconfig is stored encrypted (AES-GCM via security.encryption_key) so the backend can register via Kom.
	// Excluded from API responses; only used internally. Legacy plaintext rows are accepted on read.
	Kubeconfig string `json:"-" gorm:"type:longtext;not null;comment:可写凭证 kubeconfig(加密)"`

	// KubeconfigReadonly 可选只读凭证；空则只读操作回退到 Kubeconfig。
	KubeconfigReadonly string `json:"-" gorm:"type:longtext;comment:只读凭证 kubeconfig(加密)"`

	// DirectConfig 直连配置 JSON（加密），当 ConnectionMode=direct 时使用
	DirectConfig string `json:"-" gorm:"type:longtext;comment:直连配置JSON(加密)"`

	// ImpersonateEnabled 开启后 Rest/kom 客户端附加 Impersonate（需网关 SA 具备 impersonate 权限）。
	ImpersonateEnabled bool `json:"impersonate_enabled" gorm:"not null;default:0;comment:是否启用用户伪装"`

	// ImpersonateUserPrefix 伪装用户名前缀，默认 yunshu:
	ImpersonateUserPrefix string `json:"impersonate_user_prefix" gorm:"size:64;default:'yunshu:';comment:伪装用户名前缀"`

	// RequireDestructiveConfirm 高危操作（drain/helm uninstall/rbac apply）须 confirm=true
	RequireDestructiveConfirm bool `json:"require_destructive_confirm" gorm:"not null;default:1;comment:高危操作须确认"`

	Status int `json:"status" gorm:"not null;default:1;index;comment:状态 1启用 0禁用"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 k8s_clusters。
func (K8sCluster) TableName() string {
	return "k8s_clusters"
}
