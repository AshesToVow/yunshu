package cicd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

// --- Service CRUD ---

type ServiceListQuery struct {
	ProjectID   uint              `form:"project_id"`
	Keyword     string            `form:"keyword"`
	ServiceType string            `form:"service_type"`
	Page        int               `form:"page"`
	PageSize    int               `form:"page_size"`
	Actor       *auth.CurrentUser `form:"-"`
}

type ServiceItem struct {
	model.CicdService
	HasCiConfig     bool            `json:"has_ci_config"`
	DeployConfigCnt int             `json:"deploy_config_count"`
	LastBuildResult string          `json:"last_build_result,omitempty"`
	LastBuildAt     *time.Time      `json:"last_build_at,omitempty"`
	Access          *CicdAccessPerm `json:"access,omitempty"`
}

type ServiceUpsertRequest struct {
	ProjectID   uint   `json:"project_id"`
	Identifier  string `json:"identifier" binding:"required,max=128"`
	Name        string `json:"name" binding:"required,max=128"`
	ServiceType string `json:"service_type" binding:"required,max=32"`
	Owner       string `json:"owner" binding:"omitempty,max=64"`
	ProductLine string `json:"product_line" binding:"omitempty,max=128"`
	Remark      string `json:"remark" binding:"omitempty,max=512"`
	Status      *int   `json:"status" binding:"omitempty,oneof=0 1"`
	JenkinsJob  string `json:"jenkins_job" binding:"omitempty,max=256"`
}

