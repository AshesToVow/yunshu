package handler

import (
	"context"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/plugin"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单管理：树查询、增删改；非 super-admin 会过滤 AdminOnly 菜单。
type MenuHandler struct {
	service *service.MenuService
	plugins *config.PluginsConfig
}

// NewMenuHandler 构造菜单处理器。
func NewMenuHandler(svc *service.MenuService, plugins *config.PluginsConfig) *MenuHandler {
	return &MenuHandler{service: svc, plugins: plugins}
}

// Tree 返回菜单树；非 super-admin 时移除仅管理员可见项。
func (h *MenuHandler) Tree(c *gin.Context) {
	list, err := h.service.Tree(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	// 如果不是 super-admin，需要从菜单树中移除仅管理员可见项
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		filtered := plugin.FilterMenusByPlugins(list, h.plugins)
		response.Success(c, filtered)
		return
	}

	isSuper := false
	for _, rc := range user.RoleCodes {
		if strings.TrimSpace(rc) == "super-admin" {
			isSuper = true
			break
		}
	}

	if isSuper {
		filtered := plugin.FilterMenusByPlugins(list, h.plugins)
		response.Success(c, filtered)
		return
	}

	// 递归过滤 AdminOnly 项
	var filter func([]model.Menu) []model.Menu
	filter = func(items []model.Menu) []model.Menu {
		var out []model.Menu
		for _, it := range items {
			if it.AdminOnly {
				continue
			}
			if len(it.Children) > 0 {
				it.Children = filter(it.Children)
			}
			out = append(out, it)
		}
		return out
	}

	filtered := filter(list)
	filtered = plugin.FilterMenusByPlugins(filtered, h.plugins)
	response.Success(c, filtered)
}

// Create 创建菜单项。
func (h *MenuHandler) Create(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.MenuCreatePayload) (*model.Menu, error) {
		return h.service.Create(ctx, req)
	})
}

// Update 按 ID 更新菜单。
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.MenuUpdatePayload) (*model.Menu, error) {
		return h.service.Update(ctx, id, req)
	})
}

// Delete 按 ID 删除菜单。
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// BatchStatus 批量设置菜单状态（启用/停用）。
func (h *MenuHandler) BatchStatus(c *gin.Context) {
	ServeJSONOK(c, gin.H{"message": "updated"}, func(ctx context.Context, req service.MenuBatchStatusPayload) error {
		return h.service.BatchSetStatus(ctx, req)
	})
}
