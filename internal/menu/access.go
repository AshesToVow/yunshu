package menu

import (
	"fmt"
	"strings"

	"yunshu/internal/model"

	"github.com/casbin/casbin/v2"
)

// BindingStore 菜单 path / menu_id 的自定义入口权限（DB 覆盖静态映射）。
type BindingStore struct {
	ByMenuID map[uint][]EntryPermission
	ByPath   map[string][]EntryPermission
}

func NewBindingStore(bindings []model.MenuPermissionBinding, menus []model.Menu) BindingStore {
	menuPathByID := make(map[uint]string, len(menus))
	for _, m := range menus {
		if p := normalizeMenuPath(m.Path); p != "" {
			menuPathByID[m.ID] = p
		}
	}
	byMenuID := make(map[uint][]EntryPermission)
	byPath := make(map[string][]EntryPermission)
	for _, b := range bindings {
		ep := EntryPermission{Resource: strings.TrimSpace(b.Resource), Action: strings.TrimSpace(b.Action)}
		if ep.Resource == "" || ep.Action == "" {
			continue
		}
		byMenuID[b.MenuID] = append(byMenuID[b.MenuID], ep)
		if path, ok := menuPathByID[b.MenuID]; ok {
			byPath[path] = append(byPath[path], ep)
		}
	}
	return BindingStore{ByMenuID: byMenuID, ByPath: byPath}
}

func (s BindingStore) Resolve(menu model.Menu) []EntryPermission {
	if custom, ok := s.ByMenuID[menu.ID]; ok && len(custom) > 0 {
		return custom
	}
	path := normalizeMenuPath(menu.Path)
	if path == "" {
		return nil
	}
	if custom, ok := s.ByPath[path]; ok && len(custom) > 0 {
		return custom
	}
	return DefaultPathBindings()[path]
}

func normalizeMenuPath(path string) string {
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

// UserCanAccessMenu 判断用户是否具备菜单入口权限（bindings 为空则放行以兼容旧数据）。
func UserCanAccessMenu(enforcer *casbin.SyncedEnforcer, userID uint, bindings []EntryPermission) bool {
	if len(bindings) == 0 {
		return true
	}
	subject := fmt.Sprintf("user:%d", userID)
	for _, b := range bindings {
		allowed, err := enforcer.Enforce(subject, b.Resource, strings.ToUpper(b.Action))
		if err != nil {
			continue
		}
		if allowed {
			return true
		}
	}
	return false
}

// FilterMenusByAccess 按 Casbin 入口权限递归过滤菜单树。
func FilterMenusByAccess(items []model.Menu, enforcer *casbin.SyncedEnforcer, userID uint, store BindingStore) []model.Menu {
	var walk func([]model.Menu) []model.Menu
	walk = func(nodes []model.Menu) []model.Menu {
		out := make([]model.Menu, 0, len(nodes))
		for _, it := range nodes {
			child := it
			if len(it.Children) > 0 {
				child.Children = walk(it.Children)
			}
			path := normalizeMenuPath(it.Path)
			if path != "" {
				if !UserCanAccessMenu(enforcer, userID, store.Resolve(it)) {
					continue
				}
			} else if len(child.Children) == 0 {
				continue
			}
			if path == "" && len(child.Children) == 0 {
				continue
			}
			out = append(out, child)
		}
		return out
	}
	return walk(items)
}

// RoleCanAccessMenu 判断角色是否具备菜单入口权限。
func RoleCanAccessMenu(enforcer *casbin.SyncedEnforcer, roleCode string, bindings []EntryPermission) bool {
	if len(bindings) == 0 {
		return true
	}
	for _, b := range bindings {
		allowed, err := enforcer.Enforce(roleCode, b.Resource, strings.ToUpper(b.Action))
		if err != nil {
			continue
		}
		if allowed {
			return true
		}
	}
	return false
}
