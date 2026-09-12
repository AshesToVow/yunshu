package handler

import (
	"context"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"
	"yunshu/internal/service/alert"
	k8ssvc "yunshu/internal/service/k8s"

	"github.com/gin-gonic/gin"
)

// PlatformFeaturesHandler 聚合新增平台能力 HTTP 入口（避免频繁改 Wire 构造签名）。
type PlatformFeaturesHandler struct {
	monitorRules    *service.AlertMonitorRuleService
	ruleChangeRepo  interfaces.AlertRuleChangeRepository
	promqlSavedRepo interfaces.PromqlSavedQueryRepository
	crTemplateRepo  interfaces.K8sCrTemplateRepository
}

func NewPlatformFeaturesHandler(
	monitorRules *service.AlertMonitorRuleService,
	ruleChangeRepo interfaces.AlertRuleChangeRepository,
	promqlSavedRepo interfaces.PromqlSavedQueryRepository,
	crTemplateRepo interfaces.K8sCrTemplateRepository,
) *PlatformFeaturesHandler {
	return &PlatformFeaturesHandler{
		monitorRules:    monitorRules,
		ruleChangeRepo:  ruleChangeRepo,
		promqlSavedRepo: promqlSavedRepo,
		crTemplateRepo:  crTemplateRepo,
	}
}

func (h *PlatformFeaturesHandler) ruleChangeSvc() *alert.AlertRuleChangeService {
	return alert.NewAlertRuleChangeService(h.ruleChangeRepo, h.monitorRules)
}

func (h *PlatformFeaturesHandler) crTemplateSvc() *k8ssvc.K8sCrTemplateService {
	return k8ssvc.NewK8sCrTemplateService(h.crTemplateRepo)
}

func (h *PlatformFeaturesHandler) ListPromqlSavedQueries(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	ServeQuery(c, func(ctx context.Context, _ struct{}) (gin.H, error) {
		svc := alert.NewPromqlSavedQueryService(h.promqlSavedRepo)
		list, err := svc.List(ctx, userID)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

func (h *PlatformFeaturesHandler) CreatePromqlSavedQuery(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	ServeJSON(c, func(ctx context.Context, req alert.PromqlSavedQueryUpsertRequest) (any, error) {
		return alert.NewPromqlSavedQueryService(h.promqlSavedRepo).Create(ctx, userID, req)
	})
}

func (h *PlatformFeaturesHandler) DeletePromqlSavedQuery(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := alert.NewPromqlSavedQueryService(h.promqlSavedRepo).Delete(c.Request.Context(), userID, id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *PlatformFeaturesHandler) ProposeRuleChange(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	ServeJSON(c, func(ctx context.Context, req alert.ProposeRuleChangeRequest) (any, error) {
		return h.ruleChangeSvc().Propose(ctx, userID, req)
	})
}

func (h *PlatformFeaturesHandler) ListPendingRuleChanges(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, _ struct{}) (gin.H, error) {
		list, err := h.ruleChangeSvc().ListPending(ctx)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

func (h *PlatformFeaturesHandler) ApproveRuleChange(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	userID, _ := currentAlertUser(c)
	ServeJSON(c, func(ctx context.Context, _ struct{}) (any, error) {
		return h.ruleChangeSvc().Approve(ctx, id, userID)
	})
}

func (h *PlatformFeaturesHandler) RejectRuleChange(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	userID, _ := currentAlertUser(c)
	ServeJSON(c, func(ctx context.Context, req struct {
		Comment string `json:"comment"`
	}) (any, error) {
		return nil, h.ruleChangeSvc().Reject(ctx, id, userID, req.Comment)
	})
}

func (h *PlatformFeaturesHandler) ListCrTemplates(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q struct {
		ProjectID uint   `form:"project_id"`
		Kind      string `form:"kind"`
	}) (gin.H, error) {
		list, err := h.crTemplateSvc().List(ctx, q.ProjectID, q.Kind)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

func (h *PlatformFeaturesHandler) GetCrTemplate(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.crTemplateSvc().Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *PlatformFeaturesHandler) CreateCrTemplate(c *gin.Context) {
	ServeJSON(c, h.crTemplateSvc().Create)
}

func (h *PlatformFeaturesHandler) UpdateCrTemplate(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		abortService(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req k8ssvc.K8sCrTemplateUpsertRequest) (any, error) {
		return h.crTemplateSvc().Update(ctx, id, req)
	})
}

func (h *PlatformFeaturesHandler) DeleteCrTemplate(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.crTemplateSvc().Delete(c.Request.Context(), id); err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
