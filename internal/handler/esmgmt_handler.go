package handler

import (
	"strconv"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	esmgmtsvc "yunshu/internal/service/esmgmt"

	"github.com/gin-gonic/gin"
)

type EsmgmtHandler struct {
	svc *esmgmtsvc.Service
}

func NewEsmgmtHandler(svc *esmgmtsvc.Service) *EsmgmtHandler {
	return &EsmgmtHandler{svc: svc}
}

func actorFrom(c *gin.Context) *auth.CurrentUser {
	u, _ := auth.CurrentUserFromContext(c)
	return u
}

func (h *EsmgmtHandler) ListConnections(c *gin.Context) {
	list, err := h.svc.ListConnections(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *EsmgmtHandler) CreateConnection(c *gin.Context) {
	var req esmgmtsvc.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.CreateConnection(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EsmgmtHandler) UpdateConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req esmgmtsvc.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.UpdateConnection(c.Request.Context(), id, req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EsmgmtHandler) DeleteConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteConnection(c.Request.Context(), id, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) PingConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.PingConnection(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *EsmgmtHandler) TestConnection(c *gin.Context) {
	var req esmgmtsvc.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.TestConnection(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *EsmgmtHandler) ClusterHealth(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	out, err := h.svc.ClusterHealth(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) ListIndices(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	pattern := strings.TrimSpace(c.Query("pattern"))
	list, err := h.svc.ListIndices(c.Request.Context(), connID, pattern)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *EsmgmtHandler) CreateIndex(c *gin.Context) {
	var req esmgmtsvc.CreateIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.CreateIndex(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true, "name": strings.TrimSpace(req.Name)})
}

func (h *EsmgmtHandler) DeleteIndex(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	name := strings.TrimSpace(c.Param("name"))
	force := parseBoolQuery(c, "force")
	if err := h.svc.DeleteIndex(c.Request.Context(), connID, name, force); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) OpenIndex(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	name := strings.TrimSpace(c.Param("name"))
	if err := h.svc.OpenIndex(c.Request.Context(), connID, name); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) CloseIndex(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	name := strings.TrimSpace(c.Param("name"))
	if err := h.svc.CloseIndex(c.Request.Context(), connID, name); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) CatNodes(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	out, err := h.svc.CatNodes(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) ProxyREST(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	var req esmgmtsvc.ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.ProxyREST(c.Request.Context(), connID, req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) CreateIndexBackup(c *gin.Context) {
	var req esmgmtsvc.BackupIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.CreateIndexBackup(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *EsmgmtHandler) ListBackupJobs(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	limit := 50
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.svc.ListBackupJobs(c.Request.Context(), connID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *EsmgmtHandler) GetBackupJob(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.GetBackupJob(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *EsmgmtHandler) DownloadBackup(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	artifact := strings.TrimSpace(c.Query("artifact"))
	out, err := h.svc.PresignBackupDownload(c.Request.Context(), id, artifact)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) CreateIndexRestore(c *gin.Context) {
	var req esmgmtsvc.RestoreIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.CreateIndexRestore(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *EsmgmtHandler) ListRestoreJobs(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	limit := 50
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.svc.ListRestoreJobs(c.Request.Context(), connID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *EsmgmtHandler) GetRestoreJob(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.GetRestoreJob(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *EsmgmtHandler) ListSchedules(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	list, err := h.svc.ListSchedules(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *EsmgmtHandler) CreateSchedule(c *gin.Context) {
	var req esmgmtsvc.ScheduleUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.CreateSchedule(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *EsmgmtHandler) UpdateSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req esmgmtsvc.ScheduleUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdateSchedule(c.Request.Context(), id, req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *EsmgmtHandler) DeleteSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteSchedule(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func parseBoolQuery(c *gin.Context, key string) bool {
	raw := strings.ToLower(strings.TrimSpace(c.Query(key)))
	if raw == "" {
		return false
	}
	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	return raw == "1" || raw == "yes"
}
