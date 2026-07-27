package handler

import (
	"context"
	"strings"
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

// ReceiveAlertmanager godoc
// @Summary Receive Alertmanager webhook
// @Description Ingest Alertmanager notifications. Auth via header X-Alert-Token or Authorization Bearer (not query token).
// @Tags AlertsWebhook
// @Accept json
// @Produce json
// @Param X-Alert-Token header string false "Webhook token (preferred)"
// @Param payload body service.AlertManagerPayload true "Alertmanager payload"
// @Success 200 {object} response.Body "success"
// @Failure 401 {object} response.Body "invalid webhook token"
// @Failure 400 {object} response.Body "bad request"
// @Router /api/v1/alerts/webhook/alertmanager [post]
func (h *AlertHandler) ReceiveAlertmanager(c *gin.Context) {
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
	if !h.svc.ValidateWebhookToken(token) {
		response.Error(c, constants.ErrAlertWebhookTokenInvalid)
		return
	}

	var payload service.AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(bindErrorMessage(err)))
		return
	}

	// Service 内 Redis 入队成功即返回；失败时同步降级处理，避免 handler 再套无界 goroutine。
	if err := h.svc.ReceiveAlertmanager(c.Request.Context(), payload); err != nil {
		abortService(c, err)
		return
	}

	c.JSON(202, gin.H{"message": "accepted"})
}
