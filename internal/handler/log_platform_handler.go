package handler

import (
	"context"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type LogPlatformHandler struct {
	retention *service.LogRetentionService
	kafka     *service.KafkaToESService
}

func NewLogPlatformHandler(retention *service.LogRetentionService, kafka *service.KafkaToESService) *LogPlatformHandler {
	return &LogPlatformHandler{retention: retention, kafka: kafka}
}

func (h *LogPlatformHandler) GetGlobalRetention(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	it, err := h.retention.GetGlobal(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, it)
}

func (h *LogPlatformHandler) UpsertGlobalRetention(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LogRetentionUpsertRequest) (service.LogRetentionItem, error) {
		return h.retention.UpsertGlobal(ctx, req)
	})
}

func (h *LogPlatformHandler) ListRetentionPolicies(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	list, err := h.retention.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *LogPlatformHandler) StorageStats(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	stats, err := h.retention.StorageStats(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *LogPlatformHandler) RunCleanup(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	res, err := h.retention.RunCleanup(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *LogPlatformHandler) KafkaStats(c *gin.Context) {
	if h.kafka == nil {
		response.Success(c, service.KafkaQueueStats{Message: "Kafka 服务未初始化"})
		return
	}
	stats, err := h.kafka.Stats(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *LogPlatformHandler) KafkaConfigPreview(c *gin.Context) {
	if h.kafka == nil {
		response.Success(c, service.KafkaConfigPreviewItem{})
		return
	}
	cfg, err := h.kafka.ConfigPreview(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *LogPlatformHandler) GetProjectRetention(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	it, err := h.retention.GetProject(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, it)
}

func (h *LogPlatformHandler) UpsertProjectRetention(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LogRetentionUpsertRequest) (service.LogRetentionItem, error) {
		return h.retention.UpsertProject(ctx, projectID, req)
	})
}

func (h *LogPlatformHandler) DeleteProjectRetention(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.retention.DeleteProjectOverride(c.Request.Context(), projectID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
