package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sSearchHandler struct {
	svc *service.K8sSearchService
}

func NewK8sSearchHandler(svc *service.K8sSearchService) *K8sSearchHandler {
	return &K8sSearchHandler{svc: svc}
}

// Search GET /api/v1/k8s/search — 多集群并行检索 Pod/Service/Ingress/Event。
func (h *K8sSearchHandler) Search(c *gin.Context) {
	ServeQuery(c, h.svc.Search)
}
