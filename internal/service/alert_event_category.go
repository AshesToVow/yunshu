package service

import "strings"

const (
	AlertEventCategoryDelivery   = "delivery"
	AlertEventCategoryRouting    = "routing"
	AlertEventCategorySilence    = "silence"
	AlertEventCategoryInhibition = "inhibition"
	AlertEventCategoryTiming     = "timing"
	AlertEventCategoryResolved   = "resolved"
	AlertEventCategoryFailure    = "failure"
	AlertEventCategoryOther      = "other"
)

func ValidAlertEventCategory(category string) bool {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case AlertEventCategoryDelivery,
		AlertEventCategoryRouting,
		AlertEventCategorySilence,
		AlertEventCategoryInhibition,
		AlertEventCategoryTiming,
		AlertEventCategoryResolved,
		AlertEventCategoryFailure,
		AlertEventCategoryOther:
		return true
	default:
		return false
	}
}
