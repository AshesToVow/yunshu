package project

import (
	"yunshu/internal/model"

	"gorm.io/gorm"
)

func migrateProjectsDefaultMeta(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.Project{}) {
		return nil
	}
	if err := db.Model(&model.Project{}).Where("project_type = '' OR project_type IS NULL").Update("project_type", model.ProjectTypeBusiness).Error; err != nil {
		return err
	}
	return db.Model(&model.Project{}).Where("lifecycle_status = '' OR lifecycle_status IS NULL").Update("lifecycle_status", model.ProjectLifecycleActive).Error
}
