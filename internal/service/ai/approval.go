package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	"yunshu/internal/service/k8s"
)

func (s *Service) createToolApproval(ctx context.Context, userID uint, toolName, argsJSON string, clusterID uint, ns, resource, reason string) (map[string]any, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("数据库不可用")
	}
	row := model.AiToolApproval{
		UserID:    userID,
		ToolName:  toolName,
		ArgsJSON:  argsJSON,
		ClusterID: clusterID,
		Namespace: ns,
		Resource:  resource,
		Reason:    truncateStr(reason, 500),
		Status:    "pending",
	}
	if err := s.repo.CreateApproval(ctx, &row); err != nil {
		return nil, err
	}
	if err := s.createAIWorkflowTicket(ctx, &row); err != nil {
		_ = s.repo.DeleteApproval(ctx, row.ID)
		return nil, fmt.Errorf("创建统一审批工单失败: %w", err)
	}
	return map[string]any{
		"approval_id": row.ID,
		"status":      row.Status,
		"message":     "已创建高危操作审批单，需审批通过后才会执行",
		"tool_name":   toolName,
	}, nil
}

type ApprovalListQuery struct {
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	MineOnly bool   `form:"mine_only"`
	All      bool   `form:"all"` // 仅审批角色/超管可看全部
}

func (s *Service) ListApprovals(ctx context.Context, actor *auth.CurrentUser, q ApprovalListQuery) (*pagination.Result[model.AiToolApproval], error) {
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	p := repository.AiApprovalListParams{
		Status: strings.TrimSpace(q.Status),
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
	}
	if q.MineOnly || !q.All || !canReviewApprovals(actor) {
		p.RestrictUser = true
		p.UserID = actor.ID
	}
	list, total, err := s.repo.ListApprovals(ctx, p)
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.AiToolApproval]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

type ReviewApprovalRequest struct {
	Approve bool   `json:"approve"`
	Note    string `json:"note"`
	Execute bool   `json:"execute"` // 审批通过后是否立即执行
}

func (s *Service) ReviewApproval(ctx context.Context, actor *auth.CurrentUser, id uint, req ReviewApprovalRequest) (*model.AiToolApproval, error) {
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	if !canReviewApprovals(actor) {
		return nil, constants.ErrForbiddenWithMsg("无权审批 AI 高危操作")
	}
	row, err := s.repo.GetApprovalByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFound
	}
	if row.Status != "pending" {
		return nil, constants.ErrBadRequestWithMsg("审批单状态不可变更")
	}
	if row.UserID == actor.ID && !auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil, constants.ErrForbiddenWithMsg("不能审批自己发起的操作")
	}

	wf := s.workflowEngine()
	if wf.HasLinkedTicket(ctx, model.WorkflowRefAiToolApproval, id) {
		detail, err := s.reviewAIViaWorkflow(ctx, row, req.Approve, req.Note, actor)
		if err != nil {
			return nil, err
		}
		if detail != nil && detail.Status == model.WorkflowTicketStatusPending {
			return row, nil
		}
		uid := actor.ID
		row.ReviewerID = &uid
		row.ReviewNote = truncateStr(req.Note, 500)
		if !req.Approve || (detail != nil && detail.Status == model.WorkflowTicketStatusRejected) {
			row.Status = "rejected"
			if err := s.repo.SaveApproval(ctx, row); err != nil {
				return nil, err
			}
			s.syncInvestigationAfterApproval(ctx, row)
			return row, nil
		}
		if detail == nil || detail.Status != model.WorkflowTicketStatusApproved {
			return row, nil
		}
		row.Status = "approved"
		if err := s.repo.SaveApproval(ctx, row); err != nil {
			return nil, err
		}
		if req.Execute {
			return s.ExecuteApproval(ctx, actor, id)
		}
		s.syncInvestigationAfterApproval(ctx, row)
		return row, nil
	}

	uid := actor.ID
	row.ReviewerID = &uid
	row.ReviewNote = truncateStr(req.Note, 500)
	if !req.Approve {
		row.Status = "rejected"
		if err := s.repo.SaveApproval(ctx, row); err != nil {
			return nil, err
		}
		s.syncInvestigationAfterApproval(ctx, row)
		return row, nil
	}
	row.Status = "approved"
	if err := s.repo.SaveApproval(ctx, row); err != nil {
		return nil, err
	}
	if req.Execute {
		return s.ExecuteApproval(ctx, actor, id)
	}
	s.syncInvestigationAfterApproval(ctx, row)
	return row, nil
}

