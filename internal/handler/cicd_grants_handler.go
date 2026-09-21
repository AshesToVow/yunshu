package handler

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

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
