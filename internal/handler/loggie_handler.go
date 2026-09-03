package handler

import (
	"context"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type LoggieHandler struct {
	svc *service.LoggieAgentService
}

func NewLoggieHandler(svc *service.LoggieAgentService) *LoggieHandler {
	return &LoggieHandler{svc: svc}
}

func (h *LoggieHandler) ReportHeartbeat(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.LoggieHeartbeatRequest) (gin.H, error) {
		if err := h.svc.ReportHeartbeat(ctx, req); err != nil {
			return nil, err
		}
		return gin.H{"message": "ok"}, nil
	})
}

func (h *LoggieHandler) Bootstrap(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieBootstrapRequest) (*service.LoggieBootstrapResult, error) {
		return h.svc.Bootstrap(ctx, projectID, req)
	})
}

func (h *LoggieHandler) ListStatus(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListStatus(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *LoggieHandler) DownloadPipeline(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintQuery(c, "server_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	bundle, err := h.svc.GeneratePipelineBundle(c.Request.Context(), projectID, serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	kind := c.DefaultQuery("file", "pipeline")
	switch kind {
	case "env":
		exportutil.ServeBytes(c, bundle.EnvFilename, "text/plain; charset=utf-8", []byte(bundle.EnvFile))
	case "heartbeat":
		exportutil.ServeBytes(c, bundle.HeartbeatFilename, "text/x-sh; charset=utf-8", []byte(bundle.HeartbeatScript))
	case "pipelines":
		name := bundle.PipelinesFilename
		if name == "" {
			name = "pipelines.yml"
		}
		content := bundle.PipelinesOnlyYAML
		if content == "" {
			content = bundle.PipelineYAML
		}
		exportutil.ServeBytes(c, name, "application/x-yaml; charset=utf-8", []byte(content))
	case "start":
		name := bundle.StartFilename
		if name == "" {
			name = "start.sh"
		}
		exportutil.ServeBytes(c, name, "text/x-sh; charset=utf-8", []byte(bundle.StartScript))
	default:
		exportutil.ServeBytes(c, bundle.PipelineFilename, "application/x-yaml; charset=utf-8", []byte(bundle.PipelineYAML))
	}
}

func (h *LoggieHandler) PreviewBootstrapSources(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintQuery(c, "server_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.PreviewBootstrapSources(c.Request.Context(), projectID, serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *LoggieHandler) DeployConfig(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieDeployRequest) (*service.LoggieDeployResult, error) {
		return h.svc.DeployConfig(ctx, projectID, req)
	})
}

func (h *LoggieHandler) RestartLoggie(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieDeployRequest) (*service.LoggieDeployResult, error) {
		return h.svc.RestartLoggie(ctx, projectID, req)
	})
}

func (h *LoggieHandler) StartLoggie(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieDeployRequest) (*service.LoggieDeployResult, error) {
		return h.svc.StartLoggie(ctx, projectID, req)
	})
}

func (h *LoggieHandler) StopLoggie(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieDeployRequest) (*service.LoggieDeployResult, error) {
		return h.svc.StopLoggie(ctx, projectID, req)
	})
}

func (h *LoggieHandler) InstallLoggie(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieInstallRequest) (*service.LoggieDeployResult, error) {
		return h.svc.InstallLoggie(ctx, projectID, req)
	})
}

func (h *LoggieHandler) UninstallLoggie(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieUninstallRequest) (*service.LoggieDeployResult, error) {
		return h.svc.UninstallLoggie(ctx, projectID, req)
	})
}

func (h *LoggieHandler) SyncFromLogSources(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.LoggieDeployRequest) (*service.LoggieDeployResult, error) {
		return h.svc.SyncFromLogSources(ctx, projectID, req)
	})
}

func (h *LoggieHandler) ESConfigPreview(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("服务未就绪"))
		return
	}
	cfg, err := h.svc.ESConfigForUI(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *LoggieHandler) SetESConnection(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("服务未就绪"))
		return
	}
	var req service.SetESConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	cfg, err := h.svc.SetESConnection(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, cfg)
}
