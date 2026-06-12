package k8s

import (
	"yunshu/internal/model"

	"gorm.io/gorm"
)

type migrateK8sGrantLegacy struct {
	RoleCode string `gorm:"column:role_code"`
}

func (migrateK8sGrantLegacy) TableName() string { return "k8s_cluster_access_grants" }

type migrateK8sDenyLegacy struct {
	RoleCode string `gorm:"column:role_code"`
}

func (migrateK8sDenyLegacy) TableName() string { return "k8s_namespace_deny_rules" }

func migrateK8sLegacyRoleCodeToPrincipal(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var gProbe migrateK8sGrantLegacy
	if db.Migrator().HasTable(&model.K8sClusterAccessGrant{}) && db.Migrator().HasColumn(&gProbe, "RoleCode") {
		if err := db.Exec(`UPDATE k8s_cluster_access_grants SET principal_kind = ?, principal_ref = TRIM(role_code) WHERE TRIM(COALESCE(role_code,'')) <> ''`, model.K8sPrincipalRole).Error; err != nil {
			return err
		}
		_ = db.Migrator().DropColumn(&gProbe, "RoleCode")
	}
	var dProbe migrateK8sDenyLegacy
	if db.Migrator().HasTable(&model.K8sNamespaceDenyRule{}) && db.Migrator().HasColumn(&dProbe, "RoleCode") {
		if err := db.Exec(`UPDATE k8s_namespace_deny_rules SET principal_kind = ?, principal_ref = TRIM(role_code) WHERE TRIM(COALESCE(role_code,'')) <> ''`, model.K8sPrincipalRole).Error; err != nil {
			return err
		}
		_ = db.Migrator().DropColumn(&dProbe, "RoleCode")
	}
	return nil
}

func migrateDropLegacyK8sCasbinPolicies(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("casbin_rule") {
		return nil
	}
	return db.Exec("DELETE FROM casbin_rule WHERE ptype = 'p' AND v1 LIKE 'k8s:cluster%'").Error
}
