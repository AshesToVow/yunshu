package handler

import (
	"context"
	"strconv"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/plugin"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type PolicyHandler struct {
	service     *service.PolicyService
	governance  *service.PolicyGovernanceService
	permSvc     *service.PermissionService
	plugins     *config.PluginsConfig
}

// NewPolicyHandler 创建相关逻辑。
func NewPolicyHandler(
	svc *service.PolicyService,
	governance *service.PolicyGovernanceService,
	permSvc *service.PermissionService,
	plugins *config.PluginsConfig,
) *PolicyHandler {
	return &PolicyHandler{service: svc, governance: governance, permSvc: permSvc, plugins: plugins}
}

// List godoc
// @Summary List policies
// @Description List current role-permission policy bindings.
// @Tags Policy
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]service.PolicyItemResponse} "success"
// @Router /api/v1/policies [get]
func (h *PolicyHandler) List(c *gin.Context) {
	data, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	out := make([]service.PolicyItemResponse, 0, len(data))
	for _, item := range data {
		if plugin.IsAPIResourceAllowed(item.Resource, h.plugins) {
			out = append(out, item)
		}
	}
	response.Success(c, out)
}

// MenuLinks 返回 permission key → 关联菜单 path。
func (h *PolicyHandler) MenuLinks(c *gin.Context) {
	if h.governance == nil {
		response.Success(c, service.PermissionMenuLinksResponse{Links: map[string][]service.MenuLink{}})
		return
	}
	response.Success(c, h.governance.MenuLinks(c.Request.Context()))
}

// Simulate 分层模拟 API 鉴权结果。
func (h *PolicyHandler) Simulate(c *gin.Context) {
	if h.governance == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.PolicySimulateRequest) (*service.PolicySimulateResponse, error) {
		return h.governance.Simulate(ctx, req)
	})
}

// Conflicts 分析角色权限与菜单治理冲突。
func (h *PolicyHandler) Conflicts(c *gin.Context) {
	if h.governance == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	roleID, err := strconv.ParseUint(c.Query("role_id"), 10, 32)
	if err != nil || roleID == 0 {
		response.Error(c, constants.ErrInvalidRequestParam)
		return
	}
	data, err := h.governance.Conflicts(c.Request.Context(), uint(roleID))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// FixMenuEntryAPIs 一键补齐菜单入口 API（缺失权限项会自动创建）。
func (h *PolicyHandler) FixMenuEntryAPIs(c *gin.Context) {
	if h.governance == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	roleID, err := strconv.ParseUint(c.Query("role_id"), 10, 32)
	if err != nil || roleID == 0 {
		response.Error(c, constants.ErrInvalidRequestParam)
		return
	}
	data, err := h.governance.FixMenuEntryAPIs(c.Request.Context(), uint(roleID))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// FixDisabledPluginPolicies 清理角色上属于未启用插件的 Casbin 策略。
func (h *PolicyHandler) FixDisabledPluginPolicies(c *gin.Context) {
	if h.governance == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	roleID, err := strconv.ParseUint(c.Query("role_id"), 10, 32)
	if err != nil || roleID == 0 {
		response.Error(c, constants.ErrInvalidRequestParam)
		return
	}
	data, err := h.governance.FixDisabledPluginPolicies(c.Request.Context(), uint(roleID))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// PermissionTree 返回菜单+API 合一授权树。
func (h *PolicyHandler) PermissionTree(c *gin.Context) {
	if h.governance == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	roleID, err := strconv.ParseUint(c.Query("role_id"), 10, 32)
	if err != nil || roleID == 0 {
		response.Error(c, constants.ErrInvalidRequestParam)
		return
	}
	data, err := h.governance.PermissionTree(c.Request.Context(), uint(roleID))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Grant godoc
// @Summary Grant policy
// @Router /api/v1/policies [post]
func (h *PolicyHandler) Grant(c *gin.Context) {
	ServeJSONOK(c, gin.H{"message": "granted"}, func(ctx context.Context, req service.PolicyGrantRequest) error {
		if err := h.validatePermissionPlugin(ctx, req.PermissionID); err != nil {
			return err
		}
		return h.service.Grant(ctx, req)
	})
}

// Revoke godoc
// @Summary Revoke policy
// @Router /api/v1/policies [delete]
func (h *PolicyHandler) Revoke(c *gin.Context) {
	ServeJSONOK(c, gin.H{"message": "revoked"}, func(ctx context.Context, req service.PolicyGrantRequest) error {
		if err := h.validatePermissionPlugin(ctx, req.PermissionID); err != nil {
			return err
		}
		return h.service.Revoke(ctx, req)
	})
}

func (h *PolicyHandler) validatePermissionPlugin(ctx context.Context, permissionID uint) error {
	if h.plugins == nil || permissionID == 0 {
		return nil
	}
	item, err := h.permSvc.Detail(ctx, permissionID)
	if err != nil {
		return err
	}
	if !plugin.IsAPIResourceAllowed(item.Resource, h.plugins) {
		return constants.ErrBadRequestWithMsg("该权限所属插件未启用，无法授权")
	}
	return nil
}
