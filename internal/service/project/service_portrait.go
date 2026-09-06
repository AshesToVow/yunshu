package project

import (
	"context"
	"fmt"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// ServicePortrait 服务画像聚合视图（运行/发布/告警/日志入口由 links 推导）。
type ServicePortrait struct {
	Service       ServiceCatalogItem   `json:"service"`
	RecentChanges []model.ChangeEvent  `json:"recent_changes"`
	EntryPoints   []PortraitEntryPoint `json:"entry_points"`
	CicdSummary   *PortraitCicdSummary `json:"cicd_summary,omitempty"`
	Health        *PortraitHealth      `json:"health,omitempty"`
	LogSummary    *PortraitLogSummary  `json:"log_summary,omitempty"`
}

// PortraitLogSummary 日志智能摘要（异常 + 模板统计）。
type PortraitLogSummary struct {
	OpenAnomalyCount int64                  `json:"open_anomaly_count"`
	PatternCount     int64                  `json:"pattern_count"`
	RecentAnomalies  []PortraitLogAnomalyBrief `json:"recent_anomalies"`
}

type PortraitLogAnomalyBrief struct {
	ID          uint   `json:"id"`
	AnomalyType string `json:"anomaly_type"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	DetectedAt  string `json:"detected_at"`
}

type PortraitEntryPoint struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Path  string `json:"path"`
	Hint  string `json:"hint,omitempty"`
}

type PortraitCicdSummary struct {
	CicdServiceID uint   `json:"cicd_service_id"`
	Identifier    string `json:"identifier"`
	Name          string `json:"name"`
	LastReleaseID *uint  `json:"last_release_id,omitempty"`
	LastStatus    string `json:"last_status,omitempty"`
	LastTitle     string `json:"last_title,omitempty"`
	LastAt        string `json:"last_at,omitempty"`
}

func (s *ServiceCatalogService) Get(ctx context.Context, projectID, id uint) (*ServiceCatalogItem, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	row, err := s.repo.GetByID(ctx, projectID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	links, _ := s.repo.ListLinks(ctx, row.ID)
	return &ServiceCatalogItem{ServiceCatalog: *row, Links: links}, nil
}

func (s *ServiceCatalogService) Portrait(ctx context.Context, projectID, catalogID uint, recentChanges []model.ChangeEvent) (*ServicePortrait, error) {
	item, err := s.Get(ctx, projectID, catalogID)
	if err != nil {
		return nil, err
	}
	return &ServicePortrait{
		Service:       *item,
		RecentChanges: recentChanges,
		EntryPoints:   buildPortraitEntries(item),
		CicdSummary:   loadCicdSummary(ctx, s.db, item),
		Health:        s.buildHealth(ctx, item),
		LogSummary:    s.loadLogSummary(ctx, item),
	}, nil
}

func buildPortraitEntries(item *ServiceCatalogItem) []PortraitEntryPoint {
	if item == nil {
		return nil
	}
	var out []PortraitEntryPoint
	seen := map[string]struct{}{}
	add := func(kind, label, path, hint string) {
		key := kind + "|" + path
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, PortraitEntryPoint{Kind: kind, Label: label, Path: path, Hint: hint})
	}
	for _, l := range item.Links {
		switch l.LinkType {
		case model.ServiceLinkCicdService:
			add("cicd", "CI/CD 发布记录", "/cicd/release-records",
				fmt.Sprintf("cicd_service_id=%v", derefUint(l.RefID)))
		case model.ServiceLinkK8sWorkload:
			add("k8s", "K8s 工作负载", "/deployments", l.RefKey)
		case model.ServiceLinkLogSource:
			add("logs", "日志检索", "/project-logs",
				fmt.Sprintf("log_source_id=%v", derefUint(l.RefID)))
			add("logs", "日志异常", "/project-logs",
				"tab=anomalies")
		case model.ServiceLinkAlertMonitorRule:
			add("alert", "告警监控", "/alert-monitor-platform",
				fmt.Sprintf("rule_id=%v", derefUint(l.RefID)))
		case model.ServiceLinkCmdbService:
			add("cmdb", "服务器服务实例", "/project-servers",
				fmt.Sprintf("cmdb_service_id=%v", derefUint(l.RefID)))
		case model.ServiceLinkDbInstance:
			add("dbmgmt", "数据库实例", "/dbmgmt/instances",
				fmt.Sprintf("instance_id=%v", derefUint(l.RefID)))
		}
	}
	return out
}

func derefUint(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

func loadCicdSummary(ctx context.Context, db *gorm.DB, item *ServiceCatalogItem) *PortraitCicdSummary {
	if db == nil || item == nil {
		return nil
	}
	var cicdID uint
	for _, l := range item.Links {
		if l.LinkType == model.ServiceLinkCicdService && l.RefID != nil && *l.RefID > 0 {
			cicdID = *l.RefID
			break
		}
	}
	if cicdID == 0 {
		return nil
	}
	var svc model.CicdService
	if err := db.WithContext(ctx).Where("id = ? AND project_id = ?", cicdID, item.ProjectID).First(&svc).Error; err != nil {
		return nil
	}
	sum := &PortraitCicdSummary{
		CicdServiceID: svc.ID,
		Identifier:    svc.Identifier,
		Name:          svc.Name,
	}
	var run model.CicdReleaseRun
	if err := db.WithContext(ctx).
		Where("project_id = ? AND service_id = ?", item.ProjectID, svc.ID).
		Order("id DESC").
		First(&run).Error; err == nil {
		id := run.ID
		sum.LastReleaseID = &id
		sum.LastStatus = run.Status
		sum.LastTitle = run.Title
		if run.StartedAt != nil {
			sum.LastAt = run.StartedAt.Format(time.RFC3339)
		} else {
			sum.LastAt = run.CreatedAt.Format(time.RFC3339)
		}
	}
	return sum
}

func (s *ServiceCatalogService) loadLogSummary(ctx context.Context, item *ServiceCatalogItem) *PortraitLogSummary {
	if s.db == nil || item == nil || item.ProjectID == 0 {
		return nil
	}
	var openCnt, patCnt int64
	_ = s.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ?", item.ProjectID, model.LogAnomalyStatusOpen).
		Count(&openCnt).Error
	_ = s.db.WithContext(ctx).Model(&model.LogPattern{}).
		Where("project_id = ?", item.ProjectID).
		Count(&patCnt).Error
	var rows []model.LogAnomaly
	_ = s.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ?", item.ProjectID, model.LogAnomalyStatusOpen).
		Order("detected_at DESC").Limit(5).Find(&rows).Error
	recent := make([]PortraitLogAnomalyBrief, 0, len(rows))
	for _, r := range rows {
		recent = append(recent, PortraitLogAnomalyBrief{
			ID: r.ID, AnomalyType: r.AnomalyType, Title: r.Title,
			Severity: r.Severity, DetectedAt: r.DetectedAt.Format(time.RFC3339),
		})
	}
	if openCnt == 0 && patCnt == 0 && len(recent) == 0 {
		return &PortraitLogSummary{OpenAnomalyCount: 0, PatternCount: patCnt}
	}
	return &PortraitLogSummary{
		OpenAnomalyCount: openCnt,
		PatternCount:     patCnt,
		RecentAnomalies:  recent,
	}
}
