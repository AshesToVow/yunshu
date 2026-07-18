package system

import (
	"context"
	"errors"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type RoleService struct {
	roleRepo interfaces.RoleRepository
	userRepo interfaces.UserRepository
	enforcer *casbin.SyncedEnforcer
}

// NewRoleService 创建相关逻辑。
func NewRoleService(roleRepo interfaces.RoleRepository, userRepo interfaces.UserRepository, enforcer *casbin.SyncedEnforcer) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		userRepo: userRepo,
		enforcer: enforcer,
	}
}

// Create 创建相关的业务逻辑。
func (s *RoleService) Create(ctx context.Context, req RoleCreateRequest) (*RoleItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	role := model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      status,
	}
	if err := s.roleRepo.Create(ctx, &role); err != nil {
		return nil, bizerrors.Pass(ctx, "role", "Create", err)
	}
	response := NewRoleItem(role)
	return &response, nil
}

// Update 更新相关的业务逻辑。
func (s *RoleService) Update(ctx context.Context, id uint, req RoleUpdateRequest) (*RoleItem, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrRoleNotFound
		}
		return nil, bizerrors.Pass(ctx, "role", "Update", err)
	}

	oldCode := role.Code
	oldStatus := role.Status
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		role.Code = *req.Code
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}

	if err = s.roleRepo.Save(ctx, role); err != nil {
		return nil, bizerrors.Pass(ctx, "role", "Update", err)
	}
	if err = ReplaceRoleCode(s.enforcer, oldCode, role.Code); err != nil {
		return nil, bizerrors.Pass(ctx, "role", "Update", err)
	}
	// 状态切换即时生效：禁用移除 user→role 分组，启用按 DB 绑定补回。
	if err = s.applyStatusTransition(ctx, role.Code, oldStatus, role.Status); err != nil {
		return nil, bizerrors.Pass(ctx, "role", "Update", err)
	}
	response := NewRoleItem(*role)
	return &response, nil
}

// applyStatusTransition 在角色启用/禁用状态发生变化时同步 Casbin user→role 分组。
func (s *RoleService) applyStatusTransition(ctx context.Context, roleCode string, oldStatus, newStatus int) error {
	if oldStatus == newStatus {
		return nil
	}
	if newStatus == model.StatusDisabled {
		return DisableRoleGroupings(s.enforcer, roleCode)
	}
	userIDs, err := s.userRepo.ListUserIDsByRoleCode(ctx, roleCode)
	if err != nil {
		return err
	}
	return EnableRoleGroupings(s.enforcer, roleCode, userIDs)
}

// Delete 删除相关的业务逻辑。
func (s *RoleService) Delete(ctx context.Context, id uint) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrRoleNotFound
		}
		return bizerrors.Pass(ctx, "role", "Delete", err)
	}

	if err = s.roleRepo.Delete(ctx, role); err != nil {
		return bizerrors.Pass(ctx, "role", "Delete", err)
	}
	return RemoveRolePolicies(s.enforcer, role.Code)
}

// Detail 查询详情相关的业务逻辑。
func (s *RoleService) Detail(ctx context.Context, id uint) (*RoleItem, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrRoleNotFound
		}
		return nil, bizerrors.Pass(ctx, "role", "Detail", err)
	}
	response := NewRoleItem(*role)
	return &response, nil
}

// List 查询列表相关的业务逻辑。
func (s *RoleService) List(ctx context.Context, query RoleListQuery) (*pagination.Result[RoleItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	roles, total, err := s.roleRepo.List(ctx, repository.RoleListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "role", "List", err)
	}

	list := make([]RoleItem, 0, len(roles))
	for _, role := range roles {
		list = append(list, NewRoleItem(role))
	}

	return &pagination.Result[RoleItem]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
