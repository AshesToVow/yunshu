package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ServiceLinkCicdService      = "cicd_service"
	ServiceLinkCmdbService      = "cmdb_service"
	ServiceLinkLogSource        = "log_source"
	ServiceLinkK8sWorkload      = "k8s_workload"
	ServiceLinkAlertMonitorRule = "alert_monitor_rule"
	ServiceLinkDbInstance       = "db_instance"
)

// ServiceCatalog 项目级统一服务标识（与 CMDB services / cicd_services 解耦）。
type ServiceCatalog struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ProjectID   uint           `json:"project_id" gorm:"not null;uniqueIndex:uk_svc_catalog_proj_ident,priority:1;index"`
	Identifier  string         `json:"identifier" gorm:"size:128;not null;uniqueIndex:uk_svc_catalog_proj_ident,priority:2;comment:项目内唯一标识"`
	Name        string         `json:"name" gorm:"size:128;not null"`
	Owner       string         `json:"owner" gorm:"size:64"`
	ProductLine string         `json:"product_line" gorm:"size:128"`
	Criticality string         `json:"criticality" gorm:"size:32;default:'normal';comment:critical|high|normal|low"`
	Status      int            `json:"status" gorm:"not null;default:1;comment:1启用 0停用"`
	Remark      string         `json:"remark" gorm:"size:512"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceCatalog) TableName() string { return "service_catalog" }

// ServiceLink 将统一服务关联到各域实体。
type ServiceLink struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ServiceID uint           `json:"service_id" gorm:"not null;uniqueIndex:uk_svc_link,priority:1;index"`
	LinkType  string         `json:"link_type" gorm:"size:32;not null;uniqueIndex:uk_svc_link,priority:2;comment:cicd_service|cmdb_service|..."`
	RefID     *uint          `json:"ref_id" gorm:"uniqueIndex:uk_svc_link,priority:3;comment:数值型外键"`
	RefKey    string         `json:"ref_key" gorm:"size:256;uniqueIndex:uk_svc_link,priority:4;comment:字符串外键如 cluster/ns/kind/name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceLink) TableName() string { return "service_links" }
