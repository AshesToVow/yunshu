from pathlib import Path

p = Path("internal/service/workflow/service.go")
text = p.read_text(encoding="utf-8")

replacements = []

replacements.append((
'''	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		def, _, err := s.loadDefinitionTx(tx, key)
		if err != nil {
			return err
		}
		if def == nil {
			def = &model.WorkflowDefinition{
				Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
				Name: key.Domain + " workflow", Enabled: true, ForbidSelfApprove: true,
			}
			if err := tx.Create(def).Error; err != nil {
				return err
			}
		}
		keys := make([]string, 0, len(normalized))
		for _, st := range normalized {
			keys = append(keys, st.Key)
			var existing model.WorkflowStage
			err := tx.Where("definition_id = ? AND stage_key = ?", def.ID, st.Key).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row := model.WorkflowStage{
					DefinitionID: def.ID, StageKey: st.Key, StageName: st.Name, SortOrder: st.Sort,
					Enabled: st.Enabled, AssigneeRuleType: st.RuleType,
					UserGroupID: st.UserGroupID, DutyMonitorRuleID: st.DutyRuleID,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"stage_name": st.Name, "sort_order": st.Sort, "enabled": st.Enabled,
				"assignee_rule_type": st.RuleType, "user_group_id": st.UserGroupID,
				"duty_monitor_rule_id": st.DutyRuleID,
			}).Error; err != nil {
				return err
			}
		}
		q := tx.Where("definition_id = ?", def.ID)
		if len(keys) > 0 {
			q = q.Where("stage_key NOT IN ?", keys)
		}
		return q.Delete(&model.WorkflowStage{}).Error
	})''',
'''	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		def, _, err := tx.LoadDefinition(ctx, key.Domain, key.ProjectID, key.TicketType)
		if err != nil {
			return err
		}
		if def == nil {
			def = &model.WorkflowDefinition{
				Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
				Name: key.Domain + " workflow", Enabled: true, ForbidSelfApprove: true,
			}
			if err := tx.CreateDefinition(ctx, def); err != nil {
				return err
			}
		}
		keys := make([]string, 0, len(normalized))
		for _, st := range normalized {
			keys = append(keys, st.Key)
			existing, err := tx.GetStageByKey(ctx, def.ID, st.Key)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row := model.WorkflowStage{
					DefinitionID: def.ID, StageKey: st.Key, StageName: st.Name, SortOrder: st.Sort,
					Enabled: st.Enabled, AssigneeRuleType: st.RuleType,
					UserGroupID: st.UserGroupID, DutyMonitorRuleID: st.DutyRuleID,
				}
				if err := tx.CreateStage(ctx, &row); err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.UpdateStageFields(ctx, existing.ID, map[string]any{
				"stage_name": st.Name, "sort_order": st.Sort, "enabled": st.Enabled,
				"assignee_rule_type": st.RuleType, "user_group_id": st.UserGroupID,
				"duty_monitor_rule_id": st.DutyRuleID,
			}); err != nil {
				return err
			}
		}
		return tx.DeleteStagesNotIn(ctx, def.ID, keys)
	})'''
))

