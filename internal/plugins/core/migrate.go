package core

import (
	"strings"

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
	if err := dropLegacyPermissionUniqueIndex(db); err != nil {
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

// dropLegacyPermissionUniqueIndex 删除旧的两列 idx_resource_action，使其重建为含 deleted_at 的复合索引。
func dropLegacyPermissionUniqueIndex(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Permission{}) {
		return nil
	}
	// 仅当旧索引不含 deleted_at 时才需重建：无法在此廉价判定列组成，直接尝试删除，AutoMigrate 会按新定义补回。
	return dropIndexIfPresent(db, "permissions", "idx_resource_action")
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
	return migrateFixWecomNotifyModeDictTypo(db)
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
