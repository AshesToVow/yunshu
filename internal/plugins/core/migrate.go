package core

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/database"
	workflowsvc "yunshu/internal/service/workflow"

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
	// 先去重，再删含 deleted_at 的失效唯一索引，避免 AutoMigrate 建 (resource,action) 唯一键失败。
	if err := cleanupPermissionsDuplicatesOnBoot(db); err != nil {
		return err
	}
	if err := dropLegacyPermissionUniqueIndexes(db); err != nil {
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
	if dialect != "mysql" && dialect != "postgres" {
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

func dropIndexIfPresent(db *gorm.DB, table, index string) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" {
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
	if err := migrateEnableDingTalkSignSecretDictSeed(db); err != nil {
		return err
	}
	if err := migrateFixWecomNotifyModeDictTypo(db); err != nil {
		return err
	}
	if err := workflowsvc.MigrateLegacyDefinitions(context.Background(), db); err != nil {
		return err
	}
	return workflowsvc.MigrateLegacyTickets(context.Background(), db)
}

func dropDictEntriesLegacyCompositeIndex(db *gorm.DB) error {
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" {
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
	if dialect != "mysql" && dialect != "postgres" {
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
