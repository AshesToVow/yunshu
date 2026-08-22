package repository

import (
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

func applyAlertEventProjectFilter(tx *gorm.DB, db *gorm.DB, projectID uint) *gorm.DB {
	if projectID == 0 || tx == nil || db == nil {
		return tx
	}
	dsSub := db.Model(&model.AlertDatasource{}).Select("id").Where("project_id = ?", projectID)
	// Prefer project_id column (backfilled on read). Avoid LIKE on request_payload longtext —
	// that full-scans ~10k+ rows and makes /history/stats time out (UI shows all zeros).
	// 优先 project_id 列（已回填）；仅历史未解析行才走 datasource / 策略关联，避免 FIND_IN_SET 扫全表。
	return tx.Where(
		`project_id = ?
OR (IFNULL(project_id, 0) = 0 AND datasource_id IN (?))
OR (IFNULL(project_id, 0) = 0 AND matched_policy_ids <> '' AND EXISTS (
	SELECT 1 FROM alert_subscription_nodes n
	WHERE n.project_id = ? AND n.deleted_at IS NULL
	  AND FIND_IN_SET(n.id, alert_events.matched_policy_ids)
))`,
		projectID,
		dsSub,
		projectID,
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
			"group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled", "ack_active",
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
				"group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled", "ack_active",
				"resolved_aggregate_suppressed", "resolved_no_prior_firing_delivery",
				"no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription",
				"all_channel_delivery_failed",
			},
		)
	default:
		return tx
	}
}
