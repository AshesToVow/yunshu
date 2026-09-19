package handler

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

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
