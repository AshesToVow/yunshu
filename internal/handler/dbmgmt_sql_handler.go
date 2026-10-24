package handler

import (
	"context"
	"yunshu/internal/service"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *DbmgmtHandler) Query(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.QueryRequest) (*service.QueryResult, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ExecuteQuery(ctx, projectID, instanceID, req, actor)
	})
}

func (h *DbmgmtHandler) Execute(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.ExecuteRequest) (*service.ExecuteResponse, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ExecuteSQL(ctx, projectID, instanceID, req, actor)
	})
}

func (h *DbmgmtHandler) CheckSQL(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.SQLCheckRequest) (*service.SQLCheckResponse, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.CheckSQL(ctx, projectID, instanceID, req, actor)
	})
}

func (h *DbmgmtHandler) Import(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.ImportRequest) (*service.ExecuteResponse, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ImportSQL(ctx, projectID, instanceID, req, actor)
	})
}

func (h *DbmgmtHandler) ListColumnMaskRules(c *gin.Context) {
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
	list, err := h.svc.ListColumnMaskRules(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *DbmgmtHandler) UpsertColumnMaskRule(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.ColumnMaskRuleUpsertRequest) (any, error) {
		return h.svc.UpsertColumnMaskRule(ctx, projectID, instanceID, req, actor)
	})
}

func (h *DbmgmtHandler) DeleteColumnMaskRule(c *gin.Context) {
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
	ruleID, err := parseUintParam(c, "ruleId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.DeleteColumnMaskRule(c.Request.Context(), projectID, instanceID, ruleID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
