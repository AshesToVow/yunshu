package handler

import (
	"context"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"
	"yunshu/internal/service/alert"
	k8ssvc "yunshu/internal/service/k8s"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PlatformFeaturesHandler 聚合新增平台能力 HTTP 入口（避免频繁改 Wire 构造签名）。
type PlatformFeaturesHandler struct {
	db           *gorm.DB
	monitorRules *service.AlertMonitorRuleService
}

func NewPlatformFeaturesHandler(db *gorm.DB, monitorRules *service.AlertMonitorRuleService) *PlatformFeaturesHandler {
	return &PlatformFeaturesHandler{db: db, monitorRules: monitorRules}
}

func (h *PlatformFeaturesHandler) ruleChangeSvc() *alert.AlertRuleChangeService {
	return alert.NewAlertRuleChangeService(h.db, h.monitorRules)
}

func (h *PlatformFeaturesHandler) crTemplateSvc() *k8ssvc.K8sCrTemplateService {
	return k8ssvc.NewK8sCrTemplateService(h.db)
}

func (h *PlatformFeaturesHandler) ListPromqlSavedQueries(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	ServeQuery(c, func(ctx context.Context, _ struct{}) (gin.H, error) {
		svc := alert.NewPromqlSavedQueryService(h.db)
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
		return alert.NewPromqlSavedQueryService(h.db).Create(ctx, userID, req)
	})
}

func (h *PlatformFeaturesHandler) DeletePromqlSavedQuery(c *gin.Context) {
	userID, _ := currentAlertUser(c)
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := alert.NewPromqlSavedQueryService(h.db).Delete(c.Request.Context(), userID, id); err != nil {
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
