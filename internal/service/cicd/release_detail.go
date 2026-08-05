package cicd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

type ReleaseHandlerItem struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type ReleaseOperationLogItem struct {
	Action     string `json:"action"`
	ActorName  string `json:"actor_name"`
	OperatedAt string `json:"operated_at"`
	Message    string `json:"message"`
}

type ReleaseRunDetailResponse struct {
	ReleaseRunItem
	ApprovalSteps    []ReleaseApprovalStepItem `json:"approval_steps"`
	ApprovalFlowText string                    `json:"approval_flow_text"`
	CurrentHandlers  []ReleaseHandlerItem      `json:"current_handlers"`
	OperationLogs    []ReleaseOperationLogItem `json:"operation_logs"`
	DestHosts        []string                  `json:"dest_hosts"`
	DeployConfigName string                    `json:"deploy_config_name"`
	DestPath         string                    `json:"dest_path"`
}

func (s *Service) GetReleaseRunDetail(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) (*ReleaseRunDetailResponse, error) {
	var row model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, row.ServiceID, actor, "view"); err != nil {
		return nil, err
	}
	item := ReleaseRunItem{CicdReleaseRun: row}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err == nil {
		item.ServiceName = svc.Name
		item.ServiceIdentifier = svc.Identifier
	}
	var proj model.Project
	if err := s.db.WithContext(ctx).Select("name").Where("id = ?", projectID).First(&proj).Error; err == nil {
		item.ProjectName = proj.Name
	}
	if row.CurrentStageKey != "" {
		item.CurrentStageName = stageNameByKey(row.CurrentStageKey)
	}

	steps, err := s.buildReleaseApprovalStepItems(ctx, runID)
	if err != nil {
		return nil, err
	}
	flowText := buildApprovalFlowText(steps)
	handlers, _ := s.loadCurrentHandlers(ctx, steps)
	logs := buildReleaseOperationLogs(row, steps, flowText)
	destHosts, deployName, destPath := s.loadReleaseDeployMeta(ctx, projectID, &row)

	return &ReleaseRunDetailResponse{
		ReleaseRunItem:   item,
		ApprovalSteps:    steps,
		ApprovalFlowText: flowText,
		CurrentHandlers:  handlers,
		OperationLogs:    logs,
		DestHosts:        destHosts,
		DeployConfigName: deployName,
		DestPath:         destPath,
	}, nil
}

func (s *Service) buildReleaseApprovalStepItems(ctx context.Context, runID uint) ([]ReleaseApprovalStepItem, error) {
	var steps []model.CicdReleaseApprovalStep
	if err := s.db.WithContext(ctx).Where("release_run_id = ?", runID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	groupIDs := make([]uint, 0)
	seen := map[uint]struct{}{}
	for _, st := range steps {
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if _, ok := seen[*st.UserGroupID]; !ok {
				seen[*st.UserGroupID] = struct{}{}
				groupIDs = append(groupIDs, *st.UserGroupID)
			}
		}
	}
	groupNames := map[uint]string{}
	if len(groupIDs) > 0 {
		var groups []model.UserGroup
		_ = s.db.WithContext(ctx).Select("id, name").Where("id IN ?", groupIDs).Find(&groups).Error
		for _, g := range groups {
			groupNames[g.ID] = g.Name
		}
	}
	items := make([]ReleaseApprovalStepItem, 0, len(steps))
	for _, st := range steps {
		item := ReleaseApprovalStepItem{
			ID:             st.ID,
			StageKey:       st.StageKey,
			StageName:      st.StageName,
			SortOrder:      st.SortOrder,
			Status:         st.Status,
			UserGroupID:    st.UserGroupID,
			ReviewerUserID: st.ReviewerUserID,
			ReviewerName:   st.ReviewerName,
			ReviewComment:  st.ReviewComment,
		}
		if st.UserGroupID != nil {
			item.UserGroupName = groupNames[*st.UserGroupID]
		}
		if st.ReviewedAt != nil {
			ts := st.ReviewedAt.Format("2006-01-02 15:04:05")
			item.ReviewedAt = &ts
		}
		items = append(items, item)
	}
	return items, nil
}

