from pathlib import Path

# linked.go
p = Path("internal/service/workflow/linked.go")
text = p.read_text(encoding="utf-8")

reps = []
reps.append((
'''func (s *Service) GetTicketByRefType(ctx context.Context, refType string, refID uint, ticketType string) (*model.WorkflowTicket, error) {
	refType = strings.TrimSpace(refType)
	if refType == "" || refID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	q := s.db.WithContext(ctx).Where("ref_type = ? AND ref_id = ?", refType, refID)
	if tt := strings.TrimSpace(ticketType); tt != "" {
		q = q.Where("ticket_type = ?", tt)
	}
	var row model.WorkflowTicket
	err := q.Order("id DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}''',
'''func (s *Service) GetTicketByRefType(ctx context.Context, refType string, refID uint, ticketType string) (*model.WorkflowTicket, error) {
	refType = strings.TrimSpace(refType)
	if refType == "" || refID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.repo.GetTicketByRef(ctx, refType, refID, strings.TrimSpace(ticketType))
}'''
))

reps.append((
'''	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: ticketType,
			ProjectID: in.ProjectID, Title: title, Status: model.WorkflowTicketStatusPending,
			SubmitterUserID: in.SubmitterUserID, RefType: strings.TrimSpace(in.RefType), RefID: in.RefID,
			PayloadJSON: payloadJSON,
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		now := time.Now()
		for i, st := range stages {
			assigneeID, err := s.resolveDutyAssignee(ctx, st, now)
			if err != nil {
				return err
			}
			step := model.WorkflowTicketStep{
				TicketID: ticket.ID, StageKey: st.StageKey, StageName: st.StageName,
				SortOrder: st.SortOrder, Status: model.WorkflowStepPending,
				AssigneeRuleType: st.AssigneeRuleType, UserGroupID: st.UserGroupID,
				DutyMonitorRuleID: st.DutyMonitorRuleID, AssigneeUserID: assigneeID,
			}
			if i == 0 {
				step.ActivatedAt = &now
			}
			if err := tx.Create(&step).Error; err != nil {
				return err
			}
		}
		return nil
	})''',
'''	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: ticketType,
			ProjectID: in.ProjectID, Title: title, Status: model.WorkflowTicketStatusPending,
			SubmitterUserID: in.SubmitterUserID, RefType: strings.TrimSpace(in.RefType), RefID: in.RefID,
			PayloadJSON: payloadJSON,
		}
		if err := tx.CreateTicket(ctx, &ticket); err != nil {
			return err
		}
		now := time.Now()
		for i, st := range stages {
			assigneeID, err := s.resolveDutyAssignee(ctx, st, now)
			if err != nil {
				return err
			}
			step := model.WorkflowTicketStep{
				TicketID: ticket.ID, StageKey: st.StageKey, StageName: st.StageName,
				SortOrder: st.SortOrder, Status: model.WorkflowStepPending,
				AssigneeRuleType: st.AssigneeRuleType, UserGroupID: st.UserGroupID,
				DutyMonitorRuleID: st.DutyMonitorRuleID, AssigneeUserID: assigneeID,
			}
			if i == 0 {
				step.ActivatedAt = &now
			}
			if err := tx.CreateStep(ctx, &step); err != nil {
				return err
			}
		}
		return nil
	})'''
))

reps.append((
'''	if err := s.db.WithContext(ctx).Create(&ticket).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateInfoTicket", err)
	}''',
'''	if err := s.repo.CreateTicket(ctx, &ticket); err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateInfoTicket", err)
	}'''
))

reps.append((
'''func (s *Service) GetActiveStep(ctx context.Context, ticketID uint) (*model.WorkflowTicketStep, error) {
	var step model.WorkflowTicketStep
	err := s.db.WithContext(ctx).
		Where("ticket_id = ? AND status = ? AND activated_at IS NOT NULL", ticketID, model.WorkflowStepPending).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}''',
'''func (s *Service) GetActiveStep(ctx context.Context, ticketID uint) (*model.WorkflowTicketStep, error) {
	return s.repo.GetActiveStep(ctx, ticketID)
}'''
))

reps.append((
'''	now := time.Now()
	return s.db.WithContext(ctx).Model(ticket).Updates(map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	}).Error
}

// CloseLinkedTicketTyped 按工单类型关闭关联统一工单。
func (s *Service) CloseLinkedTicketTyped(ctx context.Context, refType string, refID uint, ticketType string) error {
	ticket, err := s.GetTicketByRefType(ctx, refType, refID, ticketType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if ticket.Status == model.WorkflowTicketStatusClosed || ticket.Status == model.WorkflowTicketStatusCancelled {
		return nil
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(ticket).Updates(map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	}).Error
}''',
'''	now := time.Now()
	return s.repo.UpdateTicketFields(ctx, ticket.ID, map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	})
}

// CloseLinkedTicketTyped 按工单类型关闭关联统一工单。
func (s *Service) CloseLinkedTicketTyped(ctx context.Context, refType string, refID uint, ticketType string) error {
	ticket, err := s.GetTicketByRefType(ctx, refType, refID, ticketType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if ticket.Status == model.WorkflowTicketStatusClosed || ticket.Status == model.WorkflowTicketStatusCancelled {
		return nil
	}
	now := time.Now()
	return s.repo.UpdateTicketFields(ctx, ticket.ID, map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	})
}'''
))

