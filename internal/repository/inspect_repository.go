package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type InspectRepository struct {
	db *gorm.DB
}

func NewInspectRepository(db *gorm.DB) InspectRepo {
	if db == nil {
		return &InspectRepository{}
	}
	return &InspectRepository{db: db}
}

func (r *InspectRepository) GetPlanByProject(ctx context.Context, projectID uint) (*model.InspectPlan, error) {
	var plan model.InspectPlan
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *InspectRepository) GetPlanByID(ctx context.Context, id uint) (*model.InspectPlan, error) {
	var plan model.InspectPlan
	if err := r.db.WithContext(ctx).First(&plan, id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *InspectRepository) CreatePlan(ctx context.Context, plan *model.InspectPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *InspectRepository) SavePlan(ctx context.Context, plan *model.InspectPlan) error {
	return r.db.WithContext(ctx).Save(plan).Error
}

func (r *InspectRepository) UpdatePlanLastRunAt(ctx context.Context, planID uint, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.InspectPlan{}).Where("id = ?", planID).Update("last_run_at", at).Error
}

func (r *InspectRepository) ListEnabledPlans(ctx context.Context) ([]model.InspectPlan, error) {
	var list []model.InspectPlan
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND datasource_id > 0 AND cron_spec <> ''", true).
		Find(&list).Error
	return list, err
}

func (r *InspectRepository) ListPlansWithRetain(ctx context.Context) ([]model.InspectPlan, error) {
	var list []model.InspectPlan
	err := r.db.WithContext(ctx).Where("retain_days > 0").Find(&list).Error
	return list, err
}

func (r *InspectRepository) FirstEnabledPrometheusDS(ctx context.Context, projectID uint) (*model.AlertDatasource, error) {
	var ds model.AlertDatasource
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND enabled = ? AND type = ?", projectID, true, "prometheus").
		Order("id ASC").First(&ds).Error
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *InspectRepository) GetDatasource(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	var ds model.AlertDatasource
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ds).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *InspectRepository) CountItemsByProject(ctx context.Context, projectID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.InspectItem{}).Where("project_id = ?", projectID).Count(&n).Error
	return n, err
}

func (r *InspectRepository) ListItemsByProject(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var list []model.InspectItem
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *InspectRepository) ListGlobalItems(ctx context.Context) ([]model.InspectItem, error) {
	var list []model.InspectItem
	err := r.db.WithContext(ctx).Where("project_id = 0").Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *InspectRepository) ListEnabledItemsByProject(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var list []model.InspectItem
	err := r.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).
		Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *InspectRepository) ListEnabledGlobalItems(ctx context.Context) ([]model.InspectItem, error) {
	var list []model.InspectItem
	err := r.db.WithContext(ctx).Where("project_id = 0 AND enabled = ?", true).
		Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *InspectRepository) GetItem(ctx context.Context, projectID, itemID uint) (*model.InspectItem, error) {
	var item model.InspectItem
	err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *InspectRepository) CreateItem(ctx context.Context, item *model.InspectItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *InspectRepository) SaveItem(ctx context.Context, item *model.InspectItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *InspectRepository) DeleteItem(ctx context.Context, projectID, itemID uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).Delete(&model.InspectItem{})
	return res.RowsAffected, res.Error
}

func (r *InspectRepository) DeleteItemsByProject(ctx context.Context, projectID uint) error {
	return r.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&model.InspectItem{}).Error
}

func (r *InspectRepository) GetGlobalItemByTypeName(ctx context.Context, typ, name string) (*model.InspectItem, error) {
	var row model.InspectItem
	err := r.db.WithContext(ctx).
		Where("project_id = 0 AND type = ? AND name = ?", typ, name).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *InspectRepository) UpdateItemLinkedRuleID(ctx context.Context, itemID, ruleID uint) error {
	return r.db.WithContext(ctx).Model(&model.InspectItem{}).Where("id = ?", itemID).
		Update("linked_rule_id", ruleID).Error
}

func (r *InspectRepository) ListRuns(ctx context.Context, p InspectRunListParams) ([]model.InspectRun, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.InspectRun{}).Where("project_id = ?", p.ProjectID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InspectRun
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *InspectRepository) ListRunTrends(ctx context.Context, projectID uint, limit int) ([]model.InspectRun, error) {
	var list []model.InspectRun
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func (r *InspectRepository) GetRun(ctx context.Context, projectID, runID uint) (*model.InspectRun, error) {
	var run model.InspectRun
	err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *InspectRepository) GetRunByID(ctx context.Context, runID uint) (*model.InspectRun, error) {
	var run model.InspectRun
	if err := r.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *InspectRepository) CreateRun(ctx context.Context, run *model.InspectRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *InspectRepository) SaveRun(ctx context.Context, run *model.InspectRun) error {
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *InspectRepository) UpdateRunFields(ctx context.Context, runID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.InspectRun{}).Where("id = ?", runID).Updates(fields).Error
}

func (r *InspectRepository) MarkRunRunning(ctx context.Context, runID uint, startedAt time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.InspectRun{}).
		Where("id = ? AND status IN ?", runID, []string{"pending", "running"}).
		Updates(map[string]any{"status": "running", "started_at": startedAt})
	return res.RowsAffected, res.Error
}

func (r *InspectRepository) ReclaimOrphanRunning(ctx context.Context, staleBefore, finishedAt time.Time, errMsg string) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.InspectRun{}).
		Where("status = ? AND (started_at IS NULL OR started_at < ?)", "running", staleBefore).
		Updates(map[string]any{
			"status": "failed", "error_message": errMsg, "finished_at": finishedAt,
		})
	return res.RowsAffected, res.Error
}

