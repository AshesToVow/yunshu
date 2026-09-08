package cicd

import (
	"context"
	"fmt"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/service/changeevent"
)

func recordReleaseChange(ctx context.Context, catalogRepo interfaces.ServiceCatalogRepository, release *model.CicdReleaseRun, action, status, summary string) {
	if release == nil || release.ProjectID == 0 {
		return
	}
	if summary == "" {
		summary = fmt.Sprintf("CI/CD release #%d %s", release.ID, action)
	}
	var catalogID *uint
	if catalogRepo != nil && release.ServiceID > 0 {
		if id, err := catalogRepo.FindServiceIDByLinkRef(ctx, release.ProjectID, model.ServiceLinkCicdService, release.ServiceID); err == nil && id > 0 {
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
