package menu

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// Sync 将 DefaultCatalog 写入数据库（按 parent_id + path upsert），并应用历史数据补丁。
func Sync(ctx context.Context, db *gorm.DB) error {
	if err := CleanupDuplicateRootMenus(ctx, db); err != nil {
		return err
	}
	if err := syncSpecSubtree(ctx, db, DefaultCatalog(), nil); err != nil {
		return err
	}
	if err := applyLegacyPatches(ctx, db); err != nil {
		return err
	}
	return CleanupDuplicateRootMenus(ctx, db)
}

// syncSpecSubtree 递归 upsert 菜单子树，深度不限。
func syncSpecSubtree(ctx context.Context, db *gorm.DB, specs []Spec, parentID *uint) error {
	for _, spec := range specs {
		m, err := upsertByPath(ctx, db, spec, parentID)
		if err != nil {
			return err
		}
		if len(spec.Children) == 0 {
			continue
		}
		pid := m.ID
		if err := syncSpecSubtree(ctx, db, spec.Children, &pid); err != nil {
			return err
		}
	}
	return nil
}

func upsertByPath(ctx context.Context, db *gorm.DB, spec Spec, parentID *uint) (*model.Menu, error) {
	path := strings.TrimSpace(spec.Path)
	var existing model.Menu
	q := db.WithContext(ctx).Where("path = ?", path)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	err := q.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.WithContext(ctx).Where("path = ?", path).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				m := specToModel(spec, parentID)
				if err := db.WithContext(ctx).Create(&m).Error; err != nil {
					return nil, err
				}
				return &m, nil
			}
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	existing.ParentID = parentID
	existing.Name = spec.Name
	existing.Icon = spec.Icon
	existing.Sort = spec.Sort
	existing.Hidden = spec.Hidden
	existing.AdminOnly = spec.AdminOnly
	existing.Component = spec.Component
	existing.Redirect = spec.Redirect
	existing.Status = spec.statusOrDefault()
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func specToModel(s Spec, parentID *uint) model.Menu {
	return model.Menu{
		ParentID:  parentID,
		Path:      strings.TrimSpace(s.Path),
		Name:      s.Name,
		Icon:      s.Icon,
		Sort:      s.Sort,
		Hidden:    s.Hidden,
		AdminOnly: s.AdminOnly,
		Component: s.Component,
		Redirect:  s.Redirect,
		Status:    s.statusOrDefault(),
	}
}

func applyLegacyPatches(ctx context.Context, db *gorm.DB) error {
	var all []model.Menu
	if err := db.WithContext(ctx).Find(&all).Error; err != nil {
		return err
	}
	for i := range all {
		m := &all[i]
		if err := patchLegacyMenu(ctx, db, m); err != nil {
			return err
		}
	}
	return nil
}

func patchLegacyMenu(ctx context.Context, db *gorm.DB, m *model.Menu) error {
	p := strings.TrimSpace(m.Path)
	name := strings.TrimSpace(m.Name)
	comp := strings.TrimSpace(m.Component)
	needSave := false

	if name == "Kubernetes 管理" && p == "/kubernetes" {
		m.Name = "Kubernetes 容器管理"
		m.Icon = "KubernetesOutlined"
		needSave = true
	}

	if name == "Event 事件" && p == "/alert-events" {
		m.Path = "/events"
		p = "/events"
		needSave = true
	}
	if p == "/events" {
		if comp != "events-page" {
			m.Component = "events-page"
			needSave = true
		}
		if strings.TrimSpace(m.Icon) == "NotificationOutlined" {
			m.Icon = "FileSearchOutlined"
			needSave = true
		}
	}

	switch p {
	case "/alert-events":
		if !m.Hidden {
			m.Hidden = true
			needSave = true
		}
		want := "/alert-monitor-platform?tab=config&cfg=history"
		if strings.TrimSpace(m.Redirect) != want {
			m.Redirect = want
			needSave = true
		}
	case "/alert-config-center":
		if !m.Hidden {
			m.Hidden = true
			needSave = true
		}
		want := "/alert-monitor-platform?tab=config&cfg=policies"
		if strings.TrimSpace(m.Redirect) != want {
			m.Redirect = want
			needSave = true
		}
	case "/runtime-config":
		if !m.Hidden {
			m.Hidden = true
			needSave = true
		}
		if strings.TrimSpace(m.Redirect) != "/dict-entries" {
			m.Redirect = "/dict-entries"
			needSave = true
		}
	}

	if m.Hidden && m.Status != 0 {
		m.Status = 0
		needSave = true
	}

	if !needSave {
		return nil
	}
	return db.WithContext(ctx).Save(m).Error
}

type rootMenuDedupSpec struct {
	paths              []string
	keepName           string
	keepIcon           string
	keepSort           int
	preferNameContains string
	knownChildPaths    []string
	penaltyChildPaths  []string
}

