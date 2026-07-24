package alert

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/promapi"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// pingPromQL 为轻量即时查询，用于连通性检测（不依赖具体指标是否存在）。
const pingPromQL = "vector(1)"

type AlertDatasourceListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type AlertDatasourceUpsertRequest struct {
	ProjectID        uint   `json:"project_id" binding:"required"`
	Name             string `json:"name" binding:"required,max=128"`
	Type             string `json:"type" binding:"omitempty,max=32"`
	BaseURL          string `json:"base_url" binding:"required,max=512"`
	BearerToken      string `json:"bearer_token"`
	BasicUser        string `json:"basic_user" binding:"omitempty,max=128"`
	BasicPassword    string `json:"basic_password" binding:"omitempty,max=256"`
	SkipTLSVerify    *bool  `json:"skip_tls_verify"`
	Enabled          *bool  `json:"enabled"`
	Remark           string `json:"remark" binding:"omitempty,max=512"`
	ClearBearerToken bool   `json:"clear_bearer_token"`
	ClearBasicAuth   bool   `json:"clear_basic_auth"`
}

type AlertDatasourceItem struct {
	model.AlertDatasource
	ProjectName string `json:"project_name,omitempty" gorm:"column:project_name"`
}

type PromQueryRequest struct {
	Query string `json:"query" binding:"required"`
	Time  string `json:"time"`
}

type PromQueryRangeRequest struct {
	Query string `json:"query" binding:"required"`
	Start string `json:"start" binding:"required"`
	End   string `json:"end" binding:"required"`
	Step  string `json:"step" binding:"required"`
}

// DatasourcePingResult 数据源连通性检测结果（即使失败也 HTTP 200 返回本结构，便于前端展示）。
type DatasourcePingResult struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
}

type AlertDatasourceService struct {
	repo interfaces.AlertDatasourceRepository
}

func NewAlertDatasourceService(repo interfaces.AlertDatasourceRepository) *AlertDatasourceService {
	return &AlertDatasourceService{repo: repo}
}

func (s *AlertDatasourceService) mask(ds *model.AlertDatasource) {
	if strings.TrimSpace(ds.BearerToken) != "" {
		ds.BearerToken = "***"
	}
	if strings.TrimSpace(ds.BasicPassword) != "" {
		ds.BasicPassword = "***"
	}
}

func (s *AlertDatasourceService) List(ctx context.Context, q AlertDatasourceListQuery) ([]AlertDatasourceItem, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	rows, total, err := s.repo.ListWithProject(ctx, repository.AlertDatasourceListFilter{
		ProjectID: q.ProjectID,
		Keyword:   q.Keyword,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.datasource", "List", err)
	}
	list := make([]AlertDatasourceItem, len(rows))
	for i, row := range rows {
		list[i] = AlertDatasourceItem{AlertDatasource: row.AlertDatasource, ProjectName: row.ProjectName}
		s.mask(&list[i].AlertDatasource)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDatasourceService) Get(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, bizerrors.Pass(ctx, "alert.datasource", "Get", err)
	}
	s.mask(row)
	return row, nil
}

func (s *AlertDatasourceService) getRaw(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AlertDatasourceService) Create(ctx context.Context, req AlertDatasourceUpsertRequest) (*model.AlertDatasource, error) {
	t := strings.TrimSpace(req.Type)
	if t == "" {
		t = "prometheus"
	}
	if t != "prometheus" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg480bba83b97b)
	}
	if req.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg9a7f154a70af)
	}
	row := model.AlertDatasource{
		ProjectID:     req.ProjectID,
		Name:          strings.TrimSpace(req.Name),
		Type:          t,
		BaseURL:       strings.TrimSpace(req.BaseURL),
		BearerToken:   strings.TrimSpace(req.BearerToken),
		BasicUser:     strings.TrimSpace(req.BasicUser),
		BasicPassword: strings.TrimSpace(req.BasicPassword),
		SkipTLSVerify: req.SkipTLSVerify != nil && *req.SkipTLSVerify,
		Enabled:       req.Enabled == nil || *req.Enabled,
		Remark:        strings.TrimSpace(req.Remark),
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "Create", err)
	}
	s.mask(&row)
	return &row, nil
}

