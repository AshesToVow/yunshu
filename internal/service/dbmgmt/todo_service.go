package dbmgmt

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"

	"gorm.io/gorm"
)

const (
	dbMineStatusPending = "mine_pending"
	dbMineStatusDone    = "mine_done"
)

func isNewDatabaseRequest(req model.DbAccessRequest) bool {
	privs := parsePrivilegesJSON(req.PrivilegesJSON)
	return hasPrivilege(privs, "create_database") && len(parseTableNamesJSON(req.TableNamesJSON)) == 0
}

func (s *Service) filterAccessRequestsApprovalPending(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.accessRequestPendingSubquery(actor))
}

func (s *Service) accessRequestPendingSubquery(actor *auth.CurrentUser) *gorm.DB {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.db.Table("db_access_requests AS r").
			Select("1").
			Where("r.id = db_access_requests.id").
			Where("r.status = ?", model.DbAccessRequestStatusPending)
	}
	userID := actorUserID(actor)
	wf := s.workflowPendingSubquery(userID, model.WorkflowRefDbAccessRequest, "db_access_requests.id")
	currentStep := s.db.Table("db_access_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.access_request_id = db_access_requests.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_access_request_steps s2
			WHERE s2.access_request_id = db_access_requests.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (s *Service) filterAccessRequestsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.accessRequestDoneSubquery(userID))
}

func (s *Service) accessRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_access_requests AS r").Select("1").Where("1 = 0")
	}
	wf := s.workflowDoneSubquery(userID, model.WorkflowRefDbAccessRequest, "db_access_requests.id")
	acted := s.db.Table("db_access_request_steps AS s").
		Select("1").
		Where("s.access_request_id = db_access_requests.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (s *Service) filterAccessRequestsApprovalMine(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?) OR EXISTS (?)", s.accessRequestPendingSubquery(actor), s.accessRequestDoneSubquery(actorUserID(actor)))
}

func (s *Service) filterTicketsApprovalPending(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.ticketPendingSubquery(actor))
}

func (s *Service) ticketPendingSubquery(actor *auth.CurrentUser) *gorm.DB {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.db.Table("db_sql_tickets AS r").
			Select("1").
			Where("r.id = db_sql_tickets.id").
			Where("r.status = ?", model.DbTicketStatusPendingApproval)
	}
	userID := actorUserID(actor)
	wf := s.workflowPendingSubquery(userID, model.WorkflowRefDbSqlTicket, "db_sql_tickets.id")
	currentStep := s.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.ticket_id = db_sql_tickets.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_sql_ticket_steps s2
			WHERE s2.ticket_id = db_sql_tickets.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("r.status = ?", model.DbTicketStatusPendingApproval).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (s *Service) filterTicketsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.ticketDoneSubquery(userID))
}

func (s *Service) ticketDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_sql_tickets AS r").Select("1").Where("1 = 0")
	}
	wf := s.workflowDoneSubquery(userID, model.WorkflowRefDbSqlTicket, "db_sql_tickets.id")
	acted := s.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Where("s.ticket_id = db_sql_tickets.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (s *Service) filterTicketsApprovalMine(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?) OR EXISTS (?)", s.ticketPendingSubquery(actor), s.ticketDoneSubquery(actorUserID(actor)))
}

func (s *Service) filterTicketsExecutionPending(dbq *gorm.DB, userID uint) *gorm.DB {
	if userID == 0 {
		return dbq.Where("1 = 0")
	}
	return dbq.Where("status = ? AND submitter_user_id = ?", model.DbTicketStatusPendingExecution, userID)
}

func (s *Service) filterTicketsExecutionDone(dbq *gorm.DB, userID uint) *gorm.DB {
	if userID == 0 {
		return dbq.Where("1 = 0")
	}
	return dbq.Where("submitter_user_id = ?", userID).
		Where("status IN ?", []string{model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting})
}

func (s *Service) filterTicketsExecutionMine(dbq *gorm.DB, userID uint) *gorm.DB {
	pending := s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status = ?", model.DbTicketStatusPendingExecution)
	done := s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status IN ?", []string{model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting})
	return dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
}

func (s *Service) workflowPendingSubquery(userID uint, refType, refIDCol string) *gorm.DB {
	return s.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = "+refIDCol+" AND t.deleted_at IS NULL", refType).
		Where("t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL",
			model.WorkflowTicketStatusPending, model.WorkflowStepPending).
		Where(`(s.assignee_user_id = ? OR (s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
			SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
		)))`, userID, userID)
}

func (s *Service) workflowDoneSubquery(userID uint, refType, refIDCol string) *gorm.DB {
	return s.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = "+refIDCol+" AND t.deleted_at IS NULL", refType).
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.WorkflowStepApproved, model.WorkflowStepRejected})
}

