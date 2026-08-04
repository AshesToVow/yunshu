package cicd

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// syncCicdToServiceCatalog 将 CI/CD 服务同步到统一服务目录并绑定 cicd_service link。
func syncCicdToServiceCatalog(ctx context.Context, db *gorm.DB, cicd *model.CicdService) {
	if db == nil || cicd == nil || cicd.ProjectID == 0 || cicd.Identifier == "" {
		return
	}
	repo := repository.NewServiceCatalogRepository(db)
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
