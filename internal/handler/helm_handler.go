package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type HelmHandler struct {
	svc *service.K8sHelmService
}

func NewHelmHandler(svc *service.K8sHelmService) *HelmHandler {
	return &HelmHandler{svc: svc}
}

func (h *HelmHandler) ListReleases(c *gin.Context) {
	ServeQuery(c, h.svc.ListReleases)
}

func (h *HelmHandler) GetRelease(c *gin.Context) {
	ServeQuery(c, h.svc.GetRelease)
}

func (h *HelmHandler) GetReleaseHistory(c *gin.Context) {
	ServeQuery(c, h.svc.GetReleaseHistory)
}

func (h *HelmHandler) GetReleaseValues(c *gin.Context) {
	ServeQuery(c, h.svc.GetReleaseValues)
}

func (h *HelmHandler) Install(c *gin.Context) {
	ServeJSON201(c, h.svc.Install)
}

func (h *HelmHandler) Upgrade(c *gin.Context) {
	ServeJSON(c, h.svc.Upgrade)
}

func (h *HelmHandler) Rollback(c *gin.Context) {
	ServeJSON(c, h.svc.Rollback)
}

func (h *HelmHandler) Uninstall(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.Uninstall)
}

func (h *HelmHandler) HarborInfo(c *gin.Context) {
	data, err := h.svc.GetHarborInfo(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *HelmHandler) ListCharts(c *gin.Context) {
	ServeQuery(c, h.svc.ListCharts)
}

func (h *HelmHandler) ChartVersions(c *gin.Context) {
	ServeQuery(c, h.svc.ChartVersions)
}
