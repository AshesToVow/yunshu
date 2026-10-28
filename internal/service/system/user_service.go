package system

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/exportutil"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"
	"yunshu/internal/service/alert"
	"yunshu/internal/store"

	"github.com/casbin/casbin/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo          interfaces.UserRepository
	roleRepo          interfaces.RoleRepository
	departmentRepo    interfaces.DepartmentRepository
	projectMemberRepo interfaces.ProjectMemberRepository
	assigneeSvc       *alert.AlertRuleAssigneeService
	enforcer          *casbin.SyncedEnforcer
	resolvePolicy     PasswordPolicyResolver
	redis             *redis.Client
}

// NewUserService 创建相关逻辑。
func NewUserService(
	userRepo interfaces.UserRepository,
	roleRepo interfaces.RoleRepository,
	departmentRepo interfaces.DepartmentRepository,
	enforcer *casbin.SyncedEnforcer,
	projectMemberRepo interfaces.ProjectMemberRepository,
	assigneeSvc *alert.AlertRuleAssigneeService,
	resolvePolicy PasswordPolicyResolver,
	redisClient *redis.Client,
) *UserService {
	return &UserService{
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		departmentRepo:    departmentRepo,
		projectMemberRepo: projectMemberRepo,
		assigneeSvc:       assigneeSvc,
		enforcer:          enforcer,
		resolvePolicy:     resolvePolicy,
		redis:             redisClient,
	}
}

func (s *UserService) accessibleDepartmentIDs(ctx context.Context, actor *auth.CurrentUser) ([]uint, error) {
	if actor == nil || actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, nil
	}
	return s.departmentRepo.ListDescendantIDsAndSelf(ctx, *actor.DepartmentID)
}

func (s *UserService) canAccessUser(ctx context.Context, actor *auth.CurrentUser, target *model.User) (bool, error) {
	if actor == nil || target == nil {
		return false, nil
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) || actor.ID == target.ID {
		return true, nil
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return false, nil
	}
	if target.DepartmentID == nil || *target.DepartmentID == 0 {
		return false, nil
	}
	ids, err := s.accessibleDepartmentIDs(ctx, actor)
	if err != nil {
		return false, bizerrors.Pass(ctx, "user", "canAccessUser", err)
	}
	return slices.Contains(ids, *target.DepartmentID), nil
}

// ensureTargetManageable 部门管理员即便能看到超级管理员（如同部门），也不得改动其账号。
func (s *UserService) ensureTargetManageable(actor *auth.CurrentUser, target *model.User) error {
	if actor == nil || target == nil {
		return constants.ErrForbidden
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if auth.IsSuperAdminRole(model.ExtractRoleCodes(target.Roles)) {
		return constants.ErrForbiddenWithMsg("仅内置超级管理员可以操作内置超级管理员账号")
	}
	return nil
}

// ensureAssignableRoles 非超管只能分配自己已持有的角色，防止通过用户管理提权。
func ensureAssignableRoles(actor *auth.CurrentUser, roles []model.Role) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	allowed := make(map[string]struct{}, len(actor.RoleCodes))
	for _, code := range actor.RoleCodes {
		allowed[strings.TrimSpace(code)] = struct{}{}
	}
	for _, role := range roles {
		if _, ok := allowed[strings.TrimSpace(role.Code)]; !ok {
			return constants.ErrForbiddenWithMsg("不能分配超出自身权限范围的角色")
		}
	}
	return nil
}

