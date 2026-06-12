package handler

import (
	"yunshu/internal/config"
	"yunshu/internal/pkg/response"
	"yunshu/internal/plugin"

	"github.com/gin-gonic/gin"
)

// PluginHandler 业务插件清单（GVA 风格模块管理）。
type PluginHandler struct {
	cfg *config.PluginsConfig
}

// NewPluginHandler 创建插件清单处理器。
func NewPluginHandler(cfg *config.PluginsConfig) *PluginHandler {
	return &PluginHandler{cfg: cfg}
}

// List godoc
// @Summary List business plugins
// @Description Returns registered plugins and enabled names from config.plugins.enabled.
// @Tags Plugins
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Router /api/v1/plugins [get]
func (h *PluginHandler) List(c *gin.Context) {
	response.Success(c, gin.H{
		"plugins":    plugin.Catalog(h.cfg),
		"enabled":    plugin.EnabledNames(h.cfg),
		"registered": plugin.RegisteredNames(),
	})
}
