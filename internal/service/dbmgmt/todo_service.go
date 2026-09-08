package dbmgmt

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

const (
	dbMineStatusPending = "mine_pending"
	dbMineStatusDone    = "mine_done"
)

func isNewDatabaseRequest(req model.DbAccessRequest) bool {
	privs := parsePrivilegesJSON(req.PrivilegesJSON)
	return hasPrivilege(privs, "create_database") && len(parseTableNamesJSON(req.TableNamesJSON)) == 0
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
	steps, _ := s.repo.ListAccessRequestStepsByRequestIDs(ctx, ids)
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
	steps, _ := s.repo.ListSqlTicketStepsByTicketIDs(ctx, ids)
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
				// 统一工单无 legacy steps 时，提交人待执行也应标成 mine_pending（详情/列表都能出执行按钮）
				if item.SubmitterUserID == userID && item.MineStatus == "" {
					item.MineStatus = dbMineStatusPending
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
				if item.SubmitterUserID == userID && item.MineStatus == "" {
					item.MineStatus = dbMineStatusPending
				}
			}
		} else if item.Status == model.DbTicketStatusPendingExecution {
			item.CurrentStageName = "待提交人执行"
			if item.SubmitterUserID == userID && item.MineStatus == "" {
				item.MineStatus = dbMineStatusPending
			}
		}
	}
}

func (s *Service) enrichMineFromWorkflow(
	ctx context.Context, refID uint, refType string, pendingBiz bool,
	mineStatus, stageName *string, actor *auth.CurrentUser,
) {
	userID := actorUserID(actor)
	steps, err := s.repo.ListWorkflowStepsByRef(ctx, refType, refID)
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

func (s *Service) enrichAppUserRequestMineStatus(ctx context.Context, items []AppUserRequestItem, actor *auth.CurrentUser) {
	userID := actorUserID(actor)
	if userID == 0 || len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	steps, _ := s.repo.ListAppUserRequestStepsByRequestIDs(ctx, ids)
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
