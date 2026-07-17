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
	currentStep := s.db.Table("db_access_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.access_request_id = r.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_access_request_steps s2
			WHERE s2.access_request_id = r.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?)", currentStep)
}

func (s *Service) filterAccessRequestsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.accessRequestDoneSubquery(userID))
}

func (s *Service) accessRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_access_requests AS r").Select("1").Where("1 = 0")
	}
	acted := s.db.Table("db_access_request_steps AS s").
		Select("1").
		Where("s.access_request_id = r.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("EXISTS (?)", acted)
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
	currentStep := s.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.ticket_id = r.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_sql_ticket_steps s2
			WHERE s2.ticket_id = r.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("r.status = ?", model.DbTicketStatusPendingApproval).
		Where("EXISTS (?)", currentStep)
}

func (s *Service) filterTicketsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.ticketDoneSubquery(userID))
}

func (s *Service) ticketDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_sql_tickets AS r").Select("1").Where("1 = 0")
	}
	acted := s.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Where("s.ticket_id = r.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("EXISTS (?)", acted)
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
				case model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting:
					item.MineStatus = dbMineStatusDone
				}
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
		}
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
	currentStep := s.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.app_user_request_id = r.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_app_user_request_steps s2
			WHERE s2.app_user_request_id = r.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return s.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?)", currentStep)
}

func (s *Service) filterAppUserRequestsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("EXISTS (?)", s.appUserRequestDoneSubquery(userID))
}

func (s *Service) appUserRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return s.db.Table("db_app_user_requests AS r").Select("1").Where("1 = 0")
	}
	acted := s.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Where("s.app_user_request_id = r.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return s.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("EXISTS (?)", acted)
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
