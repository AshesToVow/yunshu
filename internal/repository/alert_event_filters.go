package repository

import (
	"fmt"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

func applyAlertEventProjectFilter(tx *gorm.DB, db *gorm.DB, projectID uint) *gorm.DB {
	if projectID == 0 || tx == nil || db == nil {
		return tx
	}
	dsSub := db.Model(&model.AlertDatasource{}).Select("id").Where("project_id = ?", projectID)
	pid := fmt.Sprintf("%d", projectID)
	return tx.Where(
		"datasource_id IN (?) OR request_payload LIKE ? OR request_payload LIKE ?",
		dsSub,
		`%"project_id":"`+pid+`"%`,
		`%"project_id":`+pid+`,%`,
	)
}

func applyAlertEventCategoryFilter(tx *gorm.DB, category string) *gorm.DB {
	cat := strings.TrimSpace(strings.ToLower(category))
	if cat == "" {
		return tx
	}
	switch cat {
	case "inhibition":
		return tx.Where("error_message LIKE ?", "inhibition_suppressed:%")
	case "silence":
		return tx.Where("error_message IN ?", []string{"silence_suppressed", "subscription_suppressed"})
	case "timing":
		return tx.Where("error_message IN ?", []string{
			"group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled",
		})
	case "resolved":
		return tx.Where("error_message IN ?", []string{
			"resolved_aggregate_suppressed", "resolved_no_prior_firing_delivery",
		})
	case "routing":
		return tx.Where("error_message IN ?", []string{
			"no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription",
		})
	case "failure":
		return tx.Where("success = ? OR error_message = ?", false, "all_channel_delivery_failed")
	case "delivery":
		return tx.Where(
			"success = ? AND channel_id > 0 AND (error_message IS NULL OR TRIM(error_message) = '')",
			true,
		)
	case "other":
		return tx.Where(
			`success = ? AND TRIM(COALESCE(error_message, '')) != '' 
AND error_message NOT LIKE ? 
AND error_message NOT IN ?`,
			true,
			"inhibition_suppressed:%",
			[]string{
				"silence_suppressed", "subscription_suppressed",
				"group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled",
				"resolved_aggregate_suppressed", "resolved_no_prior_firing_delivery",
				"no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription",
				"all_channel_delivery_failed",
			},
		)
	default:
		return tx
	}
}