func (s *UserService) resolveAssignableRoles(ctx context.Context, actor *auth.CurrentUser, roleIDs []uint) ([]model.Role, error) {
	roles, err := s.roleRepo.GetByIDs(ctx, roleIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "resolveAssignableRoles", err)
	}
	if len(roleIDs) > 0 && len(roles) != len(roleIDs) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgbc90b8ad5f29)
	}
	if err := ensureAssignableRoles(actor, roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// ensureSuperAdminRemains 禁止禁用/删除最后一名启用状态的内置超级管理员，避免系统失去管理入口。
func (s *UserService) ensureSuperAdminRemains(ctx context.Context, target *model.User) error {
	if target == nil || target.Status != model.StatusEnabled {
		return nil
	}
	if !auth.IsSuperAdminRole(model.ExtractEnabledRoleCodes(target.Roles)) {
		return nil
	}
	if s.userRepo == nil {
		return constants.ErrInternal
	}
	n, err := s.userRepo.CountEnabledUsersByRoleCode(ctx, auth.SuperAdminRoleCode)
	if err != nil {
		return bizerrors.Pass(ctx, "user", "ensureSuperAdminRemains", err)
	}
	if n <= 1 {
		return constants.ErrForbiddenWithMsg("系统至少需要保留一名启用状态的内置超级管理员")
	}
	return nil
}

// invalidateUserSessions 踢出目标用户的全部 access/refresh 会话（管理员重置密码后必须重新登录）。
func (s *UserService) invalidateUserSessions(ctx context.Context, userID uint) {
	if s.redis == nil || userID == 0 {
		return
	}
	_ = store.InvalidateAllUserAccessTokens(ctx, s.redis, userID)
	_ = store.InvalidateAllUserRefreshTokens(ctx, s.redis, userID)
}

// Create 创建相关的业务逻辑。
func (s *UserService) Create(ctx context.Context, req UserCreateRequest) (*UserDetailResponse, error) {
	if err := s.ensureUserUnique(ctx, 0, req.Username, req.Email); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Create", err)
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Create", err)
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgbc90b8ad5f29)
	}
	if err := enforcePasswordComplexity(ctx, s.resolvePolicy, req.Password, strings.TrimSpace(req.Username)); err != nil {
		return nil, err
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Create", err)
	}

	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	email := normalizeEmail(req.Email)
	var departmentID *uint
	if req.DepartmentID != nil && *req.DepartmentID > 0 {
		if _, err = s.departmentRepo.GetByID(ctx, *req.DepartmentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg94d63d947b0e)
			}
			return nil, bizerrors.Pass(ctx, "user", "Create", err)
		}
		departmentID = req.DepartmentID
	}
	now := time.Now()
	user := model.User{
		Username:          strings.TrimSpace(req.Username),
		Email:             &email,
		Phone:             strings.TrimSpace(req.Phone),
		Password:          hashedPassword,
		PasswordChangedAt: &now,
		Nickname:          strings.TrimSpace(req.Nickname),
		Status:            status,
		DepartmentID:      departmentID,
		Roles:             roles,
	}
	if err = s.userRepo.Create(ctx, &user); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Create", err)
	}
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Create", err)
	}

	response := NewUserDetailResponse(user)
	return &response, nil
}

func (s *UserService) CreateByActor(ctx context.Context, actor *auth.CurrentUser, req UserCreateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.Create(ctx, req)
	}
	if _, err := s.resolveAssignableRoles(ctx, actor, req.RoleIDs); err != nil {
		return nil, err
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgc8caf91c1d57)
	}
	allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "CreateByActor", err)
	}
	if req.DepartmentID == nil {
		deptID := *actor.DepartmentID
		req.DepartmentID = &deptID
	} else if *req.DepartmentID > 0 && !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
		return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgd672e80435d4)
	}
	return s.Create(ctx, req)
}

