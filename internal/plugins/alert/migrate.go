package alert

import (
	"strings"

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
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	err := db.Exec("ALTER TABLE `alert_monitor_rules` DROP COLUMN `duty_schedule_id`").Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that column/key exists") ||
		strings.Contains(msg, "unknown column") || strings.Contains(msg, "1054") {
		return nil
	}
	return err
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