func (s *Service) enrichAccessRequestMineStatus(ctx context.Context, items []AccessRequestItem, actor *auth.CurrentUser) {
	userID := actorUserID(actor)
	if userID == 0 || len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	var steps []model.DbAccessRequestStep
	_ = s.db.WithContext(ctx).Where("access_request_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	byReq := make(map[uint][]model.DbAccessRequestStep, len(items))
	for _, st := range steps {
		byReq[st.AccessRequestID] = append(byReq[st.AccessRequestID], st)
	}
	for i := range items {
		item := &items[i]
		sts := byReq[item.ID]
		if len(sts) == 0 {
			s.enrichMineFromWorkflow(ctx, item.ID, model.WorkflowRefDbAccessRequest, item.Status == model.DbAccessRequestStatusPending, &item.MineStatus, &item.CurrentStageName, actor)
			continue
		}
		if item.Status == model.DbAccessRequestStatusPending {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
					(st.Status == model.DbApprovalStepApproved || st.Status == model.DbApprovalStepRejected) {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
			if item.MineStatus == "" {
				for j := range sts {
					if sts[j].Status == model.DbApprovalStepPending {
						ok, _ := s.userCanApproveStep(ctx, actor, sts[j].UserGroupID)
						if ok {
							item.MineStatus = dbMineStatusPending
						}
						break
					}
				}
			}
		} else if item.Status == model.DbAccessRequestStatusApproved || item.Status == model.DbAccessRequestStatusRejected {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
		}
		if item.CurrentStageName == "" && item.Status == model.DbAccessRequestStatusPending {
			for _, st := range sts {
				if st.Status == model.DbApprovalStepPending {
					item.CurrentStageName = st.StageName
					item.IsFinalApproval = isFinalAccessApprovalStep(sts, &st)
					break
				}
			}
		} else if item.Status == model.DbAccessRequestStatusPending {
			for j := range sts {
				if sts[j].Status == model.DbApprovalStepPending {
					item.IsFinalApproval = isFinalAccessApprovalStep(sts, &sts[j])
					break
				}
			}
		}
	}
}

func (s *Service) enrichTicketMineStatus(ctx context.Context, items []TicketItem, actor *auth.CurrentUser, mineTab string) {
	userID := actorUserID(actor)
	if userID == 0 || len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	var steps []model.DbSqlTicketStep
	_ = s.db.WithContext(ctx).Where("ticket_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	byTicket := make(map[uint][]model.DbSqlTicketStep, len(items))
	for _, st := range steps {
		byTicket[st.TicketID] = append(byTicket[st.TicketID], st)
	}
	for i := range items {
		item := &items[i]
		sts := byTicket[item.ID]
		if mineTab == "execution" {
			if item.SubmitterUserID == userID {
				switch item.Status {
				case model.DbTicketStatusPendingExecution:
					item.MineStatus = dbMineStatusPending
					item.CurrentStageName = "待提交人执行"
				case model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting:
					item.MineStatus = dbMineStatusDone
				}
			}
			continue
		}
		if len(sts) == 0 {
			pending := item.Status == model.DbTicketStatusPendingApproval
			s.enrichMineFromWorkflow(ctx, item.ID, model.WorkflowRefDbSqlTicket, pending, &item.MineStatus, &item.CurrentStageName, actor)
			if item.Status == model.DbTicketStatusPendingExecution {
				item.CurrentStageName = "待提交人执行"
			}
			continue
		}
		if item.Status == model.DbTicketStatusPendingApproval {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
					(st.Status == model.DbApprovalStepApproved || st.Status == model.DbApprovalStepRejected) {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
			if item.MineStatus == "" {
				for j := range sts {
					if sts[j].Status == model.DbApprovalStepPending {
						ok, _ := s.userCanApproveStep(ctx, actor, sts[j].UserGroupID)
						if ok {
							item.MineStatus = dbMineStatusPending
						}
						break
					}
				}
			}
		} else if item.Status == model.DbTicketStatusRejected {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
		}
		if item.CurrentStageName == "" {
			switch item.Status {
			case model.DbTicketStatusPendingApproval:
				for _, st := range sts {
					if st.Status == model.DbApprovalStepPending {
						item.CurrentStageName = st.StageName
						break
					}
				}
			case model.DbTicketStatusPendingExecution:
				item.CurrentStageName = "待提交人执行"
			}
		} else if item.Status == model.DbTicketStatusPendingExecution {
			item.CurrentStageName = "待提交人执行"
		}
	}
}

func (s *Service) enrichMineFromWorkflow(
	ctx context.Context, refID uint, refType string, pendingBiz bool,
	mineStatus, stageName *string, actor *auth.CurrentUser,
) {
	userID := actorUserID(actor)
	var steps []model.WorkflowTicketStep
	err := s.db.WithContext(ctx).Raw(`
SELECT s.* FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.ref_type = ? AND t.ref_id = ? AND s.deleted_at IS NULL
ORDER BY s.sort_order ASC, s.id ASC
`, refType, refID).Scan(&steps).Error
	if err != nil || len(steps) == 0 {
		return
	}
	for _, st := range steps {
		if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
			(st.Status == model.WorkflowStepApproved || st.Status == model.WorkflowStepRejected) {
			*mineStatus = dbMineStatusDone
			return
		}
	}
	if !pendingBiz {
		return
	}
	for i := range steps {
		st := &steps[i]
		if st.Status != model.WorkflowStepPending || st.ActivatedAt == nil {
			continue
		}
		if stageName != nil && *stageName == "" {
			*stageName = st.StageName
		}
		ok := false
		if st.AssigneeUserID != nil && *st.AssigneeUserID == userID {
			ok = true
		} else {
			ok, _ = s.userCanApproveStep(ctx, actor, st.UserGroupID)
		}
		if ok {
			*mineStatus = dbMineStatusPending
		}
		return
	}
}

func (s *Service) filterAppUserRequestsApprovalPending(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.appUserRequestPendingSubquery(actor))
}

func (s *Service) appUserRequestPendingSubquery(actor *auth.CurrentUser) *gorm.DB {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.db.Table("db_app_user_requests AS r").
			Select("1").
			Where("r.id = db_app_user_requests.id").
			Where("r.status = ?", model.DbAccessRequestStatusPending)
	}
	userID := actorUserID(actor)
	wf := s.workflowPendingSubquery(userID, model.WorkflowRefDbAppUserRequest, "db_app_user_requests.id")
	currentStep := s.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.app_user_request_id = db_app_user_requests.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_app_user_request_steps s2
			WHERE s2.app_user_request_id = db_app_user_requests.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (s *Service) filterAppUserRequestsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.appUserRequestDoneSubquery(userID))
}

func (s *Service) appUserRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_app_user_requests AS r").Select("1").Where("1 = 0")
	}
	wf := s.workflowDoneSubquery(userID, model.WorkflowRefDbAppUserRequest, "db_app_user_requests.id")
	acted := s.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Where("s.app_user_request_id = db_app_user_requests.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (s *Service) filterAppUserRequestsApprovalMine(dbq *gorm.DB, actor *auth.CurrentUser) *gorm.DB {
	return dbq.Where("EXISTS (?) OR EXISTS (?)", s.appUserRequestPendingSubquery(actor), s.appUserRequestDoneSubquery(actorUserID(actor)))
}

