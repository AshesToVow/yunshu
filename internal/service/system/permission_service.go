package system

import (
	"context"
	"errors"
	"strings"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type PermissionService struct {
	permissionRepo interfaces.PermissionRepository
	enforcer       *casbin.SyncedEnforcer
}

// NewPermissionService 创建相关逻辑。
func NewPermissionService(permissionRepo interfaces.PermissionRepository, enforcer *casbin.SyncedEnforcer) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
		enforcer:       enforcer,
	}
}

// Create 创建相关的业务逻辑。
func (s *PermissionService) Create(ctx context.Context, req PermissionCreateRequest) (*PermissionItem, error) {
	resource := strings.TrimSpace(req.Resource)
	action := strings.TrimSpace(req.Action)
	if existing, err := s.permissionRepo.GetByResourceAction(ctx, resource, action); err == nil && existing != nil {
		response := NewPermissionItem(*existing)
		return &response, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerrors.Pass(ctx, "permission", "Create", err)
	}
	permission := model.Permission{
		Name:            req.Name,
		Resource:        resource,
		Action:          action,
		Description:     req.Description,
		K8sScopeEnabled: req.K8sScopeEnabled,
	}
	if err := s.permissionRepo.Create(ctx, &permission); err != nil {
		if existing, getErr := s.permissionRepo.GetByResourceAction(ctx, resource, action); getErr == nil && existing != nil {
			response := NewPermissionItem(*existing)
			return &response, nil
		}
		return nil, bizerrors.Pass(ctx, "permission", "Create", err)
	}
	response := NewPermissionItem(permission)
	return &response, nil
}

// Update 更新相关的业务逻辑。
func (s *PermissionService) Update(ctx context.Context, id uint, req PermissionUpdateRequest) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrPermissionNotFound
		}
		return nil, bizerrors.Pass(ctx, "permission", "Update", err)
	}

	oldResource := permission.Resource
	oldAction := permission.Action
	if req.Name != nil {
		permission.Name = *req.Name
	}
	if req.Resource != nil {
		permission.Resource = *req.Resource
	}
	if req.Action != nil {
		permission.Action = *req.Action
	}
	if req.Description != nil {
		permission.Description = *req.Description
	}
	if req.K8sScopeEnabled != nil {
		permission.K8sScopeEnabled = *req.K8sScopeEnabled
	}

	if err = s.permissionRepo.Save(ctx, permission); err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "Update", err)
	}
	if err = ReplacePermissionResource(s.enforcer, oldResource, oldAction, permission.Resource, permission.Action); err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "Update", err)
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

// Delete 删除相关的业务逻辑。
func (s *PermissionService) Delete(ctx context.Context, id uint) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrPermissionNotFound
		}
		return bizerrors.Pass(ctx, "permission", "Delete", err)
	}

	if err = s.permissionRepo.Delete(ctx, permission); err != nil {
		return bizerrors.Pass(ctx, "permission", "Delete", err)
	}
	return RemovePermissionPolicies(s.enforcer, permission.Resource, permission.Action)
}

// Detail 查询详情相关的业务逻辑。
func (s *PermissionService) Detail(ctx context.Context, id uint) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrPermissionNotFound
		}
		return nil, bizerrors.Pass(ctx, "permission", "Detail", err)
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

// List 查询列表相关的业务逻辑。
func (s *PermissionService) List(ctx context.Context, query PermissionListQuery) (*pagination.Result[PermissionItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	permissions, total, err := s.permissionRepo.List(ctx, repository.PermissionListParams{
		Keyword:    query.Keyword,
		Page:       page,
		PageSize:   pageSize,
		K8sScope:   query.K8sScope,
		K8sRelated: query.K8sRelated,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "List", err)
	}

	list := make([]PermissionItem, 0, len(permissions))
	for _, permission := range permissions {
		list = append(list, NewPermissionItem(permission))
	}

	return &pagination.Result[PermissionItem]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListAllFiltered 返回符合筛选条件的全部权限（不受分页 100 条上限约束）。
func (s *PermissionService) ListAllFiltered(ctx context.Context, query PermissionListQuery) ([]PermissionItem, error) {
	permissions, err := s.permissionRepo.ListFiltered(ctx, repository.PermissionListParams{
		Keyword:    query.Keyword,
		K8sScope:   query.K8sScope,
		K8sRelated: query.K8sRelated,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "ListAllFiltered", err)
	}
	list := make([]PermissionItem, 0, len(permissions))
	for _, permission := range permissions {
		list = append(list, NewPermissionItem(permission))
	}
	return list, nil
}

// BatchSetK8sScope 批量更新 K8s 范围校验开关（默认仅集群资源接口路径）。
// 关闭时写入 [k8s-scope=off] 描述标记，开启时移除该标记，与 isScopedK8sPermission 对齐。
func (s *PermissionService) BatchSetK8sScope(ctx context.Context, req PermissionBatchK8sScopeRequest) (*PermissionBatchK8sScopeResponse, error) {
	k8sRelated := strings.TrimSpace(req.K8sRelated)
	if k8sRelated == "" {
		k8sRelated = "on"
	}
	params := repository.PermissionListParams{
		Keyword:    strings.TrimSpace(req.Keyword),
		K8sRelated: k8sRelated,
	}
	affected, err := s.permissionRepo.BatchSetK8sScopeEnabled(ctx, params, req.Enabled)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "BatchSetK8sScope", err)
	}
	list, err := s.permissionRepo.ListFiltered(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission", "BatchSetK8sScope", err)
	}
	const offTag = "[k8s-scope=off]"
	for i := range list {
		p := &list[i]
		desc := p.Description
		hasOff := strings.Contains(strings.ToLower(desc), strings.ToLower(offTag))
		if req.Enabled {
			if !hasOff {
				continue
			}
			p.Description = strings.TrimSpace(strings.ReplaceAll(desc, offTag, ""))
			p.Description = strings.TrimSpace(strings.ReplaceAll(p.Description, "  ", " "))
			p.K8sScopeEnabled = true
		} else {
			if hasOff {
				continue
			}
			if strings.TrimSpace(desc) == "" {
				p.Description = offTag
			} else {
				p.Description = strings.TrimSpace(desc) + " " + offTag
			}
			p.K8sScopeEnabled = false
		}
		if err := s.permissionRepo.Save(ctx, p); err != nil {
			return nil, bizerrors.Pass(ctx, "permission", "BatchSetK8sScope", err)
		}
	}
	return &PermissionBatchK8sScopeResponse{Affected: affected}, nil
}
