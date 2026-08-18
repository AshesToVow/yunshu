package alert

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/consulclient"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
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
	db *gorm.DB
}

func NewAlertConsulService(db *gorm.DB) *AlertConsulService {
	return &AlertConsulService{db: db}
}

func (s *AlertConsulService) mask(ep *model.AlertConsulEndpoint) {
	if ep != nil && strings.TrimSpace(ep.Token) != "" {
		ep.Token = "***"
	}
}

func (s *AlertConsulService) ListEndpoints(ctx context.Context, q AlertConsulEndpointListQuery) ([]model.AlertConsulEndpoint, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AlertConsulEndpoint{})
	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("name LIKE ? OR address LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.consul", "ListEndpoints", err)
	}
	var list []model.AlertConsulEndpoint
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "CreateEndpoint", err)
	}
	s.mask(row)
	return row, nil
}

func (s *AlertConsulService) UpdateEndpoint(ctx context.Context, id uint, req AlertConsulEndpointUpsertRequest) (*model.AlertConsulEndpoint, error) {
	var row model.AlertConsulEndpoint
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "UpdateEndpoint", err)
	}
	s.mask(&row)
	return &row, nil
}

func (s *AlertConsulService) DeleteEndpoint(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("endpoint_id = ?", id).Delete(&model.AlertMonitorObject{}).Error; err != nil {
			return bizerrors.Pass(ctx, "alert.consul", "DeleteEndpoint.objects", err)
		}
		if err := tx.Delete(&model.AlertConsulEndpoint{}, id).Error; err != nil {
			return bizerrors.Pass(ctx, "alert.consul", "DeleteEndpoint", err)
		}
		return nil
	})
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
	var row model.AlertConsulEndpoint
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.consul", "loadEndpoint", err)
	}
	return &row, nil
}

func (s *AlertConsulService) ListObjects(ctx context.Context, q AlertMonitorObjectListQuery) ([]model.AlertMonitorObject, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AlertMonitorObject{})
	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if q.EndpointID > 0 {
		db = db.Where("endpoint_id = ?", q.EndpointID)
	}
	if role := strings.TrimSpace(q.ExporterRole); role != "" {
		db = db.Where("exporter_role = ?", role)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("service_name LIKE ? OR service_id LIKE ? OR address LIKE ? OR yunshu_project LIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.consul", "ListObjects", err)
	}
	var list []model.AlertMonitorObject
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
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
		_ = s.db.WithContext(ctx).Model(ep).Updates(map[string]any{
			"last_error": truncateConsulErr(err.Error()),
		}).Error
		return nil, constants.ErrBadRequestWithMsg("列出 Consul 服务失败: " + err.Error())
	}

	now := time.Now().UTC()
	upserted := 0
	removed := 0
	keepKeys := map[string]struct{}{}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

				var row model.AlertMonitorObject
				qerr := tx.Unscoped().Where("endpoint_id = ? AND service_name = ? AND service_id = ?", ep.ID, sname, sid).First(&row).Error
				row.EndpointID = ep.ID
				row.ProjectID = ep.ProjectID
				row.ServiceName = sname
				row.ServiceID = sid
				row.Node = strings.TrimSpace(inst.Node)
				row.Address = strings.TrimSpace(inst.Address)
				row.Port = inst.ServicePort
				row.TagsJSON = string(tagsJSON)
				row.MetaJSON = string(metaJSON)
				row.ExporterRole = strings.TrimSpace(meta["exporter_role"])
				row.YunshuProject = strings.TrimSpace(firstMeta(meta, "yunshu_project", "project"))
				row.Health = consulclient.AggregateHealth(inst.Checks)
				row.ProbeURL = strings.TrimSpace(firstMeta(meta, "probe_url", "probe_host"))
				row.SyncedAt = now
				row.DeletedAt = gorm.DeletedAt{}
				if qerr == gorm.ErrRecordNotFound {
					if err := tx.Create(&row).Error; err != nil {
						return err
					}
				} else if qerr != nil {
					return qerr
				} else {
					if err := tx.Save(&row).Error; err != nil {
						return err
					}
				}
				upserted++
			}
		}

		var existing []model.AlertMonitorObject
		if err := tx.Where("endpoint_id = ?", ep.ID).Find(&existing).Error; err != nil {
			return err
		}
		for _, old := range existing {
			key := old.ServiceName + "\x00" + old.ServiceID
			if _, ok := keepKeys[key]; !ok {
				if err := tx.Delete(&old).Error; err != nil {
					return err
				}
				removed++
			}
		}
		return nil
	})
	if err != nil {
		_ = s.db.WithContext(ctx).Model(ep).Updates(map[string]any{
			"last_error": truncateConsulErr(err.Error()),
		}).Error
		return nil, constants.ErrBadRequestWithMsg("同步 Consul 失败: " + err.Error())
	}

	_ = s.db.WithContext(ctx).Model(ep).Updates(map[string]any{
		"last_sync_at": now,
		"last_error":   "",
	}).Error

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