func (s *Service) ExecuteApproval(ctx context.Context, actor *auth.CurrentUser, id uint) (*model.AiToolApproval, error) {
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	if !canReviewApprovals(actor) && !auth.IsSuperAdminRole(actor.RoleCodes) {
		peek, err := s.repo.GetApprovalByID(ctx, id)
		if err != nil {
			return nil, constants.ErrNotFound
		}
		if peek.UserID != actor.ID {
			return nil, constants.ErrForbiddenWithMsg("无权执行该审批单")
		}
	}
	affected, err := s.repo.ClaimApprovalExecution(ctx, id, []string{"approved", "failed"})
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, constants.ErrBadRequestWithMsg("仅已批准的审批单可执行（可能已被他人执行）")
	}

	row, err := s.repo.GetApprovalByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFound
	}

	var args map[string]any
	_ = json.Unmarshal([]byte(row.ArgsJSON), &args)
	if args == nil {
		args = map[string]any{}
	}
	getUint := func(key string, fb uint) uint {
		if v, ok := args[key].(float64); ok {
			return uint(v)
		}
		return fb
	}
	getStr := func(key string) string {
		if v, ok := args[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	clusterID := getUint("cluster_id", row.ClusterID)
	ns := getStr("namespace")
	if ns == "" {
		ns = row.Namespace
	}
	name := getStr("name")
	if name == "" {
		name = row.Resource
	}

	execCtx := withActorContext(ctx, actor)

	var execErr error
	switch row.ToolName {
	case "scale_deployment", "restart_deployment", "delete_pod":
		if err := s.assertK8sClusterAccess(ctx, actor, clusterID, ns, k8s.K8sAccessRankAdmin); err != nil {
			_ = s.repo.UpdateApprovalFields(ctx, id, map[string]any{
				"status": "failed", "result_msg": truncateStr(err.Error(), 1000),
			})
			return nil, err
		}
		switch row.ToolName {
		case "scale_deployment":
			if s.workloadSvc == nil {
				execErr = fmt.Errorf("Workload 服务不可用")
				break
			}
			replicas := int32(getUint("replicas", 1))
			execErr = s.workloadSvc.DeploymentScale(execCtx, k8s.WorkloadScaleRequest{
				ClusterID: clusterID, Namespace: ns, Name: name, Replicas: replicas,
			})
		case "restart_deployment":
			if s.workloadSvc == nil {
				execErr = fmt.Errorf("Workload 服务不可用")
				break
			}
			execErr = s.workloadSvc.DeploymentRestart(execCtx, k8s.NamespacedDetailQuery{
				ClusterID: clusterID, Namespace: ns, Name: name,
			})
		case "delete_pod":
			if s.podSvc == nil {
				execErr = fmt.Errorf("Pod 服务不可用")
				break
			}
			execErr = s.podSvc.Delete(execCtx, k8s.PodDeleteRequest{
				ClusterID: clusterID, Namespace: ns, Name: name,
			})
		}
	case "create_alert_silence":
		out, err := s.executeCreateAlertSilence(execCtx, actor, getUint("project_id", 0), getUint, getStr)
		if err != nil {
			execErr = err
		} else {
			raw, _ := json.Marshal(out)
			row.ResultMsg = truncateStr(string(raw), 1000)
		}
	default:
		reg, regErr := s.repo.GetToolByName(ctx, row.ToolName)
		if regErr != nil || reg == nil || !strings.EqualFold(reg.Runtime, "script") {
			execErr = fmt.Errorf("不支持执行的工具: %s", row.ToolName)
			break
		}
		out, err := s.runScriptTool(execCtx, toolDefRow{
			Name: reg.Name, ScriptLang: reg.ScriptLang, ScriptPath: reg.ScriptPath, TimeoutSec: reg.TimeoutSec,
		}, row.ArgsJSON)
		if err != nil {
			execErr = err
		} else {
			row.ResultMsg = truncateStr(out, 1000)
		}
	}
	if execErr != nil {
		row.Status = "failed"
		row.ResultMsg = truncateStr(execErr.Error(), 1000)
	} else {
		row.Status = "executed"
		if strings.TrimSpace(row.ResultMsg) == "" {
			row.ResultMsg = "执行成功"
		}
	}
	if err := s.repo.SaveApproval(ctx, row); err != nil {
		return row, fmt.Errorf("执行结果落库失败: %w", err)
	}
	s.syncInvestigationAfterApproval(ctx, row)
	return row, execErr
}

// syncInvestigationAfterApproval 审批终态后回写关联调查，闭合 awaiting_approval。
func (s *Service) syncInvestigationAfterApproval(ctx context.Context, appr *model.AiToolApproval) {
	if s.repo == nil || appr == nil || appr.ID == 0 {
		return
	}
	switch appr.Status {
	case "rejected", "executed", "failed", "approved":
		// approved 且未执行时：若还有 pending 兄弟单则保持 awaiting；仅全部终态才推进
	default:
		return
	}
	inv, err := s.repo.FindInvestigationLinkingApproval(ctx, appr.ID)
	if err != nil || inv == nil || inv.Status != "awaiting_approval" {
		return
	}
	ids := linkedApprovalIDsFromInvestigation(inv)
	if len(ids) == 0 {
		ids = []uint{appr.ID}
	}
	allTerminal := true
	anyFailed := false
	for _, id := range ids {
		a, e := s.repo.GetApprovalByID(ctx, id)
		if e != nil || a == nil {
			allTerminal = false
			break
		}
		switch a.Status {
		case "pending", "approved", "executing":
			allTerminal = false
		case "rejected", "executed":
			// ok
		case "failed":
			anyFailed = true
		default:
			allTerminal = false
		}
	}
	if !allTerminal {
		return
	}
	if anyFailed {
		inv.Status = "failed"
	} else {
		inv.Status = "done"
	}
	inv.UpdatedAt = time.Now()
	_ = s.repo.SaveInvestigation(ctx, inv)
}

func linkedApprovalIDsFromInvestigation(inv *model.AiInvestigation) []uint {
	if inv == nil {
		return nil
	}
	seen := map[uint]struct{}{}
	var ids []uint
	add := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if inv.ApprovalID != nil {
		add(*inv.ApprovalID)
	}
	for _, raw := range []string{inv.AnalysisJSON, inv.ReportJSON} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var blob map[string]any
		if json.Unmarshal([]byte(raw), &blob) != nil {
			continue
		}
		for _, key := range []string{"approvals", "actions"} {
			arr, ok := blob[key].([]any)
			if !ok {
				continue
			}
			for _, item := range arr {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				add(uintFromAny(m["approval_id"], 0))
			}
		}
	}
	return ids
}
