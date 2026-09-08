package handler

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"

	"github.com/gin-gonic/gin"
)

func (h *DbmgmtHandler) ListInstances(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q dbmgmtsvc.InstanceListQuery) (*pagination.Result[dbmgmtsvc.InstanceItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		return h.svc.ListInstances(ctx, q)
	})
}

func (h *DbmgmtHandler) GetInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.GetInstance(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DbmgmtHandler) CreateInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req dbmgmtsvc.InstanceUpsertRequest) (*dbmgmtsvc.InstanceItem, error) {
		req.ProjectID = projectID
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.UpsertInstance(ctx, 0, req, actor)
	})
}

func (h *DbmgmtHandler) UpdateInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req dbmgmtsvc.InstanceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ProjectID = projectID
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.UpsertInstance(c.Request.Context(), instanceID, req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DbmgmtHandler) DeleteInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.DeleteInstance(c.Request.Context(), projectID, instanceID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *DbmgmtHandler) PingInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.PingInstance(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *DbmgmtHandler) ListDatabases(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.ListDatabases(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DbmgmtHandler) ListTables(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.ListTables(c.Request.Context(), projectID, instanceID, c.Query("database"), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DbmgmtHandler) ListColumns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.ListColumns(c.Request.Context(), projectID, instanceID, c.Query("database"), c.Query("table"), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}