func (r *InspectRepository) ReclaimStaleRunning(ctx context.Context, cutoff, finishedAt time.Time, errMsg string) error {
	return r.db.WithContext(ctx).Model(&model.InspectRun{}).
		Where("status = ? AND started_at IS NOT NULL AND started_at < ?", "running", cutoff).
		Updates(map[string]any{
			"status": "failed", "error_message": errMsg, "finished_at": finishedAt,
		}).Error
}

func (r *InspectRepository) ListPendingRuns(ctx context.Context, limit int) ([]model.InspectRun, error) {
	var list []model.InspectRun
	err := r.db.WithContext(ctx).Where("status = ?", "pending").Order("id ASC").Limit(limit).Find(&list).Error
	return list, err
}

func (r *InspectRepository) ListExpiredSuccessRuns(ctx context.Context, projectID uint, cutoff time.Time) ([]model.InspectRun, error) {
	var list []model.InspectRun
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND created_at < ? AND status = ?", projectID, cutoff, "success").
		Find(&list).Error
	return list, err
}

func (r *InspectRepository) DeleteRun(ctx context.Context, run *model.InspectRun) error {
	return r.db.WithContext(ctx).Delete(run).Error
}

func (r *InspectRepository) ListLocalStorageRuns(ctx context.Context, projectID uint) ([]model.InspectRun, error) {
	var list []model.InspectRun
	err := r.db.WithContext(ctx).Where("project_id = ? AND storage = ?", projectID, "local").Find(&list).Error
	return list, err
}

func (r *InspectRepository) PluckPreviousFindingRunID(ctx context.Context, projectID, excludeRunID uint) (uint, error) {
	var prevID uint
	err := r.db.WithContext(ctx).Model(&model.InspectFinding{}).
		Where("project_id = ? AND run_id <> ?", projectID, excludeRunID).
		Order("run_id DESC").Limit(1).Pluck("run_id", &prevID).Error
	return prevID, err
}

func (r *InspectRepository) ListFindingsByRun(ctx context.Context, projectID, runID uint) ([]model.InspectFinding, error) {
	var list []model.InspectFinding
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND run_id = ? AND state <> ?", projectID, runID, "recovered").
		Find(&list).Error
	return list, err
}

func (r *InspectRepository) CreateFindingsBatch(ctx context.Context, rows []model.InspectFinding) error {
	return r.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func (r *InspectRepository) GetReportTemplate(ctx context.Context, projectID, templateID uint) (*model.InspectReportTemplate, error) {
	var row model.InspectReportTemplate
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = 1 AND (project_id = 0 OR project_id = ?)", templateID, projectID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *InspectRepository) GetGlobalDefaultReportTemplate(ctx context.Context) (*model.InspectReportTemplate, error) {
	var def model.InspectReportTemplate
	err := r.db.WithContext(ctx).
		Where("project_id = 0 AND code = ? AND status = 1", "default").First(&def).Error
	if err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *InspectRepository) GetGlobalReportTemplateByCode(ctx context.Context, code string) (*model.InspectReportTemplate, error) {
	var row model.InspectReportTemplate
	err := r.db.WithContext(ctx).
		Where("project_id = 0 AND code = ?", code).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *InspectRepository) ListReportTemplates(ctx context.Context, projectID uint) ([]model.InspectReportTemplate, error) {
	var list []model.InspectReportTemplate
	err := r.db.WithContext(ctx).
		Where("(project_id = 0 OR project_id = ?) AND status = 1", projectID).
		Order("project_id ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *InspectRepository) GetProjectReportTemplate(ctx context.Context, projectID, tid uint) (*model.InspectReportTemplate, error) {
	var row model.InspectReportTemplate
	err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", tid, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *InspectRepository) GetCopyableReportTemplate(ctx context.Context, projectID, sourceID uint) (*model.InspectReportTemplate, error) {
	var row model.InspectReportTemplate
	err := r.db.WithContext(ctx).
		Where("id = ? AND (project_id = 0 OR project_id = ?)", sourceID, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *InspectRepository) CreateReportTemplate(ctx context.Context, row *model.InspectReportTemplate) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *InspectRepository) SaveReportTemplate(ctx context.Context, row *model.InspectReportTemplate) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *InspectRepository) DeleteProjectReportTemplate(ctx context.Context, projectID, tid uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("id = ? AND project_id = ? AND is_builtin = ?", tid, projectID, false).
		Delete(&model.InspectReportTemplate{})
	return res.RowsAffected, res.Error
}

func (r *InspectRepository) GetMonitorRule(ctx context.Context, id uint) (*model.AlertMonitorRule, error) {
	var rule model.AlertMonitorRule
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *InspectRepository) SaveMonitorRule(ctx context.Context, rule *model.AlertMonitorRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *InspectRepository) CreateMonitorRuleLinkItem(ctx context.Context, rule *model.AlertMonitorRule, itemID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rule).Error; err != nil {
			return err
		}
		return tx.Model(&model.InspectItem{}).Where("id = ?", itemID).
			Update("linked_rule_id", rule.ID).Error
	})
}

// ClaimPlanLastRunAt updates last_run_at only when still matching expected (scheduler claim).
func (r *InspectRepository) ClaimPlanLastRunAt(ctx context.Context, planID uint, expected *time.Time, claimed time.Time) (int64, error) {
	q := r.db.WithContext(ctx).Model(&model.InspectPlan{}).Where("id = ?", planID)
	if expected == nil {
		q = q.Where("last_run_at IS NULL")
	} else {
		q = q.Where("last_run_at = ?", *expected)
	}
	res := q.Update("last_run_at", claimed)
	return res.RowsAffected, res.Error
}
