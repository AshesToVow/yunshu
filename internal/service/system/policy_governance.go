package system

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/menu"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/plugingate"
	"yunshu/internal/service/k8s"
	bizerrors "yunshu/internal/pkg/errors"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

// PolicyGovernanceService 策略模拟、冲突分析与统一权限树。
type PolicyGovernanceService struct {
	userRepo       interfaces.UserRepository
	roleRepo       interfaces.RoleRepository
	permissionRepo interfaces.PermissionRepository
	menuRepo       interfaces.MenuRepository
	memberRepo     interfaces.ProjectMemberRepository
	k8sAccessRepo  interfaces.K8sClusterAccessRepository
	enforcer       *casbin.SyncedEnforcer
	plugins        *config.PluginsConfig
}

func NewPolicyGovernanceService(
	userRepo interfaces.UserRepository,
	roleRepo interfaces.RoleRepository,
	permissionRepo interfaces.PermissionRepository,
	menuRepo interfaces.MenuRepository,
	memberRepo interfaces.ProjectMemberRepository,
	k8sAccessRepo interfaces.K8sClusterAccessRepository,
	enforcer *casbin.SyncedEnforcer,
	plugins *config.PluginsConfig,
) *PolicyGovernanceService {
	return &PolicyGovernanceService{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		menuRepo:       menuRepo,
		memberRepo:     memberRepo,
		k8sAccessRepo:  k8sAccessRepo,
		enforcer:       enforcer,
		plugins:        plugins,
	}
}

type PolicySimulateRequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Method    string `json:"method" binding:"required"`
	ClusterID uint   `json:"cluster_id"`
	Namespace string `json:"namespace"`
	ProjectID uint   `json:"project_id"`
}

