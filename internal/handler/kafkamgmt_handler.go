package handler

import (
	"yunshu/internal/pkg/response"
	kafkamgmtsvc "yunshu/internal/service/kafkamgmt"

	"github.com/gin-gonic/gin"
)

type KafkamgmtHandler struct {
	svc *kafkamgmtsvc.Service
}

func NewKafkamgmtHandler(svc *kafkamgmtsvc.Service) *KafkamgmtHandler {
	return &KafkamgmtHandler{svc: svc}
}

func (h *KafkamgmtHandler) ListConnections(c *gin.Context) {
	list, err := h.svc.ListConnections(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *KafkamgmtHandler) CreateConnection(c *gin.Context) {
	var req kafkamgmtsvc.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.CreateConnection(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *KafkamgmtHandler) ImportConnectionFromDict(c *gin.Context) {
	item, err := h.svc.ImportConnectionFromDict(c.Request.Context(), actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *KafkamgmtHandler) UpdateConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req kafkamgmtsvc.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.UpdateConnection(c.Request.Context(), id, req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *KafkamgmtHandler) DeleteConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteConnection(c.Request.Context(), id, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *KafkamgmtHandler) PingConnection(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.PingConnection(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *KafkamgmtHandler) TestConnection(c *gin.Context) {
	var req kafkamgmtsvc.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.TestConnection(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *KafkamgmtHandler) ListBrokers(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	out, err := h.svc.ListBrokers(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) ListTopics(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	out, err := h.svc.ListTopics(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) GetTopicDetail(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	topic := c.Query("topic")
	out, err := h.svc.GetTopicDetail(c.Request.Context(), connID, topic)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) GetTopicConfig(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	topic := c.Query("topic")
	out, err := h.svc.GetTopicConfig(c.Request.Context(), connID, topic)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) CreateTopic(c *gin.Context) {
	var req kafkamgmtsvc.CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.CreateTopic(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *KafkamgmtHandler) DeleteTopic(c *gin.Context) {
	var req kafkamgmtsvc.DeleteTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteTopic(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *KafkamgmtHandler) CreatePartitions(c *gin.Context) {
	var req kafkamgmtsvc.CreatePartitionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.CreatePartitions(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *KafkamgmtHandler) Produce(c *gin.Context) {
	var req kafkamgmtsvc.ProduceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Produce(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *KafkamgmtHandler) Consume(c *gin.Context) {
	var req kafkamgmtsvc.ConsumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Consume(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) ListGroups(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	out, err := h.svc.ListGroups(c.Request.Context(), connID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) GetGroupMembers(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	groupID := c.Query("group_id")
	out, err := h.svc.GetGroupMembers(c.Request.Context(), connID, groupID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) GetGroupLag(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	groupID := c.Query("group_id")
	topic := c.Query("topic")
	out, err := h.svc.GetGroupLag(c.Request.Context(), connID, groupID, topic)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *KafkamgmtHandler) DeleteGroup(c *gin.Context) {
	var req kafkamgmtsvc.DeleteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteGroup(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
