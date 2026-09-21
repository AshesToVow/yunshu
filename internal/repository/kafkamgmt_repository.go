package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type KafkamgmtRepository struct {
	db *gorm.DB
}

func NewKafkamgmtRepository(db *gorm.DB) KafkamgmtRepo {
	if db == nil {
		return &KafkamgmtRepository{}
	}
	return &KafkamgmtRepository{db: db}
}

func (r *KafkamgmtRepository) ListConnections(ctx context.Context) ([]model.KafkamgmtConnection, error) {
	var list []model.KafkamgmtConnection
	if err := r.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *KafkamgmtRepository) GetConnection(ctx context.Context, id uint) (*model.KafkamgmtConnection, error) {
	var row model.KafkamgmtConnection
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *KafkamgmtRepository) GetConnectionOwner(ctx context.Context, id uint) (*model.KafkamgmtConnection, error) {
	var row model.KafkamgmtConnection
	if err := r.db.WithContext(ctx).Select("id", "owner_user_id").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *KafkamgmtRepository) FindDictImportConnection(ctx context.Context, remarkMarker, name string) (*model.KafkamgmtConnection, error) {
	var existing model.KafkamgmtConnection
	err := r.db.WithContext(ctx).
		Where("remark LIKE ? OR name = ?", "%"+remarkMarker+"%", name).
		Order("id ASC").
		First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *KafkamgmtRepository) CountDefaultConnections(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.KafkamgmtConnection{}).Where("is_default = ?", true).Count(&n).Error
	return n, err
}

func (r *KafkamgmtRepository) GetDefaultConnection(ctx context.Context) (*model.KafkamgmtConnection, error) {
	var def model.KafkamgmtConnection
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&def).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *KafkamgmtRepository) CreateConnectionClearDefaults(ctx context.Context, row *model.KafkamgmtConnection) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.KafkamgmtConnection{}).
				Where("is_default = ?", true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(row).Error
	})
}

func (r *KafkamgmtRepository) UpdateConnectionClearDefaults(ctx context.Context, id uint, row *model.KafkamgmtConnection) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.KafkamgmtConnection{}).
				Where("is_default = ? AND id <> ?", true, id).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(row).Error
	})
}

func (r *KafkamgmtRepository) DeleteConnection(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.KafkamgmtConnection{}, id)
	return res.RowsAffected, res.Error
}
