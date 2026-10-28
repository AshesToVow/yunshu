package system

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/menu"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

// allowedMenuBindingActions 限定绑定动作，避免写入 Casbin 永远不会命中的非法方法。
var allowedMenuBindingActions = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {},
}

// menuPermissionExists 由调用方注入权限点存在性判定，便于单测替换。
type menuPermissionExists func(ctx context.Context, resource, action string) (bool, error)

func normalizeMenuBindingMode(mode string) (string, bool) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "":
		return "any", true
	case "any", "all":
		return m, true
	default:
		return "", false
	}
}

// buildMenuBindings 校验并归一化绑定项：动作白名单、mode、重复项、权限点真实存在。
// 权限点不存在时拒绝保存，否则会出现「菜单可见、接口 403」的假权限。
func buildMenuBindings(ctx context.Context, items []MenuPermissionBindingItem, exists menuPermissionExists) ([]model.MenuPermissionBinding, error) {
	out := make([]model.MenuPermissionBinding, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, b := range items {
		resource := strings.TrimSpace(b.Resource)
		if resource == "" {
			return nil, constants.ErrBadRequestWithMsg("权限资源不能为空")
		}
		if constants.HasPermissionResourceWildcard(resource) {
			return nil, constants.ErrBadRequestWithMsg("权限资源不能包含通配符 *")
		}
		if !strings.HasPrefix(resource, "/") {
			return nil, constants.ErrBadRequestWithMsg("权限资源必须以 / 开头：" + resource)
		}
		action := strings.ToUpper(strings.TrimSpace(b.Action))
		if _, ok := allowedMenuBindingActions[action]; !ok {
			return nil, constants.ErrBadRequestWithMsg("不支持的权限动作：" + strings.TrimSpace(b.Action))
		}
		mode, ok := normalizeMenuBindingMode(b.Mode)
		if !ok {
			return nil, constants.ErrBadRequestWithMsg("权限模式仅支持 any 或 all")
		}
		key := resource + "\x00" + action
		if _, dup := seen[key]; dup {
			return nil, constants.ErrBadRequestWithMsg("重复的权限绑定：" + action + " " + resource)
		}
		if exists != nil {
			found, err := exists(ctx, resource, action)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "menu", "ReplacePermissionBindings", err)
			}
			if !found {
				return nil, constants.ErrBadRequestWithMsg("权限点不存在：" + action + " " + resource + "（请先在权限管理中登记或同步路由）")
			}
		}
		seen[key] = struct{}{}
		out = append(out, model.MenuPermissionBinding{Resource: resource, Action: action, Mode: mode})
	}
	return out, nil
}

func (s *MenuService) permissionExists(ctx context.Context, resource, action string) (bool, error) {
	if s.permissionRepo == nil {
		return true, nil
	}
	if _, err := s.permissionRepo.GetByResourceAction(ctx, resource, action); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, bizerrors.Pass(ctx, "menu", "permissionExists", err)
	}
	return true, nil
}

type MenuPermissionBindingItem struct {
	ID       uint   `json:"id,omitempty"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
	Mode     string `json:"mode"`
}

type MenuPermissionBindingsResponse struct {
	MenuID   uint                        `json:"menu_id"`
	MenuPath string                      `json:"menu_path"`
	Custom   []MenuPermissionBindingItem `json:"custom"`
	Default  []menu.EntryPermission      `json:"default"`
	Effective []menu.EntryPermission     `json:"effective"`
}

type MenuPermissionBindingsReplaceRequest struct {
	Bindings []MenuPermissionBindingItem `json:"bindings"`
}

func (s *MenuService) GetPermissionBindings(ctx context.Context, menuID uint) (*MenuPermissionBindingsResponse, error) {
	m, err := s.menuRepo.GetByID(ctx, menuID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "GetPermissionBindings", err)
	}
	custom, err := s.menuRepo.ListPermissionBindingsByMenuID(ctx, menuID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "GetPermissionBindings", err)
	}
	customItems := make([]MenuPermissionBindingItem, 0, len(custom))
	for _, b := range custom {
		customItems = append(customItems, MenuPermissionBindingItem{
			ID:       b.ID,
			Resource: b.Resource,
			Action:   b.Action,
			Mode:     b.Mode,
		})
	}
	path := strings.TrimSpace(m.Path)
	defaultPerms := menu.DefaultPathBindings()[normalizeMenuPathForBindings(path)]
	effective := menu.NewBindingStore(custom, []model.Menu{*m}).Resolve(*m)
	return &MenuPermissionBindingsResponse{
		MenuID:    m.ID,
		MenuPath:  path,
		Custom:    customItems,
		Default:   defaultPerms,
		Effective: effective,
	}, nil
}

func (s *MenuService) ReplacePermissionBindings(ctx context.Context, menuID uint, req MenuPermissionBindingsReplaceRequest) error {
	if _, err := s.menuRepo.GetByID(ctx, menuID); err != nil {
		return bizerrors.Pass(ctx, "menu", "ReplacePermissionBindings", err)
	}
	models, err := buildMenuBindings(ctx, req.Bindings, s.permissionExists)
	if err != nil {
		return err
	}
	if err := s.menuRepo.ReplacePermissionBindings(ctx, menuID, models); err != nil {
		return bizerrors.Pass(ctx, "menu", "ReplacePermissionBindings", err)
	}
	s.invalidateCache()
	return nil
}

func normalizeMenuPathForBindings(path string) string {
	p := strings.TrimSpace(strings.ToLower(path))
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	return p
}
