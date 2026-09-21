package handler

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

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
