package core

import (
	"context"
	"strings"
	"yunshu/internal/service"

	"yunshu/internal/model"
	"yunshu/internal/pkg/database"

	"gorm.io/gorm"
)

func bootstrapPreMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := dropDictEntriesLegacyCompositeIndex(db); err != nil {
		return err
	}
	if err := dropLegacyRoleUniqueIndexes(db); err != nil {
		return err
	}
	if err := dropLegacyRegistrationUniqueIndexes(db); err != nil {
		return err
	}
	// 先去重，再删含 deleted_at 的失效唯一索引，避免 AutoMigrate 建 (resource,action) 唯一键失败。
	if err := cleanupPermissionsDuplicatesOnBoot(db); err != nil {
		return err
	}
	if err := dropLegacyPermissionUniqueIndexes(db); err != nil {
		return err
	}
	// 工单 (ref_type, ref_id, ticket_type) 幂等唯一键：先清存量重复，再删旧索引。
	if err := cleanupWorkflowTicketRefDuplicatesOnBoot(db); err != nil {
		return err
	}
	if err := dropIndexIfPresent(db, "workflow_tickets", "idx_wf_ticket_ref"); err != nil {
		return err
	}
	return cleanupDictEntriesDuplicatesOnBoot(db)
}

// dropLegacyRoleUniqueIndexes 删除不含 deleted_at 的旧唯一索引，使 AutoMigrate 重建为软删除友好的复合索引。
func dropLegacyRoleUniqueIndexes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Role{}) {
		return nil
	}
	for _, idx := range []string{"idx_roles_name", "idx_roles_code"} {
		if err := dropIndexIfPresent(db, "roles", idx); err != nil {
			return err
		}
	}
	return nil
}

// dropLegacyRegistrationUniqueIndexes 删除注册申请的历史唯一索引：
// 被拒申请需保留审计记录且不得永久占用用户名/邮箱，改由服务层按待审状态判重。
func dropLegacyRegistrationUniqueIndexes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.RegistrationRequest{}) {
		return nil
	}
	for _, idx := range []string{"idx_registration_requests_username", "idx_registration_requests_email"} {
		if err := dropIndexIfPresent(db, "registration_requests", idx); err != nil {
			return err
		}
	}
	return nil
}

// dropLegacyPermissionUniqueIndexes 删除历史权限唯一索引，由 AutoMigrate 重建为 idx_perm_resource_action。
func dropLegacyPermissionUniqueIndexes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Permission{}) {
		return nil
	}
	for _, idx := range []string{"idx_resource_action", "idx_perm_resource_action_deleted"} {
		if err := dropIndexIfPresent(db, "permissions", idx); err != nil {
			return err
		}
	}
	return nil
}

