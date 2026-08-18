package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type K8sEventForwardRepository struct {
	db *gorm.DB
}

func NewK8sEventForwardRepository(db *gorm.DB) K8sEventForwardRepo {
	return &K8sEventForwardRepository{db: db}
}

func (r *K8sEventForwardRepository) ListRules(ctx context.Context, f K8sEventForwardRuleListFilter) (*pagination.Result[model.K8sEventForwardRule], error) {
	page, pageSize := pagination.Normalize(f.Page, f.PageSize)
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.K8sEventForwardRule{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.K8sEventForwardRule
	err := r.db.WithContext(ctx).Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.K8sEventForwardRule]{
		List: list, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (r *K8sEventForwardRepository) GetRule(ctx context.Context, id uint) (*model.K8sEventForwardRule, error) {
	var rule model.K8sEventForwardRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	return &rule, err
}

func (r *K8sEventForwardRepository) CreateRule(ctx context.Context, rule *model.K8sEventForwardRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *K8sEventForwardRepository) SaveRule(ctx context.Context, rule *model.K8sEventForwardRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *K8sEventForwardRepository) DeleteRule(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.K8sEventForwardRule{}, id)
	return res.RowsAffected, res.Error
}

func (r *K8sEventForwardRepository) GetSettings(ctx context.Context) (model.K8sEventForwardSetting, error) {
	var st model.K8sEventForwardSetting
	err := r.db.WithContext(ctx).First(&st, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.K8sEventForwardSetting{ID: 1}, nil
	}
	return st, err
}

func (r *K8sEventForwardRepository) SaveSettings(ctx context.Context, st *model.K8sEventForwardSetting) error {
	if st == nil {
		return nil
	}
	st.ID = 1
	existing, err := r.GetSettings(ctx)
	if err != nil {
		return err
	}
	if existing.ID == 0 {
		return r.db.WithContext(ctx).Create(st).Error
	}
	return r.db.WithContext(ctx).Model(&model.K8sEventForwardSetting{}).Where("id = ?", 1).
		Select("ProcessIntervalSeconds", "BatchSize", "MaxRetries", "WatcherBufferSize", "UpdatedAt").
		Updates(st).Error
}

func (r *K8sEventForwardRepository) EnsureDefaultSettings(ctx context.Context, defaults model.K8sEventForwardSetting) error {
	var st model.K8sEventForwardSetting
	err := r.db.WithContext(ctx).First(&st, 1).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	defaults.ID = 1
	return r.db.WithContext(ctx).Create(&defaults).Error
}

func (r *K8sEventForwardRepository) SaveForwardedEvent(ctx context.Context, ev *model.K8sForwardedEvent) error {
	if ev == nil {
		return nil
	}
	ev.Processed = false
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "evt_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"cluster_id": ev.ClusterID,
			"namespace":  ev.Namespace,
			"name":       ev.Name,
			"type":       ev.Type,
			"reason":     ev.Reason,
			"level":      ev.Level,
			"message":    ev.Message,
			"timestamp":  ev.Timestamp,
			"processed":  false,
			"attempts":   0,
		}),
	}).Create(ev).Error
}

func (r *K8sEventForwardRepository) ListUnprocessedEvents(ctx context.Context, limit int) ([]model.K8sForwardedEvent, error) {
	var list []model.K8sForwardedEvent
	err := r.db.WithContext(ctx).
		Where("processed = ?", false).
		Order("timestamp ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// ClaimUnprocessedEvents 在事务内加锁领取待转发事件，降低多实例重复推送。
// MySQL 5.7 无 SKIP LOCKED：对 settings 行 FOR UPDATE 串行化领取，并先标记 processed。
func (r *K8sEventForwardRepository) ClaimUnprocessedEvents(ctx context.Context, limit int) ([]model.K8sForwardedEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []model.K8sForwardedEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var st model.K8sEventForwardSetting
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&st, 1).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := tx.Where("processed = ?", false).
			Order("timestamp ASC").
			Limit(limit).
			Find(&list).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(list))
		for _, ev := range list {
			ids = append(ids, ev.ID)
		}
		return tx.Model(&model.K8sForwardedEvent{}).
			Where("id IN ? AND processed = ?", ids, false).
			Update("processed", true).Error
	})
	return list, err
}

func (r *K8sEventForwardRepository) MarkEventProcessed(ctx context.Context, id int64, processed bool) error {
	return r.db.WithContext(ctx).Model(&model.K8sForwardedEvent{}).
		Where("id = ?", id).
		Update("processed", processed).Error
}

func (r *K8sEventForwardRepository) IncrementEventAttempts(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.K8sForwardedEvent{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + ?", 1)).Error
}

func (r *K8sEventForwardRepository) ListEnabledRules(ctx context.Context) ([]model.K8sEventForwardRule, error) {
	var rules []model.K8sEventForwardRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("id ASC").Find(&rules).Error
	return rules, err
}

func (r *K8sEventForwardRepository) HasEnabledRules(ctx context.Context) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.K8sEventForwardRule{}).
		Where("enabled = ?", true).Count(&n).Error
	return n > 0, err
}

func (r *K8sEventForwardRepository) ListEnabledClusterIDs(ctx context.Context) ([]uint, error) {
	rules, err := r.ListEnabledRules(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[uint]struct{}{}
	var ids []uint
	for _, rule := range rules {
		for _, p := range strings.Split(rule.ClusterIDs, ",") {
			c := strings.TrimSpace(p)
			if c == "" {
				continue
			}
			n, err := strconv.ParseUint(c, 10, 64)
			if err != nil || n == 0 {
				continue
			}
			id := uint(n)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []uint{}, nil
	}
	// 仅保留仍启用的集群
	var enabled []uint
	err = r.db.WithContext(ctx).Model(&model.K8sCluster{}).
		Where("status = ? AND id IN ?", 1, ids).
		Pluck("id", &enabled).Error
	return enabled, err
}

func (r *K8sEventForwardRepository) GetClusterName(ctx context.Context, id uint) string {
	var c model.K8sCluster
	if err := r.db.WithContext(ctx).Select("name").First(&c, id).Error; err != nil {
		return ""
	}
	return c.Name
}

func (r *K8sEventForwardRepository) GetClusterOwningProjectID(ctx context.Context, id uint) uint {
	if id == 0 {
		return 0
	}
	var c model.K8sCluster
	if err := r.db.WithContext(ctx).Select("owning_project_id").First(&c, id).Error; err != nil {
		return 0
	}
	if c.OwningProjectID != nil && *c.OwningProjectID > 0 {
		return *c.OwningProjectID
	}
	return 0
}
