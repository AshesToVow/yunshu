package handler

import (
	"context"
	"strconv"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"

	"github.com/gin-gonic/gin"
)

func (h *DbmgmtHandler) ListGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var instanceID uint
	if v := c.Query("instance_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		instanceID = uint(n)
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.ListGrants(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DbmgmtHandler) CreateGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req dbmgmtsvc.GrantUpsertRequest) (*dbmgmtsvc.GrantItem, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.CreateGrant(ctx, projectID, req, actor)
	})
}

func (h *DbmgmtHandler) UpdateGrant(c *gin.Context) {
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
	var req dbmgmtsvc.GrantUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.UpdateGrant(c.Request.Context(), projectID, grantID, req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DbmgmtHandler) DeleteGrant(c *gin.Context) {
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
	if err := h.svc.DeleteGrant(c.Request.Context(), projectID, grantID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *DbmgmtHandler) GetEffectiveGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	n, err := strconv.ParseUint(c.Query("instance_id"), 10, 64)
	if err != nil || n == 0 {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	perm, err := h.svc.GetEffectivePermission(c.Request.Context(), projectID, uint(n), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, perm)
}

func (h *DbmgmtHandler) GetApprovalFlow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.GetApprovalFlow(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *DbmgmtHandler) UpsertApprovalFlow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.ApprovalFlowUpsertRequest) (*dbmgmtsvc.ApprovalFlowResponse, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.UpsertApprovalFlow(ctx, projectID, req, actor)
	})
}

func (h *DbmgmtHandler) ListAccessRequests(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q dbmgmtsvc.AccessRequestListQuery) (*pagination.Result[dbmgmtsvc.AccessRequestItem], error) {
		q.ProjectID = projectID
		if q.Mine {
			if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
				q.MineViewer = u
			}
		}
		return h.svc.ListAccessRequests(ctx, q)
	})
}

func (h *DbmgmtHandler) CreateAccessRequest(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req dbmgmtsvc.AccessRequestCreateRequest) (*dbmgmtsvc.AccessRequestItem, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.CreateAccessRequest(ctx, projectID, req, actor)
	})
}

func (h *DbmgmtHandler) ApproveAccessRequest(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	requestID, err := parseUintParam(c, "requestId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"ok": true}, func(ctx context.Context, req dbmgmtsvc.AccessApproveRequest) error {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ApproveAccessRequest(ctx, projectID, requestID, req, actor)
	})
}

func (h *DbmgmtHandler) RejectAccessRequest(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	requestID, err := parseUintParam(c, "requestId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"ok": true}, func(ctx context.Context, req dbmgmtsvc.ReviewRequest) error {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.RejectAccessRequest(ctx, projectID, requestID, req.Comment, actor)
	})
}
