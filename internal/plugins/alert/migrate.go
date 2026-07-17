package alert

import (
	"yunshu/internal/model"

	"gorm.io/gorm"
)

func dropAlertDutyBlocksLegacyScheduleID(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertDutyBlock{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.AlertDutyBlock{}, "schedule_id") {
		return nil
	}
	return db.Migrator().DropColumn(&model.AlertDutyBlock{}, "schedule_id")
}

func dropAlertMonitorRulesLegacyDutyScheduleID(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertMonitorRule{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.AlertMonitorRule{}, "duty_schedule_id") {
		return nil
	}
	return db.Migrator().DropColumn(&model.AlertMonitorRule{}, "duty_schedule_id")
}

func migrateNormalizeAlertEventStatus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertEvent{}) {
		return nil
	}
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return db.Exec(`
UPDATE alert_events SET status = LOWER(TRIM(status))
WHERE deleted_at IS NULL AND status IS NOT NULL AND status <> LOWER(TRIM(status))`).Error
	case "sqlite":
		return db.Exec(`
UPDATE alert_events SET status = LOWER(TRIM(status))
WHERE deleted_at IS NULL AND status IS NOT NULL AND LOWER(TRIM(status)) <> status`).Error
	default:
		return nil
	}
}

func dropLegacyUnusedTables(db *gorm.DB) error {
	for _, t := range []string{"alert_rule_templates"} {
		if !db.Migrator().HasTable(t) {
			continue
		}
		if err := db.Migrator().DropTable(t); err != nil {
			return err
		}
	}
	return nil
}
