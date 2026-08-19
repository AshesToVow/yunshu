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
	"yunshu/internal/service/k8s"
)

func (s *Service) createToolApproval(ctx context.Context, userID uint, toolName, argsJSON string, clusterID uint, ns, resource, reason string) (map[string]any, error) {
	if s.db == nil {
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
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
	if s.db == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AiToolApproval{})
	if st := strings.TrimSpace(q.Status); st != "" {
		db = db.Where("status = ?", st)
	}
	// 默认仅本人；审批角色显式 all=true 时可看全部
	if q.MineOnly || !q.All || !canReviewApprovals(actor) {
		db = db.Where("user_id = ?", actor.ID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.AiToolApproval
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
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
	if s.db == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	if !canReviewApprovals(actor) {
		return nil, constants.ErrForbiddenWithMsg("无权审批 AI 高危操作")
	}
	var row model.AiToolApproval
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if row.Status != "pending" {
		return nil, constants.ErrBadRequestWithMsg("审批单状态不可变更")
	}
	if row.UserID == actor.ID && !auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil, constants.ErrForbiddenWithMsg("不能审批自己发起的操作")
	}
	uid := actor.ID
	row.ReviewerID = &uid
	row.ReviewNote = truncateStr(req.Note, 500)
	if !req.Approve {
		row.Status = "rejected"
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	row.Status = "approved"
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	if req.Execute {
		return s.ExecuteApproval(ctx, actor, id)
	}
	return &row, nil
}

func (s *Service) ExecuteApproval(ctx context.Context, actor *auth.CurrentUser, id uint) (*model.AiToolApproval, error) {
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	if !canReviewApprovals(actor) && !auth.IsSuperAdminRole(actor.RoleCodes) {
		// 允许申请人在已批准后触发执行，但仍注入本人上下文做 K8s ACL
		var peek model.AiToolApproval
		if err := s.db.WithContext(ctx).First(&peek, id).Error; err != nil {
			return nil, constants.ErrNotFound
		}
		if peek.UserID != actor.ID {
			return nil, constants.ErrForbiddenWithMsg("无权执行该审批单")
		}
	}
	// 乐观锁：仅 approved/failed → executing
	res := s.db.WithContext(ctx).Model(&model.AiToolApproval{}).
		Where("id = ? AND status IN ?", id, []string{"approved", "failed"}).
		Updates(map[string]any{"status": "executing", "updated_at": time.Now()})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, constants.ErrBadRequestWithMsg("仅已批准的审批单可执行（可能已被他人执行）")
	}

	var row model.AiToolApproval
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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

	if err := s.assertK8sClusterAccess(ctx, actor, clusterID, ns, k8s.K8sAccessRankAdmin); err != nil {
		_ = s.db.WithContext(ctx).Model(&model.AiToolApproval{}).Where("id = ?", id).
			Updates(map[string]any{"status": "failed", "result_msg": truncateStr(err.Error(), 1000)})
		return nil, err
	}

	execCtx := withActorContext(ctx, actor)

	var execErr error
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
	default:
		execErr = fmt.Errorf("不支持执行的工具: %s", row.ToolName)
	}
	if execErr != nil {
		row.Status = "failed"
		row.ResultMsg = truncateStr(execErr.Error(), 1000)
	} else {
		row.Status = "executed"
		row.ResultMsg = "执行成功"
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return &row, fmt.Errorf("执行结果落库失败: %w", err)
	}
	return &row, execErr
}
