package project

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type ServiceCatalogService struct {
	repo        interfaces.ServiceCatalogRepository
	projectRepo interfaces.ProjectRepository
	db          *gorm.DB
}

func NewServiceCatalogService(
	repo interfaces.ServiceCatalogRepository,
	projectRepo interfaces.ProjectRepository,
	db *gorm.DB,
) *ServiceCatalogService {
	return &ServiceCatalogService{repo: repo, projectRepo: projectRepo, db: db}
}

type ServiceCatalogListQuery struct {
	ProjectID uint   `form:"-"`
	Keyword   string `form:"keyword"`
	Status    *int   `form:"status"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type ServiceCatalogUpsertRequest struct {
	ProjectID   uint   `json:"-"`
	ID          uint   `json:"id"`
	Identifier  string `json:"identifier" binding:"required,max=128"`
	Name        string `json:"name" binding:"required,max=128"`
	Owner       string `json:"owner" binding:"omitempty,max=64"`
	ProductLine string `json:"product_line" binding:"omitempty,max=128"`
	Criticality string `json:"criticality" binding:"omitempty,max=32"`
	Status      *int   `json:"status"`
	Remark      string `json:"remark" binding:"omitempty,max=512"`
}

type ServiceLinkRequest struct {
	ProjectID uint   `json:"-"`
	ServiceID uint   `json:"-"`
	LinkType  string `json:"link_type" binding:"required,max=32"`
	RefID     *uint  `json:"ref_id"`
	RefKey    string `json:"ref_key" binding:"omitempty,max=256"`
}

type ServiceCatalogItem struct {
	model.ServiceCatalog
	Links []model.ServiceLink `json:"links,omitempty"`
}

func (s *ServiceCatalogService) ensureProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project_id required")
	}
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ServiceCatalogService) List(ctx context.Context, q ServiceCatalogListQuery) (*pagination.Result[ServiceCatalogItem], error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.List(ctx, repository.ServiceCatalogListParams{
		ProjectID: q.ProjectID,
		Keyword:   q.Keyword,
		Status:    q.Status,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]ServiceCatalogItem, 0, len(list))
	for _, row := range list {
		links, _ := s.repo.ListLinks(ctx, row.ID)
		items = append(items, ServiceCatalogItem{ServiceCatalog: row, Links: links})
	}
	return &pagination.Result[ServiceCatalogItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ServiceCatalogService) Upsert(ctx context.Context, req ServiceCatalogUpsertRequest) (*ServiceCatalogItem, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	identifier := strings.TrimSpace(req.Identifier)
	name := strings.TrimSpace(req.Name)
	if identifier == "" || name == "" {
		return nil, constants.ErrBadRequestWithMsg("identifier and name required")
	}
	criticality := strings.TrimSpace(req.Criticality)
	if criticality == "" {
		criticality = "normal"
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	var row *model.ServiceCatalog
	var err error
	if req.ID > 0 {
		row, err = s.repo.GetByID(ctx, req.ProjectID, req.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, constants.ErrNotFound
			}
			return nil, err
		}
		row.Identifier = identifier
		row.Name = name
		row.Owner = strings.TrimSpace(req.Owner)
		row.ProductLine = strings.TrimSpace(req.ProductLine)
		row.Criticality = criticality
		row.Status = status
		row.Remark = strings.TrimSpace(req.Remark)
		if err := s.repo.Save(ctx, row); err != nil {
			return nil, err
		}
	} else {
		row = &model.ServiceCatalog{
			ProjectID:   req.ProjectID,
			Identifier:  identifier,
			Name:        name,
			Owner:       strings.TrimSpace(req.Owner),
			ProductLine: strings.TrimSpace(req.ProductLine),
			Criticality: criticality,
			Status:      status,
			Remark:      strings.TrimSpace(req.Remark),
		}
		if err := s.repo.UpsertByIdentifier(ctx, row); err != nil {
			return nil, err
		}
	}
	links, _ := s.repo.ListLinks(ctx, row.ID)
	return &ServiceCatalogItem{ServiceCatalog: *row, Links: links}, nil
}

func (s *ServiceCatalogService) Delete(ctx context.Context, projectID, id uint) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, projectID, id)
}

func (s *ServiceCatalogService) AddLink(ctx context.Context, req ServiceLinkRequest) (*model.ServiceLink, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, req.ProjectID, req.ServiceID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	linkType := strings.TrimSpace(req.LinkType)
	if !validLinkType(linkType) {
		return nil, constants.ErrBadRequestWithMsg("invalid link_type")
	}
	refKey := strings.TrimSpace(req.RefKey)
	if req.RefID == nil && refKey == "" {
		return nil, constants.ErrBadRequestWithMsg("ref_id or ref_key required")
	}
	if existing, err := s.repo.FindLink(ctx, req.ServiceID, linkType, req.RefID, refKey); err == nil && existing != nil {
		return existing, nil
	}
	link := &model.ServiceLink{
		ServiceID: req.ServiceID,
		LinkType:  linkType,
		RefID:     req.RefID,
		RefKey:    refKey,
	}
	if err := s.repo.AddLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *ServiceCatalogService) DeleteLink(ctx context.Context, projectID, serviceID, linkID uint) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	if _, err := s.repo.GetByID(ctx, projectID, serviceID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFound
		}
		return err
	}
	return s.repo.DeleteLink(ctx, serviceID, linkID)
}

// UpsertFromCicd 由 CI/CD 服务写入时同步目录并绑定 cicd_service link。
func (s *ServiceCatalogService) UpsertFromCicd(ctx context.Context, cicd *model.CicdService) {
	if s == nil || s.repo == nil || cicd == nil || cicd.ProjectID == 0 || cicd.Identifier == "" {
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
	if err := s.repo.UpsertByIdentifier(ctx, row); err != nil || row.ID == 0 {
		return
	}
	refID := cicd.ID
	if _, err := s.repo.FindLink(ctx, row.ID, model.ServiceLinkCicdService, &refID, ""); err == nil {
		return
	}
	_ = s.repo.AddLink(ctx, &model.ServiceLink{
		ServiceID: row.ID,
		LinkType:  model.ServiceLinkCicdService,
		RefID:     &refID,
	})
}

func validLinkType(t string) bool {
	switch t {
	case model.ServiceLinkCicdService,
		model.ServiceLinkCmdbService,
		model.ServiceLinkLogSource,
		model.ServiceLinkK8sWorkload,
		model.ServiceLinkAlertMonitorRule,
		model.ServiceLinkDbInstance:
		return true
	default:
		return false
	}
}

type ChangeEventService struct {
	repo        interfaces.ChangeEventRepository
	projectRepo interfaces.ProjectRepository
	db          *gorm.DB
}

func NewChangeEventService(
	repo interfaces.ChangeEventRepository,
	projectRepo interfaces.ProjectRepository,
	db *gorm.DB,
) *ChangeEventService {
	return &ChangeEventService{repo: repo, projectRepo: projectRepo, db: db}
}

type ChangeEventListQuery struct {
	ProjectID uint   `form:"-"`
	ServiceID *uint  `form:"service_id"`
	Source    string `form:"source"`
	Status    string `form:"status"`
	Keyword   string `form:"keyword"`
	From      string `form:"from"`
	To        string `form:"to"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (s *ChangeEventService) List(ctx context.Context, q ChangeEventListQuery) (*pagination.Result[model.ChangeEvent], error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	if _, err := s.projectRepo.GetByID(ctx, q.ProjectID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	var from, to *time.Time
	if strings.TrimSpace(q.From) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(q.From)); err == nil {
			from = &t
		}
	}
	if strings.TrimSpace(q.To) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(q.To)); err == nil {
			to = &t
		}
	}
	list, total, err := s.repo.List(ctx, repository.ChangeEventListParams{
		ProjectID: q.ProjectID,
		ServiceID: q.ServiceID,
		Source:    q.Source,
		Status:    q.Status,
		From:      from,
		To:        to,
		Keyword:   q.Keyword,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.ChangeEvent]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}
