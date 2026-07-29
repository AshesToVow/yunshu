package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type ServiceCatalogRepo interface {
	GetByID(ctx context.Context, projectID, id uint) (*model.ServiceCatalog, error)
	GetByIdentifier(ctx context.Context, projectID uint, identifier string) (*model.ServiceCatalog, error)
	List(ctx context.Context, p ServiceCatalogListParams) ([]model.ServiceCatalog, int64, error)
	Create(ctx context.Context, row *model.ServiceCatalog) error
	Save(ctx context.Context, row *model.ServiceCatalog) error
	Delete(ctx context.Context, projectID, id uint) error
	UpsertByIdentifier(ctx context.Context, row *model.ServiceCatalog) error
	ListLinks(ctx context.Context, serviceID uint) ([]model.ServiceLink, error)
	AddLink(ctx context.Context, link *model.ServiceLink) error
	DeleteLink(ctx context.Context, serviceID, linkID uint) error
	FindLink(ctx context.Context, serviceID uint, linkType string, refID *uint, refKey string) (*model.ServiceLink, error)
}

type ServiceCatalogListParams struct {
	ProjectID uint
	Keyword   string
	Status    *int
	Page      int
	PageSize  int
}

type ServiceCatalogRepository struct{ db *gorm.DB }

func NewServiceCatalogRepository(db *gorm.DB) ServiceCatalogRepo {
	return &ServiceCatalogRepository{db: db}
}

func (r *ServiceCatalogRepository) GetByID(ctx context.Context, projectID, id uint) (*model.ServiceCatalog, error) {
	var row model.ServiceCatalog
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", id, projectID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ServiceCatalogRepository) GetByIdentifier(ctx context.Context, projectID uint, identifier string) (*model.ServiceCatalog, error) {
	var row model.ServiceCatalog
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND identifier = ?", projectID, identifier).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ServiceCatalogRepository) List(ctx context.Context, p ServiceCatalogListParams) ([]model.ServiceCatalog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ServiceCatalog{}).Where("project_id = ?", p.ProjectID)
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("identifier LIKE ? OR name LIKE ? OR owner LIKE ?", like, like, like)
	}
	if p.Status != nil {
		q = q.Where("status = ?", *p.Status)
	}
	var list []model.ServiceCatalog
	total, err := listWithPagination(q, p.Page, p.PageSize, "id DESC", &list)
	return list, total, err
}

func (r *ServiceCatalogRepository) Create(ctx context.Context, row *model.ServiceCatalog) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ServiceCatalogRepository) Save(ctx context.Context, row *model.ServiceCatalog) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *ServiceCatalogRepository) Delete(ctx context.Context, projectID, id uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND project_id = ?", id, projectID).
		Delete(&model.ServiceCatalog{}).Error
}

func (r *ServiceCatalogRepository) UpsertByIdentifier(ctx context.Context, row *model.ServiceCatalog) error {
	existing, err := r.GetByIdentifier(ctx, row.ProjectID, row.Identifier)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.Create(ctx, row)
		}
		return err
	}
	existing.Name = row.Name
	existing.Owner = row.Owner
	existing.ProductLine = row.ProductLine
	if row.Status != 0 {
		existing.Status = row.Status
	}
	if row.Criticality != "" {
		existing.Criticality = row.Criticality
	}
	*row = *existing
	return r.Save(ctx, existing)
}

func (r *ServiceCatalogRepository) ListLinks(ctx context.Context, serviceID uint) ([]model.ServiceLink, error) {
	var list []model.ServiceLink
	err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *ServiceCatalogRepository) AddLink(ctx context.Context, link *model.ServiceLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *ServiceCatalogRepository) DeleteLink(ctx context.Context, serviceID, linkID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND service_id = ?", linkID, serviceID).
		Delete(&model.ServiceLink{}).Error
}

func (r *ServiceCatalogRepository) FindLink(
	ctx context.Context,
	serviceID uint,
	linkType string,
	refID *uint,
	refKey string,
) (*model.ServiceLink, error) {
	q := r.db.WithContext(ctx).Model(&model.ServiceLink{}).
		Where("service_id = ? AND link_type = ?", serviceID, linkType)
	if refID != nil {
		q = q.Where("ref_id = ?", *refID)
	} else {
		q = q.Where("ref_id IS NULL")
	}
	q = q.Where("ref_key = ?", refKey)
	var row model.ServiceLink
	if err := q.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type ChangeEventRepo interface {
	List(ctx context.Context, p ChangeEventListParams) ([]model.ChangeEvent, int64, error)
	Create(ctx context.Context, row *model.ChangeEvent) error
}

type ChangeEventListParams struct {
	ProjectID uint
	ServiceID *uint
	Source    string
	Status    string
	From      *time.Time
	To        *time.Time
	Keyword   string
	Page      int
	PageSize  int
}

type ChangeEventRepository struct{ db *gorm.DB }

func NewChangeEventRepository(db *gorm.DB) ChangeEventRepo {
	return &ChangeEventRepository{db: db}
}

func (r *ChangeEventRepository) Create(ctx context.Context, row *model.ChangeEvent) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ChangeEventRepository) List(ctx context.Context, p ChangeEventListParams) ([]model.ChangeEvent, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ChangeEvent{}).Where("project_id = ?", p.ProjectID)
	if p.ServiceID != nil && *p.ServiceID > 0 {
		q = q.Where("service_id = ?", *p.ServiceID)
	}
	if s := strings.TrimSpace(p.Source); s != "" {
		q = q.Where("source = ?", s)
	}
	if s := strings.TrimSpace(p.Status); s != "" {
		q = q.Where("status = ?", s)
	}
	if p.From != nil {
		q = q.Where("started_at >= ?", *p.From)
	}
	if p.To != nil {
		q = q.Where("started_at <= ?", *p.To)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("summary LIKE ? OR action LIKE ?", like, like)
	}
	var list []model.ChangeEvent
	total, err := listWithPagination(q, p.Page, p.PageSize, "started_at DESC, id DESC", &list)
	return list, total, err
}
