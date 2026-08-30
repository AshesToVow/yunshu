package inspect

// 巡检项（inspect_item）：项目自有项 CRUD，以及与全局模板项的同步/重置。

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

type ItemUpsertRequest struct {
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Query         string  `json:"query"`
	Threshold     float64 `json:"threshold"`
	ThresholdType string  `json:"threshold_type"`
	Unit          string  `json:"unit"`
	LabelsJSON    string  `json:"labels_json"`
	Enabled       *bool   `json:"enabled"`
	SortOrder     *int    `json:"sort_order"`
}

func (s *Service) ListItems(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var projectItems []model.InspectItem
	if err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sort_order ASC, id ASC").
		Find(&projectItems).Error; err != nil {
		return nil, err
	}
	if len(projectItems) > 0 {
		return projectItems, nil
	}
	var globals []model.InspectItem
	err := s.db.WithContext(ctx).
		Where("project_id = 0").
		Order("sort_order ASC, id ASC").
		Find(&globals).Error
	return globals, err
}

func (s *Service) CreateItem(ctx context.Context, projectID uint, req ItemUpsertRequest) (*model.InspectItem, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	name := strings.TrimSpace(req.Name)
	query := strings.TrimSpace(req.Query)
	if name == "" || query == "" {
		return nil, constants.ErrBadRequestWithMsg("name and query required")
	}
	tt := strings.TrimSpace(req.ThresholdType)
	if tt == "" {
		tt = "greater"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sortOrder := 1000
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	item := model.InspectItem{
		ProjectID:     projectID,
		Type:          strings.TrimSpace(req.Type),
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		Query:         query,
		Threshold:     req.Threshold,
		ThresholdType: tt,
		Unit:          strings.TrimSpace(req.Unit),
		LabelsJSON:    strings.TrimSpace(req.LabelsJSON),
		Enabled:       enabled,
		SortOrder:     sortOrder,
	}
	if item.Type == "" {
		item.Type = "自定义"
	}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) UpdateItem(ctx context.Context, projectID, itemID uint, req ItemUpsertRequest) (*model.InspectItem, error) {
	var item model.InspectItem
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("巡检项不存在或不可修改全局模板（请先同步到项目）")
		}
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		item.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Type) != "" {
		item.Type = strings.TrimSpace(req.Type)
	}
	if strings.TrimSpace(req.Query) != "" {
		item.Query = strings.TrimSpace(req.Query)
	}
	item.Description = strings.TrimSpace(req.Description)
	item.Threshold = req.Threshold
	if strings.TrimSpace(req.ThresholdType) != "" {
		item.ThresholdType = strings.TrimSpace(req.ThresholdType)
	}
	item.Unit = strings.TrimSpace(req.Unit)
	if req.LabelsJSON != "" {
		item.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) DeleteItem(ctx context.Context, projectID, itemID uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).Delete(&model.InspectItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("巡检项不存在")
	}
	return nil
}

// SyncItemsFromTemplate 将全局模板复制为项目项（已存在同名则跳过；含默认关闭项，便于按需启用）。
func (s *Service) SyncItemsFromTemplate(ctx context.Context, projectID uint) (int, error) {
	if projectID == 0 {
		return 0, constants.ErrBadRequestWithMsg("project_id required")
	}
	var globals []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = 0").Order("sort_order ASC, id ASC").Find(&globals).Error; err != nil {
		return 0, err
	}
	var existing []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&existing).Error; err != nil {
		return 0, err
	}
	have := map[string]bool{}
	for _, e := range existing {
		have[e.Type+"|"+e.Name] = true
	}
	created := 0
	for _, g := range globals {
		key := g.Type + "|" + g.Name
		if have[key] {
			continue
		}
		cp := g
		cp.ID = 0
		cp.ProjectID = projectID
		cp.CreatedAt = time.Time{}
		cp.UpdatedAt = time.Time{}
		if err := s.db.WithContext(ctx).Create(&cp).Error; err != nil {
			return created, err
		}
		created++
		have[key] = true
	}
	return created, nil
}

// ResetItemsFromTemplate 删除项目自有巡检项后，重新从全局模板全量同步（用于切换 Telegraf 模板等）。
func (s *Service) ResetItemsFromTemplate(ctx context.Context, projectID uint) (int, error) {
	if projectID == 0 {
		return 0, constants.ErrBadRequestWithMsg("project_id required")
	}
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&model.InspectItem{}).Error; err != nil {
		return 0, err
	}
	return s.SyncItemsFromTemplate(ctx, projectID)
}

// effectiveItems 执行时生效的巡检项：优先项目自有启用项，为空则回退全局启用项。
func (s *Service) effectiveItems(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var projectItems []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).
		Order("sort_order ASC, id ASC").Find(&projectItems).Error; err != nil {
		return nil, err
	}
	if len(projectItems) > 0 {
		return projectItems, nil
	}
	var globals []model.InspectItem
	err := s.db.WithContext(ctx).Where("project_id = 0 AND enabled = ?", true).
		Order("sort_order ASC, id ASC").Find(&globals).Error
	return globals, err
}

// SeedGlobalTemplates 幂等写入/刷新全局巡检模板项（按 type+name upsert）。
func (s *Service) SeedGlobalTemplates(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	for _, want := range defaultTemplateItems() {
		var row model.InspectItem
		err := s.db.WithContext(ctx).
			Where("project_id = 0 AND type = ? AND name = ?", want.Type, want.Name).
			First(&row).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.WithContext(ctx).Create(&want).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		row.Description = want.Description
		row.Query = want.Query
		row.Threshold = want.Threshold
		row.ThresholdType = want.ThresholdType
		row.Unit = want.Unit
		row.SortOrder = want.SortOrder
		// 不覆盖管理员已改的 Enabled
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
