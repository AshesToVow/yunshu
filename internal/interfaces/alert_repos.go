package interfaces

import "yunshu/internal/repository"

type (
	AlertEventRepository          = repository.AlertEventRepo
	AlertChannelRepository        = repository.AlertChannelRepo
	AlertSilenceRepository        = repository.AlertSilenceRepo
	AlertInhibitionRuleRepository = repository.AlertInhibitionRuleRepo
	AlertSubscriptionRepository   = repository.AlertSubscriptionRepo
	AlertDatasourceRepository     = repository.AlertDatasourceRepo
	AlertMonitorRuleRepository    = repository.AlertMonitorRuleRepo
	AlertReceiverGroupRepository  = repository.AlertReceiverGroupRepo
	AlertDutyRepository           = repository.AlertDutyRepo
	AlertRuleAssigneeRepository   = repository.AlertRuleAssigneeRepo
	AlertFiringDeliveryRepository = repository.AlertFiringDeliveryRepo
	CloudExpiryRuleRepository     = repository.CloudExpiryRuleRepo
)
