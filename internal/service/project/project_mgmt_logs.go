package project

import (
	"context"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type ServiceItem struct {
	ID        uint    `json:"id"`
	ServerID  uint    `json:"server_id"`
	Name      string  `json:"name"`
	Env       *string `json:"env"`
	Labels    *string `json:"labels"`
	Remark    *string `json:"remark"`
	Status    int     `json:"status"`
	CreatedAt string  `json:"created_at"`
}

func toServiceItem(it model.Service) ServiceItem {
	return ServiceItem{
		ID:        it.ID,
		ServerID:  it.ServerID,
		Name:      it.Name,
		Env:       it.Env,
		Labels:    it.Labels,
		Remark:    it.Remark,
		Status:    it.Status,
		CreatedAt: it.CreatedAt.Format(time.RFC3339),
	}
}

type ServiceListQuery struct {
	ProjectID uint   `form:"project_id" binding:"required"`
	ServerID  *uint  `form:"server_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (s *ProjectMgmtService) ListServices(ctx context.Context, q ServiceListQuery) (*pagination.Result[ServiceItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.serviceRepo.List(ctx, repository.ServiceListParams{
		ProjectID: q.ProjectID,
		ServerID:  q.ServerID,
		Keyword:   strings.TrimSpace(q.Keyword),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "project", "ListServices", err)
	}
	out := make([]ServiceItem, 0, len(list))
	for _, it := range list {
		out = append(out, toServiceItem(it))
	}
	return &pagination.Result[ServiceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type ServiceUpsertRequest struct {
	ID       *uint   `json:"id"`
	ServerID uint    `json:"server_id" binding:"required"`
	Name     string  `json:"name" binding:"required"`
	Env      *string `json:"env"`
	Labels   *string `json:"labels"`
	Remark   *string `json:"remark"`
	Status   int     `json:"status"`
}

// UpsertService 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertService(ctx context.Context, req ServiceUpsertRequest) (*ServiceItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	var it *model.Service
	var err error
	if req.ID != nil && *req.ID > 0 {
		it, err = s.serviceRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgac7e51a53391)
			}
			return nil, bizerrors.Pass(ctx, "project", "UpsertService", err)
		}
	} else {
		it = &model.Service{}
	}
	it.ServerID = req.ServerID
	it.Name = strings.TrimSpace(req.Name)
	it.Env = req.Env
	it.Labels = req.Labels
	it.Remark = req.Remark
	it.Status = status
	if it.ID == 0 {
		if err := s.serviceRepo.Create(ctx, it); err != nil {
			return nil, bizerrors.Pass(ctx, "project", "UpsertService", err)
		}
	} else {
		if err := s.serviceRepo.Save(ctx, it); err != nil {
			return nil, bizerrors.Pass(ctx, "project", "UpsertService", err)
		}
	}
	out := toServiceItem(*it)
	return &out, nil
}

// DeleteService 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteService(ctx context.Context, id uint) error {
	return s.serviceRepo.DeleteByID(ctx, id)
}

type LogSourceItem struct {
	ID            uint    `json:"id"`
	ServiceID     uint    `json:"service_id"`
	LogType       string  `json:"log_type"`
	Path          string  `json:"path"`
	Encoding      *string `json:"encoding"`
	Timezone      *string `json:"timezone"`
	MultilineRule *string `json:"multiline_rule"`
	IncludeRegex  *string `json:"include_regex"`
	ExcludeRegex  *string `json:"exclude_regex"`
	Status        int     `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

func toLogSourceItem(it model.ServiceLogSource) LogSourceItem {
	return LogSourceItem{
		ID:            it.ID,
		ServiceID:     it.ServiceID,
		LogType:       it.LogType,
		Path:          it.Path,
		Encoding:      it.Encoding,
		Timezone:      it.Timezone,
		MultilineRule: it.MultilineRule,
		IncludeRegex:  it.IncludeRegex,
		ExcludeRegex:  it.ExcludeRegex,
		Status:        it.Status,
		CreatedAt:     it.CreatedAt.Format(time.RFC3339),
	}
}

type LogSourceListQuery struct {
	ProjectID uint  `form:"project_id" binding:"required"`
	ServiceID *uint `form:"service_id"`
	Page      int   `form:"page"`
	PageSize  int   `form:"page_size"`
}

// ListLogSources 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListLogSources(ctx context.Context, q LogSourceListQuery) (*pagination.Result[LogSourceItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.logRepo.List(ctx, repository.LogSourceListParams{ProjectID: q.ProjectID, ServiceID: q.ServiceID, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "project", "ListLogSources", err)
	}
	out := make([]LogSourceItem, 0, len(list))
	for _, it := range list {
		out = append(out, toLogSourceItem(it))
	}
	return &pagination.Result[LogSourceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type LogSourceUpsertRequest struct {
	ID            *uint   `json:"id"`
	ServiceID     uint    `json:"service_id" binding:"required"`
	LogType       string  `json:"log_type"`
	Path          string  `json:"path" binding:"required"`
	Encoding      *string `json:"encoding"`
	Timezone      *string `json:"timezone"`
	MultilineRule *string `json:"multiline_rule"`
	IncludeRegex  *string `json:"include_regex"`
	ExcludeRegex  *string `json:"exclude_regex"`
	Status        int     `json:"status"`
}

// UpsertLogSource 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertLogSource(ctx context.Context, req LogSourceUpsertRequest) (*LogSourceItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	logType := strings.TrimSpace(req.LogType)
	if logType == "" {
		logType = "file"
	}
	var it *model.ServiceLogSource
	var err error
	if req.ID != nil && *req.ID > 0 {
		it, err = s.logRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg9d63941807e2)
			}
			return nil, bizerrors.Pass(ctx, "project", "UpsertLogSource", err)
		}
	} else {
		it = &model.ServiceLogSource{}
	}
	it.ServiceID = req.ServiceID
	it.LogType = logType
	it.Path = strings.TrimSpace(req.Path)
	it.Encoding = req.Encoding
	it.Timezone = req.Timezone
	it.MultilineRule = req.MultilineRule
	it.IncludeRegex = req.IncludeRegex
	it.ExcludeRegex = req.ExcludeRegex
	it.Status = status
	if it.ID == 0 {
		if err := s.logRepo.Create(ctx, it); err != nil {
			return nil, bizerrors.Pass(ctx, "project", "UpsertLogSource", err)
		}
	} else {
		if err := s.logRepo.Save(ctx, it); err != nil {
			return nil, bizerrors.Pass(ctx, "project", "UpsertLogSource", err)
		}
	}
	out := toLogSourceItem(*it)
	return &out, nil
}

// DeleteLogSource 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteLogSource(ctx context.Context, id uint) error {
	return s.logRepo.DeleteByID(ctx, id)
}

// ValidateLogSearchFilters 校验 ES 检索可选过滤条件属于当前项目。
func (s *ProjectMgmtService) ValidateLogSearchFilters(ctx context.Context, projectID uint, serverID, logSourceID *uint) error {
	if projectID == 0 {
		return constants.ErrProjectIDRequired
	}
	if serverID != nil && *serverID > 0 {
		sv, err := s.serverRepo.GetByID(ctx, *serverID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return constants.ErrLogSourceServerNotFound
			}
			return bizerrors.Pass(ctx, "project", "ValidateLogSearchFilters", err)
		}
		if sv.ProjectID != projectID {
			return constants.ErrServerNotInCurrentProject
		}
	}
	if logSourceID != nil && *logSourceID > 0 {
		if _, err := s.logRepo.GetByIDInProject(ctx, projectID, *logSourceID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return constants.ErrNotFoundWithMsg(constants.ErrMsg9d63941807e2)
			}
			return bizerrors.Pass(ctx, "project", "ValidateLogSearchFilters", err)
		}
	}
	return nil
}
