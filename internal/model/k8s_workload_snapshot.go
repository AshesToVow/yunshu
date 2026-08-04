package model

import (
	"time"

	"gorm.io/gorm"
)

// K8sWorkloadSnapshot 写操作前快照，用于回滚。
type K8sWorkloadSnapshot struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ProjectID uint           `json:"project_id" gorm:"not null;index"`
	ClusterID uint           `json:"cluster_id" gorm:"not null;index"`
	Namespace string         `json:"namespace" gorm:"size:128;not null;index"`
	Kind      string         `json:"kind" gorm:"size:64;not null"`
	Name      string         `json:"name" gorm:"size:256;not null;index"`
	YAML      string         `json:"yaml" gorm:"type:longtext;not null"`
	ActorID   *uint          `json:"actor_id"`
	Reason    string         `json:"reason" gorm:"size:128"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (K8sWorkloadSnapshot) TableName() string { return "k8s_workload_snapshots" }
