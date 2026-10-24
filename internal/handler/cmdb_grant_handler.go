package handler

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *CMDBHandler) ListServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	var userID, serverID uint
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			userID = uint(parsed)
		}
	}
	if v := strings.TrimSpace(c.Query("server_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			serverID = uint(parsed)
		}
	}
	list, err := h.svc.ListServerGrants(c.Request.Context(), projectID, actor, userID, serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// MyServerAccess 当前用户对指定服务器的有效权限（含 owner/admin 隐式全量）。
func (h *CMDBHandler) MyServerAccess(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	perm, err := h.svc.EffectiveServerAccess(c.Request.Context(), projectID, serverID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, perm)
}

func (h *CMDBHandler) UpsertServerGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req service.ServerGrantUpsertRequest) (*model.ServerAccessGrant, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		return h.svc.UpsertServerGrant(ctx, req, actor)
	})
}

func (h *CMDBHandler) BulkUpsertServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req service.ServerGrantBulkRequest) (gin.H, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		n, err := h.svc.BulkUpsertServerGrants(ctx, req, actor)
		if err != nil {
			return nil, err
		}
		return gin.H{"upserted": n}, nil
	})
}

func (h *CMDBHandler) DeleteServerGrant(c *gin.Context) {
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
	if err := h.svc.DeleteServerGrant(c.Request.Context(), projectID, grantID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *CMDBHandler) BootstrapServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	req := service.BootstrapServerGrantsRequest{ProjectID: projectID}
	if actor != nil {
		id := actor.ID
		req.CreatedBy = &id
	}
	stats, err := h.svc.BootstrapServerGrantsForMembers(c.Request.Context(), req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}
