package cicd

import (
	"context"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
)

// syncCicdToServiceCatalog 将 CI/CD 服务同步到统一服务目录并绑定 cicd_service link。
func syncCicdToServiceCatalog(ctx context.Context, repo interfaces.ServiceCatalogRepository, cicd *model.CicdService) {
	if repo == nil || cicd == nil || cicd.ProjectID == 0 || cicd.Identifier == "" {
		return
	}
	row := &model.ServiceCatalog{
		ProjectID:   cicd.ProjectID,
		Identifier:  cicd.Identifier,
		Name:        cicd.Name,
		Owner:       cicd.Owner,
		ProductLine: cicd.ProductLine,
		Criticality: "normal",
		Status:      cicd.Status,
	}
	if row.Status == 0 {
		row.Status = 1
	}
	if err := repo.UpsertByIdentifier(ctx, row); err != nil || row.ID == 0 {
		return
	}
	refID := cicd.ID
	if _, err := repo.FindLink(ctx, row.ID, model.ServiceLinkCicdService, &refID, ""); err == nil {
		return
	}
	_ = repo.AddLink(ctx, &model.ServiceLink{
		ServiceID: row.ID,
		LinkType:  model.ServiceLinkCicdService,
		RefID:     &refID,
	})
}
