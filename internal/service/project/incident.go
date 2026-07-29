package project

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type IncidentListQuery struct {
	ProjectID uint   `form:"-"`
	Status    string `form:"status"`
	Severity  string `form:"severity"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type IncidentOpenRequest struct {
	ProjectID        uint   `json:"-"`
	Title            string `json:"title" binding:"required,max=256"`
	Severity         string `json:"severity" binding:"omitempty,max=16"`
	Summary          string `json:"summary" binding:"omitempty,max=1024"`
	ServiceID        *uint  `json:"service_id"`
	AlertFingerprint string `json:"alert_fingerprint" binding:"omitempty,max=256"`
	AssigneeUserID   *uint  `json:"assignee_user_id"`
	OpenedBy         *uint  `json:"-"`
}

type IncidentUpdateRequest struct {
	ProjectID      uint   `json:"-"`
	ID             uint   `json:"-"`
	Status         string `json:"status" binding:"omitempty,max=32"`
	Summary        string `json:"summary" binding:"omitempty,max=1024"`
	AssigneeUserID *uint  `json:"assignee_user_id"`
	ActorID        *uint  `json:"-"`
}

type IncidentTimeline struct {
	Incident model.Incident       `json:"incident"`
	Notes    []model.IncidentNote `json:"notes"`
	Changes  []model.ChangeEvent  `json:"changes"`
	Alerts   []model.AlertEvent   `json:"alerts"`
	Releases []IncidentRelease    `json:"releases"`
	MTTA     *int64               `json:"mtta_seconds,omitempty"`
	MTTR     *int64               `json:"mttr_seconds,omitempty"`
}

func (s *ChangeEventService) ListIncidents(ctx context.Context, q IncidentListQuery) (*pagination.Result[model.Incident], error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.Incident{}).Where("project_id = ?", q.ProjectID)
	if st := strings.TrimSpace(q.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	if sev := strings.TrimSpace(q.Severity); sev != "" {
		dbq = dbq.Where("severity = ?", sev)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.Incident
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.Result[model.Incident]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ChangeEventService) OpenIncident(ctx context.Context, req IncidentOpenRequest) (*model.Incident, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	sev := strings.ToLower(strings.TrimSpace(req.Severity))
	if sev == "" {
		sev = model.IncidentSeverityP1
	}
	if sev != model.IncidentSeverityP1 && sev != model.IncidentSeverityP2 {
		return nil, constants.ErrBadRequestWithMsg("severity 仅支持 p1/p2")
	}
	fp := strings.TrimSpace(req.AlertFingerprint)
	if fp != "" {
		var existing model.Incident
		err := s.db.WithContext(ctx).
			Where("project_id = ? AND alert_fingerprint = ? AND status IN ?",
				req.ProjectID, fp, []string{model.IncidentStatusOpen, model.IncidentStatusMitigating}).
			First(&existing).Error
		if err == nil {
			return &existing, nil
		}
	}
	openedBy := req.OpenedBy
	if openedBy == nil {
		if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
			id := u.ID
			openedBy = &id
		}
	}
	row := model.Incident{
		ProjectID:        req.ProjectID,
		ServiceID:        req.ServiceID,
		Title:            strings.TrimSpace(req.Title),
		Severity:         sev,
		Status:           model.IncidentStatusOpen,
		Summary:          strings.TrimSpace(req.Summary),
		AlertFingerprint: fp,
		AssigneeUserID:   req.AssigneeUserID,
		OpenedBy:         openedBy,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ChangeEventService) UpdateIncident(ctx context.Context, req IncidentUpdateRequest) (*model.Incident, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	var row model.Incident
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", req.ID, req.ProjectID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	updates := map[string]any{}
	if sum := strings.TrimSpace(req.Summary); sum != "" {
		updates["summary"] = sum
	}
	if req.AssigneeUserID != nil {
		updates["assignee_user_id"] = *req.AssigneeUserID
	}
	st := strings.ToLower(strings.TrimSpace(req.Status))
	now := time.Now()
	switch st {
	case "":
	case model.IncidentStatusOpen, model.IncidentStatusMitigating, model.IncidentStatusResolved, model.IncidentStatusPostmortem:
		updates["status"] = st
		if st == model.IncidentStatusMitigating && row.AcknowledgedAt == nil {
			updates["acknowledged_at"] = now
			mtta := int64(now.Sub(row.CreatedAt).Seconds())
			updates["mtta_seconds"] = mtta
		}
		if st == model.IncidentStatusResolved || st == model.IncidentStatusPostmortem {
			updates["resolved_at"] = now
			mttr := int64(now.Sub(row.CreatedAt).Seconds())
			updates["mttr_seconds"] = mttr
			if row.AcknowledgedAt == nil {
				updates["acknowledged_at"] = now
				updates["mtta_seconds"] = mttr
			}
		}
	default:
		return nil, constants.ErrBadRequestWithMsg("status 无效")
	}
	if len(updates) == 0 {
		return &row, nil
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Where("id = ?", row.ID).First(&row).Error
	return &row, nil
}

func (s *ChangeEventService) AddIncidentNote(ctx context.Context, projectID, incidentID, authorID uint, body string) (*model.IncidentNote, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, constants.ErrBadRequestWithMsg("备注不能为空")
	}
	var inc model.Incident
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", incidentID, projectID).First(&inc).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	note := model.IncidentNote{IncidentID: incidentID, AuthorID: authorID, Body: body}
	if err := s.db.WithContext(ctx).Create(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *ChangeEventService) GetIncidentTimeline(ctx context.Context, projectID, incidentID uint, windowMinutes int) (*IncidentTimeline, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	var inc model.Incident
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", incidentID, projectID).First(&inc).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	from := inc.CreatedAt.Add(-time.Duration(windowMinutes) * time.Minute)
	to := time.Now()
	if inc.ResolvedAt != nil {
		to = *inc.ResolvedAt
	}
	var notes []model.IncidentNote
	_ = s.db.WithContext(ctx).Where("incident_id = ?", inc.ID).Order("id ASC").Find(&notes).Error

	cq := s.db.WithContext(ctx).Model(&model.ChangeEvent{}).
		Where("project_id = ? AND started_at >= ? AND started_at <= ?", projectID, from, to)
	if inc.ServiceID != nil {
		cq = cq.Where("service_id = ?", *inc.ServiceID)
	}
	var changes []model.ChangeEvent
	_ = cq.Order("id DESC").Limit(100).Find(&changes).Error

	var alerts []model.AlertEvent
	aq := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("created_at >= ? AND created_at <= ?", from, to)
	if fp := strings.TrimSpace(inc.AlertFingerprint); fp != "" {
		aq = aq.Where("fingerprint = ?", fp)
	} else {
		aq = aq.Where("severity IN ?", []string{"critical", "warning"})
	}
	_ = aq.Order("id DESC").Limit(50).Find(&alerts).Error

	var runs []model.CicdReleaseRun
	_ = s.db.WithContext(ctx).
		Where("project_id = ? AND created_at >= ? AND created_at <= ?", projectID, from, to).
		Order("id DESC").Limit(30).Find(&runs).Error
	var releases []IncidentRelease
	for _, r := range runs {
		started := ""
		if r.StartedAt != nil {
			started = r.StartedAt.Format(time.RFC3339)
		}
		releases = append(releases, IncidentRelease{
			ID: r.ID, ServiceID: r.ServiceID, Title: r.Title,
			Status: r.Status, Tenv: r.Tenv, StartedAt: started,
		})
	}
	return &IncidentTimeline{
		Incident: inc,
		Notes:    notes,
		Changes:  changes,
		Alerts:   alerts,
		Releases: releases,
		MTTA:     inc.MTTASeconds,
		MTTR:     inc.MTTRSeconds,
	}, nil
}
