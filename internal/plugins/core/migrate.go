package core

import (
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

func bootstrapPreMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := dropDictEntriesLegacyCompositeIndex(db); err != nil {
		return err
	}
	return cleanupDictEntriesDuplicatesOnBoot(db)
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
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	err := db.Exec("ALTER TABLE `dict_entries` DROP INDEX `idx_dict_type_value_deleted`").Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that it exists") {
		return nil
	}
	return err
}

func cleanupDictEntriesDuplicatesOnBoot(db *gorm.DB) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	sqlByLabel := `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.label) = TRIM(d2.label) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	if err := db.Exec(sqlByLabel).Error; err != nil {
		return err
	}
	sqlByValue := `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.value) = TRIM(d2.value) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	return db.Exec(sqlByValue).Error
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