var duplicateRootMenuSpecs = []rootMenuDedupSpec{
	{
		paths:              []string{"/kubernetes", "/kubernetes/"},
		keepName:           "Kubernetes 容器管理",
		keepIcon:           "KubernetesOutlined",
		keepSort:           5,
		preferNameContains: "容器",
		knownChildPaths:    []string{"/pods", "/clusters", "/cronjobs", "/jobs", "/events", "/k8s-resource-topology"},
		penaltyChildPaths:  []string{"/pod", "/cluster"},
	},
	{
		paths:              []string{"/alert-notify", "/alert-notify/"},
		keepName:           "告警通知",
		keepIcon:           "BellOutlined",
		keepSort:           2,
		preferNameContains: "告警",
		knownChildPaths:    []string{"/alert-channels", "/alert-monitor-platform", "/alert-duty", "/alert-maintenance"},
	},
	{
		paths:              []string{"/project-management", "/project-management/"},
		keepName:           "项目管理",
		keepIcon:           "ProjectOutlined",
		keepSort:           3,
		preferNameContains: "项目",
		knownChildPaths:    []string{"/projects", "/application-topology", "/project-logs", "/project-log-sources", "/project-servers"},
	},
}

// CleanupDuplicateRootMenus 合并重复的一级目录（/kubernetes、/alert-notify、/project-management 等）。
func CleanupDuplicateRootMenus(ctx context.Context, db *gorm.DB) error {
	for _, spec := range duplicateRootMenuSpecs {
		if err := cleanupDuplicateRootMenu(ctx, db, spec); err != nil {
			return err
		}
	}
	return nil
}

func cleanupDuplicateRootMenu(ctx context.Context, db *gorm.DB, spec rootMenuDedupSpec) error {
	var roots []model.Menu
	if err := db.WithContext(ctx).
		Where("TRIM(path) IN ? AND (parent_id IS NULL OR parent_id = 0)", spec.paths).
		Find(&roots).Error; err != nil {
		return err
	}
	if len(roots) <= 1 {
		return nil
	}

	keepID := roots[0].ID
	keepScore := -1
	for _, r := range roots {
		score := rootMenuScore(ctx, db, r, spec)
		if score > keepScore {
			keepScore = score
			keepID = r.ID
		}
	}

	for _, r := range roots {
		if r.ID == keepID {
			continue
		}
		if err := mergeMenuChildrenToParent(ctx, db, r.ID, keepID); err != nil {
			return err
		}
		ids, err := collectMenuSubtreeIDs(ctx, db, r.ID)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			continue
		}
		if err := db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
	}

	return db.WithContext(ctx).Model(&model.Menu{}).Where("id = ?", keepID).Updates(map[string]interface{}{
		"name": spec.keepName,
		"icon": spec.keepIcon,
		"sort": spec.keepSort,
	}).Error
}

func rootMenuScore(ctx context.Context, db *gorm.DB, root model.Menu, spec rootMenuDedupSpec) int {
	var children []model.Menu
	if err := db.WithContext(ctx).Where("parent_id = ?", root.ID).Find(&children).Error; err != nil {
		return 0
	}
	score := len(children) * 2
	if spec.preferNameContains != "" && strings.Contains(root.Name, spec.preferNameContains) {
		score += 10
	}
	childSet := map[string]struct{}{}
	for _, p := range spec.knownChildPaths {
		childSet[p] = struct{}{}
	}
	penaltySet := map[string]struct{}{}
	for _, p := range spec.penaltyChildPaths {
		penaltySet[p] = struct{}{}
	}
	for _, c := range children {
		if _, ok := childSet[c.Path]; ok {
			score++
		}
		if _, ok := penaltySet[c.Path]; ok {
			score--
		}
	}
	return score
}

func mergeMenuChildrenToParent(ctx context.Context, db *gorm.DB, fromRootID, toRootID uint) error {
	var children []model.Menu
	if err := db.WithContext(ctx).Where("parent_id = ?", fromRootID).Find(&children).Error; err != nil {
		return err
	}
	for _, c := range children {
		var exists model.Menu
		err := db.WithContext(ctx).Where("parent_id = ? AND path = ?", toRootID, c.Path).First(&exists).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pid := toRootID
			c.ParentID = &pid
			if err := db.WithContext(ctx).Save(&c).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		ids, err := collectMenuSubtreeIDs(ctx, db, c.ID)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			continue
		}
		if err := db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func collectMenuSubtreeIDs(ctx context.Context, db *gorm.DB, rootID uint) ([]uint, error) {
	var out []uint
	queue := []uint{rootID}
	seen := map[uint]bool{}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)

		var children []model.Menu
		if err := db.WithContext(ctx).
			Where("parent_id = ?", id).
			Find(&children).Error; err != nil {
			return nil, err
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return out, nil
}
