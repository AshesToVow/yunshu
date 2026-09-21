package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *AIHandler) CenterOverview(c *gin.Context) {
	response.Success(c, h.svc.CenterOverview(auth.RequestContext(c)))
}

func (h *AIHandler) ToolRuntimeHealth(c *gin.Context) {
	response.Success(c, h.svc.ToolRuntimeHealth(auth.RequestContext(c)))
}

func (h *AIHandler) ReseedCenter(c *gin.Context) {
	rep, err := h.svc.ReseedCenter(auth.RequestContext(c))
	if err != nil {

		if rep != nil {
			response.Success(c, gin.H{"ok": false, "error": err.Error(), "report": rep})
			return
		}
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true, "report": rep})
}
