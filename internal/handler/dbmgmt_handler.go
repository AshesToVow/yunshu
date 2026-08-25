package handler

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/goinception"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"

	"github.com/gin-gonic/gin"
)

type DbmgmtHandler struct {
	svc *dbmgmtsvc.Service
}

func NewDbmgmtHandler(svc *dbmgmtsvc.Service) *DbmgmtHandler {
	return &DbmgmtHandler{svc: svc}
}

func (h *DbmgmtHandler) ListInstances(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q dbmgmtsvc.InstanceListQuery) (*pagination.Result[dbmgmtsvc.InstanceItem], error) {
		q.ProjectID = projectID
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
	item, err := h.svc.GetInstance(c.Request.Context(), projectID, instanceID)
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
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.QueryRequest) (*dbmgmtsvc.QueryResult, error) {
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
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.ExecuteRequest) (*dbmgmtsvc.ExecuteResponse, error) {
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
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.SQLCheckRequest) (*dbmgmtsvc.SQLCheckResponse, error) {
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
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.ImportRequest) (*dbmgmtsvc.ExecuteResponse, error) {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ImportSQL(ctx, projectID, instanceID, req, actor)
	})
}

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

func (h *DbmgmtHandler) ListTickets(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q dbmgmtsvc.TicketListQuery) (*pagination.Result[dbmgmtsvc.TicketItem], error) {
		q.ProjectID = projectID
		if q.Mine {
			if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
				q.MineViewer = u
				tabStatus := strings.TrimSpace(q.Status)
				if tabStatus == model.DbTicketStatusPendingExecution {
					q.MineTab = "execution"
				} else {
					q.MineTab = "approval"
				}
			}
		}
		return h.svc.ListTickets(ctx, q)
	})
}

func (h *DbmgmtHandler) GetTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.GetTicket(c.Request.Context(), projectID, ticketID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DbmgmtHandler) GetTicketRollback(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.GetTicketRollback(c.Request.Context(), projectID, ticketID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DbmgmtHandler) PreviewRollbackTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	item, err := h.svc.PreviewRollbackTicket(c.Request.Context(), projectID, ticketID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DbmgmtHandler) SubmitRollbackTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.SubmitRollbackTicketRequest) (*dbmgmtsvc.ExecuteResponse, error) {
		return h.svc.SubmitRollbackTicket(ctx, projectID, ticketID, req, actor)
	})
}

func (h *DbmgmtHandler) ListTicketOSCJobs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	list, err := h.svc.ListTicketOSCJobs(c.Request.Context(), projectID, ticketID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DbmgmtHandler) GetTicketOSC(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	sqlsha1 := c.Param("sqlsha1")
	actor, _ := auth.CurrentUserFromContext(c)
	rs, err := h.svc.GetTicketOSCPercent(c.Request.Context(), projectID, ticketID, sqlsha1, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rs)
}

func (h *DbmgmtHandler) ControlTicketOSC(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	sqlsha1 := c.Param("sqlsha1")
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.OSCControlRequest) (*goinception.ReviewSet, error) {
		return h.svc.ControlTicketOSC(ctx, projectID, ticketID, sqlsha1, req.Command, actor)
	})
}

func (h *DbmgmtHandler) ApproveTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"ok": true}, func(ctx context.Context, req dbmgmtsvc.ReviewRequest) error {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.ApproveTicket(ctx, projectID, ticketID, req.Comment, actor)
	})
}

func (h *DbmgmtHandler) RejectTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"ok": true}, func(ctx context.Context, req dbmgmtsvc.ReviewRequest) error {
		actor, _ := auth.CurrentUserFromContext(c)
		return h.svc.RejectTicket(ctx, projectID, ticketID, req.Comment, actor)
	})
}

func (h *DbmgmtHandler) ExecuteTicket(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.ExecuteTicket(c.Request.Context(), projectID, ticketID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *DbmgmtHandler) ListExecutions(c *gin.Context) {
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
	queryOnly := c.Query("query_only") == "1" || c.Query("query_only") == "true"
	var executorUserID uint
	if v := c.Query("executor_user_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		executorUserID = uint(n)
	}
	actor, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.ListExecutions(c.Request.Context(), projectID, instanceID, executorUserID, queryOnly, actor, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

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

func (h *DbmgmtHandler) ListTicketSteps(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseUintParam(c, "ticketId")
	if err != nil {
		response.Error(c, err)
		return
	}
	steps, err := h.svc.ListTicketSteps(c.Request.Context(), projectID, ticketID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, steps)
}

func (h *DbmgmtHandler) ListColumnMaskRules(c *gin.Context) {
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListColumnMaskRules(c.Request.Context(), instanceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *DbmgmtHandler) UpsertColumnMaskRule(c *gin.Context) {
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req dbmgmtsvc.ColumnMaskRuleUpsertRequest) (any, error) {
		return h.svc.UpsertColumnMaskRule(ctx, instanceID, req)
	})
}

func (h *DbmgmtHandler) DeleteColumnMaskRule(c *gin.Context) {
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
	if err := h.svc.DeleteColumnMaskRule(c.Request.Context(), instanceID, ruleID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
