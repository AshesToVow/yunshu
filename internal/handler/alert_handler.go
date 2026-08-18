package handler

import (
	"context"
	"strings"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	svc *service.AlertService
}

// NewAlertHandler 创建相关逻辑。
func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// ListChannels 查询列表对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) ListChannels(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertChannelListQuery) (gin.H, error) {
		list, err := h.svc.ListChannels(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// CreateChannel 创建对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	ServeJSON(c, h.svc.CreateChannel)
}

// UpdateChannel 更新对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertChannelUpsertRequest) (*model.AlertChannel, error) {
		return h.svc.UpdateChannel(ctx, id, req)
	})
}

// DeleteChannel 删除对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.svc.DeleteChannel(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// TestChannel 测试对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) TestChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertTestRequest) (*service.AlertChannelTestResult, error) {
		return h.svc.TestChannel(ctx, id, req)
	})
}

func (h *AlertHandler) DebugRouting(c *gin.Context) {
	ServeJSON(c, h.svc.DebugRouting)
}

func (h *AlertHandler) ListEventsGrouped(c *gin.Context) {
	var q service.AlertEventListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(err.Error()))
		return
	}
	list, total, page, pageSize, err := h.svc.ListEventsGrouped(c.Request.Context(), q)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{
		"items": list, "list": list, "total": total, "page": page, "page_size": pageSize,
	})
}

// PreviewChannelTemplate 预览通道模板渲染结果。
func (h *AlertHandler) PreviewChannelTemplate(c *gin.Context) {
	ServeJSON(c, h.svc.PreviewChannelTemplate)
}

// ListEvents 查询列表对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) ListEvents(c *gin.Context) {
	var q service.AlertEventListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(err.Error()))
		return
	}
	list, total, page, pageSize, err := h.svc.ListEvents(c.Request.Context(), q)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":     list,
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AlertHandler) ListCurEvents(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertCurEventListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.ListCurEvents(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	userID := uint(0)
	userName := ""
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		userID = u.ID
		userName = strings.TrimSpace(u.Nickname)
		if userName == "" {
			userName = strings.TrimSpace(u.Username)
		}
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertAckRequest) (*model.AlertAck, error) {
		return h.svc.AcknowledgeAlert(ctx, userID, userName, req)
	})
}

func (h *AlertHandler) ClearAlertAck(c *gin.Context) {
	fp := strings.TrimSpace(c.Query("fingerprint"))
	if fp == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("fingerprint required"))
		return
	}
	if err := h.svc.ClearAlertAck(c.Request.Context(), fp); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ack cleared"})
}

func (h *AlertHandler) GetActiveAck(c *gin.Context) {
	fp := strings.TrimSpace(c.Query("fingerprint"))
	if fp == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("fingerprint required"))
		return
	}
	info, err := h.svc.GetActiveAck(c.Request.Context(), fp)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, info)
}

func (h *AlertHandler) ListHisEvents(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertHisEventListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.ListHisEvents(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

// ExplainFingerprintDelivery 按 fingerprint 查询投递/跳过原因与 firing_delivered 状态。
func (h *AlertHandler) ExplainFingerprintDelivery(c *gin.Context) {
	fp := strings.TrimSpace(c.Query("fingerprint"))
	if fp == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("fingerprint required"))
		return
	}
	out, err := h.svc.ExplainFingerprintDelivery(c.Request.Context(), fp)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, out)
}

// HistoryStats 处理对应的 HTTP 请求并返回统一响应。
func (h *AlertHandler) HistoryStats(c *gin.Context) {
	var q struct {
		ProjectID uint `form:"project_id"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	stats, err := h.svc.HistoryStats(c.Request.Context(), q.ProjectID)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, stats)
}

// ReceiveK8sEventIngress godoc
// @Summary Ingest K8s forwarded events (internal)
// @Description Platform-internal ingress for K8s Event forwarder. Not an Alertmanager endpoint. Auth: X-Alert-Token or Bearer.
// @Tags AlertsIngress
// @Accept json
// @Produce json
// @Param X-Alert-Token header string false "Ingress token (preferred)"
// @Param payload body service.AlertManagerPayload true "Event batch payload (AM-shaped transport)"
// @Success 202 {object} response.Body "accepted"
// @Failure 401 {object} response.Body "invalid token"
// @Failure 400 {object} response.Body "bad request"
// @Router /api/v1/alerts/ingress/k8s-events [post]
func (h *AlertHandler) ReceiveK8sEventIngress(c *gin.Context) {
	token := c.GetHeader("X-Alert-Token")
	if token == "" {
		token = c.GetHeader("X-Webhook-Token")
	}
	if token == "" {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		} else {
			token = authHeader
		}
	}
	if !h.svc.ValidateK8sEventIngressToken(token, c.ClientIP()) {
		response.Error(c, constants.ErrAlertWebhookTokenInvalid)
		return
	}

	var payload service.AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(bindErrorMessage(err)))
		return
	}
	if err := h.svc.ReceiveK8sEventIngress(c.Request.Context(), payload); err != nil {
		abortService(c, err)
		return
	}

	c.JSON(202, gin.H{"message": "accepted"})
}

// QualityReport 告警质量治理报告。
func (h *AlertHandler) QualityReport(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q struct {
		WindowHours int  `form:"window_hours"`
		ProjectID   uint `form:"project_id"`
	}) (*service.AlertQualityReport, error) {
		return h.svc.QualityReport(ctx, q.WindowHours, q.ProjectID)
	})
}
