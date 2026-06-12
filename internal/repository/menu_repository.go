package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type MenuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepo {
	return &MenuRepository{db: db}
}

func (r *MenuRepository) Create(ctx context.Context, menu *model.Menu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *MenuRepository) GetByID(ctx context.Context, id uint) (*model.Menu, error) {
	var menu model.Menu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepository) Update(ctx context.Context, menu *model.Menu) error {
	return r.db.WithContext(ctx).Save(menu).Error
}

func (r *MenuRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Menu{}, id).Error
}

func (r *MenuRepository) ListAll(ctx context.Context) ([]model.Menu, error) {
	var list []model.Menu
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *MenuRepository) Tree(ctx context.Context) ([]model.Menu, error) {
	list, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(list), nil
}

// buildMenuTree 单次遍历构建菜单树，避免递归扫描全表导致的 O(n²)。
func buildMenuTree(menus []model.Menu) []model.Menu {
	if len(menus) == 0 {
		return nil
	}

	childrenByParent := make(map[uint][]model.Menu, len(menus))
	roots := make([]model.Menu, 0, 16)
	for _, m := range menus {
		if m.ParentID == nil {
			roots = append(roots, m)
			continue
		}
		pid := *m.ParentID
		childrenByParent[pid] = append(childrenByParent[pid], m)
	}

	var attach func(m model.Menu) model.Menu
	attach = func(m model.Menu) model.Menu {
		kids := childrenByParent[m.ID]
		if len(kids) == 0 {
			return m
		}
		m.Children = make([]model.Menu, len(kids))
		for i, child := range kids {
			m.Children[i] = attach(child)
		}
		return m
	}

	tree := make([]model.Menu, len(roots))
	for i, root := range roots {
		tree[i] = attach(root)
	}
	return tree
}

func (r *MenuRepository) CountChildren(ctx context.Context, parentID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Menu{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

func (r *MenuRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.Menu{}).Where("id IN ?", ids).Update("status", status).Error
}
