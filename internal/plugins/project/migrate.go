package project

import (
	"yunshu/internal/model"
	"yunshu/internal/pkg/database"

	"gorm.io/gorm"
)

func migrateLogAgentsClearPlaceholderListenPort(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.LogAgent{}) {
		return nil
	}
	return db.Model(&model.LogAgent{}).Where("listen_port = ?", 12580).Update("listen_port", 0).Error
}

func migrateProjectsDefaultMeta(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Project{}) {
		return nil
	}
	if err := db.Model(&model.Project{}).Where("project_type = '' OR project_type IS NULL").Update("project_type", model.ProjectTypeBusiness).Error; err != nil {
		return err
	}
	return db.Model(&model.Project{}).Where("lifecycle_status = '' OR lifecycle_status IS NULL").Update("lifecycle_status", model.ProjectLifecycleActive).Error
}

func migrateAgentDiscoveryUniqueIndex(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	dialect := database.DialectName(db)
	if dialect != "mysql" && dialect != "postgres" {
		return nil
	}
	if !db.Migrator().HasTable("agent_discoveries") {
		return nil
	}
	if db.Migrator().HasIndex("agent_discoveries", "idx_agent_discovery_unique") {
		return nil
	}
	sql, err := database.SQLCreateAgentDiscoveryUniqueIndex(dialect)
	if err != nil {
		return err
	}
	if err := db.Exec(sql).Error; err != nil && !database.IsDuplicateIndexError(err) {
		return err
	}
	return nil
}
