package overview

import (
	"context"
	"time"

	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/repository"
)

// OverviewScreenResponse 智慧运维大屏聚合数据（KPI 不重复堆叠，以列表/分解为主）。
type OverviewScreenResponse struct {
	GeneratedAt time.Time `json:"generated_at"`

	KPI OverviewScreenKPI `json:"kpi"`

	Health OverviewScreenHealth `json:"health"`

	AlertsTop          []OverviewScreenAlert   `json:"alerts_top"`
	RecentReleases     []OverviewScreenRelease `json:"recent_releases"`
	RecentChanges      []OverviewScreenChange  `json:"recent_changes"`
	LoggieOfflineSample []OverviewScreenLoggie `json:"loggie_offline_sample"`
}

type OverviewScreenKPI struct {
	UsersCount                int64 `json:"users_count"`
	ClustersCount             int64 `json:"clusters_count"`
	ServersCount              int64 `json:"servers_count"`
	ServersEnabled            int64 `json:"servers_enabled"`
	ServersDisabled           int64 `json:"servers_disabled"`
	PendingRegistrationsCount int64 `json:"pending_registrations_count"`

	PodNormalCount   int64 `json:"pod_normal_count"`
	PodAbnormalCount int64 `json:"pod_abnormal_count"`
	EventWarningCount int64 `json:"event_warning_count"`

	AlertFiringCount      int64 `json:"alert_firing_count"`
	AlertEventsTodayCount int64 `json:"alert_events_today_count"`

	LoggieAgentsOnlineCount  int64 `json:"loggie_agents_online_count"`
	LoggieAgentsOfflineCount int64 `json:"loggie_agents_offline_count"`

	AIInvestigationsToday int64 `json:"ai_investigations_today"`
	AIInvestigationsOpen  int64 `json:"ai_investigations_open"`
}

type OverviewScreenHealth struct {
	PodHealthPct     int                   `json:"pod_health_pct"`
	AgentOnlinePct   int                   `json:"agent_online_pct"`
	AlertBySeverity  []OverviewLabelCount  `json:"alert_by_severity"`
	LoggieByHealth   []OverviewLabelCount  `json:"loggie_by_health"`
	ServersByStatus  []OverviewLabelCount  `json:"servers_by_status"`
}

type OverviewLabelCount struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type OverviewScreenAlert struct {
	ID          uint      `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Alertname   string    `json:"alertname"`
	Severity    string    `json:"severity"`
	Cluster     string    `json:"cluster"`
	ProjectID   uint      `json:"project_id"`
	Summary     string    `json:"summary"`
	StartsAt    time.Time `json:"starts_at"`
}

type OverviewScreenRelease struct {
	ID            uint       `json:"id"`
	ProjectID     uint       `json:"project_id"`
	ProjectName   string     `json:"project_name"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	Tenv          string     `json:"tenv"`
	SubmitterName string     `json:"submitter_name"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type OverviewScreenChange struct {
	ID        uint      `json:"id"`
	ProjectID uint      `json:"project_id"`
	Source    string    `json:"source"`
	Action    string    `json:"action"`
	RiskLevel string    `json:"risk_level"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	StartedAt time.Time `json:"started_at"`
}

