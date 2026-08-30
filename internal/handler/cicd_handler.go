package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

type CicdHandler struct {
	svc *cicd.Service
}

func NewCicdHandler(svc *cicd.Service) *CicdHandler {
	return &CicdHandler{svc: svc}
}

func (h *CicdHandler) cicdActor(c *gin.Context) *auth.CurrentUser {
	actor, _ := auth.CurrentUserFromContext(c)
	return actor
}

func (h *CicdHandler) requireCicdServiceAccess(c *gin.Context, projectID, serviceID uint, need string) bool {
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, h.cicdActor(c), need); err != nil {
		response.Error(c, err)
		return false
	}
	return true
}

func (h *CicdHandler) ListServices(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q cicd.ServiceListQuery) (*pagination.Result[cicd.ServiceItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		return h.svc.ListServices(ctx, q)
	})
}

func (h *CicdHandler) GetService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "view"); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.GetService(c.Request.Context(), projectID, serviceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CicdHandler) CreateService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCanCreateCicdService(c.Request.Context(), projectID, actor); err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req cicd.ServiceUpsertRequest) (*model.CicdService, error) {
		req.ProjectID = projectID
		return h.svc.UpsertService(ctx, 0, req)
	})
}

func (h *CicdHandler) UpdateService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "manage"); err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ProjectID = projectID
	row, err := h.svc.UpsertService(c.Request.Context(), serviceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *CicdHandler) DeleteService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "manage"); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), projectID, serviceID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) GetCiConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	row, err := h.svc.GetCiConfigView(c.Request.Context(), projectID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *CicdHandler) UpsertCiConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.CiConfigUpsertRequest) (*cicd.CiConfigUpsertResult, error) {
		return h.svc.UpsertCiConfig(ctx, projectID, serviceID, req)
	})
}

func (h *CicdHandler) ListDeployConfigs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	rows, err := h.svc.ListDeployConfigs(c.Request.Context(), projectID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) CreateDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	ServeJSON201(c, func(ctx context.Context, req cicd.DeployConfigUpsertRequest) (*cicd.DeployConfigUpsertResult, error) {
		return h.svc.UpsertDeployConfig(ctx, projectID, serviceID, 0, req)
	})
}

func (h *CicdHandler) UpdateDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	configID, err := parseUintParam(c, "configId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.DeployConfigUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.svc.UpsertDeployConfig(c.Request.Context(), projectID, serviceID, configID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CicdHandler) DeleteDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	configID, err := parseUintParam(c, "configId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteDeployConfig(c.Request.Context(), projectID, serviceID, configID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) TriggerBuild(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "build"); err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.TriggerBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	var userID *uint
	builderName := ""
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		userID = &u.ID
		builderName = u.Username
		if builderName == "" {
			builderName = u.Nickname
		}
	}
	run, err := h.svc.TriggerBuild(c.Request.Context(), projectID, serviceID, req, userID, builderName)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) TriggerRelease(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "release"); err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.TriggerReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	var userID *uint
	submitterName := ""
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		userID = &u.ID
		submitterName = u.Username
		if submitterName == "" {
			submitterName = u.Nickname
		}
	}
	run, err := h.svc.TriggerRelease(c.Request.Context(), projectID, serviceID, req, userID, submitterName)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) ListArtifacts(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	list, err := h.svc.ListArtifacts(c.Request.Context(), projectID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *CicdHandler) ListBuildRuns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q cicd.BuildRunListQuery) (*pagination.Result[cicd.BuildRunItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		return h.svc.ListBuildRuns(ctx, q)
	})
}

func (h *CicdHandler) GetBuildRun(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.GetBuildRun(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CicdHandler) GetBuildRunLog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	logText, err := h.svc.GetBuildRunLog(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"log": logText})
}

