package handler

import (
	"context"
	"fmt"
	"net/url"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

func (h *CicdHandler) ListServices(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q cicd.ServiceListQuery) (*pagination.Result[cicd.ServiceItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		return h.svc.ListServices(ctx, q)
	})
}

func (h *CicdHandler) GetService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "view"); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.GetService(c.Request.Context(), projectID, serviceID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CicdHandler) CreateService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCanCreateCicdService(c.Request.Context(), projectID, actor); err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req cicd.ServiceUpsertRequest) (*model.CicdService, error) {
		req.ProjectID = projectID
		return h.svc.UpsertService(ctx, 0, req)
	})
}

func (h *CicdHandler) UpdateService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "manage"); err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ProjectID = projectID
	row, err := h.svc.UpsertService(c.Request.Context(), serviceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *CicdHandler) DeleteService(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertCicdAccess(c.Request.Context(), projectID, serviceID, actor, "manage"); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), projectID, serviceID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) GetCiConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	row, err := h.svc.GetCiConfigView(c.Request.Context(), projectID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *CicdHandler) UpsertCiConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.CiConfigUpsertRequest) (*cicd.CiConfigUpsertResult, error) {
		return h.svc.UpsertCiConfig(ctx, projectID, serviceID, req)
	})
}

func (h *CicdHandler) ListDeployConfigs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	rows, err := h.svc.ListDeployConfigs(c.Request.Context(), projectID, serviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) CreateDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	ServeJSON201(c, func(ctx context.Context, req cicd.DeployConfigUpsertRequest) (*cicd.DeployConfigUpsertResult, error) {
		return h.svc.UpsertDeployConfig(ctx, projectID, serviceID, 0, req)
	})
}

func (h *CicdHandler) UpdateDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	configID, err := parseUintParam(c, "configId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req cicd.DeployConfigUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.svc.UpsertDeployConfig(c.Request.Context(), projectID, serviceID, configID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CicdHandler) DeleteDeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "manage") {
		return
	}
	configID, err := parseUintParam(c, "configId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteDeployConfig(c.Request.Context(), projectID, serviceID, configID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// DownloadHelmScaffold 按服务生成 helm/ 脚手架 zip（解压到业务仓库根目录即可）。
func (h *CicdHandler) DownloadHelmScaffold(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serviceID, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if !h.requireCicdServiceAccess(c, projectID, serviceID, "view") {
		return
	}
	var q cicd.HelmScaffoldQuery
	_ = c.ShouldBindQuery(&q)
	filename, data, err := h.svc.BuildHelmScaffoldZip(c.Request.Context(), projectID, serviceID, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(filename)))
	c.Data(200, "application/zip", data)
}

// DownloadHelmScaffoldPreview 未绑定服务时按表单参数预览下载脚手架。
func (h *CicdHandler) DownloadHelmScaffoldPreview(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.RequireProjectAdmin(c.Request.Context(), projectID, h.cicdActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	var q cicd.HelmScaffoldQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	filename, data, err := h.svc.BuildHelmScaffoldZipPreview(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(filename)))
	c.Data(200, "application/zip", data)
}
