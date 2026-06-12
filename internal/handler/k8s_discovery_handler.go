package handler

import (
	"strconv"
	"strings"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sDiscoveryHandler struct {
	svc *service.K8sDiscoveryService
}

func NewK8sDiscoveryHandler(svc *service.K8sDiscoveryService) *K8sDiscoveryHandler {
	return &K8sDiscoveryHandler{svc: svc}
}

// ListAPIResources godoc
// @Summary List cluster API resources
// @Description Discovery API resource list; optional query namespaced=true|false filters namespaced resources.
// @Tags Clusters
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cluster ID"
// @Param namespaced query string false "Filter: true | false"
// @Success 200 {object} response.Body "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Router /api/v1/clusters/{id}/api-resources [get]
func (h *K8sDiscoveryHandler) ListAPIResources(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var ns *bool
	switch strings.TrimSpace(strings.ToLower(c.Query("namespaced"))) {
	case "true":
		t := true
		ns = &t
	case "false":
		f := false
		ns = &f
	}
	list, err := h.svc.ListAPIResources(c.Request.Context(), id, ns)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list, "cluster_id": strconv.FormatUint(uint64(id), 10)})
}