type OverviewScreenLoggie struct {
	ID           uint       `json:"id"`
	ProjectID    uint       `json:"project_id"`
	ServerID     uint       `json:"server_id"`
	HealthStatus string     `json:"health_status"`
	LastError    string     `json:"last_error"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
}

func pct(part, total int64) int {
	if total <= 0 {
		return 100
	}
	return int((part * 100) / total)
}

func mapLabelCounts(rows []repository.OverviewLabelCount) []OverviewLabelCount {
	out := make([]OverviewLabelCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, OverviewLabelCount{Label: r.Label, Count: r.Cnt})
	}
	return out
}

// Screen 聚合大屏面板数据：复用 Get() KPI，并补充告警榜/发布/变更/采集分解。
func (s *OverviewService) Screen(ctx context.Context) (*OverviewScreenResponse, error) {
	if s == nil || s.repo == nil {
		return nil, constants.ErrInternal
	}

	base, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	projectIDs, unrestricted, err := s.resolveProjectScope(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	agentCutoff := now.Add(-loggieAgentOnlineWindow)

	out := &OverviewScreenResponse{
		GeneratedAt: now,
		KPI: OverviewScreenKPI{
			UsersCount:                base.UsersCount,
			ClustersCount:             base.ClustersCount,
			ServersCount:              base.ServersCount,
			PendingRegistrationsCount: base.PendingRegistrationsCount,
			PodNormalCount:            base.PodNormalCount,
			PodAbnormalCount:          base.PodAbnormalCount,
			EventWarningCount:         base.EventWarningCount,
			AlertFiringCount:          base.AlertFiringCount,
			AlertEventsTodayCount:     base.AlertEventsTodayCount,
			LoggieAgentsOnlineCount:   base.LoggieAgentsOnlineCount,
			LoggieAgentsOfflineCount:  base.LoggieAgentsOfflineCount,
		},
		Health: OverviewScreenHealth{
			PodHealthPct:   pct(base.PodNormalCount, base.PodNormalCount+base.PodAbnormalCount),
			AgentOnlinePct: pct(base.LoggieAgentsOnlineCount, base.LoggieAgentsOnlineCount+base.LoggieAgentsOfflineCount),
		},
		AlertsTop:           []OverviewScreenAlert{},
		RecentReleases:      []OverviewScreenRelease{},
		RecentChanges:       []OverviewScreenChange{},
		LoggieOfflineSample: []OverviewScreenLoggie{},
	}

	if rows, err := s.repo.CountServersByStatus(ctx); err == nil {
		out.Health.ServersByStatus = mapLabelCounts(rows)
		for _, r := range rows {
			switch r.Label {
			case "1":
				out.KPI.ServersEnabled = r.Cnt
			case "0":
				out.KPI.ServersDisabled = r.Cnt
			}
		}
	}

	if rows, err := s.repo.CountAlertsBySeverity(ctx, projectIDs, unrestricted); err == nil {
		out.Health.AlertBySeverity = mapLabelCounts(rows)
	}

	if n, err := s.repo.CountFiringAlerts(ctx, projectIDs, unrestricted); err == nil {
		// 大屏 KPI 与告警榜同源：alert_cur_events（当前实例），避免与 alert_events 投递流水口径不一致
		out.KPI.AlertFiringCount = n
	}

	if rows, err := s.repo.CountLoggieByHealth(ctx); err == nil {
		out.Health.LoggieByHealth = mapLabelCounts(rows)
	}

	if today, open, err := s.repo.CountAIInvestigations(ctx, dayStart, dayEnd); err == nil {
		out.KPI.AIInvestigationsToday = today
		out.KPI.AIInvestigationsOpen = open
	}

	if alerts, err := s.repo.ListFiringAlertsTop(ctx, projectIDs, unrestricted, 12); err == nil {
		out.AlertsTop = make([]OverviewScreenAlert, 0, len(alerts))
		for _, a := range alerts {
			out.AlertsTop = append(out.AlertsTop, OverviewScreenAlert{
				ID: a.ID, Fingerprint: a.Fingerprint, Alertname: a.Alertname,
				Severity: a.Severity, Cluster: a.Cluster, ProjectID: a.ProjectID,
				Summary: a.Summary, StartsAt: a.StartsAt,
			})
		}
	} else {
		_ = bizerrors.Pass(ctx, "overview", "Screen.alerts", err)
	}

	if s.cicdChartsEnabled() {
		if releases, err := s.repo.ListRecentReleases(ctx, projectIDs, unrestricted, 10); err == nil {
			out.RecentReleases = make([]OverviewScreenRelease, 0, len(releases))
			for _, r := range releases {
				out.RecentReleases = append(out.RecentReleases, OverviewScreenRelease{
					ID: r.ID, ProjectID: r.ProjectID, ProjectName: r.ProjectName,
					Title: r.Title, Status: r.Status, Tenv: r.Tenv,
					SubmitterName: r.SubmitterName, FinishedAt: r.FinishedAt, CreatedAt: r.CreatedAt,
				})
			}
		}
	}

	if changes, err := s.repo.ListRecentChanges(ctx, projectIDs, unrestricted, 12); err == nil {
		out.RecentChanges = make([]OverviewScreenChange, 0, len(changes))
		for _, c := range changes {
			out.RecentChanges = append(out.RecentChanges, OverviewScreenChange{
				ID: c.ID, ProjectID: c.ProjectID, Source: c.Source, Action: c.Action,
				RiskLevel: c.RiskLevel, Status: c.Status, Summary: c.Summary, StartedAt: c.StartedAt,
			})
		}
	}

	if agents, err := s.repo.ListLoggieOfflineSample(ctx, agentCutoff, 8); err == nil {
		out.LoggieOfflineSample = make([]OverviewScreenLoggie, 0, len(agents))
		for _, a := range agents {
			out.LoggieOfflineSample = append(out.LoggieOfflineSample, OverviewScreenLoggie{
				ID: a.ID, ProjectID: a.ProjectID, ServerID: a.ServerID,
				HealthStatus: a.HealthStatus, LastError: a.LastError, LastSeenAt: a.LastSeenAt,
			})
		}
	}

	return out, nil
}