func buildApprovalFlowText(steps []ReleaseApprovalStepItem) string {
	if len(steps) == 0 {
		return ""
	}
	parts := make([]string, 0, len(steps))
	for _, st := range steps {
		name := strings.TrimSpace(st.StageName)
		if name == "" {
			name = st.StageKey
		}
		if !strings.HasSuffix(name, "审批") {
			name += "审批"
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, " → ")
}

func (s *Service) loadCurrentHandlers(ctx context.Context, steps []ReleaseApprovalStepItem) ([]ReleaseHandlerItem, error) {
	for _, st := range steps {
		if st.Status != model.CicdApprovalStepPending {
			continue
		}
		if st.UserGroupID == nil || *st.UserGroupID == 0 {
			return nil, nil
		}
		userIDs, err := s.userGroupRepo.ListMemberUserIDs(ctx, *st.UserGroupID)
		if err != nil || len(userIDs) == 0 {
			return nil, err
		}
		var users []model.User
		_ = s.db.WithContext(ctx).Select("id, username, nickname").Where("id IN ?", userIDs).Find(&users).Error
		out := make([]ReleaseHandlerItem, 0, len(users))
		for _, u := range users {
			name := u.Username
			if name == "" {
				name = u.Nickname
			}
			out = append(out, ReleaseHandlerItem{
				UserID:   u.ID,
				Username: u.Username,
				Nickname: u.Nickname,
			})
		}
		return out, nil
	}
	return nil, nil
}

func buildReleaseOperationLogs(release model.CicdReleaseRun, steps []ReleaseApprovalStepItem, flowText string) []ReleaseOperationLogItem {
	logs := make([]ReleaseOperationLogItem, 0, 8)
	submitAt := formatReleaseTime(release.StartedAt, release.CreatedAt)
	submitMsg := "提交发布工单"
	if release.AuditEnabled {
		if flowText != "" {
			submitMsg = "等待审批，审批流程：" + flowText
		} else {
			submitMsg = "等待审批"
		}
	}
	logs = append(logs, ReleaseOperationLogItem{
		Action:     "提交",
		ActorName:  release.SubmitterName,
		OperatedAt: submitAt,
		Message:    submitMsg,
	})

	for _, st := range steps {
		if st.Status == model.CicdApprovalStepPending {
			continue
		}
		at := submitAt
		if st.ReviewedAt != nil {
			at = *st.ReviewedAt
		}
		stageLabel := st.StageName
		if stageLabel == "" {
			stageLabel = st.StageKey
		}
		switch st.Status {
		case model.CicdApprovalStepApproved:
			msg := fmt.Sprintf("%s：通过", stageLabel)
			if st.ReviewComment != "" {
				msg += "，" + st.ReviewComment
			}
			if st.UserGroupName != "" {
				msg += "（" + st.UserGroupName + "）"
			}
			logs = append(logs, ReleaseOperationLogItem{
				Action:     "审批通过",
				ActorName:  st.ReviewerName,
				OperatedAt: at,
				Message:    msg,
			})
		case model.CicdApprovalStepRejected:
			msg := fmt.Sprintf("%s：驳回", stageLabel)
			if st.ReviewComment != "" {
				msg += "，" + st.ReviewComment
			}
			logs = append(logs, ReleaseOperationLogItem{
				Action:     "审批驳回",
				ActorName:  st.ReviewerName,
				OperatedAt: at,
				Message:    msg,
			})
		}
	}

	switch release.Status {
	case model.CicdRunStatusPendingExecution:
		logs = append(logs, ReleaseOperationLogItem{
			Action:     "待执行",
			ActorName:  release.SubmitterName,
			OperatedAt: formatReleaseTime(release.ReviewedAt, release.UpdatedAt),
			Message:    "全部审批通过，等待提交人执行发布",
		})
	case model.CicdRunStatusRunning:
		logs = append(logs, ReleaseOperationLogItem{
			Action:     "执行发布",
			ActorName:  release.SubmitterName,
			OperatedAt: formatReleaseTime(release.StartedAt, release.UpdatedAt),
			Message:    "提交人已触发 Jenkins 发布",
		})
	case model.CicdRunStatusSuccess:
		logs = append(logs, ReleaseOperationLogItem{
			Action:     "执行完成",
			ActorName:  "系统",
			OperatedAt: formatReleaseTime(release.FinishedAt, release.UpdatedAt),
			Message:    "发布执行成功",
		})
	case model.CicdRunStatusFailure:
		logs = append(logs, ReleaseOperationLogItem{
			Action:     "执行失败",
			ActorName:  "系统",
			OperatedAt: formatReleaseTime(release.FinishedAt, release.UpdatedAt),
			Message:    "发布执行失败",
		})
	case model.CicdRunStatusRejected:
		if len(steps) == 0 && release.ReviewerName != "" {
			logs = append(logs, ReleaseOperationLogItem{
				Action:     "审批驳回",
				ActorName:  release.ReviewerName,
				OperatedAt: formatReleaseTime(release.ReviewedAt, release.UpdatedAt),
				Message:    release.ReviewComment,
			})
		}
	case model.CicdRunStatusCancelled:
		logs = append(logs, ReleaseOperationLogItem{
			Action:     "终止",
			ActorName:  release.ReviewerName,
			OperatedAt: formatReleaseTime(release.ReviewedAt, timeFromPtr(release.FinishedAt)),
			Message:    strings.TrimSpace(release.ReviewComment),
		})
	}
	return logs
}

func formatReleaseTime(primary *time.Time, fallback time.Time) string {
	if primary != nil && !primary.IsZero() {
		return primary.Format("2006-01-02 15:04:05")
	}
	if !fallback.IsZero() {
		return fallback.Format("2006-01-02 15:04:05")
	}
	return ""
}

func timeFromPtr(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func (s *Service) loadReleaseDeployMeta(ctx context.Context, projectID uint, release *model.CicdReleaseRun) ([]string, string, string) {
	if release == nil || release.DeployConfigID == nil {
		return nil, "", ""
	}
	var dc model.CicdDeployConfig
	if err := s.db.WithContext(ctx).Where("id = ?", *release.DeployConfigID).First(&dc).Error; err != nil {
		return nil, "", ""
	}
	destPath := strings.TrimSpace(dc.DestPath)
	destStr, _ := s.resolveDestIPs(ctx, projectID, &dc)
	var hosts []string
	if destStr != "" {
		for _, h := range strings.Split(destStr, ",") {
			if t := strings.TrimSpace(h); t != "" {
				hosts = append(hosts, t)
			}
		}
	}
	if strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		ns := strings.TrimSpace(dc.K8sNamespace)
		if ns != "" {
			destPath = "namespace: " + ns
		}
	}
	return hosts, dc.Name, destPath
}
