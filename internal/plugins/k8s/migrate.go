package k8s

import (
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8scaps"
	"yunshu/internal/pkg/k8srole"

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

func migrateDropLegacyK8sGrantUnique(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.K8sClusterAccessGrant{}) {
		return nil
	}
	// 旧唯一键 (kind, ref, cluster_id) 无法容纳 k8m 同一集群的只读+Exec 两行。
	if db.Migrator().HasIndex(&model.K8sClusterAccessGrant{}, "uk_k8s_cluster_grant") {
		_ = db.Migrator().DropIndex(&model.K8sClusterAccessGrant{}, "uk_k8s_cluster_grant")
	}
	return nil
}

func migrateK8sGrantsToK8mRoles(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.K8sClusterAccessGrant{}) {
		return nil
	}
	var grants []model.K8sClusterAccessGrant
	if err := db.Find(&grants).Error; err != nil {
		return err
	}
	type nsKey struct {
		kind, ref string
		cluster   uint
	}
	allowBy := map[nsKey][]string{}
	denyBy := map[nsKey][]string{}
	if db.Migrator().HasTable(&model.K8sNamespaceAllowRule{}) {
		var allows []model.K8sNamespaceAllowRule
		if err := db.Find(&allows).Error; err != nil {
			return err
		}
		for _, a := range allows {
			k := nsKey{kind: a.PrincipalKind, ref: a.PrincipalRef, cluster: a.ClusterID}
			allowBy[k] = append(allowBy[k], a.Namespace)
		}
	}
	if db.Migrator().HasTable(&model.K8sNamespaceDenyRule{}) {
		var denys []model.K8sNamespaceDenyRule
		if err := db.Find(&denys).Error; err != nil {
			return err
		}
		for _, d := range denys {
			k := nsKey{kind: d.PrincipalKind, ref: d.PrincipalRef, cluster: d.ClusterID}
			denyBy[k] = append(denyBy[k], d.Namespace)
		}
	}

	seenExec := map[nsKey]bool{}
	for i := range grants {
		g := &grants[i]
		k := nsKey{kind: g.PrincipalKind, ref: g.PrincipalRef, cluster: g.ClusterID}
		if strings.TrimSpace(g.Namespaces) == "" {
			g.Namespaces = k8srole.JoinNamespaces(allowBy[k])
		}
		if strings.TrimSpace(g.BlacklistNamespaces) == "" {
			g.BlacklistNamespaces = k8srole.JoinNamespaces(denyBy[k])
		}
		preset := strings.ToLower(strings.TrimSpace(g.Preset))
		caps := k8scaps.FromGrant(*g)
		roles := k8srole.ExpandToStoredRoles(preset, nil)
		if preset == k8scaps.PresetCustom || preset == "" {
			roles = k8srole.ExpandToStoredRoles("", caps)
		}
		if len(roles) == 0 {
			roles = []string{k8srole.ClusterReadonly}
		}
		primary := roles[0]
		g.Preset = primary
		g.Capabilities = k8scaps.Marshal(k8srole.CapsForRole(primary))
		if err := db.Save(g).Error; err != nil {
			return err
		}
		if len(roles) > 1 && !seenExec[k] {
			seenExec[k] = true
			extra := model.K8sClusterAccessGrant{
				PrincipalKind:       g.PrincipalKind,
				PrincipalRef:        g.PrincipalRef,
				ClusterID:           g.ClusterID,
				Preset:              k8srole.ClusterPodExec,
				Namespaces:          g.Namespaces,
				BlacklistNamespaces: g.BlacklistNamespaces,
				Capabilities:        k8scaps.Marshal(k8srole.CapsForRole(k8srole.ClusterPodExec)),
			}
			if err := db.Where("principal_kind = ? AND principal_ref = ? AND cluster_id = ? AND preset = ?",
				extra.PrincipalKind, extra.PrincipalRef, extra.ClusterID, extra.Preset).
				FirstOrCreate(&extra).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
