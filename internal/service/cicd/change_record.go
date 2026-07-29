package cicd

import (
	"context"
	"fmt"

	"yunshu/internal/model"
	"yunshu/internal/service/changeevent"

	"gorm.io/gorm"
)

func recordReleaseChange(ctx context.Context, db *gorm.DB, release *model.CicdReleaseRun, action, status, summary string) {
	if release == nil || release.ProjectID == 0 {
		return
	}
	if summary == "" {
		summary = fmt.Sprintf("CI/CD release #%d %s", release.ID, action)
	}
	var catalogID *uint
	if db != nil && release.ServiceID > 0 {
		var link model.ServiceLink
		err := db.WithContext(ctx).
			Joins("JOIN service_catalog sc ON sc.id = service_links.service_id AND sc.project_id = ? AND sc.deleted_at IS NULL", release.ProjectID).
			Where("service_links.link_type = ? AND service_links.ref_id = ? AND service_links.deleted_at IS NULL",
				model.ServiceLinkCicdService, release.ServiceID).
			Order("service_links.id DESC").
			First(&link).Error
		if err == nil {
			id := link.ServiceID
			catalogID = &id
		}
	}
	changeevent.Record(ctx, changeevent.Input{
		ProjectID:   release.ProjectID,
		ServiceID:   catalogID,
		Source:      model.ChangeSourceCicd,
		Action:      action,
		RiskLevel:   model.ChangeRiskHigh,
		Status:      status,
		ActorUserID: release.SubmitterUserID,
		Summary:     summary,
		Payload: map[string]any{
			"release_id":      release.ID,
			"cicd_service_id": release.ServiceID,
			"release_type":    release.ReleaseType,
			"tenv":            release.Tenv,
			"title":           release.Title,
		},
		StartedAt: release.StartedAt,
	})
}