func (s *AlertDatasourceService) Update(ctx context.Context, id uint, req AlertDatasourceUpsertRequest) (*model.AlertDatasource, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, bizerrors.Pass(ctx, "alert.datasource", "Update", err)
	}
	if req.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg9a7f154a70af)
	}
	if req.ProjectID != row.ProjectID {
		row.ProjectID = req.ProjectID
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.BaseURL) != "" {
		row.BaseURL = strings.TrimSpace(req.BaseURL)
	}
	if req.SkipTLSVerify != nil {
		row.SkipTLSVerify = *req.SkipTLSVerify
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	row.Remark = strings.TrimSpace(req.Remark)
	if req.ClearBearerToken {
		row.BearerToken = ""
	} else if strings.TrimSpace(req.BearerToken) != "" && req.BearerToken != "***" {
		row.BearerToken = strings.TrimSpace(req.BearerToken)
	}
	if req.ClearBasicAuth {
		row.BasicUser = ""
		row.BasicPassword = ""
	} else {
		if strings.TrimSpace(req.BasicUser) != "" {
			row.BasicUser = strings.TrimSpace(req.BasicUser)
		}
		if strings.TrimSpace(req.BasicPassword) != "" && req.BasicPassword != "***" {
			row.BasicPassword = strings.TrimSpace(req.BasicPassword)
		}
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "Update", err)
	}
	s.mask(row)
	return row, nil
}

func (s *AlertDatasourceService) Delete(ctx context.Context, id uint) error {
	n, err := s.repo.Delete(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.datasource", "Delete", err)
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
	}
	return nil
}

// PrometheusClient 返回已启用的 Prometheus 客户端（含凭据明文，仅供服务端内部调用）。
func (s *AlertDatasourceService) PrometheusClient(ctx context.Context, id uint) (*promapi.Client, *model.AlertDatasource, error) {
	return s.clientFor(ctx, id)
}

func (s *AlertDatasourceService) clientFor(ctx context.Context, id uint) (*promapi.Client, *model.AlertDatasource, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, nil, bizerrors.Pass(ctx, "alert.datasource", "clientFor", err)
	}
	if !row.Enabled {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfa357d889ce0)
	}
	if row.Type != "prometheus" {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsg9a8a590cfc72)
	}
	return &promapi.Client{
		BaseURL:       row.BaseURL,
		BearerToken:   row.BearerToken,
		BasicUser:     row.BasicUser,
		BasicPassword: row.BasicPassword,
		SkipTLSVerify: row.SkipTLSVerify,
	}, row, nil
}

func (s *AlertDatasourceService) PromQuery(ctx context.Context, id uint, req PromQueryRequest) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "PromQuery", err)
	}
	qctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	body, _, err := cli.QueryInstant(qctx, strings.TrimSpace(req.Query), strings.TrimSpace(req.Time))
	return body, bizerrors.Pass(ctx, "alert.datasource", "PromQuery", err)
}

func (s *AlertDatasourceService) PromQueryRange(ctx context.Context, id uint, req PromQueryRangeRequest) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "PromQueryRange", err)
	}
	qctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	body, _, err := cli.QueryRange(qctx, strings.TrimSpace(req.Query), strings.TrimSpace(req.Start), strings.TrimSpace(req.End), strings.TrimSpace(req.Step))
	return body, bizerrors.Pass(ctx, "alert.datasource", "PromQueryRange", err)
}

func (s *AlertDatasourceService) PrometheusActiveAlerts(ctx context.Context, id uint) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "PrometheusActiveAlerts", err)
	}
	qctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	body, _, err := cli.ActiveAlerts(qctx)
	return body, bizerrors.Pass(ctx, "alert.datasource", "PrometheusActiveAlerts", err)
}

func (s *AlertDatasourceService) PingDatasource(ctx context.Context, id uint) (*DatasourcePingResult, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, bizerrors.Pass(ctx, "alert.datasource", "PingDatasource", err)
	}
	t := strings.TrimSpace(row.Type)
	if t == "" {
		t = "prometheus"
	}
	if t != "prometheus" {
		return &DatasourcePingResult{OK: false, Message: "非 prometheus 类型暂不支持连通性检测", LatencyMs: 0}, nil
	}
	if strings.TrimSpace(row.BaseURL) == "" {
		return &DatasourcePingResult{OK: false, Message: "base_url 为空", LatencyMs: 0}, nil
	}
	cli := &promapi.Client{
		BaseURL:       row.BaseURL,
		BearerToken:   row.BearerToken,
		BasicUser:     row.BasicUser,
		BasicPassword: row.BasicPassword,
		SkipTLSVerify: row.SkipTLSVerify,
	}
	qctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	start := time.Now()
	body, _, err := cli.QueryInstant(qctx, pingPromQL, "")
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &DatasourcePingResult{OK: false, Message: err.Error(), LatencyMs: latency}, nil
	}
	if !promapi.QueryResponseStatusSuccess(body) {
		return &DatasourcePingResult{OK: false, Message: "Prometheus 返回非 success 状态", LatencyMs: latency}, nil
	}
	return &DatasourcePingResult{OK: true, Message: "ok", LatencyMs: latency}, nil
}
