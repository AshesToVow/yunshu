package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// --- Build runs ---

func (r *CicdRepository) ListBuildRuns(ctx context.Context, p CicdBuildRunListParams) ([]model.CicdBuildRun, int64, error) {
	q := r.dbq(ctx).Model(&model.CicdBuildRun{})
	if p.ProjectID > 0 {
		q = q.Where("project_id = ?", p.ProjectID)
	}
	if p.ServiceID > 0 {
		q = q.Where("service_id = ?", p.ServiceID)
	} else if p.RestrictSvc {
		if len(p.ServiceIDs) == 0 {
			return nil, 0, nil
		}
		q = q.Where("service_id IN ?", p.ServiceIDs)
	} else if len(p.ServiceIDs) > 0 {
		q = q.Where("service_id IN ?", p.ServiceIDs)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("builder_name LIKE ? OR branch_name LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CicdBuildRun
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) GetBuildRun(ctx context.Context, projectID, runID uint) (*model.CicdBuildRun, error) {
	var row model.CicdBuildRun
	err := r.dbq(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetBuildRunByID(ctx context.Context, runID uint) (*model.CicdBuildRun, error) {
	var row model.CicdBuildRun
	err := r.dbq(ctx).Where("id = ?", runID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateBuildRun(ctx context.Context, row *model.CicdBuildRun) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveBuildRun(ctx context.Context, row *model.CicdBuildRun) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) UpdateBuildRunFields(ctx context.Context, runID uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.CicdBuildRun{}).Where("id = ?", runID).Updates(fields).Error
}

func (r *CicdRepository) UpdateBuildRunFieldsIfStatus(ctx context.Context, runID uint, statuses []string, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdBuildRun{}).
		Where("id = ? AND build_result IN ?", runID, statuses).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) DeleteBuildRun(ctx context.Context, projectID, runID uint) error {
	return r.dbq(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdBuildRun{}).Error
}

func (r *CicdRepository) ListActiveBuildRuns(ctx context.Context, limit int) ([]model.CicdBuildRun, error) {
	var rows []model.CicdBuildRun
	err := r.dbq(ctx).
		Where("build_result IN ?", []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}).
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListSuccessfulBuildRunsMissingArtifacts(ctx context.Context, limit int) ([]model.CicdBuildRun, error) {
	var rows []model.CicdBuildRun
	q := r.dbq(ctx).
		Where("build_result = ? AND (package_path = '' OR package_path IS NULL OR image_address = '' OR image_address IS NULL)", model.CicdRunStatusSuccess).
		Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) FindBuildRunByServiceJenkins(ctx context.Context, serviceID uint, buildNumber int, queueID int64) (*model.CicdBuildRun, error) {
	q := r.dbq(ctx).Where("service_id = ?", serviceID)
	switch {
	case buildNumber > 0 && queueID > 0:
		q = q.Where("build_number = ? OR jenkins_queue_id = ?", buildNumber, queueID)
	case buildNumber > 0:
		q = q.Where("build_number = ?", buildNumber)
	case queueID > 0:
		q = q.Where("jenkins_queue_id = ?", queueID)
	default:
		return nil, gorm.ErrRecordNotFound
	}
	var br model.CicdBuildRun
	err := q.Order("id DESC").First(&br).Error
	if err != nil {
		return nil, err
	}
	return &br, nil
}

// --- Release runs ---

func (r *CicdRepository) ListReleaseRuns(ctx context.Context, p CicdReleaseRunListParams) ([]model.CicdReleaseRun, int64, error) {
	q := r.dbq(ctx).Model(&model.CicdReleaseRun{})
	q = r.applyReleaseRunListParams(q, p)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CicdReleaseRun
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) applyReleaseRunListParams(q *gorm.DB, p CicdReleaseRunListParams) *gorm.DB {
	if p.ProjectID > 0 {
		q = q.Where("project_id = ?", p.ProjectID)
	}
	if p.ServiceID > 0 {
		q = q.Where("service_id = ?", p.ServiceID)
	} else if p.RestrictSvc {
		if len(p.ServiceIDs) == 0 {
			return q.Where("1 = 0")
		}
		q = q.Where("service_id IN ?", p.ServiceIDs)
	} else if len(p.ServiceIDs) > 0 {
		q = q.Where("service_id IN ?", p.ServiceIDs)
	}
	if st := strings.TrimSpace(p.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	if rt := strings.TrimSpace(p.ReleaseType); rt != "" {
		q = q.Where("release_type = ?", rt)
	}
	if env := strings.TrimSpace(p.Tenv); env != "" {
		q = q.Where("tenv = ?", env)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR submitter_name LIKE ?", like, like)
	}
	if p.ApproverUserID != nil && *p.ApproverUserID > 0 {
		q = r.filterReleaseRunsForApprover(q, *p.ApproverUserID)
	}
	if p.ApprovalDoneUserID != nil && *p.ApprovalDoneUserID > 0 {
		q = r.filterReleaseRunsApprovalDone(q, *p.ApprovalDoneUserID)
	}
	if p.ApprovalMineUserID != nil && *p.ApprovalMineUserID > 0 {
		q = r.filterReleaseRunsApprovalMine(q, *p.ApprovalMineUserID)
	}
	if p.ExecutionUserID != nil && *p.ExecutionUserID > 0 {
		q = q.Where("status = ?", model.CicdRunStatusPendingExecution).
			Where("submitter_user_id = ?", *p.ExecutionUserID)
	}
	if p.ExecutionDoneUserID != nil && *p.ExecutionDoneUserID > 0 {
		q = r.filterReleaseRunsExecutionDone(q, *p.ExecutionDoneUserID)
	}
	if p.ExecutionMineUserID != nil && *p.ExecutionMineUserID > 0 {
		q = r.filterReleaseRunsExecutionMine(q, *p.ExecutionMineUserID)
	}
	return q
}

func (r *CicdRepository) filterReleaseRunsForApprover(dbq *gorm.DB, userID uint) *gorm.DB {
	wfPending := r.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = cicd_release_runs.id AND t.ticket_type = ? AND t.deleted_at IS NULL",
			model.WorkflowRefCicdReleaseRun, model.WorkflowTicketTypeRelease).
		Where("t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL",
			model.WorkflowTicketStatusPending, model.WorkflowStepPending).
		Where(`(s.assignee_user_id = ? OR (s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
			SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
		)))`, userID, userID)
	noSteps := r.db.Table("cicd_release_approval_steps AS s0").
		Select("1").
		Where("s0.release_run_id = cicd_release_runs.id")
	noWF := r.db.Table("workflow_tickets AS tw").
		Select("1").
		Where("tw.ref_type = ? AND tw.ref_id = cicd_release_runs.id AND tw.ticket_type = ? AND tw.deleted_at IS NULL",
			model.WorkflowRefCicdReleaseRun, model.WorkflowTicketTypeRelease)
	currentStep := r.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.release_run_id = cicd_release_runs.id").
		Where("s.status = ?", model.CicdApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
			WHERE s2.release_run_id = cicd_release_runs.id AND s2.status = ?
		)`, model.CicdApprovalStepPending)
	return dbq.Where("cicd_release_runs.status = ?", model.CicdRunStatusPendingApproval).
		Where("EXISTS (?) OR (NOT EXISTS (?) AND NOT EXISTS (?)) OR EXISTS (?)",
			wfPending, noWF, noSteps, currentStep)
}

func (r *CicdRepository) filterReleaseRunsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	wfActed := r.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = cicd_release_runs.id AND t.ticket_type = ? AND t.deleted_at IS NULL",
			model.WorkflowRefCicdReleaseRun, model.WorkflowTicketTypeRelease).
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.WorkflowStepApproved, model.WorkflowStepRejected})
	actedStep := r.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Where("s.release_run_id = cicd_release_runs.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.CicdApprovalStepApproved, model.CicdApprovalStepRejected})
	legacy := r.db.Table("cicd_release_runs AS lr").
		Select("1").
		Where("lr.id = cicd_release_runs.id").
		Where("lr.reviewer_user_id = ?", userID).
		Where("lr.reviewed_at IS NOT NULL")
	return dbq.Where("EXISTS (?) OR EXISTS (?) OR EXISTS (?)", wfActed, actedStep, legacy)
}

func (r *CicdRepository) filterReleaseRunsExecutionDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("submitter_user_id = ?", userID).
		Where("status NOT IN ?", []string{
			model.CicdRunStatusPendingApproval,
			model.CicdRunStatusPendingExecution,
		})
}

func (r *CicdRepository) filterReleaseRunsApprovalMine(dbq *gorm.DB, userID uint) *gorm.DB {
	pending := r.approvalPendingExistsSubquery(userID)
	done := r.approvalDoneExistsSubquery(userID)
	return dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
}

func (r *CicdRepository) filterReleaseRunsExecutionMine(dbq *gorm.DB, userID uint) *gorm.DB {
	pending := r.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status = ?", model.CicdRunStatusPendingExecution)
	done := r.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status NOT IN ?", []string{
			model.CicdRunStatusPendingApproval,
			model.CicdRunStatusPendingExecution,
		})
	return dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
}

func (r *CicdRepository) approvalPendingExistsSubquery(userID uint) *gorm.DB {
	noSteps := r.db.Table("cicd_release_approval_steps AS s0").
		Select("1").
		Where("s0.release_run_id = r.id")
	currentStep := r.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.release_run_id = r.id").
		Where("s.status = ?", model.CicdApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
			WHERE s2.release_run_id = r.id AND s2.status = ?
		)`, model.CicdApprovalStepPending)
	return r.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.status = ?", model.CicdRunStatusPendingApproval).
		Where("NOT EXISTS (?) OR EXISTS (?)", noSteps, currentStep)
}

func (r *CicdRepository) approvalDoneExistsSubquery(userID uint) *gorm.DB {
	actedStep := r.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Where("s.release_run_id = r.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.CicdApprovalStepApproved, model.CicdApprovalStepRejected})
	legacy := r.db.Table("cicd_release_runs AS lr").
		Select("1").
		Where("lr.id = r.id").
		Where("lr.reviewer_user_id = ?", userID).
		Where("lr.reviewed_at IS NOT NULL")
	return r.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("EXISTS (?) OR EXISTS (?)", actedStep, legacy)
}

func (r *CicdRepository) GetReleaseRun(ctx context.Context, projectID, runID uint) (*model.CicdReleaseRun, error) {
	var row model.CicdReleaseRun
	err := r.dbq(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetReleaseRunByID(ctx context.Context, runID uint) (*model.CicdReleaseRun, error) {
	var row model.CicdReleaseRun
	err := r.dbq(ctx).Where("id = ?", runID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateReleaseRun(ctx context.Context, row *model.CicdReleaseRun) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveReleaseRun(ctx context.Context, row *model.CicdReleaseRun) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) UpdateReleaseRunFields(ctx context.Context, runID uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.CicdReleaseRun{}).Where("id = ?", runID).Updates(fields).Error
}

func (r *CicdRepository) UpdateReleaseRunFieldsIfStatus(ctx context.Context, runID uint, statuses []string, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdReleaseRun{}).
		Where("id = ? AND status IN ?", runID, statuses).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) DeleteReleaseRun(ctx context.Context, projectID, runID uint) error {
	return r.dbq(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdReleaseRun{}).Error
}

func (r *CicdRepository) ListActiveReleaseRuns(ctx context.Context, limit int) ([]model.CicdReleaseRun, error) {
	var rows []model.CicdReleaseRun
	err := r.dbq(ctx).
		Where("status IN ?", []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}).
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListStuckPendingExecutionReleases(ctx context.Context, olderThan time.Time, limit int) ([]model.CicdReleaseRun, error) {
	var rows []model.CicdReleaseRun
	q := r.dbq(ctx).
		Where("status = ? AND updated_at < ?", model.CicdRunStatusPendingExecution, olderThan).
		Order("updated_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) FindReleaseRunByServiceJenkins(ctx context.Context, serviceID uint, buildNumber int, queueID int64) (*model.CicdReleaseRun, error) {
	q := r.dbq(ctx).Where("service_id = ?", serviceID)
	switch {
	case buildNumber > 0:
		q = q.Where("jenkins_build_number = ?", buildNumber)
	case queueID > 0:
		q = q.Where("jenkins_queue_url LIKE ?", "%/queue/item/"+strconv.FormatInt(queueID, 10)+"/%")
	default:
		return nil, gorm.ErrRecordNotFound
	}
	var rr model.CicdReleaseRun
	err := q.Order("id DESC").First(&rr).Error
	if err != nil {
		return nil, err
	}
	return &rr, nil
}

func (r *CicdRepository) ListPendingApprovalReleases(ctx context.Context, projectID uint) ([]model.CicdReleaseRun, error) {
	var runs []model.CicdReleaseRun
	q := r.dbq(ctx).
		Where("status = ? AND audit_enabled = ?", model.CicdRunStatusPendingApproval, true)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	err := q.Find(&runs).Error
	return runs, err
}

func (r *CicdRepository) ClaimReleaseRunStatus(ctx context.Context, runID uint, fromStatuses []string, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdReleaseRun{}).
		Where("id = ? AND status IN ?", runID, fromStatuses).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) UpdateReleaseRunFieldsIfBuildNumberZero(ctx context.Context, runID uint, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdReleaseRun{}).
		Where("id = ? AND jenkins_build_number = 0", runID).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) ClaimReleaseRunNoBuildNumber(ctx context.Context, runID uint, fromStatus string, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdReleaseRun{}).
		Where("id = ? AND jenkins_build_number = 0 AND status = ?", runID, fromStatus).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) ListBuildRunsByImageAddress(ctx context.Context, imageAddress string, limit int) ([]model.CicdBuildRun, error) {
	imageAddress = strings.TrimSpace(imageAddress)
	if imageAddress == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []model.CicdBuildRun
	err := r.dbq(ctx).
		Select("id", "project_id", "service_id", "build_number", "image_address").
		Where("image_address = ? OR image_address LIKE ?", imageAddress, imageAddress+"%").
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// --- Stages / Artifacts ---

func (r *CicdRepository) GetRunStage(ctx context.Context, runKind string, runID uint, stageType string, stageOrder int) (*model.CicdRunStage, error) {
	var existing model.CicdRunStage
	err := r.dbq(ctx).
		Where("run_kind = ? AND run_id = ? AND stage_type = ? AND stage_order = ?",
			runKind, runID, stageType, stageOrder).
		First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *CicdRepository) CreateRunStage(ctx context.Context, row *model.CicdRunStage) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) UpdateRunStage(ctx context.Context, row *model.CicdRunStage, fields map[string]any) error {
	return r.dbq(ctx).Model(row).Updates(fields).Error
}

func (r *CicdRepository) ListRunStages(ctx context.Context, runKind string, runID uint) ([]model.CicdRunStage, error) {
	var rows []model.CicdRunStage
	err := r.dbq(ctx).
		Where("run_kind = ? AND run_id = ?", runKind, runID).
		Order("stage_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) FindArtifact(ctx context.Context, buildRunID uint, artifactType, name, version string) (*model.CicdArtifact, error) {
	q := r.dbq(ctx).Where("build_run_id = ? AND artifact_type = ?", buildRunID, artifactType)
	if v := strings.TrimSpace(name); v != "" {
		q = q.Where("name = ?", v)
	}
	if v := strings.TrimSpace(version); v != "" {
		q = q.Where("digest = ? OR name LIKE ?", v, "%"+v+"%")
	}
	var existing model.CicdArtifact
	err := q.First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *CicdRepository) FindArtifactByStoragePath(ctx context.Context, buildRunID uint, artifactType, storagePath string) (*model.CicdArtifact, error) {
	path := strings.TrimSpace(storagePath)
	if path == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var existing model.CicdArtifact
	err := r.dbq(ctx).
		Where("build_run_id = ? AND artifact_type = ? AND storage_path = ?", buildRunID, artifactType, path).
		First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *CicdRepository) CreateArtifact(ctx context.Context, row *model.CicdArtifact) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) UpdateArtifact(ctx context.Context, row *model.CicdArtifact, fields map[string]any) error {
	return r.dbq(ctx).Model(row).Updates(fields).Error
}

func (r *CicdRepository) ListArtifactsByBuildRun(ctx context.Context, buildRunID uint) ([]model.CicdArtifact, error) {
	var rows []model.CicdArtifact
	err := r.dbq(ctx).Where("build_run_id = ?", buildRunID).Order("id ASC").Find(&rows).Error
	return rows, err
}
