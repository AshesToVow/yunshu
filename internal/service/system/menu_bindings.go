package system

import (
	"context"
	"strings"

	"yunshu/internal/menu"
	"yunshu/internal/model"
	bizerrors "yunshu/internal/pkg/errors"
)

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
	models := make([]model.MenuPermissionBinding, 0, len(req.Bindings))
	for _, b := range req.Bindings {
		mode := strings.TrimSpace(b.Mode)
		if mode == "" {
			mode = "any"
		}
		models = append(models, model.MenuPermissionBinding{
			Resource: strings.TrimSpace(b.Resource),
			Action:   strings.ToUpper(strings.TrimSpace(b.Action)),
			Mode:     mode,
		})
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
