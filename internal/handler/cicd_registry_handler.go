package handler

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/cicd"

	"github.com/gin-gonic/gin"
)

// --- Image registries (platform) ---

func (h *CicdHandler) ListRegistries(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q cicd.RegistryListQuery) (*pagination.Result[cicd.RegistryItem], error) {
		return h.svc.ListRegistries(ctx, q)
	})
}

func (h *CicdHandler) GetRegistry(c *gin.Context) {
	id, err := parseUintParam(c, "registryId")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.GetRegistry(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CicdHandler) CreateRegistry(c *gin.Context) {
	ServeJSON201(c, func(ctx context.Context, req cicd.RegistryUpsertRequest) (*cicd.RegistryItem, error) {
		return h.svc.UpsertRegistry(ctx, 0, req)
	})
}

func (h *CicdHandler) UpdateRegistry(c *gin.Context) {
	id, err := parseUintParam(c, "registryId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.RegistryUpsertRequest) (*cicd.RegistryItem, error) {
		return h.svc.UpsertRegistry(ctx, id, req)
	})
}

func (h *CicdHandler) DeleteRegistry(c *gin.Context) {
	id, err := parseUintParam(c, "registryId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteRegistry(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) PingRegistry(c *gin.Context) {
	id, err := parseUintParam(c, "registryId")
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.PingRegistry(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *CicdHandler) GetProjectRegistryBinding(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	bind, err := h.svc.GetProjectRegistryBinding(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, bind)
}

func (h *CicdHandler) UpsertProjectRegistryBinding(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.ProjectRegistryBindingRequest) (*model.ProjectRegistryBinding, error) {
		return h.svc.UpsertProjectRegistryBinding(ctx, projectID, req)
	})
}

func (h *CicdHandler) DeleteProjectRegistryBinding(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteProjectRegistryBinding(c.Request.Context(), projectID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) ListHarborProjects(c *gin.Context) {
	registryID := parseOptionalUintQuery(c, "registry_id")
	projectID := parseOptionalUintQuery(c, "project_id")
	rows, err := h.svc.ListHarborProjects(c.Request.Context(), registryID, projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) ListHarborRepositories(c *gin.Context) {
	registryID := parseOptionalUintQuery(c, "registry_id")
	projectID := parseOptionalUintQuery(c, "project_id")
	hp := strings.TrimSpace(c.Query("harbor_project"))
	rows, err := h.svc.ListHarborRepositories(c.Request.Context(), registryID, projectID, hp)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) ListHarborArtifacts(c *gin.Context) {
	registryID := parseOptionalUintQuery(c, "registry_id")
	projectID := parseOptionalUintQuery(c, "project_id")
	hp := strings.TrimSpace(c.Query("harbor_project"))
	repo := strings.TrimSpace(c.Query("repository"))
	rows, err := h.svc.ListHarborArtifacts(c.Request.Context(), registryID, projectID, hp, repo)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) DeleteHarborArtifact(c *gin.Context) {
	var req struct {
		RegistryID    uint   `json:"registry_id"`
		ProjectID     uint   `json:"project_id"`
		HarborProject string `json:"harbor_project"`
		Repository    string `json:"repository" binding:"required"`
		Reference     string `json:"reference" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteHarborArtifact(
		c.Request.Context(),
		req.RegistryID,
		req.ProjectID,
		req.HarborProject,
		req.Repository,
		req.Reference,
	); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) ListCleanupPolicies(c *gin.Context) {
	registryID := parseOptionalUintQuery(c, "registry_id")
	rows, err := h.svc.ListCleanupPolicies(c.Request.Context(), registryID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) CreateCleanupPolicy(c *gin.Context) {
	ServeJSON201(c, func(ctx context.Context, req cicd.CleanupPolicyUpsertRequest) (*model.ImageCleanupPolicy, error) {
		return h.svc.UpsertCleanupPolicy(ctx, 0, req)
	})
}

func (h *CicdHandler) UpdateCleanupPolicy(c *gin.Context) {
	id, err := parseUintParam(c, "policyId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.CleanupPolicyUpsertRequest) (*model.ImageCleanupPolicy, error) {
		return h.svc.UpsertCleanupPolicy(ctx, id, req)
	})
}

func (h *CicdHandler) DeleteCleanupPolicy(c *gin.Context) {
	id, err := parseUintParam(c, "policyId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteCleanupPolicy(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CicdHandler) RunCleanupPolicy(c *gin.Context) {
	id, err := parseUintParam(c, "policyId")
	if err != nil {
		response.Error(c, err)
		return
	}
	msg, err := h.svc.RunCleanupPolicyNow(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"result": msg})
}

func (h *CicdHandler) ListPipelineTemplates(c *gin.Context) {
	rows, err := h.svc.ListPipelineTemplates(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *CicdHandler) CreatePipelineTemplate(c *gin.Context) {
	ServeJSON201(c, func(ctx context.Context, req cicd.PipelineTemplateUpsertRequest) (*model.CicdPipelineTemplate, error) {
		return h.svc.UpsertPipelineTemplate(ctx, 0, req)
	})
}

func (h *CicdHandler) UpdatePipelineTemplate(c *gin.Context) {
	id, err := parseUintParam(c, "templateId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req cicd.PipelineTemplateUpsertRequest) (*model.CicdPipelineTemplate, error) {
		return h.svc.UpsertPipelineTemplate(ctx, id, req)
	})
}