func (s *Service) ListServices(ctx context.Context, q ServiceListQuery) (*pagination.Result[ServiceItem], error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.CicdService{}).Where("project_id = ?", q.ProjectID)
	unrestricted, ids, err := s.visibleCicdServiceScope(ctx, q.ProjectID, q.Actor)
	if err != nil {
		return nil, err
	}
	if !unrestricted {
		if len(ids) == 0 {
			return &pagination.Result[ServiceItem]{List: []ServiceItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
		}
		dbq = dbq.Where("id IN ?", ids)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("name LIKE ? OR identifier LIKE ?", like, like)
	}
	if st := strings.TrimSpace(q.ServiceType); st != "" {
		dbq = dbq.Where("service_type = ?", st)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdService
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	serviceIDs := make([]uint, len(rows))
	for i, row := range rows {
		serviceIDs[i] = row.ID
	}
	hasCI := map[uint]bool{}
	deployCnt := map[uint]int{}
	lastBuild := map[uint]model.CicdBuildRun{}
	if len(serviceIDs) > 0 {
		var ciIDs []uint
		_ = s.db.WithContext(ctx).Model(&model.CicdCiConfig{}).
			Where("service_id IN ?", serviceIDs).
			Distinct("service_id").
			Pluck("service_id", &ciIDs).Error
		for _, id := range ciIDs {
			hasCI[id] = true
		}
		type deployRow struct {
			ServiceID uint
			Cnt       int64
		}
		var deployRows []deployRow
		_ = s.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).
			Select("service_id, COUNT(*) AS cnt").
			Where("service_id IN ? AND status = 1", serviceIDs).
			Group("service_id").
			Scan(&deployRows).Error
		for _, d := range deployRows {
			deployCnt[d.ServiceID] = int(d.Cnt)
		}
		var builds []model.CicdBuildRun
		_ = s.db.WithContext(ctx).
			Where("service_id IN ?", serviceIDs).
			Order("id DESC").
			Find(&builds).Error
		for _, b := range builds {
			if _, ok := lastBuild[b.ServiceID]; !ok {
				lastBuild[b.ServiceID] = b
			}
		}
	}
	items := make([]ServiceItem, 0, len(rows))
	for _, row := range rows {
		item := ServiceItem{CicdService: row}
		item.HasCiConfig = hasCI[row.ID]
		item.DeployConfigCnt = deployCnt[row.ID]
		if b, ok := lastBuild[row.ID]; ok {
			item.LastBuildResult = b.BuildResult
			item.LastBuildAt = b.StartedAt
		}
		s.attachServiceAccess(ctx, q.ProjectID, q.Actor, &item)
		items = append(items, item)
	}
	return &pagination.Result[ServiceItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) attachServiceAccess(ctx context.Context, projectID uint, actor *auth.CurrentUser, item *ServiceItem) {
	if item == nil {
		return
	}
	perm, err := s.EffectiveCicdAccess(ctx, projectID, item.ID, actor)
	if err != nil || perm == nil {
		item.Access = &CicdAccessPerm{}
		return
	}
	p := *perm
	item.Access = &p
}

func (s *Service) GetService(ctx context.Context, projectID, serviceID uint, actor *auth.CurrentUser) (*ServiceItem, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	res, err := s.ListServices(ctx, ServiceListQuery{ProjectID: projectID, Page: 1, PageSize: 1, Actor: actor})
	if err != nil {
		return nil, err
	}
	item := ServiceItem{CicdService: *svc}
	for _, it := range res.List {
		if it.ID == serviceID {
			item.HasCiConfig = it.HasCiConfig
			item.DeployConfigCnt = it.DeployConfigCnt
			item.LastBuildResult = it.LastBuildResult
			item.LastBuildAt = it.LastBuildAt
			item.Access = it.Access
			break
		}
	}
	var ciCnt int64
	_ = s.db.WithContext(ctx).Model(&model.CicdCiConfig{}).Where("service_id = ?", serviceID).Count(&ciCnt).Error
	item.HasCiConfig = ciCnt > 0
	return &item, nil
}

func (s *Service) UpsertService(ctx context.Context, serviceID uint, req ServiceUpsertRequest) (*model.CicdService, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, constants.ErrBadRequestWithMsg("identifier required")
	}
	serviceType := strings.TrimSpace(req.ServiceType)
	if serviceType == model.CicdServiceTypeFrontend || serviceType == model.CicdServiceTypeBackend || serviceType == model.CicdServiceTypeMicro {
		// ok
	} else if serviceType == "micro" {
		serviceType = model.CicdServiceTypeMicro
	} else {
		return nil, constants.ErrBadRequestWithMsg("service_type must be frontend|backend|microservice")
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	jenkinsJob := strings.TrimSpace(req.JenkinsJob)
	if jenkinsJob == "" {
		jenkinsJob = fmt.Sprintf("cicd-p%d-%s", req.ProjectID, identifier)
	}
	var row model.CicdService
	if serviceID > 0 {
		if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", serviceID, req.ProjectID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFound
			}
			return nil, err
		}
	} else {
		var exists int64
		if err := s.db.WithContext(ctx).Model(&model.CicdService{}).
			Where("project_id = ? AND identifier = ?", req.ProjectID, identifier).Count(&exists).Error; err != nil {
			return nil, err
		}
		if exists > 0 {
			return nil, constants.ErrBadRequestWithMsg("identifier already exists in project")
		}
	}
	row.ProjectID = req.ProjectID
	row.Identifier = identifier
	row.Name = strings.TrimSpace(req.Name)
	row.ServiceType = serviceType
	row.Owner = strings.TrimSpace(req.Owner)
	row.ProductLine = strings.TrimSpace(req.ProductLine)
	row.Remark = strings.TrimSpace(req.Remark)
	row.Status = status
	row.JenkinsJob = jenkinsJob
	if serviceID > 0 {
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	syncCicdToServiceCatalog(ctx, s.db, &row)
	return &row, nil
}

func (s *Service) DeleteService(ctx context.Context, projectID, serviceID uint) error {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdCiConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdDeployConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdBuildRun{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdReleaseRun{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND project_id = ?", serviceID, projectID).Delete(&model.CicdService{}).Error
	})
}

func (s *Service) loadService(ctx context.Context, projectID, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", serviceID, projectID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}
