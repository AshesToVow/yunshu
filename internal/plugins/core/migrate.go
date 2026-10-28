package core

import (
	"context"
	"strings"
	"time"
	"yunshu/internal/service"

	"yunshu/internal/model"
	"yunshu/internal/pkg/database"
	"yunshu/internal/pkg/dictmask"

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
	if err := cleanupDictEntriesDuplicatesOnBoot(db); err != nil {
		return err
	}
	// 唯一索引依赖辅助列，必须在 AutoMigrate 之前补列并回填，否则建索引会直接失败。
	return prepareDictEntryUniqueness(db)
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

// prepareDictEntryUniqueness 为存量字典补唯一性辅助列并回填，使 AutoMigrate 能安全创建
// (dict_type, value_hash, active_key) 与 (dict_type, label, active_key) 唯一索引：
// value 为 TEXT（kubeconfig 可达 MB 级）无法直接建索引，改用 SHA-256；
// 软删除行 active_key 置为自身 id，让出唯一键，保证同类型/同值可被重复创建。
func prepareDictEntryUniqueness(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	migrator := db.Migrator()
	for _, column := range []string{"ValueHash", "ActiveKey"} {
		if migrator.HasColumn(&model.DictEntry{}, column) {
			continue
		}
		if err := migrator.AddColumn(&model.DictEntry{}, column); err != nil {
			return err
		}
	}
	if err := db.Exec("UPDATE dict_entries SET active_key = id WHERE deleted_at IS NOT NULL AND active_key = 0").Error; err != nil {
		return err
	}
	if err := backfillDictEntryValueHash(db); err != nil {
		return err
	}
	return dedupeDictEntriesByUniqueKey(db)
}

// backfillDictEntryValueHash 分批回填存量行的 value_hash（含软删除行，索引同样覆盖它们）。
func backfillDictEntryValueHash(db *gorm.DB) error {
	const batchSize = 500
	var lastID uint
	for {
		var rows []model.DictEntry
		if err := db.Unscoped().
			Select("id", "value").
			Where("id > ? AND (value_hash IS NULL OR value_hash = '')", lastID).
			Order("id ASC").Limit(batchSize).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			if err := db.Model(&model.DictEntry{}).Unscoped().
				Where("id = ?", row.ID).
				UpdateColumn("value_hash", dictmask.ValueHash(row.Value)).Error; err != nil {
				return err
			}
		}
		lastID = rows[len(rows)-1].ID
	}
}

// dedupeDictEntriesByUniqueKey 按最终唯一键口径（类型+值哈希、类型+标签）软删除活跃重复行，
// 保留最小 id。SQL 去重按 TRIM 比较，与 Go 侧 TrimSpace 存在差异，这里兜底后再建索引。
func dedupeDictEntriesByUniqueKey(db *gorm.DB) error {
	var rows []model.DictEntry
	if err := db.Unscoped().Model(&model.DictEntry{}).
		Select("id", "dict_type", "label", "value_hash").
		Where("deleted_at IS NULL").
		Order("id ASC").Find(&rows).Error; err != nil {
		return err
	}
	seenLabel := make(map[string]struct{}, len(rows))
	seenValue := make(map[string]struct{}, len(rows))
	victims := make([]uint, 0, len(rows))
	for _, row := range rows {
		labelKey := row.DictType + "\x00" + strings.TrimSpace(row.Label)
		valueKey := row.DictType + "\x00" + row.ValueHash
		_, dupLabel := seenLabel[labelKey]
		_, dupValue := seenValue[valueKey]
		if dupLabel || dupValue {
			victims = append(victims, row.ID)
			continue
		}
		seenLabel[labelKey] = struct{}{}
		seenValue[valueKey] = struct{}{}
	}
	for start := 0; start < len(victims); start += 500 {
		end := start + 500
		if end > len(victims) {
			end = len(victims)
		}
		if err := db.Exec(
			"UPDATE dict_entries SET deleted_at = ?, active_key = id WHERE deleted_at IS NULL AND id IN ?",
			time.Now(), victims[start:end],
		).Error; err != nil {
			return err
		}
	}
	return nil
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
