package handler

import (
	"context"
	"strconv"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/dictmask"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// sensitiveDictPermissionResource 敏感字典伪权限资源：GET=可读（列表可见+可获明文），PUT=可维护。
const sensitiveDictPermissionResource = "/api/v1/dict/sensitive"

type DictEntryHandler struct {
	svc      *service.DictEntryService
	enforcer *casbin.SyncedEnforcer
}

func NewDictEntryHandler(svc *service.DictEntryService, enforcer *casbin.SyncedEnforcer) *DictEntryHandler {
	return &DictEntryHandler{svc: svc, enforcer: enforcer}
}

type sensitiveDictAccess struct {
	read  bool
	write bool
}

// resolveSensitiveDictAccess 判定当前登录人对敏感字典的读写能力：
// 内置超管全放行；其余按伪权限 /api/v1/dict/sensitive 的 GET/PUT 判定；无法判定时按拒绝处理。
func resolveSensitiveDictAccess(enforcer *casbin.SyncedEnforcer, actor *auth.CurrentUser, hasActor bool) sensitiveDictAccess {
	if !hasActor || actor == nil {
		return sensitiveDictAccess{}
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return sensitiveDictAccess{read: true, write: true}
	}
	if enforcer == nil {
		return sensitiveDictAccess{}
	}
	subject := service.UserSubject(actor.ID)
	read, err := enforcer.Enforce(subject, sensitiveDictPermissionResource, "GET")
	if err != nil {
		return sensitiveDictAccess{}
	}
	write, err := enforcer.Enforce(subject, sensitiveDictPermissionResource, "PUT")
	if err != nil {
		return sensitiveDictAccess{}
	}
	// 可维护必然伴随可见：能改值的人已具备更强的改写入能力，不必强制再加 GET。
	return sensitiveDictAccess{read: read || write, write: write}
}

func (h *DictEntryHandler) sensitiveAccess(c *gin.Context) sensitiveDictAccess {
	actor, ok := auth.CurrentUserFromContext(c)
	return resolveSensitiveDictAccess(h.enforcer, actor, ok)
}

func sensitiveDictForbidden() error {
	return constants.ErrForbiddenWithMsg("无权访问敏感字典（密钥/凭据类），请联系管理员授权")
}

// List 普通字典全量可见；无敏感读权限时自动剔除敏感类型，显式查询敏感类型直接拒绝，
// 避免通过关键词模糊匹配探测密钥类明文。
func (h *DictEntryHandler) List(c *gin.Context) {
	query, ok := bindQuery[service.DictEntryListQuery](c)
	if !ok {
		return
	}
	access := h.sensitiveAccess(c)
	if !access.read {
		if dictmask.SensitiveDictType(query.DictType) {
			response.Error(c, sensitiveDictForbidden())
			return
		}
		query.ExcludeSensitive = true
	}
	data, err := h.svc.List(c.Request.Context(), query)
	if err != nil {
		abortService(c, err)
		return
	}
	response.Success(c, data)
}

func (h *DictEntryHandler) Create(c *gin.Context) {
	access := h.sensitiveAccess(c)
	ServeJSON201(c, func(ctx context.Context, req service.DictEntryCreateRequest) (*model.DictEntry, error) {
		if dictmask.SensitiveDictType(req.DictType) && !access.write {
			return nil, sensitiveDictForbidden()
		}
		return h.svc.Create(ctx, req)
	})
}

func (h *DictEntryHandler) Update(c *gin.Context) {
	access := h.sensitiveAccess(c)
	ServePatch(c, func(ctx context.Context, id uint, req service.DictEntryUpdateRequest) (*model.DictEntry, error) {
		if err := h.assertSensitiveWrite(ctx, access, id, req.DictType); err != nil {
			return nil, err
		}
		return h.svc.Update(ctx, id, req)
	}, "")
}

func (h *DictEntryHandler) Delete(c *gin.Context) {
	access := h.sensitiveAccess(c)
	ServeDelete(c, func(ctx context.Context, id uint) error {
		if err := h.assertSensitiveWrite(ctx, access, id, ""); err != nil {
			return err
		}
		return h.svc.Delete(ctx, id)
	}, "")
}

// assertSensitiveWrite 原类型（库中）或新类型（请求体）任一为敏感类型即要求敏感字典维护权限。
func (h *DictEntryHandler) assertSensitiveWrite(ctx context.Context, access sensitiveDictAccess, id uint, newDictType string) error {
	if access.write {
		return nil
	}
	if dictmask.SensitiveDictType(newDictType) {
		return sensitiveDictForbidden()
	}
	current, err := h.svc.Get(ctx, id)
	if err != nil {
		return err
	}
	if dictmask.SensitiveDictType(current.DictType) {
		return sensitiveDictForbidden()
	}
	return nil
}

func (h *DictEntryHandler) Options(c *gin.Context) {
	dictType := c.Param("dictType")
	list, err := h.svc.Options(c.Request.Context(), dictType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *DictEntryHandler) RevealValue(c *gin.Context) {
	access := h.sensitiveAccess(c)
	if !access.read {
		response.Error(c, sensitiveDictForbidden())
		return
	}
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.Error(c, constants.ErrBadRequestWithMsg("字典条目 id 无效"))
		return
	}
	val, err := h.svc.RevealValue(c.Request.Context(), uint(id64))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"value": val})
}
