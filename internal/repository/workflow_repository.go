package repository

import (
	"context"
	"errors"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

var errNilWorkflowDB = gorm.ErrInvalidDB

type WorkflowRepository struct {
	db *gorm.DB
}

func NewWorkflowRepository(db *gorm.DB) WorkflowRepo {
	if db == nil {
		return &WorkflowRepository{}
	}
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) dbq(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errNilWorkflowDB
	}
	return r.db.WithContext(ctx), nil
}

func (r *WorkflowRepository) LoadDefinition(ctx context.Context, domain string, projectID uint, ticketType string) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	if r.db == nil {
		return nil, nil, nil
	}
	var def model.WorkflowDefinition
	err := r.db.WithContext(ctx).
		Where("domain = ? AND project_id = ? AND ticket_type = ?", domain, projectID, ticketType).
		First(&def).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var stages []model.WorkflowStage
	if err := r.db.WithContext(ctx).Where("definition_id = ?", def.ID).
		Order("sort_order ASC, id ASC").Find(&stages).Error; err != nil {
		return nil, nil, err
	}
	return &def, stages, nil
}

func (r *WorkflowRepository) GetDefinitionByID(ctx context.Context, id uint) (*model.WorkflowDefinition, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var def model.WorkflowDefinition
	if err := db.First(&def, id).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *WorkflowRepository) CreateDefinition(ctx context.Context, def *model.WorkflowDefinition) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Create(def).Error
}

func (r *WorkflowRepository) GetStageByKey(ctx context.Context, definitionID uint, stageKey string) (*model.WorkflowStage, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var existing model.WorkflowStage
	err = db.Where("definition_id = ? AND stage_key = ?", definitionID, stageKey).First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *WorkflowRepository) GetStageByKeyUnscoped(ctx context.Context, definitionID uint, stageKey string) (*model.WorkflowStage, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var existing model.WorkflowStage
	err = db.Unscoped().Where("definition_id = ? AND stage_key = ?", definitionID, stageKey).First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *WorkflowRepository) CreateStage(ctx context.Context, stage *model.WorkflowStage) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Create(stage).Error
}

func (r *WorkflowRepository) UpdateStageFields(ctx context.Context, id uint, fields map[string]any) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.WorkflowStage{}).Where("id = ?", id).Updates(fields).Error
}

func (r *WorkflowRepository) RestoreStage(ctx context.Context, id uint, fields map[string]any) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Unscoped().Model(&model.WorkflowStage{}).Where("id = ?", id).Updates(fields).Error
}

func (r *WorkflowRepository) DeleteStagesNotIn(ctx context.Context, definitionID uint, keys []string) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	q := db.Where("definition_id = ?", definitionID)
	if len(keys) > 0 {
		q = q.Where("stage_key NOT IN ?", keys)
	}
	return q.Delete(&model.WorkflowStage{}).Error
}

func (r *WorkflowRepository) CreateTicket(ctx context.Context, ticket *model.WorkflowTicket) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Create(ticket).Error
}

func (r *WorkflowRepository) GetTicket(ctx context.Context, id uint) (*model.WorkflowTicket, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var row model.WorkflowTicket
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *WorkflowRepository) ListTickets(ctx context.Context, p WorkflowTicketListParams) ([]model.WorkflowTicket, int64, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, 0, err
	}
	query := db.Model(&model.WorkflowTicket{})
	if p.Domain != "" {
		query = query.Where("domain = ?", p.Domain)
	}
	if p.TicketType != "" {
		query = query.Where("ticket_type = ?", p.TicketType)
	}
	if p.ProjectID != nil {
		query = query.Where("project_id = ?", *p.ProjectID)
	}
	if p.Status != "" {
		query = query.Where("status = ?", p.Status)
	}
	if p.VisibleUserID != nil {
		uid := *p.VisibleUserID
		cond := "submitter_user_id = ?"
		args := []any{uid}
		if len(p.MemberProjectIDs) > 0 {
			cond += " OR project_id IN ?"
			args = append(args, p.MemberProjectIDs)
		}
		cond += " OR EXISTS (SELECT 1 FROM workflow_ticket_steps s WHERE s.ticket_id = workflow_tickets.id AND s.deleted_at IS NULL AND (s.assignee_user_id = ? OR s.reviewer_user_id = ?))"
		args = append(args, uid, uid)
		query = query.Where("("+cond+")", args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.WorkflowTicket
	err = query.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *WorkflowRepository) UpdateTicketFields(ctx context.Context, id uint, fields map[string]any) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.WorkflowTicket{}).Where("id = ?", id).Updates(fields).Error
}