// Update 更新相关的业务逻辑。
func (s *UserService) Update(ctx context.Context, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "Update", err)
	}

	// 只更新请求中出现的列，避免全行 Save 用陈旧快照覆盖并发修改（如登录时间、其他管理员改的字段）。
	fields := make(map[string]any, 8)
	if req.Nickname != nil {
		user.Nickname = strings.TrimSpace(*req.Nickname)
		fields["nickname"] = user.Nickname
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		email := normalizeEmail(*req.Email)
		if err = s.ensureUserUnique(ctx, user.ID, user.Username, email); err != nil {
			return nil, bizerrors.Pass(ctx, "user", "Update", err)
		}
		user.Email = &email
		fields["email"] = email
	}
	if req.Status != nil {
		if *req.Status == model.StatusDisabled {
			if err = s.ensureSuperAdminRemains(ctx, user); err != nil {
				return nil, err
			}
		}
		user.Status = *req.Status
		fields["status"] = user.Status
	}
	if req.DepartmentID != nil {
		if *req.DepartmentID == 0 {
			user.DepartmentID = nil
			fields["department_id"] = nil
		} else {
			if _, err = s.departmentRepo.GetByID(ctx, *req.DepartmentID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg94d63d947b0e)
				}
				return nil, bizerrors.Pass(ctx, "user", "Update", err)
			}
			user.DepartmentID = req.DepartmentID
			fields["department_id"] = *req.DepartmentID
		}
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
		fields["phone"] = user.Phone
	}
	passwordReset := false
	if req.Password != nil && *req.Password != "" {
		if err := enforcePasswordComplexity(ctx, s.resolvePolicy, *req.Password, user.Username); err != nil {
			return nil, err
		}
		passwordReset = true
		user.Password, err = password.Hash(*req.Password)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "user", "Update", err)
		}
		touchPasswordChanged(user)
		// 管理员重置密码后强制对方下次登录改密。
		user.MustChangePassword = true
		fields["password"] = user.Password
		fields["password_changed_at"] = user.PasswordChangedAt
		fields["must_change_password"] = true
	}

	if err = s.userRepo.UpdateFields(ctx, user.ID, fields); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "Update", err)
	}
	if passwordReset {
		s.invalidateUserSessions(ctx, user.ID)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) UpdateByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "UpdateByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "UpdateByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	if err := s.ensureTargetManageable(actor, target); err != nil {
		return nil, err
	}
	// 不能通过用户管理禁用自己的账号：避免管理员自锁，账号停用统一由其他管理员操作。
	if actor.ID == target.ID && req.Status != nil && *req.Status == model.StatusDisabled {
		return nil, constants.ErrForbiddenWithMsg("不能禁用自己的账号")
	}
	if target.ID != actor.ID && req.Email != nil && !auth.IsSuperAdminRole(actor.RoleCodes) {
		desired := normalizeEmail(*req.Email)
		if desired != "" && desired != userEmailValue(target) {
			return nil, constants.ErrForbiddenWithMsg("仅内置超级管理员可以修改其他账号的邮箱")
		}
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		if actor.ID == target.ID {
			return nil, constants.ErrForbiddenWithMsg("不能通过用户管理修改自己的登录密码")
		}
		if !auth.CanManageOtherUsersLoginPassword(actor.RoleCodes) {
			return nil, constants.ErrForbiddenWithMsg("仅管理员角色（内置超级管理员）可修改其他账号的登录密码")
		}
	}
	if !auth.IsSuperAdminRole(actor.RoleCodes) && req.DepartmentID != nil && *req.DepartmentID > 0 {
		allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "user", "UpdateByActor", err)
		}
		if !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
			return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgc1305dfff708)
		}
	}
	return s.Update(ctx, id, req)
}

// Delete 删除相关的业务逻辑。
func (s *UserService) Delete(ctx context.Context, id uint) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return bizerrors.Pass(ctx, "user", "Delete", err)
	}
	if err = s.ensureSuperAdminRemains(ctx, user); err != nil {
		return err
	}

	if err = s.userRepo.Delete(ctx, user); err != nil {
		return bizerrors.Pass(ctx, "user", "Delete", err)
	}
	if s.departmentRepo != nil {
		_ = s.departmentRepo.ClearLeaderByUserID(ctx, id)
	}
	if s.projectMemberRepo != nil {
		_ = s.projectMemberRepo.DeleteByUserID(ctx, id)
	}
	if s.assigneeSvc != nil {
		_ = s.assigneeSvc.PruneUserFromAllAssignees(ctx, id)
	}
	_, err = s.enforcer.DeleteUser(UserSubject(id))
	if err != nil {
		return bizerrors.Pass(ctx, "user", "Delete", err)
	}
	return nil
}

func (s *UserService) DeleteByActor(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return bizerrors.Pass(ctx, "user", "DeleteByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return bizerrors.Pass(ctx, "user", "DeleteByActor", err)
	}
	if !ok {
		return constants.ErrForbidden
	}
	if err := s.ensureTargetManageable(actor, target); err != nil {
		return err
	}
	// 不能删除自己的账号：避免管理员误删自锁，删除统一由其他管理员操作。
	if actor.ID == target.ID {
		return constants.ErrForbiddenWithMsg("不能删除自己的账号")
	}
	return s.Delete(ctx, id)
}

// Detail 查询详情相关的业务逻辑。
func (s *UserService) Detail(ctx context.Context, id uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "Detail", err)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) DetailByActor(ctx context.Context, actor *auth.CurrentUser, id uint) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "DetailByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "DetailByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	resp := NewUserDetailResponse(*target)
	return &resp, nil
}

