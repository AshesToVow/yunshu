package overview

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/service/k8s"

	"github.com/redis/go-redis/v9"
	corev1 "k8s.io/api/core/v1"
)

// logAgentOnlineWindow 须与 LogAgentService 心跳超时一致，用于总览「在线 Agent」计数。
const logAgentOnlineWindow = 90 * time.Second

type OverviewResponse struct {
	UsersCount    int64 `json:"users_count"`
	ClustersCount int64 `json:"clusters_count"`

	PendingRegistrationsCount int64 `json:"pending_registrations_count"`
	ServersCount              int64 `json:"servers_count"`

	PodNormalCount   int64 `json:"pod_normal_count"`
	PodAbnormalCount int64 `json:"pod_abnormal_count"`

	// Number of clusters that failed during pod aggregation.
	PodClusterErrors int64 `json:"pod_cluster_errors"`

	// Event stats (sampled per cluster to control latency).
	EventTotalCount    int64 `json:"event_total_count"`
	EventWarningCount  int64 `json:"event_warning_count"`
	EventClusterErrors int64 `json:"event_cluster_errors"`

	// Alert events（告警监控）
	AlertFiringCount      int64 `json:"alert_firing_count"`
	AlertEventsTodayCount int64 `json:"alert_events_today_count"`

	// Log agents（与 Agent 列表「在线」判定一致：最近心跳在 90s 内）
	LogAgentsOnlineCount  int64 `json:"log_agents_online_count"`
	LogAgentsOfflineCount int64 `json:"log_agents_offline_count"`
}

type OverviewProjectLaunchSeries struct {
	ProjectID   uint     `json:"project_id"`
	ProjectName string   `json:"project_name"`
	Data        []int64  `json:"data"`
	Color       string   `json:"color,omitempty"`
}

type OverviewProjectLaunchesResponse struct {
	Days   []string                      `json:"days"`
	Series []OverviewProjectLaunchSeries `json:"series"`
}

type OverviewReleaseByPersonItem struct {
	Person string `json:"person"`
	Count  int64  `json:"count"`
}

type OverviewReleaseByPersonResponse struct {
	Items []OverviewReleaseByPersonItem `json:"items"`
}

type OverviewService struct {
	repo            interfaces.OverviewRepository
	runtime         *k8s.K8sRuntimeService
	redis           *redis.Client
	memberRepo      interfaces.ProjectMemberRepository
	accessRepo      interfaces.K8sClusterAccessRepository
	pluginsEnabled  map[string]bool
}

// NewOverviewService 创建相关逻辑。
func NewOverviewService(
	repo interfaces.OverviewRepository,
	runtime *k8s.K8sRuntimeService,
	redisClient *redis.Client,
	memberRepo interfaces.ProjectMemberRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
	pluginsEnabled map[string]bool,
) *OverviewService {
	return &OverviewService{
		repo: repo, runtime: runtime, redis: redisClient,
		memberRepo: memberRepo, accessRepo: accessRepo, pluginsEnabled: pluginsEnabled,
	}
}

func (s *OverviewService) cicdChartsEnabled() bool {
	if s == nil || s.pluginsEnabled == nil {
		return true
	}
	return s.pluginsEnabled["cicd"]
}

func emptyProjectLaunches(now time.Time) *OverviewProjectLaunchesResponse {
	_, _, dayLabels, _ := overviewMonthRange(now)
	return &OverviewProjectLaunchesResponse{Days: dayLabels, Series: []OverviewProjectLaunchSeries{}}
}

const overviewChartDays = 30

var overviewSeriesColors = []string{
	"#14b8a6", "#8b5cf6", "#3b82f6", "#f97316", "#ef4444",
	"#64748b", "#eab308", "#22c55e", "#ec4899", "#06b6d4",
}

func (s *OverviewService) resolveProjectScope(ctx context.Context) (projectIDs []uint, unrestricted bool, err error) {
	unrestricted = true
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) || s.memberRepo == nil {
		return nil, unrestricted, nil
	}
	unrestricted = false
	projectIDs, err = s.memberRepo.ListProjectIDsByUser(ctx, u.ID)
	if err != nil {
		return nil, false, bizerrors.Pass(ctx, "overview", "resolveProjectScope", err)
	}
	return projectIDs, unrestricted, nil
}

func overviewMonthRange(now time.Time) (start, end time.Time, dayLabels []string, dayIndex map[string]int) {
	loc := now.Location()
	end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	start = end.AddDate(0, 0, -overviewChartDays)
	dayLabels = make([]string, 0, overviewChartDays)
	dayIndex = make(map[string]int, overviewChartDays)
	for i := 0; i < overviewChartDays; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		dayLabels = append(dayLabels, d.Format("01-02"))
		dayIndex[key] = i
	}
	return start, end, dayLabels, dayIndex
}