replacements.append((
'''	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: key.TicketType,
			ProjectID: req.ProjectID, Title: strings.TrimSpace(req.Title),
			Status: model.WorkflowTicketStatusPending, SubmitterUserID: submitter,
			RefType: strings.TrimSpace(req.RefType), RefID: req.RefID,
			PayloadJSON: payloadJSON, Remark: strings.TrimSpace(req.Remark),
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
			DefinitionID: def.ID, Domain: key.Domain, TicketType: key.TicketType,
			ProjectID: req.ProjectID, Title: strings.TrimSpace(req.Title),
			Status: model.WorkflowTicketStatusPending, SubmitterUserID: submitter,
			RefType: strings.TrimSpace(req.RefType), RefID: req.RefID,
			PayloadJSON: payloadJSON, Remark: strings.TrimSpace(req.Remark),
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

replacements.append((
'''func (s *Service) ListTickets(ctx context.Context, q TicketListQuery) (*pagination.Result[TicketDetail], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	query := s.db.WithContext(ctx).Model(&model.WorkflowTicket{})
	if d := strings.TrimSpace(q.Domain); d != "" {
		query = query.Where("domain = ?", d)
	}
	if tt := strings.TrimSpace(q.TicketType); tt != "" {
		query = query.Where("ticket_type = ?", tt)
	}
	if q.ProjectID != nil {
		query = query.Where("project_id = ?", *q.ProjectID)
	}
	if st := strings.TrimSpace(q.Status); st != "" {
		query = query.Where("status = ?", st)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListTickets", err)
	}
	var rows []model.WorkflowTicket
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListTickets", err)
	}''',
'''func (s *Service) ListTickets(ctx context.Context, q TicketListQuery) (*pagination.Result[TicketDetail], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	rows, total, err := s.repo.ListTickets(ctx, repository.WorkflowTicketListParams{
		Domain: strings.TrimSpace(q.Domain), TicketType: strings.TrimSpace(q.TicketType),
		ProjectID: q.ProjectID, Status: strings.TrimSpace(q.Status),
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListTickets", err)
	}'''
))

replacements.append((
'''func (s *Service) TicketDetail(ctx context.Context, id uint) (*TicketDetail, error) {
	var row model.WorkflowTicket
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "TicketDetail", err)
	}
	return s.ticketDetailFromRow(ctx, row)
}''',
'''func (s *Service) TicketDetail(ctx context.Context, id uint) (*TicketDetail, error) {
	row, err := s.repo.GetTicket(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "TicketDetail", err)
	}
	return s.ticketDetailFromRow(ctx, *row)
}'''
))

replacements.append((
'''func (s *Service) ReviewStep(ctx context.Context, ticketID, stepID uint, req ReviewStepRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	var ticket model.WorkflowTicket
	if err := s.db.WithContext(ctx).First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	if ticket.Status != model.WorkflowTicketStatusPending {
		return nil, constants.ErrBadRequestWithMsg("工单不在待审批状态")
	}
	var step model.WorkflowTicketStep
	if err := s.db.WithContext(ctx).Where("id = ? AND ticket_id = ?", stepID, ticketID).First(&step).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	if step.Status != model.WorkflowStepPending || step.ActivatedAt == nil {
		return nil, constants.ErrBadRequestWithMsg("该审批节点不可操作")
	}
	// 按工单绑定的 definition_id 读取职责分离开关（避免 ticket_type 回退 default 后查不到配置）
	forbidSelf := true
	if ticket.DefinitionID > 0 {
		var def model.WorkflowDefinition
		if err := s.db.WithContext(ctx).First(&def, ticket.DefinitionID).Error; err == nil {
			forbidSelf = def.ForbidSelfApprove
		}
	} else {
		def, _, _ := s.resolveFlow(ctx, DefinitionKey{Domain: ticket.Domain, ProjectID: ticket.ProjectID, TicketType: ticket.TicketType})
		if def != nil {
			forbidSelf = def.ForbidSelfApprove
		}
	}''',
'''func (s *Service) ReviewStep(ctx context.Context, ticketID, stepID uint, req ReviewStepRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	ticketPtr, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	ticket := *ticketPtr
	if ticket.Status != model.WorkflowTicketStatusPending {
		return nil, constants.ErrBadRequestWithMsg("工单不在待审批状态")
	}
	stepPtr, err := s.repo.GetStep(ctx, ticketID, stepID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	step := *stepPtr
	if step.Status != model.WorkflowStepPending || step.ActivatedAt == nil {
		return nil, constants.ErrBadRequestWithMsg("该审批节点不可操作")
	}
	// 按工单绑定的 definition_id 读取职责分离开关（避免 ticket_type 回退 default 后查不到配置）
	forbidSelf := true
	if ticket.DefinitionID > 0 {
		if def, err := s.repo.GetDefinitionByID(ctx, ticket.DefinitionID); err == nil && def != nil {
			forbidSelf = def.ForbidSelfApprove
		}
	} else {
		def, _, _ := s.resolveFlow(ctx, DefinitionKey{Domain: ticket.Domain, ProjectID: ticket.ProjectID, TicketType: ticket.TicketType})
		if def != nil {
			forbidSelf = def.ForbidSelfApprove
		}
	}'''
))

replacements.append((
'''	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status := model.WorkflowStepRejected
		if req.Approve {
			status = model.WorkflowStepApproved
		}
		res := tx.Model(&model.WorkflowTicketStep{}).
			Where("id = ? AND ticket_id = ? AND status = ? AND activated_at IS NOT NULL", stepID, ticketID, model.WorkflowStepPending).
			Updates(map[string]any{
				"status": status, "reviewer_user_id": reviewerID,
				"review_comment": strings.TrimSpace(req.Comment), "reviewed_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errStepConflict
		}
		if !req.Approve {
			return tx.Model(&ticket).Updates(map[string]any{
				"status": model.WorkflowTicketStatusRejected, "closed_at": now,
			}).Error
		}
		var steps []model.WorkflowTicketStep
		if err := tx.Where("ticket_id = ?", ticketID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
			return err
		}
		var next *model.WorkflowTicketStep
		for i := range steps {
			if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.WorkflowStepPending {
				next = &steps[i]
				break
			}
		}
		if next != nil {
			return tx.Model(next).Update("activated_at", now).Error
		}
		return tx.Model(&ticket).Updates(map[string]any{
			"status": model.WorkflowTicketStatusApproved, "closed_at": now,
		}).Error
	})''',
'''	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		status := model.WorkflowStepRejected
		if req.Approve {
			status = model.WorkflowStepApproved
		}
		n, err := tx.ClaimStepReview(ctx, stepID, ticketID, map[string]any{
			"status": status, "reviewer_user_id": reviewerID,
			"review_comment": strings.TrimSpace(req.Comment), "reviewed_at": now,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return errStepConflict
		}
		if !req.Approve {
			return tx.UpdateTicketFields(ctx, ticket.ID, map[string]any{
				"status": model.WorkflowTicketStatusRejected, "closed_at": now,
			})
		}
		steps, err := tx.ListStepsByTicket(ctx, ticketID)
		if err != nil {
			return err
		}
		var next *model.WorkflowTicketStep
		for i := range steps {
			if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.WorkflowStepPending {
				next = &steps[i]
				break
			}
		}
		if next != nil {
			return tx.ActivateStep(ctx, next.ID, now)
		}
		return tx.UpdateTicketFields(ctx, ticket.ID, map[string]any{
			"status": model.WorkflowTicketStatusApproved, "closed_at": now,
		})
	})'''
))

replacements.append((
'''func (s *Service) loadDefinition(ctx context.Context, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	return s.loadDefinitionTx(s.db.WithContext(ctx), key)
}

func (s *Service) loadDefinitionTx(tx *gorm.DB, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	key = key.normalize()
	var def model.WorkflowDefinition
	err := tx.Where("domain = ? AND project_id = ? AND ticket_type = ?", key.Domain, key.ProjectID, key.TicketType).First(&def).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var stages []model.WorkflowStage
	if err := tx.Where("definition_id = ?", def.ID).Order("sort_order ASC, id ASC").Find(&stages).Error; err != nil {
		return nil, nil, err
	}
	return &def, stages, nil
}''',
'''func (s *Service) loadDefinition(ctx context.Context, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	key = key.normalize()
	return s.repo.LoadDefinition(ctx, key.Domain, key.ProjectID, key.TicketType)
}'''
))

replacements.append((
'''func (s *Service) ticketDetailFromRow(ctx context.Context, row model.WorkflowTicket) (*TicketDetail, error) {
	var steps []model.WorkflowTicketStep
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", row.ID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ticketDetailFromRow", err)
	}''',
'''func (s *Service) ticketDetailFromRow(ctx context.Context, row model.WorkflowTicket) (*TicketDetail, error) {
	steps, err := s.repo.ListStepsByTicket(ctx, row.ID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ticketDetailFromRow", err)
	}'''
))

replacements.append((
'''func (s *Service) fillUserGroupNames(ctx context.Context, names map[uint]string) {
	if len(names) == 0 || s.db == nil {
		return
	}
	ids := make([]uint, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	var groups []model.UserGroup
	if err := s.db.WithContext(ctx).Select("id, name").Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return
	}
	for _, g := range groups {
		names[g.ID] = g.Name
	}
}''',
'''func (s *Service) fillUserGroupNames(ctx context.Context, names map[uint]string) {
	if len(names) == 0 || s.userGroupRepo == nil {
		return
	}
	ids := make([]uint, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	m, err := s.userGroupRepo.ListNamesByIDs(ctx, ids)
	if err != nil {
		return
	}
	for id, name := range m {
		names[id] = name
	}
}'''
))

for i, (old, new) in enumerate(replacements):
    if old not in text:
        raise SystemExit(f"block {i} not found")
    text = text.replace(old, new, 1)
    print(f"replaced block {i}")

if '"yunshu/internal/repository"' not in text:
    text = text.replace(
        '"yunshu/internal/pkg/pagination"\n',
        '"yunshu/internal/pkg/pagination"\n\t"yunshu/internal/repository"\n',
        1,
    )

p.write_text(text, encoding="utf-8")
print("service.go done")
