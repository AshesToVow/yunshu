package bootstrap

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExpectedSchemaVersion 与 migrate 写入的版本对齐；启动校验失败则拒绝起服。
// 每次改变 AutoMigrate 模型集合或破坏性种子时递增。
const ExpectedSchemaVersion = 20260906

const schemaMetaKey = "schema_version"

// SchemaMeta 平台 schema 版本元数据（单行键值）。
type SchemaMeta struct {
	Key   string `gorm:"primaryKey;size:64"`
	Value string `gorm:"size:64;not null"`
}

func (SchemaMeta) TableName() string { return "yunshu_schema_meta" }

// RecordSchemaVersion 在 migrate 成功后写入期望版本。
func RecordSchemaVersion(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.AutoMigrate(&SchemaMeta{}); err != nil {
		return fmt.Errorf("schema meta migrate: %w", err)
	}
	row := SchemaMeta{Key: schemaMetaKey, Value: strconv.Itoa(ExpectedSchemaVersion)}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&row).Error
}

// CheckSchemaVersion 生产启动闸门：DB 版本必须与二进制期望一致。
func CheckSchemaVersion(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("schema version check: db is nil")
	}
	if !db.Migrator().HasTable(&SchemaMeta{}) {
		return fmt.Errorf(
			"schema version missing: table yunshu_schema_meta not found (run `yunshu migrate`, expected=%d)",
			ExpectedSchemaVersion,
		)
	}
	var row SchemaMeta
	err := db.Where("`key` = ?", schemaMetaKey).First(&row).Error
	if err != nil {
		return fmt.Errorf(
			"schema version missing: %w (run `yunshu migrate`, expected=%d)",
			err, ExpectedSchemaVersion,
		)
	}
	got, _ := strconv.Atoi(row.Value)
	if got != ExpectedSchemaVersion {
		return fmt.Errorf(
			"schema version mismatch: db=%d expected=%d (run `yunshu migrate` with matching binary)",
			got, ExpectedSchemaVersion,
		)
	}
	return nil
}
