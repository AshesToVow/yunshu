package handler

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// AlertPlatformHandler 告警平台：数据源、静默、监控规则、处理人、值班、PromQL、Consul 目录。
type AlertPlatformHandler struct {
	ds          *service.AlertDatasourceService
	silence     *service.AlertSilenceService
	maintenance *service.AlertMaintenanceService
	rules       *service.AlertMonitorRuleService
	assign      *service.AlertRuleAssigneeService
	duty        *service.AlertDutyService
	consul      *service.AlertConsulService
}

func NewAlertPlatformHandler(
	ds *service.AlertDatasourceService,
	silence *service.AlertSilenceService,
	maintenance *service.AlertMaintenanceService,
	rules *service.AlertMonitorRuleService,
	assign *service.AlertRuleAssigneeService,
	duty *service.AlertDutyService,
	consul *service.AlertConsulService,
) *AlertPlatformHandler {
	return &AlertPlatformHandler{
		ds: ds, silence: silence, maintenance: maintenance,
		rules: rules, assign: assign, duty: duty, consul: consul,
	}
}

func alertPlatformUserID(c *gin.Context) uint {
	if u, ok := auth.CurrentUserFromContext(c); ok {
		return u.ID
	}
	return 0
}

func (h *AlertPlatformHandler) ListDatasources(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertDatasourceListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.ds.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateDatasource(c *gin.Context) {
	ServeJSON(c, h.ds.Create)
}

func (h *AlertPlatformHandler) UpdateDatasource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertDatasourceUpsertRequest) (any, error) {
		return h.ds.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteDatasource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.ds.Delete(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// PingDatasource GET — Prometheus 数据源连通性检测（即时查询 vector(1)）。
func (h *AlertPlatformHandler) PingDatasource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	res, err := h.ds.PingDatasource(c.Request.Context(), id)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AlertPlatformHandler) PromQuery(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.PromQueryRequest) (any, error) {
		raw, err := h.ds.PromQuery(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return gin.H{"data": raw}, nil
	})
}

func (h *AlertPlatformHandler) PromQueryRange(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.PromQueryRangeRequest) (any, error) {
		raw, err := h.ds.PromQueryRange(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return gin.H{"data": raw}, nil
	})
}

func (h *AlertPlatformHandler) PromActiveAlerts(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	raw, err := h.ds.PrometheusActiveAlerts(c.Request.Context(), id)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"data": raw})
}

func (h *AlertPlatformHandler) ListConsulEndpoints(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertConsulEndpointListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.consul.ListEndpoints(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateConsulEndpoint(c *gin.Context) {
	ServeJSON(c, h.consul.CreateEndpoint)
}

func (h *AlertPlatformHandler) UpdateConsulEndpoint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertConsulEndpointUpsertRequest) (any, error) {
		return h.consul.UpdateEndpoint(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteConsulEndpoint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.consul.DeleteEndpoint(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) PingConsulEndpoint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.consul.PingEndpoint(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true, "message": "consul ok"})
}

func (h *AlertPlatformHandler) SyncConsulEndpoint(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	res, err := h.consul.SyncEndpoint(c.Request.Context(), id)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AlertPlatformHandler) ListMonitorObjects(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertMonitorObjectListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.consul.ListObjects(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) ListSilences(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertSilenceListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.silence.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateSilence(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.AlertSilenceUpsertRequest) (any, error) {
		return h.silence.Create(ctx, alertPlatformUserID(c), req)
	})
}

func (h *AlertPlatformHandler) CreateSilenceBatch(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.AlertSilenceBatchRequest) (gin.H, error) {
		n, err := h.silence.CreateBatch(ctx, alertPlatformUserID(c), req)
		if err != nil {
			return nil, err
		}
		return gin.H{"created": n}, nil
	})
}

func (h *AlertPlatformHandler) UpdateSilence(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertSilenceUpsertRequest) (any, error) {
		return h.silence.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteSilence(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.silence.Delete(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) ListMonitorRules(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertMonitorRuleListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.rules.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateMonitorRule(c *gin.Context) {
	ServeJSON(c, h.rules.Create)
}

func (h *AlertPlatformHandler) ImportPrometheusYAML(c *gin.Context) {
	ServeJSON(c, h.rules.ImportPrometheusYAML)
}

func (h *AlertPlatformHandler) ListRuleTemplates(c *gin.Context) {
	group := c.Query("group")
	response.Success(c, gin.H{"list": service.ListAlertRuleTemplates(group)})
}

func (h *AlertPlatformHandler) CreateMonitorRuleFromTemplate(c *gin.Context) {
	ServeJSON(c, h.rules.CreateFromTemplate)
}

func (h *AlertPlatformHandler) UpdateMonitorRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertMonitorRuleUpsertRequest) (any, error) {
		return h.rules.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteMonitorRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.rules.Delete(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) GetMonitorRuleAssignees(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	list, err := h.assign.ListByRule(c.Request.Context(), id)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *AlertPlatformHandler) UpsertMonitorRuleAssignees(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertRuleAssigneeUpsertRequest) (any, error) {
		return h.assign.UpsertPrimary(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) ListDutyBlocks(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertDutyBlockListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.duty.ListBlocks(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateDutyBlock(c *gin.Context) {
	ServeJSON(c, h.duty.CreateBlock)
}

func (h *AlertPlatformHandler) UpdateDutyBlock(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertDutyBlockUpsertRequest) (any, error) {
		return h.duty.UpdateBlock(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteDutyBlock(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.duty.DeleteBlock(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) ListDutyCalendar(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertDutyCalendarQuery) (gin.H, error) {
		list, err := h.duty.ListCalendar(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list, "items": list}, nil
	})
}

func (h *AlertPlatformHandler) ValidateDutyBlocks(c *gin.Context) {
	ServeJSON(c, h.duty.ValidateBlocks)
}

func (h *AlertPlatformHandler) HandoffDutyBlock(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertDutyHandoffRequest) (any, error) {
		return h.duty.HandoffBlock(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) ListMaintenanceWindows(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertMaintenanceListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.maintenance.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateMaintenanceWindow(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.AlertMaintenanceUpsertRequest) (any, error) {
		return h.maintenance.Create(ctx, alertPlatformUserID(c), req)
	})
}

func (h *AlertPlatformHandler) UpdateMaintenanceWindow(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertMaintenanceUpsertRequest) (any, error) {
		return h.maintenance.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteMaintenanceWindow(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	if err := h.maintenance.Delete(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
