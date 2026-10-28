package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

type MenuService struct {
	menuRepo          interfaces.MenuRepository
	permissionRepo    interfaces.PermissionRepository
	mu                sync.RWMutex
	treeCache         []model.Menu
	treeCacheExpireAt time.Time
}

// NewMenuService 创建相关逻辑。
func NewMenuService(menuRepo interfaces.MenuRepository, permissionRepo interfaces.PermissionRepository) *MenuService {
	return &MenuService{menuRepo: menuRepo, permissionRepo: permissionRepo}
}

// menuPathConflict 判断 trim 后的路径是否已被其它菜单占用；空路径不参与唯一性约束。
func menuPathConflict(all []model.Menu, path string, excludeID uint) bool {
	p := strings.TrimSpace(path)
	if p == "" {
		return false
	}
	for _, it := range all {
		if it.ID == excludeID {
			continue
		}
		if strings.TrimSpace(it.Path) == p {
			return true
		}
	}
	return false
}

func sameParent(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ensureUniqueSiblingSort keeps sort unique within same parent scope.
// If requested sort is occupied, it shifts to next available value.
func (s *MenuService) ensureUniqueSiblingSort(ctx context.Context, parentID *uint, sort int, excludeID uint) (int, error) {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return sort, bizerrors.Pass(ctx, "menu", "ensureUniqueSiblingSort", err)
	}
	used := make(map[int]struct{}, 64)
	for _, it := range all {
		if excludeID > 0 && it.ID == excludeID {
			continue
		}
		if !sameParent(parentID, it.ParentID) {
			continue
		}
		used[it.Sort] = struct{}{}
	}
	for {
		if _, ok := used[sort]; !ok {
			return sort, nil
		}
		sort++
	}
}

type MenuCreatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"required,max=64"`
	Icon      string `json:"icon"`
	AdminOnly bool   `json:"admin_only"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	Status    int    `json:"status" binding:"required,oneof=0 1"`
}

type MenuUpdatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"omitempty,max=64"`
	Icon      string `json:"icon"`
	AdminOnly *bool  `json:"admin_only,omitempty"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	// 使用指针避免 JSON 省略时误把 status 写成 0（原逻辑 `id == menu.ID` 恒为真会错误停用菜单）
	Status *int `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
}

type MenuBatchStatusPayload struct {
	IDs    []uint `json:"ids" binding:"required,min=1,dive,gt=0"`
	Status int    `json:"status" binding:"oneof=0 1"`
}

// Tree 获取树形数据相关的业务逻辑。
func (s *MenuService) Tree(ctx context.Context) ([]model.Menu, error) {
	s.mu.RLock()
	if time.Now().Before(s.treeCacheExpireAt) && s.treeCache != nil {
		cached := s.treeCache
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	list, err := s.menuRepo.Tree(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "Tree", err)
	}

	s.mu.Lock()
	s.treeCache = list
	s.treeCacheExpireAt = time.Now().Add(60 * time.Second)
	s.mu.Unlock()
	return list, nil
}

// assertMenuPathAvailable 保证同一路径只对应一条菜单，避免动态路由命中哪条看运气。
func (s *MenuService) assertMenuPathAvailable(ctx context.Context, path string, excludeID uint) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "menu", "assertMenuPathAvailable", err)
	}
	if menuPathConflict(all, path, excludeID) {
		return constants.ErrBadRequestWithMsg("菜单路径已被占用：" + strings.TrimSpace(path))
	}
	return nil
}

// Create 创建相关的业务逻辑。
func (s *MenuService) Create(ctx context.Context, payload MenuCreatePayload) (*model.Menu, error) {
	path := strings.TrimSpace(payload.Path)
	if err := s.assertMenuPathAvailable(ctx, path, 0); err != nil {
		return nil, err
	}
	parentID := payload.ParentID
	if parentID == nil {
		// For nested paths like /system/security, auto-create missing parent menus.
		autoParentID, err := s.ensureMenuParentChainByPath(ctx, path)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "menu", "Create", err)
		}
		parentID = autoParentID
	}
	sortVal, err := s.ensureUniqueSiblingSort(ctx, parentID, payload.Sort, 0)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "Create", err)
	}
	menu := model.Menu{
		ParentID:  parentID,
		Path:      path,
		Name:      payload.Name,
		Icon:      payload.Icon,
		AdminOnly: payload.AdminOnly,
		Sort:      sortVal,
		Hidden:    payload.Hidden,
		Component: payload.Component,
		Redirect:  payload.Redirect,
		Status:    payload.Status,
	}
	if err := s.menuRepo.Create(ctx, &menu); err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "Create", err)
	}
	s.invalidateCache()
	return &menu, nil
}

func (s *MenuService) ensureMenuParentChainByPath(ctx context.Context, fullPath string) (*uint, error) {
	path := strings.TrimSpace(fullPath)
	if path == "" || path == "/" {
		return nil, nil
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) <= 1 {
		return nil, nil
	}
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "ensureMenuParentChainByPath", err)
	}
	pathMap := make(map[string]*model.Menu, len(all))
	for i := range all {
		p := strings.TrimSpace(all[i].Path)
		if p == "" {
			continue
		}
		pathMap[p] = &all[i]
	}

	var parentID *uint
	var currentPath strings.Builder
	for i := range len(segs) - 1 {
		currentPath.WriteString("/" + segs[i])
		if found, ok := pathMap[currentPath.String()]; ok {
			id := found.ID
			parentID = &id
			continue
		}
		name := segs[i]
		sortVal, err := s.ensureUniqueSiblingSort(ctx, parentID, 0, 0)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "menu", "ensureMenuParentChainByPath", err)
		}
		m := model.Menu{
			ParentID:  parentID,
			Path:      currentPath.String(),
			Name:      name,
			Icon:      "",
			AdminOnly: false,
			Sort:      sortVal,
			Hidden:    false,
			Component: "",
			Redirect:  "",
			Status:    1,
		}
		if err := s.menuRepo.Create(ctx, &m); err != nil {
			return nil, bizerrors.Pass(ctx, "menu", "ensureMenuParentChainByPath", err)
		}
		id := m.ID
		parentID = &id
	}
	return parentID, nil
}

// Update 更新相关的业务逻辑。
func (s *MenuService) Update(ctx context.Context, id uint, payload MenuUpdatePayload) (*model.Menu, error) {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrMenuNotFound
		}
		return nil, bizerrors.Pass(ctx, "menu", "Update", err)
	}
	path := strings.TrimSpace(payload.Path)
	if err := s.assertMenuPathAvailable(ctx, path, menu.ID); err != nil {
		return nil, err
	}
	if payload.Name != "" {
		menu.Name = payload.Name
	}
	menu.Path = path
	menu.Icon = payload.Icon
	if payload.AdminOnly != nil {
		menu.AdminOnly = *payload.AdminOnly
	}
	targetParentID := menu.ParentID
	if payload.ParentID != nil {
		if *payload.ParentID == menu.ID {
			return nil, constants.ErrBadRequestWithMsg("菜单父级不能是自身")
		}
		if err := s.ensureParentNotDescendant(ctx, menu.ID, *payload.ParentID); err != nil {
			return nil, err
		}
		targetParentID = payload.ParentID
	}
	sortVal, err := s.ensureUniqueSiblingSort(ctx, targetParentID, payload.Sort, menu.ID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "Update", err)
	}
	menu.Sort = sortVal
	menu.Hidden = payload.Hidden
	menu.Component = payload.Component
	menu.Redirect = payload.Redirect
	if payload.Status != nil {
		menu.Status = *payload.Status
	}
	if payload.ParentID != nil {
		menu.ParentID = payload.ParentID
	}
	if err := s.menuRepo.Update(ctx, menu); err != nil {
		return nil, bizerrors.Pass(ctx, "menu", "Update", err)
	}
	s.invalidateCache()
	return menu, nil
}

// ensureParentNotDescendant 防止把菜单挂到自己的子孙节点下形成环。
func (s *MenuService) ensureParentNotDescendant(ctx context.Context, menuID, newParentID uint) error {
	all, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "menu", "ensureParentNotDescendant", err)
	}
	parentOf := make(map[uint]*uint, len(all))
	for i := range all {
		parentOf[all[i].ID] = all[i].ParentID
	}
	seen := make(map[uint]struct{}, len(all))
	for cur := newParentID; cur != 0; {
		if cur == menuID {
			return constants.ErrBadRequestWithMsg("菜单父级不能是其子菜单")
		}
		if _, ok := seen[cur]; ok {
			return constants.ErrBadRequestWithMsg("菜单层级存在环")
		}
		seen[cur] = struct{}{}
		next := parentOf[cur]
		if next == nil {
			break
		}
		cur = *next
	}
	return nil
}

// Delete 删除相关的业务逻辑。
func (s *MenuService) Delete(ctx context.Context, id uint) error {
	count, err := s.menuRepo.CountChildren(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "menu", "Delete", err)
	}
	if count > 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsga70ebaf6959d)
	}
	if err := s.menuRepo.DeleteWithBindings(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrMenuNotFound
		}
		return bizerrors.Pass(ctx, "menu", "Delete", err)
	}
	s.invalidateCache()
	return nil
}

// BatchSetStatus 批量启用/停用菜单。
func (s *MenuService) BatchSetStatus(ctx context.Context, payload MenuBatchStatusPayload) error {
	if len(payload.IDs) == 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg83ecd70cfd99)
	}
	if err := s.menuRepo.BatchUpdateStatus(ctx, payload.IDs, payload.Status); err != nil {
		return bizerrors.Pass(ctx, "menu", "BatchSetStatus", err)
	}
	s.invalidateCache()
	return nil
}

func (s *MenuService) invalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.treeCache = nil
	s.treeCacheExpireAt = time.Time{}
}

func (s *MenuService) ListPermissionBindings(ctx context.Context) ([]model.MenuPermissionBinding, error) {
	return s.menuRepo.ListPermissionBindings(ctx)
}

func (s *MenuService) ListAllFlat(ctx context.Context) ([]model.Menu, error) {
	return s.menuRepo.ListAll(ctx)
}