func (r *WorkflowRepository) UpdateTicketFieldsIfStatus(ctx context.Context, id uint, expectedStatus string, fields map[string]any) (int64, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return 0, err
	}
	res := db.Model(&model.WorkflowTicket{}).Where("id = ? AND status = ?", id, expectedStatus).Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *WorkflowRepository) UpdateTicketFieldsIfActive(ctx context.Context, id uint, fields map[string]any) (int64, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return 0, err
	}
	res := db.Model(&model.WorkflowTicket{}).
		Where("id = ? AND status NOT IN ?", id,
			[]string{model.WorkflowTicketStatusClosed, model.WorkflowTicketStatusCancelled}).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *WorkflowRepository) GetTicketByRef(ctx context.Context, refType string, refID uint, ticketType string) (*model.WorkflowTicket, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	q := db.Where("ref_type = ? AND ref_id = ?", refType, refID)
	if ticketType != "" {
		q = q.Where("ticket_type = ?", ticketType)
	}
	var row model.WorkflowTicket
	if err := q.Order("id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *WorkflowRepository) IsTicketParticipant(ctx context.Context, ticketID, userID uint) (bool, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	err = db.Model(&model.WorkflowTicketStep{}).
		Where("ticket_id = ? AND deleted_at IS NULL AND (assignee_user_id = ? OR reviewer_user_id = ?)",
			ticketID, userID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *WorkflowRepository) CreateStep(ctx context.Context, step *model.WorkflowTicketStep) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Create(step).Error
}

func (r *WorkflowRepository) GetStep(ctx context.Context, ticketID, stepID uint) (*model.WorkflowTicketStep, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var step model.WorkflowTicketStep
	if err := db.Where("id = ? AND ticket_id = ?", stepID, ticketID).First(&step).Error; err != nil {
		return nil, err
	}
	return &step, nil
}

func (r *WorkflowRepository) ListStepsByTicket(ctx context.Context, ticketID uint) ([]model.WorkflowTicketStep, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var steps []model.WorkflowTicketStep
	err = db.Where("ticket_id = ?", ticketID).
		Order("sort_order ASC, id ASC").Find(&steps).Error
	return steps, err
}

func (r *WorkflowRepository) ListStepsByTicketIDs(ctx context.Context, ticketIDs []uint) ([]model.WorkflowTicketStep, error) {
	if len(ticketIDs) == 0 {
		return nil, nil
	}
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var steps []model.WorkflowTicketStep
	err = db.Where("ticket_id IN ?", ticketIDs).
		Order("sort_order ASC, id ASC").Find(&steps).Error
	return steps, err
}

func (r *WorkflowRepository) GetActiveStep(ctx context.Context, ticketID uint) (*model.WorkflowTicketStep, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, err
	}
	var step model.WorkflowTicketStep
	err = db.Where("ticket_id = ? AND status = ? AND activated_at IS NOT NULL", ticketID, model.WorkflowStepPending).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

func (r *WorkflowRepository) ClaimStepReview(ctx context.Context, stepID, ticketID uint, fields map[string]any) (int64, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return 0, err
	}
	res := db.Model(&model.WorkflowTicketStep{}).
		Where("id = ? AND ticket_id = ? AND status = ? AND activated_at IS NOT NULL",
			stepID, ticketID, model.WorkflowStepPending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *WorkflowRepository) ActivateStep(ctx context.Context, stepID uint, activatedAt time.Time) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.WorkflowTicketStep{}).
		Where("id = ?", stepID).Update("activated_at", activatedAt).Error
}

func (r *WorkflowRepository) ActivateStepIfTicketPending(ctx context.Context, stepID, ticketID uint, activatedAt time.Time) (int64, error) {
	db, err := r.dbq(ctx)
	if err != nil {
		return 0, err
	}
	res := db.Model(&model.WorkflowTicketStep{}).
		Where("id = ? AND ticket_id = ? AND EXISTS (SELECT 1 FROM workflow_tickets t WHERE t.id = workflow_ticket_steps.ticket_id AND t.status = ? AND t.deleted_at IS NULL)",
			stepID, ticketID, model.WorkflowTicketStatusPending).
		Update("activated_at", activatedAt)
	return res.RowsAffected, res.Error
}

func (r *WorkflowRepository) SkipPendingSteps(ctx context.Context, ticketID uint, at time.Time) error {
	db, err := r.dbq(ctx)
	if err != nil {
		return err
	}
	return db.Model(&model.WorkflowTicketStep{}).
		Where("ticket_id = ? AND status = ?", ticketID, model.WorkflowStepPending).
		Updates(map[string]any{"status": model.WorkflowStepSkipped, "reviewed_at": at}).Error
}