func (h *CicdHandler) ListBuildRunStages(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	rows, err := h.svc.ListBuildRunStages(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) ListBuildRunArtifactsMeta(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	rows, err := h.svc.ListBuildRunArtifactsMeta(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

// JenkinsCallback Jenkins 阶段/门禁/制品 HMAC 回调（无登录）。
func (h *CicdHandler) JenkinsCallback(c *gin.Context) {
	body, err := cicd.ReadCallbackBody(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	sig := c.GetHeader("X-Yunshu-Signature")
	if sig == "" {
		sig = c.GetHeader("X-Hub-Signature-256")
	}
	// X-Yunshu-Timestamp 存在时纳入签名并做时间窗校验，抵御回调重放；缺失则兼容旧共享库。
	ts := c.GetHeader("X-Yunshu-Timestamp")
	if err := h.svc.HandleJenkinsCallbackRaw(c.Request.Context(), body, sig, ts); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *CicdHandler) DeleteBuildRun(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteBuildRun(c.Request.Context(), projectID, runID, h.cicdActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) ListReleaseRuns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q cicd.ReleaseRunListQuery) (*pagination.Result[cicd.ReleaseRunItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		if q.Mine {
			if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
				scope := strings.TrimSpace(q.MineScope)
				if scope == "" {
					scope = "all"
				}
				tabStatus := strings.TrimSpace(q.Status)
				switch tabStatus {
				case model.CicdRunStatusPendingApproval:
					q.MineTab = "approval"
					switch scope {
					case "pending":
						q.ApproverUserID = &u.ID
					case "done":
						q.Status = ""
						q.ApprovalDoneUserID = &u.ID
					default:
						q.Status = ""
						q.ApprovalMineUserID = &u.ID
					}
				case model.CicdRunStatusPendingExecution:
					q.MineTab = "execution"
					switch scope {
					case "pending":
						q.ExecutorUserID = &u.ID
					case "done":
						q.Status = ""
						q.ExecutionDoneUserID = &u.ID
					default:
						q.Status = ""
						q.ExecutionMineUserID = &u.ID
					}
				}
				q.MineViewerUserID = &u.ID
			}
		}
		return h.svc.ListReleaseRuns(ctx, q)
	})
}

func (h *CicdHandler) DeleteReleaseRun(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteReleaseRun(c.Request.Context(), projectID, runID, h.cicdActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) GetReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	detail, err := h.svc.GetReleaseRunDetail(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *CicdHandler) GetReleaseRunLog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	logText, err := h.svc.GetReleaseRunLog(c.Request.Context(), projectID, runID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"log": logText})
}

func reviewerFromContext(c *gin.Context) (*uint, string) {
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		name := u.Username
		if name == "" {
			name = u.Nickname
		}
		return &u.ID, name
	}
	return nil, ""
}

func (h *CicdHandler) ApproveReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	req, ok := bindOptionalJSON[cicd.ReviewReleaseRequest](c)
	if !ok {
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	run, err := h.svc.ApproveReleaseRun(c.Request.Context(), projectID, runID, actor, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) RejectReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	req, ok := bindOptionalJSON[cicd.ReviewReleaseRequest](c)
	if !ok {
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	run, err := h.svc.RejectReleaseRun(c.Request.Context(), projectID, runID, actor, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) ExecuteReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	userID, _ := reviewerFromContext(c)
	run, err := h.svc.ExecuteReleaseRun(c.Request.Context(), projectID, runID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) VerifyReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	result, err := h.svc.VerifyReleaseRun(c.Request.Context(), projectID, runID, h.cicdActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CicdHandler) PlatformRollbackRelease(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	actor := h.cicdActor(c)
	ServeJSON(c, func(ctx context.Context, req cicd.PlatformRollbackRequest) (map[string]any, error) {
		return h.svc.PlatformRollbackRelease(ctx, projectID, runID, req, actor)
	})
}

func (h *CicdHandler) PromoteProgressiveRelease(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	actor := h.cicdActor(c)
	ServeJSON(c, func(ctx context.Context, req cicd.ProgressivePromoteRequest) (map[string]any, error) {
		return h.svc.PromoteProgressiveRelease(ctx, projectID, runID, req, actor)
	})
}

func (h *CicdHandler) AbortProgressiveRelease(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	actor := h.cicdActor(c)
	ServeJSON(c, func(ctx context.Context, req cicd.ProgressiveAbortRequest) (map[string]any, error) {
		return h.svc.AbortProgressiveRelease(ctx, projectID, runID, req, actor)
	})
}

func (h *CicdHandler) TerminateReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	req, ok := bindOptionalJSON[cicd.ReviewReleaseRequest](c)
	if !ok {
		return
	}
	userID, name := reviewerFromContext(c)
	run, err := h.svc.TerminateReleaseRun(c.Request.Context(), projectID, runID, userID, name, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) BatchApproveReleaseRuns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.BatchReleaseIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	n, err := h.svc.BatchApproveReleaseRuns(c.Request.Context(), projectID, req.IDs, actor, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"count": n})
}

