package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertChannelRepository struct {
	db *gorm.DB
}

func NewAlertChannelRepository(db *gorm.DB) AlertChannelRepo {
	return &AlertChannelRepository{db: db}
}

func (r *AlertChannelRepository) ListEnabled(ctx context.Context) ([]*model.AlertChannel, error) {
	var list []model.AlertChannel
	if err := r.db.WithContext(ctx).Model(&model.AlertChannel{}).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*model.AlertChannel, len(list))
	for i := range list {
		out[i] = &list[i]
	}
	return out, nil
}

func (r *AlertChannelRepository) List(ctx context.Context, f AlertChannelListFilter) ([]model.AlertChannel, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertChannel{}).Order("id DESC")
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR url LIKE ?", like, like)
	}
	var list []model.AlertChannel
	return list, tx.Find(&list).Error
}

func (r *AlertChannelRepository) GetByID(ctx context.Context, id uint) (*model.AlertChannel, error) {
	var ch model.AlertChannel
	if err := r.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *AlertChannelRepository) Create(ctx context.Context, ch *model.AlertChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *AlertChannelRepository) Save(ctx context.Context, ch *model.AlertChannel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

func (r *AlertChannelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AlertChannel{}, id).Error
}

var _ AlertChannelRepo = (*AlertChannelRepository)(nil)