func (r *WorkflowRepository) ListPending(ctx context.Context, f WorkflowPendingFilter) ([]WorkflowPendingRow, int64, error) {
	if f.UserID == 0 && !f.IsSuper {
		return []WorkflowPendingRow{}, 0, nil
	}
	db, err := r.dbq(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.
		Table("workflow_ticket_steps AS s").
		Select(`t.id AS workflow_ticket_id, s.id AS step_id, t.domain, t.ticket_type, t.project_id,
			t.title, t.status, s.stage_name AS current_stage_name, t.submitter_user_id, t.ref_type, t.ref_id,
			s.activated_at, t.created_at, s.status AS step_status, s.reviewer_user_id`).
		Joins("JOIN workflow_tickets t ON t.id = s.ticket_id").
		Where("t.deleted_at IS NULL AND s.deleted_at IS NULL")

	if len(f.Domains) > 0 {
		base = base.Where("t.domain IN ?", f.Domains)
	}
	if f.ProjectID != nil && *f.ProjectID > 0 {
		base = base.Where("t.project_id = ?", *f.ProjectID)
	}

	userID := f.UserID
	switch f.MineScope {
	case "pending":
		if f.IsSuper {
			base = base.Where(`(
				(t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL)
				OR `+workflowSQLSubmitterPendingExecution()+`
			)`, append([]any{
				model.WorkflowTicketStatusPending, model.WorkflowStepPending,
			}, workflowArgsSubmitterPendingExecution(userID)...)...)
		} else {
			base = base.Where(`(
				(t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL AND (
					s.assignee_user_id = ? OR
					((s.assignee_rule_type = ? AND ?) OR (s.assignee_rule_type = ? AND s.assignee_user_id IS NULL AND ?)) OR
					(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
						SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
					))
				))
				OR `+workflowSQLSubmitterPendingExecution()+`
			)`, append([]any{
				model.WorkflowTicketStatusPending, model.WorkflowStepPending, userID,
				model.WorkflowAssigneePlatformRole, f.CanPlatformReview,
				model.WorkflowAssigneeDuty, f.CanPlatformReview, userID,
			}, workflowArgsSubmitterPendingExecution(userID)...)...)
		}
	case "done":
		if userID == 0 {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("s.reviewer_user_id = ? AND s.status IN ?", userID,
				[]string{model.WorkflowStepApproved, model.WorkflowStepRejected})
		}
	default: // all
		if !f.IsSuper && userID > 0 {
			base = base.Where(`(
				s.assignee_user_id = ? OR s.reviewer_user_id = ? OR
				((s.assignee_rule_type = ? AND ?) OR (s.assignee_rule_type = ? AND s.assignee_user_id IS NULL AND ?)) OR
				(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
					SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
				))
				OR `+workflowSQLSubmitterPendingExecution()+`
			)`, append([]any{
				userID, userID,
				model.WorkflowAssigneePlatformRole, f.CanPlatformReview,
				model.WorkflowAssigneeDuty, f.CanPlatformReview, userID,
			}, workflowArgsSubmitterPendingExecution(userID)...)...)
		}
	}

	var total int64
	countQ := base.Session(&gorm.Session{})
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []WorkflowPendingRow
	err = base.Order("s.activated_at ASC, t.id DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&rows).Error
	return rows, total, err
}

func workflowSQLSubmitterPendingExecution() string {
	return `(
		(
			t.status = ? AND t.ticket_type = ? AND t.ref_type = ? AND t.submitter_user_id = ?
			AND EXISTS (
				SELECT 1 FROM cicd_release_runs r
				WHERE r.id = t.ref_id AND r.status = ? AND r.deleted_at IS NULL
			)
			AND s.id = (
				SELECT s2.id FROM workflow_ticket_steps s2
				WHERE s2.ticket_id = t.id AND s2.deleted_at IS NULL
				ORDER BY s2.sort_order DESC, s2.id DESC
				LIMIT 1
			)
		)
		OR (
			t.status = ? AND t.ticket_type = ? AND t.ref_type = ? AND t.submitter_user_id = ?
			AND EXISTS (
				SELECT 1 FROM db_sql_tickets st
				WHERE st.id = t.ref_id AND st.status = ?
			)
			AND s.id = (
				SELECT s2.id FROM workflow_ticket_steps s2
				WHERE s2.ticket_id = t.id AND s2.deleted_at IS NULL
				ORDER BY s2.sort_order DESC, s2.id DESC
				LIMIT 1
			)
		)
	)`
}

func workflowArgsSubmitterPendingExecution(userID uint) []any {
	return []any{
		model.WorkflowTicketStatusApproved, model.WorkflowTicketTypeRelease, model.WorkflowRefCicdReleaseRun, userID,
		model.CicdRunStatusPendingExecution,
		model.WorkflowTicketStatusApproved, model.WorkflowTicketTypeSql, model.WorkflowRefDbSqlTicket, userID,
		model.DbTicketStatusPendingExecution,
	}
}

func (r *WorkflowRepository) Transaction(ctx context.Context, fn func(WorkflowRepo) error) error {
	if r.db == nil {
		return fn(r)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&WorkflowRepository{db: tx})
	})
}