func (s *Service) enrichAppUserRequestMineStatus(ctx context.Context, items []AppUserRequestItem, actor *auth.CurrentUser) {
	userID := actorUserID(actor)
	if userID == 0 || len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	var steps []model.DbAppUserRequestStep
	_ = s.db.WithContext(ctx).Where("app_user_request_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	byReq := make(map[uint][]model.DbAppUserRequestStep, len(items))
	for _, st := range steps {
		byReq[st.AppUserRequestID] = append(byReq[st.AppUserRequestID], st)
	}
	for i := range items {
		item := &items[i]
		sts := byReq[item.ID]
		if len(sts) == 0 {
			s.enrichMineFromWorkflow(ctx, item.ID, model.WorkflowRefDbAppUserRequest,
				item.Status == model.DbAccessRequestStatusPending, &item.MineStatus, &item.CurrentStageName, actor)
			continue
		}
		if item.Status == model.DbAccessRequestStatusPending {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
					(st.Status == model.DbApprovalStepApproved || st.Status == model.DbApprovalStepRejected) {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
			if item.MineStatus == "" {
				for j := range sts {
					if sts[j].Status == model.DbApprovalStepPending {
						ok, _ := s.userCanApproveStep(ctx, actor, sts[j].UserGroupID)
						if ok {
							item.MineStatus = dbMineStatusPending
						}
						break
					}
				}
			}
		} else if item.Status == model.DbAccessRequestStatusApproved || item.Status == model.DbTicketStatusSuccess ||
			item.Status == model.DbTicketStatusFailed || item.Status == model.DbAccessRequestStatusRejected {
			for _, st := range sts {
				if st.ReviewerUserID != nil && *st.ReviewerUserID == userID {
					item.MineStatus = dbMineStatusDone
					break
				}
			}
		}
		if item.CurrentStageName == "" && item.Status == model.DbAccessRequestStatusPending {
			for _, st := range sts {
				if st.Status == model.DbApprovalStepPending {
					item.CurrentStageName = st.StageName
					break
				}
			}
		}
	}
}
