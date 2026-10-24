package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Approval steps ---

func (r *CicdRepository) CreateApprovalSteps(ctx context.Context, steps []model.CicdReleaseApprovalStep) error {
	if len(steps) == 0 {
		return nil
	}
	return r.dbq(ctx).Create(&steps).Error
}

func (r *CicdRepository) ListApprovalStepsByRun(ctx context.Context, releaseRunID uint) ([]model.CicdReleaseApprovalStep, error) {
	var steps []model.CicdReleaseApprovalStep
	err := r.dbq(ctx).
		Where("release_run_id = ?", releaseRunID).
		Order("sort_order ASC, id ASC").
		Find(&steps).Error
	return steps, err
}

func (r *CicdRepository) ListApprovalStepsByRuns(ctx context.Context, runIDs []uint) ([]model.CicdReleaseApprovalStep, error) {
	if len(runIDs) == 0 {
		return nil, nil
	}
	var steps []model.CicdReleaseApprovalStep
	err := r.dbq(ctx).
		Where("release_run_id IN ?", runIDs).
		Order("sort_order ASC, id ASC").
		Find(&steps).Error
	return steps, err
}

func (r *CicdRepository) GetPendingApprovalStep(ctx context.Context, releaseRunID uint) (*model.CicdReleaseApprovalStep, error) {
	var step model.CicdReleaseApprovalStep
	err := r.dbq(ctx).
		Where("release_run_id = ? AND status = ?", releaseRunID, model.CicdApprovalStepPending).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

func (r *CicdRepository) UpdateApprovalStepFields(ctx context.Context, stepID uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.CicdReleaseApprovalStep{}).Where("id = ?", stepID).Updates(fields).Error
}

func (r *CicdRepository) ActivateApprovalStep(ctx context.Context, releaseRunID uint, stageKey string) error {
	now := time.Now()
	return r.dbq(ctx).Model(&model.CicdReleaseApprovalStep{}).
		Where("release_run_id = ? AND stage_key = ? AND status = ?", releaseRunID, stageKey, model.CicdApprovalStepPending).
		Updates(map[string]any{"activated_at": now, "last_reminded_at": nil}).Error
}

func (r *CicdRepository) CountApprovalSteps(ctx context.Context, releaseRunID uint) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(&model.CicdReleaseApprovalStep{}).
		Where("release_run_id = ?", releaseRunID).
		Count(&n).Error
	return n, err
}

func (r *CicdRepository) ClaimPendingApprovalStep(ctx context.Context, stepID uint, fields map[string]any) (int64, error) {
	res := r.dbq(ctx).Model(&model.CicdReleaseApprovalStep{}).
		Where("id = ? AND status = ?", stepID, model.CicdApprovalStepPending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) ListStalePendingApprovalSteps(ctx context.Context, olderThan time.Time) ([]model.CicdReleaseApprovalStep, error) {
	var steps []model.CicdReleaseApprovalStep
	err := r.dbq(ctx).
		Joins("JOIN cicd_release_runs r ON r.id = cicd_release_approval_steps.release_run_id").
		Where("r.status = ? AND cicd_release_approval_steps.status = ?", model.CicdRunStatusPendingApproval, model.CicdApprovalStepPending).
		Where("cicd_release_approval_steps.activated_at IS NOT NULL AND cicd_release_approval_steps.activated_at < ?", olderThan).
		Find(&steps).Error
	return steps, err
}

func (r *CicdRepository) MarkApprovalStepsReminded(ctx context.Context, ids []uint, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return r.dbq(ctx).Model(&model.CicdReleaseApprovalStep{}).
		Where("id IN ?", ids).
		Update("last_reminded_at", at).Error
}

func (r *CicdRepository) ListLegacyApprovalReminderSeedIDs(ctx context.Context) ([]uint, error) {
	type row struct {
		ID uint
	}
	var ids []row
	err := r.dbq(ctx).Raw(`
SELECT s.id FROM cicd_release_approval_steps s
JOIN cicd_release_runs r ON r.id = s.release_run_id
WHERE r.status = ? AND s.status = ? AND s.activated_at IS NULL
AND s.sort_order = (
  SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
  WHERE s2.release_run_id = s.release_run_id AND s2.status = ?
)`, model.CicdRunStatusPendingApproval, model.CicdApprovalStepPending, model.CicdApprovalStepPending).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	out := make([]uint, len(ids))
	for i, id := range ids {
		out[i] = id.ID
	}
	return out, nil
}

func (r *CicdRepository) ListWorkflowApprovalReminderRows(ctx context.Context) ([]CicdWorkflowReminderRow, error) {
	var list []CicdWorkflowReminderRow
	err := r.dbq(ctx).Raw(`
SELECT s.id AS step_id, s.ticket_id, s.stage_name, s.activated_at, s.last_reminded_at,
       s.user_group_id, s.assignee_user_id, t.ref_id, t.title, t.project_id, t.submitter_user_id
FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.domain = ? AND t.ticket_type = ? AND t.status = ?
  AND s.status = ? AND s.activated_at IS NOT NULL AND s.deleted_at IS NULL
`, model.WorkflowDomainCicd, model.WorkflowTicketTypeRelease, model.WorkflowTicketStatusPending,
		model.WorkflowStepPending).Scan(&list).Error
	return list, err
}

func (r *CicdRepository) ListStaleWorkflowApprovalSteps(ctx context.Context, olderThan time.Time) ([]model.WorkflowTicketStep, error) {
	var steps []model.WorkflowTicketStep
	err := r.dbq(ctx).
		Joins("JOIN workflow_tickets t ON t.id = workflow_ticket_steps.ticket_id AND t.deleted_at IS NULL").
		Where("t.domain = ? AND t.ticket_type = ? AND t.status = ?",
			model.WorkflowDomainCicd, model.WorkflowTicketTypeRelease, model.WorkflowTicketStatusPending).
		Where("workflow_ticket_steps.status = ? AND workflow_ticket_steps.activated_at IS NOT NULL AND workflow_ticket_steps.activated_at < ?",
			model.WorkflowStepPending, olderThan).
		Find(&steps).Error
	return steps, err
}

// --- Access grants ---

func (r *CicdRepository) ListAccessGrants(ctx context.Context, p CicdAccessGrantListParams) ([]model.CicdAccessGrant, int64, error) {
	q := r.dbq(ctx).Model(&model.CicdAccessGrant{})
	if p.ProjectID > 0 {
		q = q.Where("project_id = ?", p.ProjectID)
	}
	if p.ServiceID > 0 {
		q = q.Where("service_id = ?", p.ServiceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CicdAccessGrant
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) UpsertAccessGrant(ctx context.Context, row *model.CicdAccessGrant) error {
	return r.dbq(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "service_id"}, {Name: "principal_kind"}, {Name: "principal_ref"}},
		DoUpdates: clause.AssignmentColumns([]string{"can_view", "can_build", "can_release", "can_manage", "remark", "updated_at"}),
	}).Create(row).Error
}

