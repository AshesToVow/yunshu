package handler

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"

	"github.com/gin-gonic/gin"
)

func (h *DbmgmtHandler) ListAuditLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	var instanceID uint
	if v := c.Query("instance_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		instanceID = uint(n)
	}
	action := strings.TrimSpace(c.Query("action"))
	actor, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.ListAuditLogs(c.Request.Context(), projectID, instanceID, action, page, pageSize, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *DbmgmtHandler) ListAppUserRequests(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q dbmgmtsvc.AppUserRequestListQuery) (*pagination.Result[dbmgmtsvc.AppUserRequestItem], error) {
		q.ProjectID = projectID
		if q.Mine {
			if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
				q.MineViewer = u
			}
		}
		return h.svc.ListAppUserRequests(ctx, q)
	})
}

func (h *DbmgmtHandler) CreateAppUserRequest(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req dbmgmtsvc.AppUserRequestCreateRequest) (*dbmgmtsvc.AppUserRequestItem, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.CreateAppUserRequest(ctx, projectID, req, actor)
	})
}

func (h *DbmgmtHandler) ApproveAppUserRequest(c *gin.Context) {
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
		return h.svc.ApproveAppUserRequest(ctx, projectID, requestID, req.Comment, actor)
	})
}

func (h *DbmgmtHandler) RejectAppUserRequest(c *gin.Context) {
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
		return h.svc.RejectAppUserRequest(ctx, projectID, requestID, req.Comment, actor)
	})
}

func (h *DbmgmtHandler) ListInstanceMySQLUsers(c *gin.Context) {
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
	items, err := h.svc.ListInstanceMySQLUsers(c.Request.Context(), projectID, instanceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *DbmgmtHandler) GetInstanceMySQLUserPrivileges(c *gin.Context) {
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
	res, err := h.svc.GetInstanceMySQLUserPrivileges(
		c.Request.Context(),
		projectID,
		instanceID,
		c.Query("mysql_user"),
		c.Query("mysql_host"),
		c.Query("priv_level"),
		c.Query("database"),
		actor,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *DbmgmtHandler) GetInstanceAccountPassword(c *gin.Context) {
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
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.GetInstanceAccountPassword(c.Request.Context(), projectID, instanceID, accountID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}
