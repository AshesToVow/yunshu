package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	svc *aisvc.Service
}

func NewAIHandler(svc *aisvc.Service) *AIHandler {
	return &AIHandler{svc: svc}
}

func (h *AIHandler) Status(c *gin.Context) {
	response.Success(c, h.svc.Status(auth.RequestContext(c)))
}

func (h *AIHandler) Ping(c *gin.Context) {
	var req aisvc.PingRequest
	_ = c.ShouldBindJSON(&req)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.Ping(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}