for i, (old, new) in enumerate(reps):
    if old not in text:
        raise SystemExit(f"linked block {i} not found")
    text = text.replace(old, new, 1)
    print(f"linked {i}")

# fix imports
text = text.replace('\n\t"gorm.io/gorm"\n', "\n")
if '"yunshu/internal/interfaces"' not in text:
    text = text.replace(
        '"yunshu/internal/model"\n',
        '"yunshu/internal/interfaces"\n\t"yunshu/internal/model"\n',
        1,
    )
# still need gorm for ErrRecordNotFound
if '"gorm.io/gorm"' not in text:
    text = text.replace(
        'bizerrors "yunshu/internal/pkg/errors"\n',
        'bizerrors "yunshu/internal/pkg/errors"\n\n\t"gorm.io/gorm"\n',
        1,
    )

p.write_text(text, encoding="utf-8")
print("linked.go done")

# pending.go - replace ListPendingForUser body query with repo
p = Path("internal/service/workflow/pending.go")
text = p.read_text(encoding="utf-8")

old = '''	base := s.db.WithContext(ctx).
		Table("workflow_ticket_steps AS s").
		Select(`t.id AS workflow_ticket_id, s.id AS step_id, t.domain, t.ticket_type, t.project_id,
			t.title, t.status, s.stage_name AS current_stage_name, t.submitter_user_id, t.ref_type, t.ref_id,
			s.activated_at, t.created_at, s.status AS step_status, s.reviewer_user_id`).
		Joins("JOIN workflow_tickets t ON t.id = s.ticket_id").
		Where("t.deleted_at IS NULL AND s.deleted_at IS NULL")

	if len(domains) > 0 {
		base = base.Where("t.domain IN ?", domains)
	}
	if q.ProjectID != nil && *q.ProjectID > 0 {
		base = base.Where("t.project_id = ?", *q.ProjectID)
	}

	switch mineScope {
	case MineScopePending:
		if isSuper {
			base = base.Where(`(
				(t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL)
				OR `+sqlSubmitterPendingExecution()+`
			)`, append([]any{
				model.WorkflowTicketStatusPending, model.WorkflowStepPending,
			}, argsSubmitterPendingExecution(userID)...)...)
		} else {
			base = base.Where(`(
				(t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL AND (
					s.assignee_user_id = ? OR
					(s.assignee_rule_type = ? AND ?) OR
					(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
						SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
					))
				))
				OR `+sqlSubmitterPendingExecution()+`
			)`, append([]any{
				model.WorkflowTicketStatusPending, model.WorkflowStepPending, userID,
				model.WorkflowAssigneePlatformRole, canPlatformReview, userID,
			}, argsSubmitterPendingExecution(userID)...)...)
		}
	case MineScopeDone:
		if userID == 0 {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("s.reviewer_user_id = ? AND s.status IN ?", userID,
				[]string{model.WorkflowStepApproved, model.WorkflowStepRejected})
		}
	default: // all
		if !isSuper && userID > 0 {
			base = base.Where(`(
				s.assignee_user_id = ? OR s.reviewer_user_id = ? OR
				(s.assignee_rule_type = ? AND ?) OR
				(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
					SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
				))
				OR `+sqlSubmitterPendingExecution()+`
			)`, append([]any{
				userID, userID, model.WorkflowAssigneePlatformRole, canPlatformReview, userID,
			}, argsSubmitterPendingExecution(userID)...)...)
		}
	}

	var total int64
	countQ := base.Session(&gorm.Session{})
	if err := countQ.Count(&total).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListPendingForUser", err)
	}

	type row struct {
		WorkflowTicketID uint
		StepID           uint
		Domain           string
		TicketType       string
		ProjectID        uint
		Title            string
		Status           string
		CurrentStageName string
		SubmitterUserID  uint
		RefType          string
		RefID            uint
		ActivatedAt      *time.Time
		CreatedAt        time.Time
		StepStatus       string
		ReviewerUserID   *uint
	}
	var rows []row
	if err := base.Order("s.activated_at ASC, t.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListPendingForUser", err)
	}'''

new = '''	rows, total, err := s.repo.ListPending(ctx, repository.WorkflowPendingFilter{
		Domains: domains, ProjectID: q.ProjectID, MineScope: mineScope,
		UserID: userID, IsSuper: isSuper, CanPlatformReview: canPlatformReview,
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListPendingForUser", err)
	}'''

if old not in text:
    raise SystemExit("pending query block not found")
text = text.replace(old, new, 1)

# fix row field access - WorkflowPendingRow has same fields
# remove unused sql helpers at bottom optionally keep for now

if '"yunshu/internal/repository"' not in text:
    text = text.replace(
        '"yunshu/internal/pkg/pagination"\n',
        '"yunshu/internal/pkg/pagination"\n\t"yunshu/internal/repository"\n',
        1,
    )
# remove gorm import if unused
if "gorm." not in text.split("import")[1].split(")")[0] and "Session" not in text:
    pass
# still might use gorm? check - after change no gorm in pending body. Remove sql helpers usages. Keep helpers unused - remove gorm import
text = text.replace('\n\t"gorm.io/gorm"\n', "\n")

# remove unused sql helper functions to avoid unused
# Keep them for now - they may be unused and cause compile error
# Delete sqlSubmitterPendingExecution and argsSubmitterPendingExecution
idx = text.find("// sqlSubmitterPendingExecution")
if idx > 0:
    text = text[:idx].rstrip() + "\n"

p.write_text(text, encoding="utf-8")
print("pending.go done")
