package logplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// LogIntelligenceService 日志智能分析：模板聚类、异常检测、多维关联。
type LogIntelligenceService struct {
	repo        interfaces.LogIntelligenceRepository
	logSearch   *LogSearchService
	projectRepo interfaces.ProjectRepository
}

func NewLogIntelligenceService(repo interfaces.LogIntelligenceRepository, logSearch *LogSearchService, projectRepo interfaces.ProjectRepository) *LogIntelligenceService {
	return &LogIntelligenceService{repo: repo, logSearch: logSearch, projectRepo: projectRepo}
}

type LogPatternItem struct {
	ID          uint   `json:"id"`
	ProjectID   uint   `json:"project_id"`
	Signature   string `json:"signature"`
	Sample      string `json:"sample"`
	Level       string `json:"level"`
	ServiceName string `json:"service_name"`
	HitCount    int64  `json:"hit_count"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
}

type LogPatternListQuery struct {
	ProjectID   uint   `form:"project_id"`
	Level       string `form:"level"`
	ServiceName string `form:"service_name"`
	Keyword     string `form:"keyword"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

func (s *LogIntelligenceService) ListPatterns(ctx context.Context, q LogPatternListQuery) (*pagination.Result[LogPatternItem], error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	rows, total, err := s.repo.ListPatterns(ctx, repository.LogPatternListFilter{
		ProjectID:   q.ProjectID,
		Level:       q.Level,
		ServiceName: q.ServiceName,
		Keyword:     q.Keyword,
		Offset:      (page - 1) * pageSize,
		Limit:       pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logintel", "ListPatterns", err)
	}
	out := make([]LogPatternItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toLogPatternItem(r))
	}
	return &pagination.Result[LogPatternItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type LogAnomalyItem struct {
	ID           uint   `json:"id"`
	ProjectID    uint   `json:"project_id"`
	PatternID    *uint  `json:"pattern_id,omitempty"`
	AnomalyType  string `json:"anomaly_type"`
	Signature    string `json:"signature"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
	Severity     string `json:"severity"`
	Status       string `json:"status"`
	AssigneeID   uint   `json:"assignee_id,omitempty"`
	AssigneeName string `json:"assignee_name,omitempty"`
	MutedUntil   string `json:"muted_until,omitempty"`
	DetectedAt   string `json:"detected_at"`
}

type LogAnomalyListQuery struct {
	ProjectID   uint   `form:"project_id"`
	Status      string `form:"status"`
	AnomalyType string `form:"anomaly_type"`
	AssigneeID  *uint  `form:"assignee_id"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// LogAnomalyUpdateRequest Error Tracking 状态/负责人更新。
type LogAnomalyUpdateRequest struct {
	Status       string `json:"status"`
	AssigneeID   *uint  `json:"assignee_id"`
	AssigneeName string `json:"assignee_name"`
	MuteMinutes  *int   `json:"mute_minutes"` // >0 则置 muted 并写 muted_until
}

func (s *LogIntelligenceService) ListAnomalies(ctx context.Context, q LogAnomalyListQuery) (*pagination.Result[LogAnomalyItem], error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	rows, total, err := s.repo.ListAnomalies(ctx, repository.LogAnomalyListFilter{
		ProjectID:   q.ProjectID,
		Status:      q.Status,
		AnomalyType: q.AnomalyType,
		AssigneeID:  q.AssigneeID,
		Offset:      (page - 1) * pageSize,
		Limit:       pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logintel", "ListAnomalies", err)
	}
	out := make([]LogAnomalyItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toLogAnomalyItem(r))
	}
	return &pagination.Result[LogAnomalyItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// UpdateAnomaly 更新错误追踪问题状态 / 负责人 / 静默。
func (s *LogIntelligenceService) UpdateAnomaly(ctx context.Context, projectID, anomalyID uint, req LogAnomalyUpdateRequest) (*LogAnomalyItem, error) {
	if projectID == 0 || anomalyID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id / anomaly_id 必填")
	}
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	row, err := s.repo.GetAnomalyByIDInProject(ctx, projectID, anomalyID)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("问题不存在")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		switch st {
		case model.LogAnomalyStatusOpen, model.LogAnomalyStatusAck, model.LogAnomalyStatusResolved, model.LogAnomalyStatusMuted:
			row.Status = st
		default:
			return nil, constants.ErrBadRequestWithMsg("status 无效")
		}
	}
	if req.AssigneeID != nil {
		row.AssigneeID = *req.AssigneeID
		// 与 assignee_id 一并更新姓名；清空指派时 id=0 且 name 为空
		row.AssigneeName = strings.TrimSpace(req.AssigneeName)
	} else if name := strings.TrimSpace(req.AssigneeName); name != "" {
		row.AssigneeName = name
	}
	if req.MuteMinutes != nil && *req.MuteMinutes > 0 {
		until := time.Now().UTC().Add(time.Duration(*req.MuteMinutes) * time.Minute)
		row.MutedUntil = &until
		row.Status = model.LogAnomalyStatusMuted
	}
	if err := s.repo.SaveAnomaly(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "logintel", "UpdateAnomaly", err)
	}
	item := toLogAnomalyItem(*row)
	return &item, nil
}

// LogContextQuery 关联上下文查询（P4）。
type LogContextQuery struct {
	ProjectID     uint   `form:"project_id"`
	AnchorTime    string `form:"anchor_time"`
	WindowMinutes int    `form:"window_minutes"`
	ServiceID     *uint  `form:"service_id"`
	ServiceName   string `form:"service_name"`
	AlertID       *uint  `form:"alert_id"`
	ChangeID      *uint  `form:"change_id"`
	Fingerprint   string `form:"fingerprint"`
	ClusterID     *uint  `form:"cluster_id"`
	Namespace     string `form:"namespace"`
	Pod           string `form:"pod"`
}

type LogContextResult struct {
	AnchorTime    string               `json:"anchor_time"`
	WindowFrom    string               `json:"window_from"`
	WindowTo      string               `json:"window_to"`
	Overview      *LogOverviewResult   `json:"overview,omitempty"`
	RecentChanges []model.ChangeEvent  `json:"recent_changes"`
	RecentAlerts  []model.AlertEvent   `json:"recent_alerts"`
	LogSummary    *LogSummaryResult    `json:"log_summary,omitempty"`
}

func (s *LogIntelligenceService) GetContext(ctx context.Context, q LogContextQuery) (*LogContextResult, error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	anchor := time.Now().UTC()
	if at := strings.TrimSpace(q.AnchorTime); at != "" {
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			anchor = t.UTC()
		}
	}
	win := q.WindowMinutes
	if win <= 0 {
		win = 5
	}
	if win > 60 {
		win = 60
	}
	from := anchor.Add(-time.Duration(win) * time.Minute)
	to := anchor.Add(time.Duration(win) * time.Minute)

	out := &LogContextResult{
		AnchorTime: anchor.Format(time.RFC3339),
		WindowFrom: from.Format(time.RFC3339),
		WindowTo:   to.Format(time.RFC3339),
	}

	// 告警 / 变更 / 指纹锚点
	if q.AlertID != nil && *q.AlertID > 0 && s.repo != nil {
		if ev, err := s.repo.GetAlertEventByID(ctx, *q.AlertID); err == nil {
			anchor = ev.CreatedAt.UTC()
			from = anchor.Add(-time.Duration(win) * time.Minute)
			to = anchor.Add(time.Duration(win) * time.Minute)
			out.AnchorTime = anchor.Format(time.RFC3339)
			out.WindowFrom = from.Format(time.RFC3339)
			out.WindowTo = to.Format(time.RFC3339)
			out.RecentAlerts = []model.AlertEvent{*ev}
		}
	}
	if q.Fingerprint != "" && s.repo != nil && len(out.RecentAlerts) == 0 {
		if ev, err := s.repo.GetAlertEventByFingerprint(ctx, q.Fingerprint); err == nil {
			anchor = ev.CreatedAt.UTC()
			from = anchor.Add(-time.Duration(win) * time.Minute)
			to = anchor.Add(time.Duration(win) * time.Minute)
			out.AnchorTime = anchor.Format(time.RFC3339)
			out.WindowFrom = from.Format(time.RFC3339)
			out.WindowTo = to.Format(time.RFC3339)
			out.RecentAlerts = []model.AlertEvent{*ev}
			if q.ProjectID == 0 && ev.ProjectID > 0 {
				q.ProjectID = ev.ProjectID
			}
		}
	}
	if q.ChangeID != nil && *q.ChangeID > 0 && s.repo != nil {
		if ch, err := s.repo.GetChangeEventByID(ctx, *q.ChangeID); err == nil {
			anchor = ch.StartedAt.UTC()
			from = anchor.Add(-time.Duration(win) * time.Minute)
			to = anchor.Add(time.Duration(win) * time.Minute)
			out.AnchorTime = anchor.Format(time.RFC3339)
			out.WindowFrom = from.Format(time.RFC3339)
			out.WindowTo = to.Format(time.RFC3339)
			out.RecentChanges = []model.ChangeEvent{*ch}
		}
	}

	if s.repo != nil {
		changes, _ := s.repo.ListChangeEventsInRange(ctx, q.ProjectID, from, to, 10)
		if len(out.RecentChanges) == 0 {
			out.RecentChanges = changes
		}
		alerts, _ := s.repo.ListAlertEventsInRange(ctx, q.ProjectID, from, to, 10)
		if len(out.RecentAlerts) == 0 {
			out.RecentAlerts = alerts
		}
	}

	if s.logSearch != nil {
		sq := LogSearchQuery{
			ProjectID:   q.ProjectID,
			ServiceID:   q.ServiceID,
			ServiceName: q.ServiceName,
			ClusterID:   q.ClusterID,
			Namespace:   q.Namespace,
			Pod:         q.Pod,
			From:        from.Format(time.RFC3339),
			To:          to.Format(time.RFC3339),
			Level:       "ERROR",
		}
		overview, err := s.logSearch.Overview(ctx, sq)
		if err == nil {
			out.Overview = overview
			out.LogSummary = overview.Summary
		}
	}
	return out, nil
}

// RunIntelligenceWorker 后台模板提取与异常检测（P2/P3）。
func RunIntelligenceWorker(ctx context.Context, svc *LogIntelligenceService) {
	if svc == nil || svc.logSearch == nil || svc.repo == nil {
		slog.Default().With("component", "logintel").Info("Log intelligence worker skipped: service unavailable")
		return
	}
	spec := "*/15 * * * *"
	log := slog.Default().With("component", "logintel")
	run := func() {
		if err := svc.runCycle(ctx); err != nil {
			log.Warn("Log intelligence cycle failed", "error", err)
			return
		}
		log.Info("Log intelligence cycle completed")
	}
	// 启动立即跑一轮，避免只等到下一个 :00/:15/:30/:45
	run()
	log.Info("Started log intelligence worker", "cron", spec)
	cronutil.RunWorker(ctx, spec, run, "0 */15 * * * *")
}

func (s *LogIntelligenceService) runCycle(ctx context.Context) error {
	if s.projectRepo == nil {
		return nil
	}
	projects, _, err := s.projectRepo.List(ctx, repository.ProjectListParams{Page: 1, PageSize: 200, LifecycleStatus: "active"})
	if err != nil {
		return err
	}
	for _, p := range projects {
		if err := s.processProject(ctx, p.ID); err != nil {
			slog.Default().With("component", "logintel", "project_id", p.ID).Warn("process project failed", "error", err)
		}
	}
	return nil
}

func (s *LogIntelligenceService) processProject(ctx context.Context, projectID uint) error {
	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour).Format(time.RFC3339)
	to := now.Format(time.RFC3339)
	res, err := s.logSearch.Search(ctx, LogSearchQuery{
		ProjectID: projectID,
		Level:     "ERROR",
		From:      from,
		To:        to,
		Page:      1,
		PageSize:  300,
	})
	if err != nil {
		return err
	}
	summary := SummarizeLogHits(res, 0)
	sigTotals := map[string]int{}
	sigSample := map[string]LogSearchItem{}
	for _, it := range res.List {
		sig := NormalizeLogSignature(it.Message)
		if sig == "" {
			continue
		}
		sigTotals[sig]++
		if _, ok := sigSample[sig]; !ok {
			sigSample[sig] = it
		}
	}

	for sig, cnt := range sigTotals {
		sample := sigSample[sig]
		lv := strings.TrimSpace(sample.Level)
		if lv == "" {
			lv = inferLevelFromMessage(sample.Message)
		}
		existing, err := s.repo.GetPatternBySignature(ctx, projectID, sig)
		if err == gorm.ErrRecordNotFound {
			row := model.LogPattern{
				ProjectID:   projectID,
				Signature:   sig,
				Sample:      truncateRunes(sample.Message, 500),
				Level:       lv,
				ServiceName: strings.TrimSpace(sample.ServiceName),
				HitCount:    int64(cnt),
				FirstSeenAt: now,
				LastSeenAt:  now,
			}
			if err := s.repo.CreatePattern(ctx, &row); err != nil {
				continue
			}
			s.createAnomalyIfNew(ctx, projectID, row)
			continue
		}
		if err != nil {
			continue
		}
		existing.HitCount += int64(cnt)
		existing.LastSeenAt = now
		if existing.Sample == "" {
			existing.Sample = truncateRunes(sample.Message, 500)
		}
		_ = s.repo.SavePattern(ctx, existing)
	}

	// 错误量突增：近 1h ERROR 总量 vs 近 24h 均值
	curTotal := res.Total
	dayRes, err := s.logSearch.Search(ctx, LogSearchQuery{
		ProjectID: projectID,
		Level:     "ERROR",
		From:      now.Add(-24 * time.Hour).Format(time.RFC3339),
		To:        now.Add(-1 * time.Hour).Format(time.RFC3339),
		Page:      1,
		PageSize:  1,
	})
	if err != nil {
		return nil
	}
	avgHour := dayRes.Total / 23
	if avgHour <= 0 {
		avgHour = 1
	}
	if curTotal >= avgHour*3 && curTotal >= 10 {
		title := fmt.Sprintf("ERROR 日志量突增：近1h %d 条（24h 均值约 %d/h）", curTotal, avgHour)
		s.upsertSpikeAnomaly(ctx, projectID, title, curTotal, avgHour)
	}
	_ = summary
	return nil
}

func (s *LogIntelligenceService) createAnomalyIfNew(ctx context.Context, projectID uint, pat model.LogPattern) {
	cnt, _ := s.repo.CountAnomaliesByTypeSignature(ctx, projectID, model.LogAnomalyTypeNewPattern, pat.Signature)
	if cnt > 0 {
		return
	}
	pid := pat.ID
	meta, _ := json.Marshal(map[string]any{"sample": pat.Sample, "level": pat.Level})
	row := model.LogAnomaly{
		ProjectID:    projectID,
		PatternID:    &pid,
		AnomalyType:  model.LogAnomalyTypeNewPattern,
		Signature:    pat.Signature,
		Title:        "发现新日志模板",
		Detail:       truncateRunes(pat.Sample, 300),
		Severity:     model.LogAnomalySeverityWarning,
		Status:       model.LogAnomalyStatusOpen,
		MetadataJSON: string(meta),
		DetectedAt:   time.Now().UTC(),
	}
	_ = s.repo.CreateAnomaly(ctx, &row)
}

func (s *LogIntelligenceService) upsertSpikeAnomaly(ctx context.Context, projectID uint, title string, cur, avg int64) {
	since := time.Now().UTC().Add(-2 * time.Hour)
	existing, err := s.repo.GetRecentSpikeAnomaly(ctx, projectID, since)
	meta, _ := json.Marshal(map[string]any{"current_hour": cur, "avg_hour": avg})
	if err == gorm.ErrRecordNotFound {
		row := model.LogAnomaly{
			ProjectID:    projectID,
			AnomalyType:  model.LogAnomalyTypeErrorSpike,
			Title:        title,
			Detail:       title,
			Severity:     model.LogAnomalySeverityCritical,
			Status:       model.LogAnomalyStatusOpen,
			MetadataJSON: string(meta),
			DetectedAt:   time.Now().UTC(),
		}
		_ = s.repo.CreateAnomaly(ctx, &row)
		return
	}
	if err == nil {
		existing.Title = title
		existing.Detail = title
		existing.MetadataJSON = string(meta)
		existing.DetectedAt = time.Now().UTC()
		_ = s.repo.SaveAnomaly(ctx, existing)
	}
}

func toLogPatternItem(r model.LogPattern) LogPatternItem {
	return LogPatternItem{
		ID: r.ID, ProjectID: r.ProjectID, Signature: r.Signature, Sample: r.Sample,
		Level: r.Level, ServiceName: r.ServiceName, HitCount: r.HitCount,
		FirstSeenAt: r.FirstSeenAt.Format(time.RFC3339),
		LastSeenAt:  r.LastSeenAt.Format(time.RFC3339),
	}
}

func toLogAnomalyItem(r model.LogAnomaly) LogAnomalyItem {
	item := LogAnomalyItem{
		ID: r.ID, ProjectID: r.ProjectID, PatternID: r.PatternID,
		AnomalyType: r.AnomalyType, Signature: r.Signature,
		Title: r.Title, Detail: r.Detail, Severity: r.Severity,
		Status: r.Status, AssigneeID: r.AssigneeID, AssigneeName: r.AssigneeName,
		DetectedAt: r.DetectedAt.Format(time.RFC3339),
	}
	if r.MutedUntil != nil {
		item.MutedUntil = r.MutedUntil.UTC().Format(time.RFC3339)
	}
	return item
}

// CountOpenAnomalies 统计项目 open 异常数（供服务画像健康分）。
func (s *LogIntelligenceService) CountOpenAnomalies(ctx context.Context, projectID uint) (int64, error) {
	if s.repo == nil || projectID == 0 {
		return 0, nil
	}
	return s.repo.CountOpenAnomalies(ctx, projectID)
}
