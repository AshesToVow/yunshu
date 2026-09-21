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