func (r *CicdRepository) DeleteAccessGrant(ctx context.Context, projectID, grantID uint) (int64, error) {
	res := r.dbq(ctx).Where("id = ? AND project_id = ?", grantID, projectID).Delete(&model.CicdAccessGrant{})
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) ListAccessGrantsForUser(ctx context.Context, projectID, userID uint) ([]model.CicdAccessGrant, error) {
	if userID == 0 {
		return nil, nil
	}
	kind := model.ResourcePrincipalUser
	ref := strconv.FormatUint(uint64(userID), 10)
	var rows []model.CicdAccessGrant
	err := r.dbq(ctx).
		Where("project_id = ? AND principal_kind = ? AND principal_ref = ? AND (can_view = ? OR can_build = ? OR can_release = ? OR can_manage = ?)",
			projectID, kind, ref, true, true, true, true).
		Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListMemberUserIDs(ctx context.Context, projectID uint) ([]model.ProjectMember, error) {
	var rows []model.ProjectMember
	err := r.dbq(ctx).Where("project_id = ?", projectID).Find(&rows).Error
	return rows, err
}

// --- Pipeline templates ---

func (r *CicdRepository) ListEnabledPipelineTemplates(ctx context.Context) ([]model.CicdPipelineTemplate, error) {
	var rows []model.CicdPipelineTemplate
	err := r.dbq(ctx).Where("status = 1").Order("sort ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) GetPipelineTemplate(ctx context.Context, id uint) (*model.CicdPipelineTemplate, error) {
	var row model.CicdPipelineTemplate
	err := r.dbq(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreatePipelineTemplate(ctx context.Context, row *model.CicdPipelineTemplate) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SavePipelineTemplate(ctx context.Context, row *model.CicdPipelineTemplate) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) CountPipelineTemplateByLang(ctx context.Context, languageType string) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(&model.CicdPipelineTemplate{}).
		Where("language_type = ?", languageType).
		Count(&n).Error
	return n, err
}

func (r *CicdRepository) GetPipelineTemplateByLang(ctx context.Context, languageType string) (*model.CicdPipelineTemplate, error) {
	var row model.CicdPipelineTemplate
	err := r.dbq(ctx).Where("language_type = ?", languageType).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetEnabledPipelineTemplateByLang(ctx context.Context, languageType string) (*model.CicdPipelineTemplate, error) {
	var row model.CicdPipelineTemplate
	err := r.dbq(ctx).Where("language_type = ? AND status = 1", languageType).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// --- Registry / Harbor / Cleanup ---

func (r *CicdRepository) GetImageRegistry(ctx context.Context, id uint) (*model.ImageRegistry, error) {
	var reg model.ImageRegistry
	err := r.dbq(ctx).Where("id = ?", id).First(&reg).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *CicdRepository) ListImageRegistries(ctx context.Context, p CicdImageRegistryListParams) ([]model.ImageRegistry, int64, error) {
	q := r.dbq(ctx).Model(&model.ImageRegistry{})
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR url LIKE ?", like, like)
	}
	if typ := strings.TrimSpace(p.Type); typ != "" {
		q = q.Where("type = ?", typ)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.ImageRegistry
	err := q.Order("is_default DESC, id ASC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) CreateImageRegistry(ctx context.Context, row *model.ImageRegistry) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveImageRegistry(ctx context.Context, row *model.ImageRegistry) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteImageRegistryCascade(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx CicdRepo) error {
		tr := tx.(*CicdRepository)
		if err := tr.dbq(ctx).Where("registry_id = ?", id).Delete(&model.ProjectRegistryBinding{}).Error; err != nil {
			return err
		}
		if err := tr.dbq(ctx).Where("registry_id = ?", id).Delete(&model.ImageCleanupPolicy{}).Error; err != nil {
			return err
		}
		res := tr.dbq(ctx).Delete(&model.ImageRegistry{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *CicdRepository) ClearDefaultImageRegistries(ctx context.Context) error {
	return r.dbq(ctx).Model(&model.ImageRegistry{}).Where("is_default = ?", true).Update("is_default", false).Error
}

func (r *CicdRepository) GetDefaultEnabledHarbor(ctx context.Context) (*model.ImageRegistry, error) {
	var reg model.ImageRegistry
	err := r.dbq(ctx).
		Where("is_default = ? AND status = 1 AND type = ?", true, model.ImageRegistryTypeHarbor).
		Order("id ASC").
		First(&reg).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *CicdRepository) GetProjectRegistryBinding(ctx context.Context, projectID uint) (*model.ProjectRegistryBinding, error) {
	var bind model.ProjectRegistryBinding
	err := r.dbq(ctx).Where("project_id = ?", projectID).First(&bind).Error
	if err != nil {
		return nil, err
	}
	return &bind, nil
}

func (r *CicdRepository) CreateProjectRegistryBinding(ctx context.Context, row *model.ProjectRegistryBinding) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveProjectRegistryBinding(ctx context.Context, row *model.ProjectRegistryBinding) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteProjectRegistryBinding(ctx context.Context, projectID uint) (int64, error) {
	res := r.dbq(ctx).Where("project_id = ?", projectID).Delete(&model.ProjectRegistryBinding{})
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) GetProjectHarborFields(ctx context.Context, projectID uint) (harborURL, harborProject string, err error) {
	var p model.Project
	err = r.dbq(ctx).Select("harbor_url", "harbor_project").Where("id = ?", projectID).First(&p).Error
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(p.HarborURL), strings.TrimSpace(p.HarborProject), nil
}

func (r *CicdRepository) UpdateProjectHarborFields(ctx context.Context, projectID uint, harborURL, harborProject string) error {
	return r.dbq(ctx).Model(&model.Project{}).Where("id = ?", projectID).
		Updates(map[string]any{
			"harbor_url":     strings.TrimSpace(harborURL),
			"harbor_project": strings.TrimSpace(harborProject),
		}).Error
}

func (r *CicdRepository) ListCleanupPolicies(ctx context.Context, p CicdCleanupPolicyListParams) ([]model.ImageCleanupPolicy, int64, error) {
	q := r.dbq(ctx).Model(&model.ImageCleanupPolicy{})
	if p.RegistryID != nil && *p.RegistryID > 0 {
		q = q.Where("registry_id = ?", *p.RegistryID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.ImageCleanupPolicy
	err := q.Order("id ASC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) GetCleanupPolicy(ctx context.Context, id uint) (*model.ImageCleanupPolicy, error) {
	var row model.ImageCleanupPolicy
	err := r.dbq(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateCleanupPolicy(ctx context.Context, row *model.ImageCleanupPolicy) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveCleanupPolicy(ctx context.Context, row *model.ImageCleanupPolicy) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteCleanupPolicy(ctx context.Context, id uint) (int64, error) {
	res := r.dbq(ctx).Delete(&model.ImageCleanupPolicy{}, id)
	return res.RowsAffected, res.Error
}

func (r *CicdRepository) ListEnabledCleanupPolicies(ctx context.Context) ([]model.ImageCleanupPolicy, error) {
	var policies []model.ImageCleanupPolicy
	err := r.dbq(ctx).Where("enabled = ?", true).Find(&policies).Error
	return policies, err
}

func (r *CicdRepository) UpdateCleanupPolicyFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.ImageCleanupPolicy{}).Where("id = ?", id).Updates(fields).Error
}
