package alert

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/consulclient"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
)

type AlertConsulEndpointListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type AlertConsulEndpointUpsertRequest struct {
	ProjectID  uint   `json:"project_id" binding:"required"`
	Name       string `json:"name" binding:"required,max=128"`
	Address    string `json:"address" binding:"required,max=512"`
	Token      string `json:"token"`
	Datacenter string `json:"datacenter" binding:"omitempty,max=64"`
	ServiceTag string `json:"service_tag" binding:"omitempty,max=128"`
	Enabled    *bool  `json:"enabled"`
	Remark     string `json:"remark" binding:"omitempty,max=512"`
	ClearToken bool   `json:"clear_token"`
}

type AlertMonitorObjectListQuery struct {
	ProjectID    uint   `form:"project_id"`
	EndpointID   uint   `form:"endpoint_id"`
	ExporterRole string `form:"exporter_role"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type AlertConsulSyncResult struct {
	EndpointID uint   `json:"endpoint_id"`
	Upserted   int    `json:"upserted"`
	Removed    int    `json:"removed"`
	Message    string `json:"message"`
}

type AlertConsulService struct {
	repo interfaces.AlertConsulRepository
}

func NewAlertConsulService(repo interfaces.AlertConsulRepository) *AlertConsulService {
	return &AlertConsulService{repo: repo}
}

func (s *AlertConsulService) mask(ep *model.AlertConsulEndpoint) {
	if ep != nil && strings.TrimSpace(ep.Token) != "" {
		ep.Token = "***"
	}
}

func (s *AlertConsulService) ListEndpoints(ctx context.Context, q AlertConsulEndpointListQuery) ([]model.AlertConsulEndpoint, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.ListEndpoints(ctx, repository.AlertConsulEndpointListFilter{
		ProjectID: q.ProjectID,
		Keyword:   q.Keyword,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.consul", "ListEndpoints", err)
	}
	for i := range list {
		s.mask(&list[i])
	}
	return list, total, page, pageSize, nil
}

func (s *AlertConsulService) CreateEndpoint(ctx context.Context, req AlertConsulEndpointUpsertRequest) (*model.AlertConsulEndpoint, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	tag := strings.TrimSpace(req.ServiceTag)
	if tag == "" {
		tag = "yunshu-metrics"
	}
	row := &model.AlertConsulEndpoint{
		ProjectID:  req.ProjectID,
		Name:       strings.TrimSpace(req.Name),
		Address:    strings.TrimSpace(req.Address),
		Token:      strings.TrimSpace(req.Token),
		Datacenter: strings.TrimSpace(req.Datacenter),
		ServiceTag: tag,
		Enabled:    enabled,
		Remark:     strings.TrimSpace(req.Remark),
	}
	if err := s.repo.CreateEndpoint(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "CreateEndpoint", err)
	}
	s.mask(row)
	return row, nil
}

func (s *AlertConsulService) UpdateEndpoint(ctx context.Context, id uint, req AlertConsulEndpointUpsertRequest) (*model.AlertConsulEndpoint, error) {
	row, err := s.repo.GetEndpointByID(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "UpdateEndpoint", err)
	}
	row.ProjectID = req.ProjectID
	row.Name = strings.TrimSpace(req.Name)
	row.Address = strings.TrimSpace(req.Address)
	row.Datacenter = strings.TrimSpace(req.Datacenter)
	tag := strings.TrimSpace(req.ServiceTag)
	if tag == "" {
		tag = "yunshu-metrics"
	}
	row.ServiceTag = tag
	row.Remark = strings.TrimSpace(req.Remark)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.ClearToken {
		row.Token = ""
	} else if t := strings.TrimSpace(req.Token); t != "" && t != "***" {
		row.Token = t
	}
	if err := s.repo.SaveEndpoint(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "UpdateEndpoint", err)
	}
	s.mask(row)
	return row, nil
}

func (s *AlertConsulService) DeleteEndpoint(ctx context.Context, id uint) error {
	if err := s.repo.DeleteEndpointWithObjects(ctx, id); err != nil {
		return bizerrors.Pass(ctx, "alert.consul", "DeleteEndpoint", err)
	}
	return nil
}

func (s *AlertConsulService) PingEndpoint(ctx context.Context, id uint) error {
	ep, err := s.loadEndpointRaw(ctx, id)
	if err != nil {
		return err
	}
	cli := &consulclient.Client{Address: ep.Address, Token: ep.Token, Datacenter: ep.Datacenter}
	if err := cli.Ping(ctx); err != nil {
		return constants.ErrBadRequestWithMsg("Consul 连通失败: " + err.Error())
	}
	return nil
}

func (s *AlertConsulService) loadEndpointRaw(ctx context.Context, id uint) (*model.AlertConsulEndpoint, error) {
	row, err := s.repo.GetEndpointByID(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "loadEndpoint", err)
	}
	return row, nil
}

func (s *AlertConsulService) ListObjects(ctx context.Context, q AlertMonitorObjectListQuery) ([]model.AlertMonitorObject, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.ListObjects(ctx, repository.AlertMonitorObjectListFilter{
		ProjectID:    q.ProjectID,
		EndpointID:   q.EndpointID,
		ExporterRole: q.ExporterRole,
		Keyword:      q.Keyword,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.consul", "ListObjects", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertConsulService) SyncEndpoint(ctx context.Context, id uint) (*AlertConsulSyncResult, error) {
	ep, err := s.loadEndpointRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ep.Enabled {
		return nil, constants.ErrBadRequestWithMsg("Consul 端点已停用")
	}
	cli := &consulclient.Client{Address: ep.Address, Token: ep.Token, Datacenter: ep.Datacenter}
	names, err := cli.ListServiceNames(ctx)
	if err != nil {
		_ = s.repo.UpdateEndpointMeta(ctx, ep.ID, map[string]any{
			"last_error": truncateConsulErr(err.Error()),
		})
		return nil, constants.ErrBadRequestWithMsg("列出 Consul 服务失败: " + err.Error())
	}

	now := time.Now().UTC()
	upserted := 0
	removed := 0
	keepKeys := map[string]struct{}{}

	err = s.repo.Transaction(ctx, func(txRepo repository.AlertConsulRepo) error {
		for svcName := range names {
			instances, err := cli.ListServiceInstances(ctx, svcName)
			if err != nil {
				continue
			}
			for _, inst := range instances {
				if !consulclient.HasTag(inst.ServiceTags, ep.ServiceTag) {
					continue
				}
				meta := inst.ServiceMeta
				if meta == nil {
					meta = map[string]string{}
				}
				tagsJSON, _ := json.Marshal(inst.ServiceTags)
				metaJSON, _ := json.Marshal(meta)
				sid := strings.TrimSpace(inst.ServiceID)
				if sid == "" {
					sid = strings.TrimSpace(inst.ServiceName) + "@" + strings.TrimSpace(inst.Address)
				}
				sname := strings.TrimSpace(inst.ServiceName)
				key := sname + "\x00" + sid
				keepKeys[key] = struct{}{}

				row := &model.AlertMonitorObject{
					EndpointID:    ep.ID,
					ProjectID:     ep.ProjectID,
					ServiceName:   sname,
					ServiceID:     sid,
					Node:          strings.TrimSpace(inst.Node),
					Address:       strings.TrimSpace(inst.Address),
					Port:          inst.ServicePort,
					TagsJSON:      string(tagsJSON),
					MetaJSON:      string(metaJSON),
					ExporterRole:  strings.TrimSpace(meta["exporter_role"]),
					YunshuProject: strings.TrimSpace(firstMeta(meta, "yunshu_project", "project")),
					Health:        consulclient.AggregateHealth(inst.Checks),
					ProbeURL:      strings.TrimSpace(firstMeta(meta, "probe_url", "probe_host")),
					SyncedAt:      now,
				}
				if err := txRepo.UpsertMonitorObject(ctx, row); err != nil {
					return err
				}
				upserted++
			}
		}

		existing, err := txRepo.ListObjectsByEndpoint(ctx, ep.ID)
		if err != nil {
			return err
		}
		for i := range existing {
			old := &existing[i]
			key := old.ServiceName + "\x00" + old.ServiceID
			if _, ok := keepKeys[key]; !ok {
				if err := txRepo.DeleteObject(ctx, old); err != nil {
					return err
				}
				removed++
			}
		}
		return nil
	})
	if err != nil {
		_ = s.repo.UpdateEndpointMeta(ctx, ep.ID, map[string]any{
			"last_error": truncateConsulErr(err.Error()),
		})
		return nil, constants.ErrBadRequestWithMsg("同步 Consul 失败: " + err.Error())
	}

	_ = s.repo.UpdateEndpointMeta(ctx, ep.ID, map[string]any{
		"last_sync_at": now,
		"last_error":   "",
	})

	return &AlertConsulSyncResult{
		EndpointID: ep.ID,
		Upserted:   upserted,
		Removed:    removed,
		Message:    "ok",
	}, nil
}

func firstMeta(meta map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(meta[k]); v != "" {
			return v
		}
	}
	return ""
}

func truncateConsulErr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}