type PolicySimulateLayer struct {
	Layer        string   `json:"layer"`
	Allowed      bool     `json:"allowed,omitempty"`
	Skipped      bool     `json:"skipped,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	MatchedRoles []string `json:"matched_roles,omitempty"`
	Required     string   `json:"required,omitempty"`
	Plugin       string   `json:"plugin,omitempty"`
}

type PolicyMenuImpact struct {
	Path    string `json:"path"`
	Name    string `json:"name,omitempty"`
	Visible bool   `json:"visible"`
	Reason  string `json:"reason,omitempty"`
}

type PolicySimulateResponse struct {
	Allowed    bool                 `json:"allowed"`
	Layers     []PolicySimulateLayer `json:"layers"`
	MenuImpact []PolicyMenuImpact   `json:"menu_impact,omitempty"`
}

type PolicyConflictItem struct {
	Type         string `json:"type"`
	Severity     string `json:"severity"`
	Message      string `json:"message"`
	MenuPath     string `json:"menu_path,omitempty"`
	MenuName     string `json:"menu_name,omitempty"`
	Resource     string `json:"resource,omitempty"`
	Action       string `json:"action,omitempty"`
	Permission   string `json:"permission_name,omitempty"`
	PermissionID uint   `json:"permission_id,omitempty"`
	Plugin       string `json:"plugin,omitempty"`
	SuggestFix   string `json:"suggest_fix,omitempty"`
}

type PolicyConflictsResponse struct {
	RoleID   uint                 `json:"role_id"`
	RoleCode string               `json:"role_code"`
	Items    []PolicyConflictItem `json:"items"`
}

type PermissionTreeNode struct {
	Key          string               `json:"key"`
	Title        string               `json:"title"`
	NodeType     string               `json:"node_type"`
	PermissionID uint                 `json:"permission_id,omitempty"`
	Resource     string               `json:"resource,omitempty"`
	Action       string               `json:"action,omitempty"`
	MenuPath     string               `json:"menu_path,omitempty"`
	Granted      bool                 `json:"granted,omitempty"`
	Plugin       string               `json:"plugin,omitempty"`
	PluginOff    bool                 `json:"plugin_disabled,omitempty"`
	Children     []PermissionTreeNode `json:"children,omitempty"`
}

type PermissionTreeResponse struct {
	RoleID   uint                 `json:"role_id"`
	RoleCode string               `json:"role_code"`
	Tree     []PermissionTreeNode `json:"tree"`
}

type PermissionMenuLinksResponse struct {
	Links map[string][]MenuLink `json:"links"`
}

// MenuLink 权限关联菜单摘要。
type MenuLink struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

func (s *PolicyGovernanceService) MenuLinks(_ context.Context) PermissionMenuLinksResponse {
	rev := menu.PermissionToMenuPaths()
	links := make(map[string][]MenuLink, len(rev))
	for key, paths := range rev {
		items := make([]MenuLink, 0, len(paths))
		for _, p := range paths {
			items = append(items, MenuLink{Path: p})
		}
		links[key] = items
	}
	return PermissionMenuLinksResponse{Links: links}
}

func (s *PolicyGovernanceService) Simulate(ctx context.Context, req PolicySimulateRequest) (*PolicySimulateResponse, error) {
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Simulate", err)
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	subject := UserSubject(req.UserID)
	resp := &PolicySimulateResponse{Layers: make([]PolicySimulateLayer, 0, 5)}

	roleCodes := roleCodesOf(user.Roles)
	isSuper := auth.IsSuperAdminRole(roleCodes)
	resp.Layers = append(resp.Layers, PolicySimulateLayer{
		Layer:   "super_admin",
		Allowed: isSuper,
	})
	if isSuper {
		resp.Allowed = true
		resp.MenuImpact = s.simulateMenuImpact(subject, true)
		return resp, nil
	}

	allowed, _ := s.enforcer.Enforce(subject, path, method)
	matched := matchedRoleCodes(s.enforcer, subject, path, method)
	resp.Layers = append(resp.Layers, PolicySimulateLayer{
		Layer:        "casbin_api",
		Allowed:      allowed,
		MatchedRoles: matched,
		Required:     method + " " + path,
	})
	if allowed {
		resp.Allowed = true
	} else if s.k8sAccessRepo != nil && method == "GET" && k8s.IsK8sReadAPIPath(path) {
		pack := k8sauth.PrincipalPack{
			RoleCodes:  roleCodes,
			UserID:     user.ID,
			GroupCodes: groupCodesOf(user.Groups),
		}
		k8sAllowed := false
		if path == "/api/v1/clusters" {
			k8sAllowed = s.k8sAccessRepo.HasAnyK8sGrant(ctx, pack)
		} else if req.ClusterID > 0 {
			k8sAllowed = s.k8sAccessRepo.EffectiveTier(ctx, pack, req.ClusterID) > 0
		}
		resp.Layers = append(resp.Layers, PolicySimulateLayer{
			Layer:   "k8s_cluster_grant_fallback",
			Allowed: k8sAllowed,
			Reason:  "K8s 只读 API 集群档位兜底",
		})
		if k8sAllowed {
			resp.Allowed = true
		}
	} else {
		resp.Layers = append(resp.Layers, PolicySimulateLayer{
			Layer:   "k8s_cluster_grant_fallback",
			Skipped: true,
		})
	}

	pluginName := plugingate.ResolveAPIResourcePlugin(path)
	pluginAllowed := plugingate.IsAPIResourceAllowed(path, s.plugins)
	resp.Layers = append(resp.Layers, PolicySimulateLayer{
		Layer:   "plugin",
		Allowed: pluginAllowed,
		Plugin:  pluginName,
		Reason:  pluginReason(pluginName, pluginAllowed),
	})
	if resp.Allowed && !pluginAllowed {
		resp.Allowed = false
	}

	if req.ProjectID > 0 && strings.Contains(path, "/projects/") {
		memberAllowed, memberReason := s.simulateProjectMember(ctx, req.UserID, req.ProjectID, method, path, isSuper)
		resp.Layers = append(resp.Layers, PolicySimulateLayer{
			Layer:   "project_member",
			Allowed: memberAllowed,
			Reason:  memberReason,
		})
		if resp.Allowed && !memberAllowed {
			resp.Allowed = false
		}
	} else {
		resp.Layers = append(resp.Layers, PolicySimulateLayer{
			Layer:   "project_member",
			Skipped: true,
		})
	}

	resp.MenuImpact = s.simulateMenuImpact(subject, resp.Allowed)
	return resp, nil
}

func (s *PolicyGovernanceService) Conflicts(ctx context.Context, roleID uint) (*PolicyConflictsResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Conflicts", err)
	}
	menus, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Conflicts", err)
	}
	bindings, err := s.menuRepo.ListPermissionBindings(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Conflicts", err)
	}
	store := menu.NewBindingStore(bindings, menus)
	permissions, err := s.permissionRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Conflicts", err)
	}
	permByKey := make(map[string]model.Permission, len(permissions))
	for _, p := range permissions {
		permByKey[p.Resource+"::"+p.Action] = p
	}

	resp := &PolicyConflictsResponse{RoleID: role.ID, RoleCode: role.Code, Items: make([]PolicyConflictItem, 0)}
	for _, m := range menus {
		if m.Status != 1 || m.Hidden {
			continue
		}
		path := strings.TrimSpace(m.Path)
		if path == "" {
			continue
		}
		entryPerms := store.Resolve(m)
		if len(entryPerms) == 0 {
			continue
		}
		pluginOK := true
		pluginName := plugingate.ResolveMenuPathPlugin(path)
		for _, ep := range entryPerms {
			if !plugingate.IsAPIResourceAllowed(ep.Resource, s.plugins) {
				pluginOK = false
				break
			}
		}
		canAccessEntry := menu.RoleCanAccessMenu(s.enforcer, role.Code, entryPerms)

		if pluginOK && !canAccessEntry {
			ep := entryPerms[0]
			item := PolicyConflictItem{
				Type:       "menu_needs_entry_api",
				Severity:   "error",
				Message:    "菜单启用但角色缺少入口 API 授权",
				MenuPath:   path,
				MenuName:   m.Name,
				Resource:   ep.Resource,
				Action:     ep.Action,
				SuggestFix: "在授权管理中勾选对应 GET 权限",
			}
			if p, ok := permByKey[ep.Key()]; ok {
				item.Permission = p.Name
				item.PermissionID = p.ID
			}
			resp.Items = append(resp.Items, item)
		}
		if canAccessEntry && !pluginOK {
			resp.Items = append(resp.Items, PolicyConflictItem{
				Type:       "api_granted_plugin_disabled",
				Severity:   "warning",
				Message:    "已授权入口 API 但插件未启用，侧栏仍不可见",
				MenuPath:   path,
				MenuName:   m.Name,
				Plugin:     pluginName,
				SuggestFix: "启用插件或撤销相关 API 授权",
			})
		}
		if canAccessEntry && m.AdminOnly && role.Code != "super-admin" {
			resp.Items = append(resp.Items, PolicyConflictItem{
				Type:       "api_granted_admin_only_menu",
				Severity:   "info",
				Message:    "已授权 API 但菜单标记为仅管理员可见",
				MenuPath:   path,
				MenuName:   m.Name,
				SuggestFix: "取消 admin_only 或使用 super-admin 角色",
			})
		}
	}

	for _, p := range permissions {
		if !hasRolePolicy(s.enforcer, role.Code, p.Resource, p.Action) {
			continue
		}
		if !plugingate.IsAPIResourceAllowed(p.Resource, s.plugins) {
			resp.Items = append(resp.Items, PolicyConflictItem{
				Type:       "plugin_disabled_policy_active",
				Severity:   "warning",
				Message:    "插件未启用但 Casbin 策略仍存在",
				Resource:   p.Resource,
				Action:     p.Action,
				Permission: p.Name,
				Plugin:     plugingate.ResolveAPIResourcePlugin(p.Resource),
				SuggestFix: "撤销该权限或启用插件",
			})
		}
	}
	return resp, nil
}

func (s *PolicyGovernanceService) PermissionTree(ctx context.Context, roleID uint) (*PermissionTreeResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "Conflicts", err)
	}
	menus, err := s.menuRepo.Tree(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "PermissionTree", err)
	}
	bindings, err := s.menuRepo.ListPermissionBindings(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "PermissionTree", err)
	}
	flatMenus, _ := s.menuRepo.ListAll(ctx)
	store := menu.NewBindingStore(bindings, flatMenus)
	permissions, err := s.permissionRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "policy", "PermissionTree", err)
	}
	permByKey := make(map[string]model.Permission, len(permissions))
	for _, p := range permissions {
		permByKey[p.Resource+"::"+strings.ToUpper(p.Action)] = p
	}

	var buildMenu func([]model.Menu) []PermissionTreeNode
	buildMenu = func(items []model.Menu) []PermissionTreeNode {
		out := make([]PermissionTreeNode, 0, len(items))
		for _, m := range items {
			if m.Status != 1 {
				continue
			}
			path := strings.TrimSpace(m.Path)
			node := PermissionTreeNode{
				Key:      "menu:" + strconv.FormatUint(uint64(m.ID), 10),
				Title:    m.Name,
				NodeType: "menu",
				MenuPath: path,
				Granted:  menu.RoleCanAccessMenu(s.enforcer, role.Code, store.Resolve(m)),
			}
			if path != "" {
				node.Plugin = plugingate.ResolveMenuPathPlugin(path)
				node.PluginOff = !plugingate.IsMenuPathAllowed(path, s.plugins)
			}
			children := buildMenu(m.Children)
			entryPerms := store.Resolve(m)
			for _, ep := range entryPerms {
				key := ep.Key()
				p, ok := permByKey[key]
				if !ok {
					continue
				}
				if !plugingate.IsAPIResourceAllowed(p.Resource, s.plugins) {
					continue
				}
				children = append(children, PermissionTreeNode{
					Key:          "perm:" + strconv.FormatUint(uint64(p.ID), 10),
					Title:        p.Name,
					NodeType:     "api",
					PermissionID: p.ID,
					Resource:     p.Resource,
					Action:       p.Action,
					Granted:      hasRolePolicy(s.enforcer, role.Code, p.Resource, p.Action),
					Plugin:       plugingate.ResolveAPIResourcePlugin(p.Resource),
				})
			}
			if len(children) > 0 {
				node.Children = children
			}
			if path != "" || len(node.Children) > 0 {
				out = append(out, node)
			}
		}
		return out
	}

	return &PermissionTreeResponse{
		RoleID:   role.ID,
		RoleCode: role.Code,
		Tree:     buildMenu(menus),
	}, nil
}

func (s *PolicyGovernanceService) simulateMenuImpact(subject string, apiAllowed bool) []PolicyMenuImpact {
	if apiAllowed {
		return nil
	}
	rev := menu.PermissionToMenuPaths()
	out := make([]PolicyMenuImpact, 0)
	for key, paths := range rev {
		parts := strings.SplitN(key, "::", 2)
		if len(parts) != 2 {
			continue
		}
		ok, _ := s.enforcer.Enforce(subject, parts[0], parts[1])
		if ok {
			continue
		}
		for _, p := range paths {
			out = append(out, PolicyMenuImpact{Path: p, Visible: false, Reason: "missing casbin"})
		}
	}
	return out
}

func (s *PolicyGovernanceService) simulateProjectMember(ctx context.Context, userID, projectID uint, method, path string, isSuper bool) (bool, string) {
	if isSuper || s.memberRepo == nil || projectID == 0 {
		return true, ""
	}
	if !strings.Contains(path, "/projects/") {
		return true, ""
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "非项目成员"
		}
		return false, "项目成员查询失败"
	}
	upper := strings.ToUpper(method)
	if upper == "GET" || upper == "HEAD" {
		return true, ""
	}
	if m.Role == "readonly" {
		return false, "项目只读成员禁止写操作"
	}
	return true, ""
}

func matchedRoleCodes(enforcer *casbin.SyncedEnforcer, subject, path, method string) []string {
	roles, _ := enforcer.GetRolesForUser(subject)
	out := make([]string, 0)
	for _, rc := range roles {
		ok, _ := enforcer.Enforce(rc, path, method)
		if ok {
			out = append(out, rc)
		}
	}
	return out
}

func hasRolePolicy(enforcer *casbin.SyncedEnforcer, roleCode, resource, action string) bool {
	ok, _ := enforcer.Enforce(roleCode, resource, strings.ToUpper(action))
	return ok
}

func roleCodesOf(roles []model.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Code)
	}
	return out
}

func groupCodesOf(groups []model.UserGroup) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if c := strings.TrimSpace(g.Code); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func pluginReason(pluginName string, allowed bool) string {
	if allowed {
		return ""
	}
	if pluginName == "" {
		return "插件未识别"
	}
	return "插件 " + pluginName + " 未启用"
}
