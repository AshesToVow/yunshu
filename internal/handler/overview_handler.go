package handler

import (
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type OverviewHandler struct {
	svc *service.OverviewService
}

// NewOverviewHandler 创建相关逻辑。
func NewOverviewHandler(svc *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// Get 获取对应的 HTTP 接口处理逻辑。
func (h *OverviewHandler) Get(c *gin.Context) {
	data, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// ProjectLaunches 近一个月项目上线数量统计。
func (h *OverviewHandler) ProjectLaunches(c *gin.Context) {
	data, err := h.svc.ProjectLaunches(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// ReleaseByPerson 近一个月工单按人统计。
func (h *OverviewHandler) ReleaseByPerson(c *gin.Context) {
	data, err := h.svc.ReleaseByPerson(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}