// List 查询列表相关的业务逻辑。
func (s *UserService) List(ctx context.Context, query UserListQuery) (*pagination.Result[UserDetailResponse], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	users, total, err := s.userRepo.List(ctx, repository.UserListParams{
		Keyword:      query.Keyword,
		DepartmentID: query.DepartmentID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "List", err)
	}

	list := make([]UserDetailResponse, 0, len(users))
	for _, user := range users {
		list = append(list, NewUserDetailResponse(user))
	}

	return &pagination.Result[UserDetailResponse]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *UserService) ListByActor(ctx context.Context, actor *auth.CurrentUser, query UserListQuery) (*pagination.Result[UserDetailResponse], error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.List(ctx, query)
	}
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params := repository.UserListParams{
		Keyword:      query.Keyword,
		DepartmentID: query.DepartmentID,
		Page:         page,
		PageSize:     pageSize,
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		uid := actor.ID
		params.OnlyUserID = &uid
	} else {
		ids, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "user", "ListByActor", err)
		}
		params.DepartmentIDs = ids
	}
	users, total, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "ListByActor", err)
	}
	list := make([]UserDetailResponse, 0, len(users))
	for _, user := range users {
		list = append(list, NewUserDetailResponse(user))
	}
	return &pagination.Result[UserDetailResponse]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// AssignRoles 执行对应的业务逻辑。
func (s *UserService) AssignRoles(ctx context.Context, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "AssignRoles", err)
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "AssignRoles", err)
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgbc90b8ad5f29)
	}

	hadSuper := auth.IsSuperAdminRole(model.ExtractEnabledRoleCodes(user.Roles))
	keepsSuper := auth.IsSuperAdminRole(model.ExtractEnabledRoleCodes(roles))
	if hadSuper && !keepsSuper {
		if err = s.ensureSuperAdminRemains(ctx, user); err != nil {
			return nil, err
		}
	}

	if err = s.userRepo.ReplaceRoles(ctx, user, roles); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "AssignRoles", err)
	}

	user.Roles = roles
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, bizerrors.Pass(ctx, "user", "AssignRoles", err)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) AssignRolesByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "user", "AssignRolesByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "AssignRolesByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	if err := s.ensureTargetManageable(actor, target); err != nil {
		return nil, err
	}
	if _, err := s.resolveAssignableRoles(ctx, actor, req.RoleIDs); err != nil {
		return nil, err
	}
	return s.AssignRoles(ctx, id, req)
}

func userEmailValue(user *model.User) string {
	if user == nil || user.Email == nil {
		return ""
	}
	return normalizeEmail(*user.Email)
}

func (s *UserService) ensureUserUnique(ctx context.Context, currentID uint, username, email string) error {
	if strings.TrimSpace(username) != "" {
		existing, err := s.userRepo.GetByUsername(ctx, strings.TrimSpace(username))
		if err == nil && existing.ID != currentID {
			return constants.ErrUsernameTaken
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerrors.Pass(ctx, "user", "ensureUserUnique", err)
		}
	}

	if strings.TrimSpace(email) != "" {
		existing, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
		if err == nil && existing.ID != currentID {
			return constants.ErrEmailAlreadyRegistered
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerrors.Pass(ctx, "user", "ensureUserUnique", err)
		}
	}

	return nil
}

// ListAll returns all users for export.
func (s *UserService) ListAll(ctx context.Context) ([]model.User, error) {
	return s.userRepo.ListAll(ctx)
}

func (s *UserService) ListAllByActor(ctx context.Context, actor *auth.CurrentUser) ([]model.User, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.ListAll(ctx)
	}
	params := repository.UserListParams{
		Page:     1,
		PageSize: 100000,
		Unbounded: true,
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		uid := actor.ID
		params.OnlyUserID = &uid
	} else {
		ids, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "user", "ListAllByActor", err)
		}
		params.DepartmentIDs = ids
	}
	users, _, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "ListAllByActor", err)
	}
	return users, nil
}

