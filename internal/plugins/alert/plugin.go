package alert

import (
	"yunshu/internal/model"
	"yunshu/internal/plugin"

	"gorm.io/gorm"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "alert" }
func (m *module) Description() string { return "告警平台：摄入、规则、值班、多渠道通知" }

func (m *module) Models() []any {
	return []any{
		&model.AlertChannel{},
		&model.AlertEvent{},
		&model.AlertDatasource{},
		&model.AlertSilence{},
		&model.AlertMaintenanceWindow{},
		&model.AlertMonitorRule{},
		&model.AlertRuleAssignee{},
		&model.AlertDutyBlock{},
		&model.AlertInhibitionRule{},
		&model.AlertInhibitionEvent{},
		&model.AlertSubscriptionNode{},
		&model.AlertReceiverGroup{},
		&model.AlertSubscriptionMatch{},
		&model.AlertFiringDelivery{},
		&model.CloudExpiryRule{},
	}
}

func (m *module) PreMigrate(db *gorm.DB) error {
	if err := dropAlertDutyBlocksLegacyScheduleID(db); err != nil {
		return err
	}
	return dropAlertMonitorRulesLegacyDutyScheduleID(db)
}

func (m *module) PostMigrate(db *gorm.DB) error {
	if err := migrateNormalizeAlertEventStatus(db); err != nil {
		return err
	}
	return dropLegacyUnusedTables(db)
}
