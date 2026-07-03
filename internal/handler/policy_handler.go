package handler

import (
	"context"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/plugin"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type PolicyHandler struct {
	service *service.PolicyService
	permSvc *service.PermissionService
	plugins *config.PluginsConfig
}

// NewPolicyHandler 创建相关逻辑。
func NewPolicyHandler(svc *service.PolicyService, permSvc *service.PermissionService, plugins *config.PluginsConfig) *PolicyHandler {
	return &PolicyHandler{service: svc, permSvc: permSvc, plugins: plugins}
}

// List godoc
// @Summary List policies
// @Description List current role-permission policy bindings.
// @Tags Policy
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]service.PolicyItemResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
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

// Grant godoc
// @Summary Grant policy
// @Description Bind one permission to one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Grant policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "服务器内部错误"
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
// @Description Remove one permission from one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Revoke policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "服务器内部错误"
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