// UserImportRowError 单行导入失败原因。
type UserImportRowError struct {
	Row      int    `json:"row"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// UserImportCredential 新建用户的临时密码（仅导入成功时返回给操作者）。
type UserImportCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserImportResult 用户批量导入结果。
type UserImportResult struct {
	Created     int                    `json:"created"`
	Skipped     int                    `json:"skipped"`
	Failed      int                    `json:"failed"`
	Errors      []UserImportRowError   `json:"errors,omitempty"`
	Credentials []UserImportCredential `json:"credentials,omitempty"`
}

// ImportUsersByActor reads an Excel file and creates users within actor scope.
func (s *UserService) ImportUsersByActor(ctx context.Context, actor *auth.CurrentUser, r io.Reader) (*UserImportResult, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	limited := exportutil.LimitedImportReader(r)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", err)
	}
	if err := exportutil.CheckImportReadSize(int64(len(data))); err != nil {
		return nil, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", err)
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", err)
	}
	if len(rows) > exportutil.MaxExcelImportRows+1 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("导入行数超过 %d 行上限", exportutil.MaxExcelImportRows))
	}

	var allowedDeptIDs []uint
	isSuper := auth.IsSuperAdminRole(actor.RoleCodes)
	if !isSuper {
		if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
			return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgc8caf91c1d57)
		}
		allowedDeptIDs, err = s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", err)
		}
	}

	result := &UserImportResult{}
	// Expect header: ID,Username,Nickname,Email,Status,DepartmentCode
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rowNum := i + 1
		if len(row) < 2 {
			continue
		}
		username := strings.TrimSpace(row[1])
		if username == "" {
			continue
		}
		var nickname string
		var emailPtr *string
		status := int(model.StatusEnabled)
		if len(row) >= 3 {
			nickname = strings.TrimSpace(row[2])
		}
		if len(row) >= 4 {
			e := strings.TrimSpace(row[3])
			if e != "" {
				emailPtr = &e
			}
		}
		if len(row) >= 5 {
			st := strings.TrimSpace(row[4])
			if st == "0" || strings.EqualFold(st, "disabled") {
				status = int(model.StatusDisabled)
			}
		}

		var departmentID *uint
		if len(row) >= 6 {
			deptCode := strings.TrimSpace(row[5])
			if deptCode != "" {
				dept, findErr := s.departmentRepo.GetByCode(ctx, deptCode)
				if findErr != nil {
					result.Failed++
					result.Errors = append(result.Errors, UserImportRowError{Row: rowNum, Username: username, Message: "部门编码不存在"})
					continue
				}
				if !isSuper && !slices.Contains(allowedDeptIDs, dept.ID) {
					result.Failed++
					result.Errors = append(result.Errors, UserImportRowError{Row: rowNum, Username: username, Message: "无权在该部门创建用户"})
					continue
				}
				departmentID = &dept.ID
			}
		}
		if !isSuper && departmentID == nil && actor.DepartmentID != nil {
			deptID := *actor.DepartmentID
			departmentID = &deptID
		}

		exists, existsErr := s.userRepo.ExistsByUsernameOrEmail(ctx, username, "")
		if existsErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, UserImportRowError{Row: rowNum, Username: username, Message: existsErr.Error()})
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		rawPass, passErr := randomImportPassword()
		if passErr != nil {
			return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", passErr)
		}
		hashed, hashErr := password.Hash(rawPass)
		if hashErr != nil {
			return nil, bizerrors.Pass(ctx, "user", "ImportUsersByActor", hashErr)
		}
		now := time.Now()
		user := model.User{
			Username:           username,
			Nickname:           nickname,
			Email:              emailPtr,
			Password:           hashed,
			PasswordChangedAt:  &now,
			MustChangePassword: true,
			Status:             status,
			DepartmentID:       departmentID,
		}
		if createErr := s.userRepo.Create(ctx, &user); createErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, UserImportRowError{Row: rowNum, Username: username, Message: createErr.Error()})
			continue
		}
		result.Created++
		result.Credentials = append(result.Credentials, UserImportCredential{Username: username, Password: rawPass})
	}
	return result, nil
}

func randomImportPassword() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// 导入账号须首次改密；随机口令避免固定弱口令进库。
	return "Tmp!" + hex.EncodeToString(b[:]), nil
}

// UsersImportTemplateExcel returns the user import template file.
func (s *UserService) UsersImportTemplateExcel() (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]any{"ID(可留空)", "Username(必填)", "Nickname", "Email", "Status(1启用/0停用)", "DepartmentCode(可选)"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"", "demo.user", "示例用户", "demo@example.com", 1, "RND-PLATFORM"})
	return f, nil
}
