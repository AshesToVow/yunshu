package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
)

// --- Build / Release Records ---

type BuildRunListQuery struct {
	ProjectID uint              `form:"project_id"`
	ServiceID uint              `form:"service_id"`
	Keyword   string            `form:"keyword"`
	Page      int               `form:"page"`
	PageSize  int               `form:"page_size"`
	Actor     *auth.CurrentUser `form:"-"`
}

type BuildRunItem struct {
	model.CicdBuildRun
	ServiceName       string `json:"service_name"`
	ServiceIdentifier string `json:"service_identifier"`
}

func (s *Service) ListBuildRuns(ctx context.Context, q BuildRunListQuery) (*pagination.Result[BuildRunItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.CicdBuildRun{})
	if q.ProjectID > 0 {
		dbq = dbq.Where("project_id = ?", q.ProjectID)
	}
	if q.ServiceID > 0 {
		dbq = dbq.Where("service_id = ?", q.ServiceID)
	} else if q.ProjectID > 0 && q.Actor != nil {
		unrestricted, ids, err := s.visibleCicdServiceScope(ctx, q.ProjectID, q.Actor)
		if err != nil {
			return nil, err
		}
		if !unrestricted {
			if len(ids) == 0 {
				return &pagination.Result[BuildRunItem]{List: []BuildRunItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
			}
			dbq = dbq.Where("service_id IN ?", ids)
		}
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("builder_name LIKE ? OR branch_name LIKE ?", like, like)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdBuildRun
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	s.enrichBuildRunPackagePaths(ctx, rows)
	svcNames := s.loadServiceNameMap(ctx, rows)
	items := make([]BuildRunItem, 0, len(rows))
	for _, row := range rows {
		item := BuildRunItem{CicdBuildRun: row}
		if meta, ok := svcNames[row.ServiceID]; ok {
			item.ServiceName = meta.Name
			item.ServiceIdentifier = meta.Identifier
		}
		items = append(items, item)
	}
	return &pagination.Result[BuildRunItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type ReleaseRunListQuery struct {
	ProjectID           uint              `form:"project_id"`
	ServiceID           uint              `form:"service_id"`
	Status              string            `form:"status"`
	ReleaseType         string            `form:"release_type"`
	Tenv                string            `form:"tenv"`
	Keyword             string            `form:"keyword"`
	Mine                bool              `form:"mine"`
	MineScope           string            `form:"mine_scope"` // pending | done | all（与 mine 联用）
	Page                int               `form:"page"`
	PageSize            int               `form:"page_size"`
	ApproverUserID      *uint             // 内部：待审核
	ExecutorUserID      *uint             // 内部：待执行（提交人）
	ApprovalDoneUserID  *uint             // 内部：我已审批
	ExecutionDoneUserID *uint             // 内部：我已执行
	ApprovalMineUserID  *uint             // 内部：审批待办全部
	ExecutionMineUserID *uint             // 内部：执行待办全部
	MineTab             string            `form:"-"` // approval | execution（mine 待办列表）
	MineViewerUserID    *uint             `form:"-"`
	Actor               *auth.CurrentUser `form:"-"`
}

type ReleaseRunItem struct {
	model.CicdReleaseRun
	ServiceName       string `json:"service_name"`
	ServiceIdentifier string `json:"service_identifier"`
	ProjectName       string `json:"project_name"`
	CurrentStageName  string `json:"current_stage_name,omitempty"`
	MineStatus        string `json:"mine_status,omitempty"` // mine_pending | mine_done（待办列表按当前用户视角）
}

func (s *Service) ListReleaseRuns(ctx context.Context, q ReleaseRunListQuery) (*pagination.Result[ReleaseRunItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if q.ProjectID > 0 && strings.TrimSpace(q.Status) == model.CicdRunStatusPendingApproval {
		_ = s.backfillPendingReleaseSteps(ctx, q.ProjectID)
	}
	dbq := s.db.WithContext(ctx).Model(&model.CicdReleaseRun{})
	if q.ProjectID > 0 {
		dbq = dbq.Where("project_id = ?", q.ProjectID)
	}
	if q.ServiceID > 0 {
		dbq = dbq.Where("service_id = ?", q.ServiceID)
	} else if q.ProjectID > 0 && q.Actor != nil {
		unrestricted, ids, err := s.visibleCicdServiceScope(ctx, q.ProjectID, q.Actor)
		if err != nil {
			return nil, err
		}
		if !unrestricted {
			if len(ids) == 0 {
				return &pagination.Result[ReleaseRunItem]{List: []ReleaseRunItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
			}
			dbq = dbq.Where("service_id IN ?", ids)
		}
	}
	if st := strings.TrimSpace(q.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	if rt := strings.TrimSpace(q.ReleaseType); rt != "" {
		dbq = dbq.Where("release_type = ?", rt)
	}
	if env := strings.TrimSpace(q.Tenv); env != "" {
		dbq = dbq.Where("tenv = ?", env)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("title LIKE ? OR submitter_name LIKE ?", like, like)
	}
	if q.ApproverUserID != nil && *q.ApproverUserID > 0 {
		dbq = s.filterReleaseRunsForApprover(dbq, *q.ApproverUserID)
	}
	if q.ApprovalDoneUserID != nil && *q.ApprovalDoneUserID > 0 {
		dbq = s.filterReleaseRunsApprovalDone(dbq, *q.ApprovalDoneUserID)
	}
	if q.ApprovalMineUserID != nil && *q.ApprovalMineUserID > 0 {
		dbq = s.filterReleaseRunsApprovalMine(dbq, *q.ApprovalMineUserID)
	}
	if q.ExecutorUserID != nil && *q.ExecutorUserID > 0 {
		dbq = dbq.Where("status = ?", model.CicdRunStatusPendingExecution).
			Where("submitter_user_id = ?", *q.ExecutorUserID)
	}
	if q.ExecutionDoneUserID != nil && *q.ExecutionDoneUserID > 0 {
		dbq = s.filterReleaseRunsExecutionDone(dbq, *q.ExecutionDoneUserID)
	}
	if q.ExecutionMineUserID != nil && *q.ExecutionMineUserID > 0 {
		dbq = s.filterReleaseRunsExecutionMine(dbq, *q.ExecutionMineUserID)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdReleaseRun
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	svcMap := make(map[uint]model.CicdService)
	for _, row := range rows {
		svcMap[row.ServiceID] = model.CicdService{}
	}
	ids := make([]uint, 0, len(svcMap))
	for id := range svcMap {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		var svcs []model.CicdService
		_ = s.db.WithContext(ctx).Where("id IN ?", ids).Find(&svcs).Error
		for _, svc := range svcs {
			svcMap[svc.ID] = svc
		}
	}
	items := make([]ReleaseRunItem, 0, len(rows))
	projectName := ""
	if q.ProjectID > 0 {
		var proj model.Project
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", q.ProjectID).First(&proj).Error; err == nil {
			projectName = proj.Name
		}
	}
	for _, row := range rows {
		item := ReleaseRunItem{CicdReleaseRun: row, ProjectName: projectName}
		if svc, ok := svcMap[row.ServiceID]; ok {
			item.ServiceName = svc.Name
			item.ServiceIdentifier = svc.Identifier
		}
		if row.CurrentStageKey != "" {
			item.CurrentStageName = stageNameByKey(row.CurrentStageKey)
		}
		items = append(items, item)
	}
	if q.MineViewerUserID != nil && *q.MineViewerUserID > 0 {
		s.enrichReleaseRunMineStatus(ctx, items, *q.MineViewerUserID, strings.TrimSpace(q.MineTab))
	}
	return &pagination.Result[ReleaseRunItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetBuildRun(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) (*BuildRunItem, error) {
	var row model.CicdBuildRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, row.ServiceID, actor, "view"); err != nil {
		return nil, err
	}
	item := BuildRunItem{CicdBuildRun: row}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err == nil {
		item.ServiceName = svc.Name
		item.ServiceIdentifier = svc.Identifier
	}
	return &item, nil
}

func (s *Service) GetBuildRunLog(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) (string, error) {
	row, err := s.GetBuildRun(ctx, projectID, runID, actor)
	if err != nil {
		return "", err
	}
	if row.BuildNumber <= 0 {
		return "", constants.ErrBadRequestWithMsg("构建编号尚未就绪")
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err != nil {
		return "", err
	}
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return "", err
	}
	return client.GetConsoleLog(ctx, svc.JenkinsJob, row.BuildNumber)
}

func (s *Service) GetReleaseRunLog(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) (string, error) {
	var row model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return "", constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, row.ServiceID, actor, "view"); err != nil {
		return "", err
	}
	if row.JenkinsBuildNumber <= 0 {
		return "", nil
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err != nil {
		return "", err
	}
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return "", err
	}
	return client.GetConsoleLog(ctx, svc.JenkinsJob, row.JenkinsBuildNumber)
}

func (s *Service) DeleteBuildRun(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) error {
	var row model.CicdBuildRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, row.ServiceID, actor, "manage"); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdBuildRun{}).Error
}

func (s *Service) DeleteReleaseRun(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) error {
	release, err := s.assertReleaseRunAccess(ctx, projectID, runID, actor, "manage")
	if err != nil {
		return err
	}
	_ = release
	return s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdReleaseRun{}).Error
}

type serviceMeta struct {
	Name       string
	Identifier string
}

func (s *Service) loadServiceNameMap(ctx context.Context, runs []model.CicdBuildRun) map[uint]serviceMeta {
	out := make(map[uint]serviceMeta)
	if len(runs) == 0 {
		return out
	}
	ids := make([]uint, 0, len(runs))
	seen := make(map[uint]struct{})
	for _, r := range runs {
		if _, ok := seen[r.ServiceID]; ok {
			continue
		}
		seen[r.ServiceID] = struct{}{}
		ids = append(ids, r.ServiceID)
	}
	var svcs []model.CicdService
	_ = s.db.WithContext(ctx).Where("id IN ?", ids).Find(&svcs).Error
	for _, svc := range svcs {
		out[svc.ID] = serviceMeta{Name: svc.Name, Identifier: svc.Identifier}
	}
	return out
}

func (s *Service) enrichBuildRunPackagePaths(ctx context.Context, rows []model.CicdBuildRun) {
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return
	}
	for i := range rows {
		if rows[i].BuildResult != model.CicdRunStatusSuccess {
			continue
		}
		if strings.TrimSpace(rows[i].PackagePath) != "" && strings.TrimSpace(rows[i].ImageAddress) != "" {
			continue
		}
		s.backfillBuildArtifacts(ctx, client, rows[i])
		var updated model.CicdBuildRun
		if err := s.db.WithContext(ctx).Select("package_path", "image_address").Where("id = ?", rows[i].ID).First(&updated).Error; err == nil {
			rows[i].PackagePath = updated.PackagePath
			rows[i].ImageAddress = updated.ImageAddress
		}
	}
}
