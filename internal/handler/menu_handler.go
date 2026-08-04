package handler

import (
	"context"

	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/plugingate"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单管理：树查询、增删改；非 super-admin 会过滤 AdminOnly 菜单。
type MenuHandler struct {
	service *service.MenuService
	plugins *config.PluginsConfig
	enforcer *casbin.SyncedEnforcer
}

// NewMenuHandler 构造菜单处理器。
func NewMenuHandler(svc *service.MenuService, plugins *config.PluginsConfig, enforcer *casbin.SyncedEnforcer) *MenuHandler {
	return &MenuHandler{service: svc, plugins: plugins, enforcer: enforcer}
}

// Tree 返回菜单树；非 super-admin 时移除仅管理员可见项并按 Casbin 入口权限过滤。
func (h *MenuHandler) Tree(c *gin.Context) {
	list, err := h.service.Tree(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		filtered := plugingate.FilterMenusByPlugins(list, h.plugins)
		response.Success(c, filtered)
		return
	}

	isSuper := auth.IsSuperAdminRole(user.RoleCodes)
	filtered := list
	if !isSuper {
		filtered = filterAdminOnlyMenus(list)
	}
	if h.enforcer != nil && !isSuper {
		bindings, _ := h.service.ListPermissionBindings(c.Request.Context())
		flat, _ := h.service.ListAllFlat(c.Request.Context())
		store := menu.NewBindingStore(bindings, flat)
		filtered = menu.FilterMenusByAccess(filtered, h.enforcer, user.ID, store)
	}
	filtered = plugingate.FilterMenusByPlugins(filtered, h.plugins)
	response.Success(c, filtered)
}

func filterAdminOnlyMenus(items []model.Menu) []model.Menu {
	var filter func([]model.Menu) []model.Menu
	filter = func(nodes []model.Menu) []model.Menu {
		out := make([]model.Menu, 0, len(nodes))
		for _, it := range nodes {
			if it.AdminOnly {
				continue
			}
			child := it
			if len(it.Children) > 0 {
				child.Children = filter(it.Children)
			}
			out = append(out, child)
		}
		return out
	}
	return filter(items)
}

// GetBindings 查询菜单入口权限绑定。
func (h *MenuHandler) GetBindings(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.service.GetPermissionBindings(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// ReplaceBindings 覆盖菜单自定义入口权限绑定。
func (h *MenuHandler) ReplaceBindings(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"message": "updated"}, func(ctx context.Context, req service.MenuPermissionBindingsReplaceRequest) error {
		return h.service.ReplacePermissionBindings(ctx, id, req)
	})
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
