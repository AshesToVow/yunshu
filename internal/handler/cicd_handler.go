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

type CicdHandler struct {
	svc *cicd.Service
}

func NewCicdHandler(svc *cicd.Service) *CicdHandler {
	return &CicdHandler{svc: svc}
}

func (h *CicdHandler) ListServices(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q cicd.ServiceListQuery) (*pagination.Result[cicd.ServiceItem], error) {
		q.ProjectID = projectID
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
	item, err := h.svc.GetService(c.Request.Context(), projectID, serviceID)
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
	ServeQuery(c, func(ctx context.Context, q cicd.BuildRunListQuery) (*pagination.Result[cicd.BuildRunItem], error) {
		q.ProjectID = projectID
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
	item, err := h.svc.GetBuildRun(c.Request.Context(), projectID, runID)
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
	logText, err := h.svc.GetBuildRunLog(c.Request.Context(), projectID, runID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"log": logText})
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
	if err := h.svc.DeleteBuildRun(c.Request.Context(), projectID, runID); err != nil {
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
	ServeQuery(c, func(ctx context.Context, q cicd.ReleaseRunListQuery) (*pagination.Result[cicd.ReleaseRunItem], error) {
		q.ProjectID = projectID
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
	if err := h.svc.DeleteReleaseRun(c.Request.Context(), projectID, runID); err != nil {
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
	detail, err := h.svc.GetReleaseRunDetail(c.Request.Context(), projectID, runID)
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
	logText, err := h.svc.GetReleaseRunLog(c.Request.Context(), projectID, runID)
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
	var req cicd.ReviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	userID, name := reviewerFromContext(c)
	run, err := h.svc.ApproveReleaseRun(c.Request.Context(), projectID, runID, userID, name, req.Comment)
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
	var req cicd.ReviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	userID, name := reviewerFromContext(c)
	run, err := h.svc.RejectReleaseRun(c.Request.Context(), projectID, runID, userID, name, req.Comment)
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

func (h *CicdHandler) TerminateReleaseRun(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	var req cicd.ReviewReleaseRequest
	_ = c.ShouldBindJSON(&req)
	userID, name := reviewerFromContext(c)
	run, err := h.svc.TerminateReleaseRun(c.Request.Context(), projectID, runID, userID, name, req.Comment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *CicdHandler) BatchApproveReleaseRuns(c *gin.Context) {
	h.batchReleaseAction(c, func(ctx context.Context, projectID uint, req cicd.BatchReleaseIDsRequest, userID *uint, name string) (any, error) {
		n, err := h.svc.BatchApproveReleaseRuns(ctx, projectID, req.IDs, userID, name, req.Comment)
		return gin.H{"count": n}, err
	})
}

func (h *CicdHandler) BatchRejectReleaseRuns(c *gin.Context) {
	h.batchReleaseAction(c, func(ctx context.Context, projectID uint, req cicd.BatchReleaseIDsRequest, userID *uint, name string) (any, error) {
		n, err := h.svc.BatchRejectReleaseRuns(ctx, projectID, req.IDs, userID, name, req.Comment)
		return gin.H{"count": n}, err
	})
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
	ServeJSON(c, func(ctx context.Context, req cicd.ApprovalFlowUpsertRequest) (*cicd.ApprovalFlowResponse, error) {
		return h.svc.UpsertApprovalFlow(ctx, projectID, req)
	})
}

func (h *CicdHandler) ListReleaseApprovalSteps(c *gin.Context) {
	projectID, runID, ok := h.releaseRunIDs(c)
	if !ok {
		return
	}
	steps, err := h.svc.ListReleaseApprovalSteps(c.Request.Context(), projectID, runID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, steps)
}
