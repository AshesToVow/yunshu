package interfaces

import "yunshu/internal/repository"

type (
	AlertEventRepository               = repository.AlertEventRepo
	AlertChannelRepository             = repository.AlertChannelRepo
	AlertSilenceRepository             = repository.AlertSilenceRepo
	AlertMaintenanceWindowRepository   = repository.AlertMaintenanceWindowRepo
	AlertInhibitionRuleRepository      = repository.AlertInhibitionRuleRepo
	AlertSubscriptionRepository        = repository.AlertSubscriptionRepo
	AlertDatasourceRepository          = repository.AlertDatasourceRepo
	AlertMonitorRuleRepository         = repository.AlertMonitorRuleRepo
	AlertReceiverGroupRepository       = repository.AlertReceiverGroupRepo
	AlertDutyRepository                = repository.AlertDutyRepo
	AlertRuleAssigneeRepository        = repository.AlertRuleAssigneeRepo
	AlertFiringDeliveryRepository      = repository.AlertFiringDeliveryRepo
	AlertAckRepository                 = repository.AlertAckRepo
	AlertProgressNoteRepository        = repository.AlertProgressNoteRepo
	AlertCurHisRepository              = repository.AlertCurHisRepo
	AlertConsulRepository              = repository.AlertConsulRepo
	AlertRuleChangeRepository          = repository.AlertRuleChangeRepo
	PromqlSavedQueryRepository         = repository.PromqlSavedQueryRepo
	CloudExpiryRuleRepository          = repository.CloudExpiryRuleRepo
)
