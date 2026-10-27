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
	indexes, err := db.Migrator().GetIndexes(&model.K8sClusterAccessGrant{})
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		if !strings.EqualFold(idx.Name(), "uk_k8s_cluster_grant") {
			continue
		}
		// 旧唯一键 (kind, ref, cluster_id) 无法容纳 k8m 同一集群的只读+Exec 两行；
		// 新键含 preset 共四列，已是目标形态，重建会丢失并发写入保护，故只删旧键。
		if len(idx.Columns()) < 4 {
			_ = db.Migrator().DropIndex(&model.K8sClusterAccessGrant{}, idx.Name())
		}
	}
	return nil
}

// nsKey 主体 + 集群维度，用于聚合命名空间黑白名单。
type nsKey struct {
	kind, ref string
	cluster   uint
}

type k8sGrantGroupKey struct {
	kind, ref, preset string
	cluster           uint
}

// k8sGrantMergePlan 迁移计划：合并归一后撞同一唯一键的行，避免 PostMigrate 因 1062 中止启动。
type k8sGrantMergePlan struct {
	Survivors []model.K8sClusterAccessGrant
	DeleteIDs []uint
	ExecRows  []model.K8sClusterAccessGrant
}

func migrateK8sGrantsToK8mRoles(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.K8sClusterAccessGrant{}) {
		return nil
	}
	var grants []model.K8sClusterAccessGrant
	if err := db.Order("id ASC").Find(&grants).Error; err != nil {
		return err
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

	plan := planK8sGrantMigration(grants, allowBy, denyBy)
	return db.Transaction(func(tx *gorm.DB) error {
		if len(plan.DeleteIDs) > 0 {
			if err := tx.Where("id IN ?", plan.DeleteIDs).Delete(&model.K8sClusterAccessGrant{}).Error; err != nil {
				return err
			}
		}
		for i := range plan.Survivors {
			if err := tx.Save(&plan.Survivors[i]).Error; err != nil {
				return err
			}
		}
		for i := range plan.ExecRows {
			extra := plan.ExecRows[i]
			if err := tx.Where("principal_kind = ? AND principal_ref = ? AND cluster_id = ? AND preset = ?",
				extra.PrincipalKind, extra.PrincipalRef, extra.ClusterID, extra.Preset).
				FirstOrCreate(&extra).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// planK8sGrantMigration 按「主体 + 集群 + 目标角色」分组：某组的首行保留并重写为归一后的角色，
// 其余同组行删除（历史数据里 readonly 与 cluster_readonly 并存会撞 uk_k8s_cluster_grant）。
func planK8sGrantMigration(
	grants []model.K8sClusterAccessGrant,
	allowBy, denyBy map[nsKey][]string,
) k8sGrantMergePlan {
	var plan k8sGrantMergePlan
	groups := map[k8sGrantGroupKey][]*model.K8sClusterAccessGrant{}
	order := make([]k8sGrantGroupKey, 0, len(grants))
	needExec := map[k8sGrantGroupKey]bool{}
	for i := range grants {
		g := &grants[i]
		k := nsKey{kind: g.PrincipalKind, ref: g.PrincipalRef, cluster: g.ClusterID}
		if strings.TrimSpace(g.Namespaces) == "" {
			g.Namespaces = k8srole.JoinNamespaces(allowBy[k])
		}
		if strings.TrimSpace(g.BlacklistNamespaces) == "" {
			g.BlacklistNamespaces = k8srole.JoinNamespaces(denyBy[k])
		}
		roles := storedRolesForGrant(g)
		gk := k8sGrantGroupKey{
			kind: g.PrincipalKind, ref: g.PrincipalRef, cluster: g.ClusterID, preset: roles[0],
		}
		if _, ok := groups[gk]; !ok {
			order = append(order, gk)
		}
		groups[gk] = append(groups[gk], g)
		if len(roles) > 1 {
			needExec[gk] = true
		}
	}
	for _, gk := range order {
		rows := groups[gk]
		survivor := *rows[0]
		survivor.Preset = gk.preset
		survivor.Namespaces = mergeGrantNamespaces(rows)
		survivor.BlacklistNamespaces = mergeGrantBlacklist(rows)
		survivor.Capabilities = k8scaps.Marshal(k8srole.CapsForRole(gk.preset))
		plan.Survivors = append(plan.Survivors, survivor)
		for _, row := range rows[1:] {
			plan.DeleteIDs = append(plan.DeleteIDs, row.ID)
		}
		if needExec[gk] {
			plan.ExecRows = append(plan.ExecRows, model.K8sClusterAccessGrant{
				PrincipalKind:       survivor.PrincipalKind,
				PrincipalRef:        survivor.PrincipalRef,
				ClusterID:           survivor.ClusterID,
				Preset:              k8srole.ClusterPodExec,
				Namespaces:          survivor.Namespaces,
				BlacklistNamespaces: survivor.BlacklistNamespaces,
				Capabilities:        k8scaps.Marshal(k8srole.CapsForRole(k8srole.ClusterPodExec)),
			})
		}
	}
	return plan
}

// storedRolesForGrant 计算一行应落库的角色：已知档位直接映射，custom/空档位按能力包展开。
func storedRolesForGrant(g *model.K8sClusterAccessGrant) []string {
	preset := strings.ToLower(strings.TrimSpace(g.Preset))
	caps := k8scaps.FromGrant(*g)
	roles := k8srole.ExpandToStoredRoles(preset, nil)
	if preset == k8scaps.PresetCustom || preset == "" {
		roles = k8srole.ExpandToStoredRoles("", caps)
	}
	if len(roles) == 0 {
		roles = []string{k8srole.ClusterReadonly}
	}
	return roles
}

// mergeGrantNamespaces 同组合并白名单：任一行留空表示不限制命名空间，合并后仍不限制；否则取并集。
func mergeGrantNamespaces(rows []*model.K8sClusterAccessGrant) string {
	var all []string
	for _, row := range rows {
		if strings.TrimSpace(row.Namespaces) == "" {
			return ""
		}
		all = append(all, k8srole.SplitNamespaces(row.Namespaces)...)
	}
	return k8srole.JoinNamespaces(all)
}

// mergeGrantBlacklist 同组合并黑名单：黑名单语义为拒绝，取并集更安全。
func mergeGrantBlacklist(rows []*model.K8sClusterAccessGrant) string {
	var all []string
	for _, row := range rows {
		all = append(all, k8srole.SplitNamespaces(row.BlacklistNamespaces)...)
	}
	return k8srole.JoinNamespaces(all)
}