func (h *CicdHandler) BatchRejectReleaseRuns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.BatchReleaseIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	n, err := h.svc.BatchRejectReleaseRuns(c.Request.Context(), projectID, req.IDs, actor, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"count": n})
}

func (h *CicdHandler) BatchExecuteReleaseRuns(c *gin.Context) {
	h.batchReleaseAction(c, func(ctx context.Context, projectID uint, req cicd.BatchReleaseIDsRequest, userID *uint, _ string) (any, error) {
		n, err := h.svc.BatchExecuteReleaseRuns(ctx, projectID, req.IDs, userID)
		return gin.H{"count": n}, err
	})
}

func (h *CicdHandler) BatchTerminateReleaseRuns(c *gin.Context) {
	h.batchReleaseAction(c, func(ctx context.Context, projectID uint, req cicd.BatchReleaseIDsRequest, userID *uint, name string) (any, error) {
		n, err := h.svc.BatchTerminateReleaseRuns(ctx, projectID, req.IDs, userID, name, req.Comment)
		return gin.H{"count": n}, err
	})
}

func (h *CicdHandler) batchReleaseAction(
	c *gin.Context,
	fn func(ctx context.Context, projectID uint, req cicd.BatchReleaseIDsRequest, userID *uint, name string) (any, error),
) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.BatchReleaseIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	userID, name := reviewerFromContext(c)
	result, err := fn(c.Request.Context(), projectID, req, userID, name)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CicdHandler) releaseRunIDs(c *gin.Context) (uint, uint, bool) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return 0, 0, false
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return 0, 0, false
	}
	return projectID, runID, true
}

func (h *CicdHandler) GetApprovalFlow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	flow, err := h.svc.GetApprovalFlow(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, flow)
}

func (h *CicdHandler) UpsertApprovalFlow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor := h.cicdActor(c)
	ServeJSON(c, func(ctx context.Context, req cicd.ApprovalFlowUpsertRequest) (*cicd.ApprovalFlowResponse, error) {
		return h.svc.UpsertApprovalFlow(ctx, projectID, req, actor)
	})
}

func (h *CicdHandler) ListReleaseApprovalSteps(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	steps, err := h.svc.ListReleaseApprovalSteps(c.Request.Context(), projectID, runID, h.cicdActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, steps)
}

// DownloadHelmScaffold 按服务生成 helm/ 脚手架 zip（解压到业务仓库根目录即可）。
func (h *CicdHandler) DownloadHelmScaffold(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	var q cicd.HelmScaffoldQuery
	_ = c.ShouldBindQuery(&q)
	filename, data, err := h.svc.BuildHelmScaffoldZip(c.Request.Context(), projectID, serviceID, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(filename)))
	c.Data(200, "application/zip", data)
}

// DownloadHelmScaffoldPreview 未绑定服务时按表单参数预览下载脚手架。
func (h *CicdHandler) DownloadHelmScaffoldPreview(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.RequireProjectAdmin(c.Request.Context(), projectID, h.cicdActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	var q cicd.HelmScaffoldQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	filename, data, err := h.svc.BuildHelmScaffoldZipPreview(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(filename)))
	c.Data(200, "application/zip", data)
}

func (h *CicdHandler) ListCicdGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	var userID, serviceID uint
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			userID = uint(parsed)
		}
	}
	if v := strings.TrimSpace(c.Query("service_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			serviceID = uint(parsed)
		}
	}
	list, err := h.svc.ListCicdGrants(c.Request.Context(), projectID, actor, userID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *CicdHandler) UpsertCicdGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req cicd.CicdGrantUpsertRequest) (*model.CicdAccessGrant, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		return h.svc.UpsertCicdGrant(ctx, req, actor)
	})
}

func (h *CicdHandler) BulkUpsertCicdGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req cicd.CicdGrantBulkRequest) (gin.H, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		n, err := h.svc.BulkUpsertCicdGrants(ctx, req, actor)
		if err != nil {
			return nil, err
		}
		return gin.H{"upserted": n}, nil
	})
}

func (h *CicdHandler) DeleteCicdGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	grantID, err := parseUintParam(c, "grantId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.DeleteCicdGrant(c.Request.Context(), projectID, grantID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *CicdHandler) BootstrapCicdGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	req := cicd.BootstrapCicdGrantsRequest{ProjectID: projectID}
	if actor != nil {
		id := actor.ID
		req.CreatedBy = &id
	}
	stats, err := h.svc.BootstrapCicdGrantsForMembers(c.Request.Context(), req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}