// ProjectLaunches 近一个月各项目上线（成功发布）数量趋势。
func (s *OverviewService) ProjectLaunches(ctx context.Context) (*OverviewProjectLaunchesResponse, error) {
	if !s.cicdChartsEnabled() {
		return emptyProjectLaunches(time.Now()), nil
	}
	if s.repo == nil {
		return nil, constants.ErrInternal
	}
	projectIDs, unrestricted, err := s.resolveProjectScope(ctx)
	if err != nil {
		return nil, err
	}
	start, end, dayLabels, dayIndex := overviewMonthRange(time.Now())
	rows, err := s.repo.CountReleaseLaunchesByProjectDay(ctx, start, end, projectIDs, unrestricted)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "overview", "ProjectLaunches", err)
	}
	totals := map[uint]int64{}
	meta := map[uint]string{}
	for _, row := range rows {
		totals[row.ProjectID] += row.Cnt
		meta[row.ProjectID] = row.ProjectName
	}
	type ranked struct {
		id    uint
		total int64
	}
	rankedList := make([]ranked, 0, len(totals))
	for id, total := range totals {
		rankedList = append(rankedList, ranked{id: id, total: total})
	}
	sort.Slice(rankedList, func(i, j int) bool {
		return rankedList[i].total > rankedList[j].total
	})
	limit := 10
	if len(rankedList) < limit {
		limit = len(rankedList)
	}
	out := &OverviewProjectLaunchesResponse{
		Days:   dayLabels,
		Series: make([]OverviewProjectLaunchSeries, 0, limit),
	}
	for i := 0; i < limit; i++ {
		pid := rankedList[i].id
		data := make([]int64, overviewChartDays)
		for _, row := range rows {
			if row.ProjectID != pid {
				continue
			}
			if idx, ok := dayIndex[row.Day]; ok {
				data[idx] = row.Cnt
			}
		}
		out.Series = append(out.Series, OverviewProjectLaunchSeries{
			ProjectID:   pid,
			ProjectName: meta[pid],
			Data:        data,
			Color:       overviewSeriesColors[i%len(overviewSeriesColors)],
		})
	}
	return out, nil
}

// ReleaseByPerson 近一个月 CD 工单按提交人统计。
func (s *OverviewService) ReleaseByPerson(ctx context.Context) (*OverviewReleaseByPersonResponse, error) {
	if !s.cicdChartsEnabled() {
		return &OverviewReleaseByPersonResponse{Items: []OverviewReleaseByPersonItem{}}, nil
	}
	if s.repo == nil {
		return nil, constants.ErrInternal
	}
	projectIDs, unrestricted, err := s.resolveProjectScope(ctx)
	if err != nil {
		return nil, err
	}
	start, end, _, _ := overviewMonthRange(time.Now())
	rows, err := s.repo.CountReleaseRunsByPerson(ctx, start, end, projectIDs, unrestricted)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "overview", "ReleaseByPerson", err)
	}
	out := &OverviewReleaseByPersonResponse{Items: make([]OverviewReleaseByPersonItem, 0, len(rows))}
	for _, row := range rows {
		out.Items = append(out.Items, OverviewReleaseByPersonItem{
			Person: row.Person,
			Count:  row.Cnt,
		})
	}
	return out, nil
}