func cleanupPermissionsDuplicatesOnBoot(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasTable(&model.Permission{}) {
		return nil
	}
	for _, sql := range []string{
		database.SQLDeletePermissionActiveDuplicates(dialect),
		database.SQLNormalizePermissionActionUpper(dialect),
		database.SQLDeletePermissionSoftDuplicatesWhenActive(dialect),
		database.SQLDeletePermissionSoftOnlyDuplicates(dialect),
	} {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// workflowTicketRefKey 工单引用幂等键。
type workflowTicketRefKey struct {
	RefType    string `gorm:"column:ref_type"`
	RefID      uint   `gorm:"column:ref_id"`
	TicketType string `gorm:"column:ticket_type"`
}

// cleanupWorkflowTicketRefDuplicatesOnBoot 清掉同 (ref_type, ref_id, ticket_type) 的重复工单
// （活跃组保留最小 id；有活跃行时删除全部软删副本；仅软删组保留最小 id），
// 否则 AutoMigrate 无法创建 idx_wf_ticket_ref_uq 唯一索引。被删工单的步骤一并物理删除。
func cleanupWorkflowTicketRefDuplicatesOnBoot(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasTable(&model.WorkflowTicket{}) {
		return nil
	}
	var keys []workflowTicketRefKey
	if err := db.Model(&model.WorkflowTicket{}).Unscoped().
		Select("ref_type, ref_id, ticket_type").
		Where("ref_type <> '' AND ref_id <> 0").
		Group("ref_type, ref_id, ticket_type").
		Having("COUNT(*) > 1").
		Scan(&keys).Error; err != nil {
		return err
	}
	for _, k := range keys {
		keeper, err := workflowTicketRefKeeperOnBoot(db, k)
		if err != nil {
			return err
		}
		if keeper == 0 {
			continue
		}
		var victims []uint
		if err := db.Model(&model.WorkflowTicket{}).Unscoped().
			Where("ref_type = ? AND ref_id = ? AND ticket_type = ? AND id <> ?",
				k.RefType, k.RefID, k.TicketType, keeper).
			Pluck("id", &victims).Error; err != nil {
			return err
		}
		if len(victims) == 0 {
			continue
		}
		if err := db.Unscoped().Where("ticket_id IN ?", victims).
			Delete(&model.WorkflowTicketStep{}).Error; err != nil {
			return err
		}
		if err := db.Unscoped().Where("id IN ?", victims).
			Delete(&model.WorkflowTicket{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// workflowTicketRefKeeperOnBoot 同键多行时决定保留行：优先活跃最小 id，其次软删最小 id。
func workflowTicketRefKeeperOnBoot(db *gorm.DB, k workflowTicketRefKey) (uint, error) {
	var keeper uint
	err := db.Model(&model.WorkflowTicket{}).
		Where("ref_type = ? AND ref_id = ? AND ticket_type = ? AND deleted_at IS NULL",
			k.RefType, k.RefID, k.TicketType).
		Order("id ASC").Limit(1).Pluck("id", &keeper).Error
	if err != nil {
		return 0, err
	}
	if keeper > 0 {
		return keeper, nil
	}
	if err := db.Model(&model.WorkflowTicket{}).Unscoped().
		Where("ref_type = ? AND ref_id = ? AND ticket_type = ? AND deleted_at IS NOT NULL",
			k.RefType, k.RefID, k.TicketType).
		Order("id ASC").Limit(1).Pluck("id", &keeper).Error; err != nil {
		return 0, err
	}
	return keeper, nil
}

// backfillWorkflowTicketRefTypeUQ 为存量工单回填幂等辅助列（新建列默认 NULL）。
// 仅回填 ref 有效（ref_type 非空且 ref_id 非 0）的行；其余工单保持 NULL 不受唯一键约束。
func backfillWorkflowTicketRefTypeUQ(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasTable(&model.WorkflowTicket{}) ||
		!db.Migrator().HasColumn(&model.WorkflowTicket{}, "ref_type_uq") {
		return nil
	}
	return db.Model(&model.WorkflowTicket{}).Unscoped().
		Where("ref_type_uq IS NULL AND ref_type <> '' AND ref_id <> 0").
		Update("ref_type_uq", gorm.Expr("ref_type")).Error
}

func dropIndexIfPresent(db *gorm.DB, table, index string) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasIndex(table, index) {
		return nil
	}
	err := db.Exec(database.SQLDropIndexIfExists(dialect, table, index)).Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that it exists") || strings.Contains(msg, "does not exist") {
		return nil
	}
	return err
}

func bootstrapPostMigrateCore(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := backfillWorkflowTicketRefTypeUQ(db); err != nil {
		return err
	}
	if err := migrateEnableDingTalkSignSecretDictSeed(db); err != nil {
		return err
	}
	if err := migrateFixWecomNotifyModeDictTypo(db); err != nil {
		return err
	}
	if err := service.MigrateLegacyDefinitions(context.Background(), db); err != nil {
		return err
	}
	if err := service.MigrateLegacyTickets(context.Background(), db); err != nil {
		return err
	}
	return service.EnsureSeeded(context.Background(), db)
}

func dropDictEntriesLegacyCompositeIndex(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	err := db.Exec(database.SQLDropDictEntriesLegacyCompositeIndex(dialect)).Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that it exists") || strings.Contains(msg, "does not exist") {
		return nil
	}
	return err
}

func cleanupDictEntriesDuplicatesOnBoot(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" && !database.IsDameng(dialect) {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	if err := db.Exec(database.SQLDeleteDictDuplicatesByLabel(dialect)).Error; err != nil {
		return err
	}
	return db.Exec(database.SQLDeleteDictDuplicatesByValue(dialect)).Error
}

func migrateEnableDingTalkSignSecretDictSeed(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	return db.Model(&model.DictEntry{}).
		Where("dict_type = ? AND label = ? AND status = ?", "dingtalk_sign_secret", "钉钉 SignSecret 示例", 0).
		Update("status", 1).Error
}

func migrateFixWecomNotifyModeDictTypo(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	return db.Model(&model.DictEntry{}).
		Where("dict_type = ?", "wwcom_notify_mode").
		Update("dict_type", "wecom_notify_mode").Error
}
