package system

import (
	"context"
	"errors"
	"strings"
	"yunshu/internal/pkg/auth"
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
	roleRepo     interfaces.RoleRepository
	userRepo     interfaces.UserRepository
	enforcer     *casbin.SyncedEnforcer
	grantCleaner PrincipalGrantCleaner
}

// NewRoleService 创建相关逻辑。
func NewRoleService(roleRepo interfaces.RoleRepository, userRepo interfaces.UserRepository, enforcer *casbin.SyncedEnforcer, grantCleaner PrincipalGrantCleaner) *RoleService {
	return &RoleService{
		roleRepo:     roleRepo,
		userRepo:     userRepo,
		enforcer:     enforcer,
		grantCleaner: grantCleaner,
	}
}

// Create 创建相关的业务逻辑。
func (s *RoleService) Create(ctx context.Context, req RoleCreateRequest) (*RoleItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	code, err := normalizeRoleCode(req.Code)
	if err != nil {
		return nil, err
	}
	exists, err := s.roleRepo.ExistsByCode(ctx, code, 0)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "role", "Create", err)
	}
	if exists {
		return nil, constants.ErrConflictWithMsg("角色编码已存在")
	}

	role := model.Role{
		Name:        req.Name,
		Code:        code,
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
	// 按旧编码校验目标：非超管不得改动内置超管角色（哪怕只改名称/描述），也不得改动自己所属角色。
	actor, _ := auth.RequestUserFromContext(ctx)
	if err := authorizeRoleTarget(actor, role.Code); err != nil {
		return nil, err
	}

	oldCode := role.Code
	oldStatus := role.Status
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		newCode := strings.TrimSpace(*req.Code)
		if newCode != role.Code {
			if isBuiltinSuperAdminRole(role.Code) {
				return nil, constants.ErrForbiddenWithMsg("内置超级管理员角色编码不可修改")
			}
			code, err := normalizeRoleCode(newCode)
			if err != nil {
				return nil, err
			}
			exists, err := s.roleRepo.ExistsByCode(ctx, code, role.ID)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "role", "Update", err)
			}
			if exists {
				return nil, constants.ErrConflictWithMsg("角色编码已存在")
			}
			role.Code = code
		}
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		if *req.Status != model.StatusEnabled && *req.Status != model.StatusDisabled {
			return nil, constants.ErrBadRequestWithMsg("角色状态仅支持 1（启用）或 0（禁用）")
		}
		if *req.Status == model.StatusDisabled && isBuiltinSuperAdminRole(role.Code) {
			return nil, constants.ErrForbiddenWithMsg("内置超级管理员角色不可禁用")
		}
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

	if isBuiltinSuperAdminRole(role.Code) {
		return constants.ErrForbiddenWithMsg("内置超级管理员角色不可删除")
	}
	// 非超管不得删除自己所属角色（自锁死），也不得触碰内置超管角色（上面已拦）。
	actor, _ := auth.RequestUserFromContext(ctx)
	if err := authorizeRoleTarget(actor, role.Code); err != nil {
		return err
	}
	// 先清各授权域中以角色编码为主体的授权再删角色：编码软删后可复用，
	// 遗留授权会被新建同编码角色继承；清理失败则中止删除，重试可收敛。
	if s.grantCleaner != nil {
		if err := s.grantCleaner.CleanupPrincipalGrants(ctx, model.K8sPrincipalRole, role.Code); err != nil {
			return bizerrors.Pass(ctx, "role", "Delete", err)
		}
	}
	// Casbin 清理先于落库：失败时角色行仍在、可重试；反过来会留下指向
	// 已删除角色的孤儿策略（ReclaimOrphans 只回收无路由映射的策略）。
	if err = RemoveRolePolicies(s.enforcer, role.Code); err != nil {
		return bizerrors.Pass(ctx, "role", "Delete", err)
	}
	return s.roleRepo.Delete(ctx, role)
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
