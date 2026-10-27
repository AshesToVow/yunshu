package handler

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type destructiveConfirmRequest struct {
	Confirm string `json:"confirm"`
	Reason  string `json:"reason"`
}

const manualCleanupConfirmKeyword = "cleanup"

// bindDestructiveConfirm 解析危险操作确认参数，参数可放在 JSON body（审计中间件会记录）或 query，便于脚本调用。
func bindDestructiveConfirm(c *gin.Context) (destructiveConfirmRequest, bool) {
	var req destructiveConfirmRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, constants.ErrBadRequestWithMsg("操作确认参数格式错误"))
			return req, false
		}
	}
	req.Confirm = strings.TrimSpace(req.Confirm)
	if req.Confirm == "" {
		req.Confirm = strings.TrimSpace(c.Query("confirm"))
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		req.Reason = strings.TrimSpace(c.Query("reason"))
	}
	if utf8.RuneCountInString(req.Reason) < 2 {
		response.Error(c, constants.ErrBadRequestWithMsg("请填写操作原因（至少 2 个字符）"))
		return req, false
	}
	return req, true
}

func logDestructiveAction(c *gin.Context, event, kind, target, reason string) {
	attrs := []any{"kind", kind, "target", target, "reason", reason}
	if user, ok := auth.CurrentUserFromContext(c); ok && user != nil {
		attrs = append(attrs, "user_id", user.ID, "username", user.Username)
	}
	slog.Default().With("component", "logplatform").Info(event, attrs...)
}

// requireDestructiveConfirm 危险删除二次确认：confirm 必须与目标名完全一致，reason 必填。
func requireDestructiveConfirm(c *gin.Context, target, kind string) (string, bool) {
	req, ok := bindDestructiveConfirm(c)
	if !ok {
		return "", false
	}
	if req.Confirm != target {
		response.Error(c, constants.ErrBadRequestWithMsg("危险操作需输入完整"+kind+"名确认删除"))
		return "", false
	}
	logDestructiveAction(c, "destructive delete requested", kind, target, req.Reason)
	return req.Reason, true
}

// requireCleanupConfirm 手动清理二次确认：confirm 固定为 cleanup，reason 必填。
func requireCleanupConfirm(c *gin.Context) (string, bool) {
	req, ok := bindDestructiveConfirm(c)
	if !ok {
		return "", false
	}
	if !strings.EqualFold(req.Confirm, manualCleanupConfirmKeyword) {
		response.Error(c, constants.ErrBadRequestWithMsg("危险操作需输入 cleanup 确认执行清理"))
		return "", false
	}
	logDestructiveAction(c, "destructive cleanup requested", "retention_cleanup", "过期索引", req.Reason)
	return req.Reason, true
}

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

func (h *LogPlatformHandler) DeleteESIndex(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	index := strings.TrimSpace(c.Param("index"))
	if index == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("索引名不能为空"))
		return
	}
	reason, ok := requireDestructiveConfirm(c, index, "索引")
	if !ok {
		return
	}
	if err := h.retention.DeleteIndex(c.Request.Context(), index); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted", "index": index, "reason": reason})
}

func (h *LogPlatformHandler) RunCleanup(c *gin.Context) {
	if h.retention == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未启用"))
		return
	}
	if _, ok := requireCleanupConfirm(c); !ok {
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

func (h *LogPlatformHandler) DeleteKafkaTopic(c *gin.Context) {
	if h.kafka == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Kafka 服务未初始化"))
		return
	}
	topic := strings.TrimSpace(c.Param("topic"))
	if topic == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("topic 不能为空"))
		return
	}
	reason, ok := requireDestructiveConfirm(c, topic, "Topic")
	if !ok {
		return
	}
	if err := h.kafka.DeleteTopic(c.Request.Context(), topic); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted", "topic": topic, "reason": reason})
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