// Get 获取相关的业务逻辑。
func (s *OverviewService) Get(ctx context.Context) (*OverviewResponse, error) {
	if s.repo == nil {
		return nil, constants.ErrInternal
	}

	cacheKey := overviewMetricsCacheKey(ctx)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && raw != "" {
			var cached OverviewResponse
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return &cached, nil
			}
		}
	}

	metrics, err := s.repo.LoadMetrics(ctx, int(model.RegistrationPending))
	if err != nil {
		return nil, bizerrors.Pass(ctx, "overview", "Get", err)
	}
	out := &OverviewResponse{
		UsersCount:                metrics.UsersCount,
		ClustersCount:             metrics.ClustersCount,
		PendingRegistrationsCount: metrics.PendingRegistrationsCount,
		ServersCount:              metrics.ServersCount,
	}
	s.fillOverviewAlertAndAgents(ctx, out)

	projectIDs, unrestricted, err := s.resolveProjectScope(ctx)
	if err != nil {
		return nil, err
	}
	var clusters []model.K8sCluster
	clusters, err = s.repo.ListEnabledClusters(ctx, projectIDs, unrestricted)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "overview", "Get", err)
	}
	clusters = s.filterOverviewClusters(ctx, clusters)
	if len(clusters) == 0 {
		if s.redis != nil {
			if b, err := json.Marshal(out); err == nil {
				_ = s.redis.Set(ctx, cacheKey, string(b), 15*time.Second).Err()
			}
		}
		return out, nil
	}

	// 产品侧体验优先：总时限内返回“可得数据 + 失败计数”，而不是让首页等待到超时。
	overallCtx, overallCancel := context.WithTimeout(ctx, 8*time.Second)
	defer overallCancel()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, c := range clusters {
		cid := c.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Guard against unexpected panics from 3rd-party k8s wrapper (kom),
			// so one cluster failure won't crash the whole backend process.
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					out.PodClusterErrors++
					out.EventClusterErrors++
					mu.Unlock()
				}
			}()

			// 连接探测保持短超时，避免不可达集群拖慢全局。
			cctx, cancel := context.WithTimeout(overallCtx, 2*time.Second)
			_, k, err := s.runtime.GetClusterKubectl(cctx, cid)
			cancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			// Pod 聚合也限制时长，超时按失败集群处理。
			pctx, pcancel := context.WithTimeout(overallCtx, 4*time.Second)
			var pods []corev1.Pod
			podQuery := k.WithContext(pctx)
			if podQuery == nil {
				pcancel()
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}
			err = podQuery.Resource(&corev1.Pod{}).AllNamespace().List(&pods).Error
			pcancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			normal := int64(0)
			abnormal := int64(0)
			for _, p := range pods {
				if isPodNormal(p) {
					normal++
				} else {
					abnormal++
				}
			}
			mu.Lock()
			out.PodNormalCount += normal
			out.PodAbnormalCount += abnormal
			mu.Unlock()

			// Event 概览仅采样最近 500 条，避免在大集群拖慢首页。
			ectx, ecancel := context.WithTimeout(overallCtx, 4*time.Second)
			var events []corev1.Event
			eventQuery := k.WithContext(ectx)
			if eventQuery == nil {
				ecancel()
				mu.Lock()
				out.EventClusterErrors++
				mu.Unlock()
				return
			}
			err = eventQuery.Resource(&corev1.Event{}).AllNamespace().Limit(500).List(&events).Error
			ecancel()
			if err != nil {
				mu.Lock()
				out.EventClusterErrors++
				mu.Unlock()
				return
			}
			total := int64(len(events))
			warnings := int64(0)
			for _, ev := range events {
				if ev.Type == "Warning" {
					warnings++
				}
			}
			mu.Lock()
			out.EventTotalCount += total
			out.EventWarningCount += warnings
			mu.Unlock()
		}()
	}
	wg.Wait()

	if s.redis != nil {
		if b, err := json.Marshal(out); err == nil {
			_ = s.redis.Set(ctx, cacheKey, string(b), 15*time.Second).Err()
		}
	}
	return out, nil
}

// fillOverviewAlertAndAgents 聚合告警与日志 Agent 指标；表不存在或查询失败时保持 0，不阻断总览。
func (s *OverviewService) fillOverviewAlertAndAgents(ctx context.Context, out *OverviewResponse) {
	if s.repo == nil || out == nil {
		return
	}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	stats, err := s.repo.FillAlertAndAgentStats(ctx, dayStart, dayEnd, now.Add(-logAgentOnlineWindow))
	if err != nil || stats == nil {
		return
	}
	out.AlertFiringCount = stats.AlertFiringCount
	out.AlertEventsTodayCount = stats.AlertEventsTodayCount
	out.LogAgentsOnlineCount = stats.LogAgentsOnlineCount
	out.LogAgentsOfflineCount = stats.LogAgentsOfflineCount
}

func isPodNormal(p corev1.Pod) bool {
	// A pragmatic definition:
	// - phase is Running
	// - all containers are ready (or no container status found -> abnormal)
	if string(p.Status.Phase) != "Running" {
		return false
	}
	if len(p.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, st := range p.Status.ContainerStatuses {
		if !st.Ready {
			return false
		}
	}
	return true
}

func overviewMetricsCacheKey(ctx context.Context) string {
	base := "overview:metrics:v4"
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return base + ":anon"
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return base + ":super"
	}
	return fmt.Sprintf("%s:u:%d", base, u.ID)
}

func (s *OverviewService) filterOverviewClusters(ctx context.Context, clusters []model.K8sCluster) []model.K8sCluster {
	if len(clusters) == 0 {
		return clusters
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) {
		return clusters
	}
	if s.accessRepo == nil {
		return nil
	}
	pack := k8sauth.PackFromCurrentUser(u)
	idx, err := s.accessRepo.BuildEffectiveTierIndex(ctx, pack)
	if err != nil {
		return nil
	}
	if idx.GlobalRank < k8s.K8sAccessRankReadonly && len(idx.PerCluster) == 0 {
		return nil
	}
	out := make([]model.K8sCluster, 0, len(clusters))
	for _, c := range clusters {
		if idx.ClusterAccessible(c.ID, k8s.K8sAccessRankReadonly) {
			out = append(out, c)
		}
	}
	return out
}

// String 的功能实现。
func (o OverviewResponse) String() string {
	return fmt.Sprintf("users=%d clusters=%d alerts_firing=%d alerts_today=%d agents_on=%d agents_off=%d pod_normal=%d pod_abnormal=%d pod_errors=%d event_total=%d event_warning=%d event_errors=%d",
		o.UsersCount, o.ClustersCount, o.AlertFiringCount, o.AlertEventsTodayCount, o.LogAgentsOnlineCount, o.LogAgentsOfflineCount,
		o.PodNormalCount, o.PodAbnormalCount, o.PodClusterErrors, o.EventTotalCount, o.EventWarningCount, o.EventClusterErrors)
}
