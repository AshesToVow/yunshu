package repository

import (
	"context"
	"errors"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AiRepository struct {
	db *gorm.DB
}

func NewAiRepository(db *gorm.DB) AiRepo {
	if db == nil {
		return &AiRepository{}
	}
	return &AiRepository{db: db}
}

func (r *AiRepository) Transaction(ctx context.Context, fn func(AiRepo) error) error {
	if r.db == nil {
		return errors.New("database unavailable")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&AiRepository{db: tx})
	})
}

func (r *AiRepository) dbq(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *AiRepository) firstByID(ctx context.Context, dest any, id uint) error {
	return r.dbq(ctx).First(dest, id).Error
}

func (r *AiRepository) reloadByID(ctx context.Context, dest any, id uint) error {
	return r.dbq(ctx).First(dest, id).Error
}

func (r *AiRepository) deleteByID(ctx context.Context, model any, id uint) (int64, error) {
	res := r.dbq(ctx).Delete(model, id)
	return res.RowsAffected, res.Error
}

func (r *AiRepository) updateModel(ctx context.Context, id uint, dest any, updates map[string]any) error {
	return r.dbq(ctx).Model(dest).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) countModel(ctx context.Context, model any) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(model).Count(&n).Error
	return n, err
}

func (r *AiRepository) countWhere(ctx context.Context, model any, query any, args ...any) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(model).Where(query, args...).Count(&n).Error
	return n, err
}

// --- Audit ---

func (r *AiRepository) CreateAuditEvent(ctx context.Context, ev *model.AiAuditEvent) error {
	if ev == nil {
		return nil
	}
	return r.dbq(ctx).Create(ev).Error
}

// --- Prompt ---

func (r *AiRepository) ListPrompts(ctx context.Context) ([]model.AiPrompt, error) {
	var rows []model.AiPrompt
	err := r.dbq(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetPromptByID(ctx context.Context, id uint) (*model.AiPrompt, error) {
	var row model.AiPrompt
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) GetPromptByCode(ctx context.Context, code string) (*model.AiPrompt, error) {
	var row model.AiPrompt
	err := r.dbq(ctx).Where("code = ?", code).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) GetPromptByCodeEnabled(ctx context.Context, code string) (*model.AiPrompt, error) {
	var row model.AiPrompt
	err := r.dbq(ctx).Where("code = ? AND enabled = ?", code, true).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreatePrompt(ctx context.Context, row *model.AiPrompt) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdatePrompt(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiPrompt{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeletePromptCascade(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		if err := tx.(*AiRepository).dbq(ctx).Where("prompt_id = ?", id).Delete(&model.AiPromptVersion{}).Error; err != nil {
			return err
		}
		res := tx.(*AiRepository).dbq(ctx).Delete(&model.AiPrompt{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *AiRepository) CreatePromptWithVersionTx(ctx context.Context, prompt *model.AiPrompt, version *model.AiPromptVersion) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if err := repo.dbq(ctx).Create(prompt).Error; err != nil {
			return err
		}
		if version != nil && version.Content != "" {
			version.PromptID = prompt.ID
			return repo.dbq(ctx).Create(version).Error
		}
		return nil
	})
}

func (r *AiRepository) ListPromptVersions(ctx context.Context, promptID uint) ([]model.AiPromptVersion, error) {
	var rows []model.AiPromptVersion
	err := r.dbq(ctx).Where("prompt_id = ?", promptID).Order("version DESC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetCurrentPromptVersion(ctx context.Context, promptID uint) (*model.AiPromptVersion, error) {
	var ver model.AiPromptVersion
	err := r.dbq(ctx).Where("prompt_id = ? AND is_current = ?", promptID, true).First(&ver).Error
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

func (r *AiRepository) GetPromptVersionByID(ctx context.Context, promptID, versionID uint) (*model.AiPromptVersion, error) {
	var ver model.AiPromptVersion
	err := r.dbq(ctx).Where("id = ? AND prompt_id = ?", versionID, promptID).First(&ver).Error
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

func (r *AiRepository) GetPromptMaxVersion(ctx context.Context, promptID uint) (int, error) {
	var maxVer int
	err := r.dbq(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
		Select("COALESCE(MAX(version),0)").Scan(&maxVer).Error
	return maxVer, err
}

func (r *AiRepository) ClearPromptCurrentVersions(ctx context.Context, promptID uint) error {
	return r.dbq(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
		Update("is_current", false).Error
}

func (r *AiRepository) PublishPromptVersion(ctx context.Context, promptID, userID uint, content, changelog string) (*model.AiPromptVersion, error) {
	maxVer, err := r.GetPromptMaxVersion(ctx, promptID)
	if err != nil {
		return nil, err
	}
	_ = r.ClearPromptCurrentVersions(ctx, promptID)
	ver := model.AiPromptVersion{
		PromptID: promptID, Version: maxVer + 1, Content: content,
		Changelog: changelog, IsCurrent: true, CreatedBy: userID,
	}
	if err := r.dbq(ctx).Create(&ver).Error; err != nil {
		return nil, err
	}
	return &ver, nil
}

func (r *AiRepository) RollbackPromptVersion(ctx context.Context, promptID, versionID uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		var ver model.AiPromptVersion
		if err := repo.dbq(ctx).Where("id = ? AND prompt_id = ?", versionID, promptID).First(&ver).Error; err != nil {
			return err
		}
		if err := repo.dbq(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
			Update("is_current", false).Error; err != nil {
			return err
		}
		return repo.dbq(ctx).Model(&ver).Update("is_current", true).Error
	})
}

func (r *AiRepository) CountPromptVersions(ctx context.Context, promptID uint) (int64, error) {
	return r.countWhere(ctx, &model.AiPromptVersion{}, "prompt_id = ?", promptID)
}

func (r *AiRepository) CountPrompts(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiPrompt{})
}

func (r *AiRepository) SeedPromptVersionUpdate(ctx context.Context, promptID uint, content string, nextVersion int) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if err := repo.dbq(ctx).Model(&model.AiPromptVersion{}).Where("prompt_id = ?", promptID).
			Update("is_current", false).Error; err != nil {
			return err
		}
		ver := model.AiPromptVersion{
			PromptID: promptID, Version: nextVersion, Content: content,
			Changelog: "seed update", IsCurrent: true,
		}
		return repo.dbq(ctx).Create(&ver).Error
	})
}

// --- LLM Model ---

func (r *AiRepository) ListLLMModels(ctx context.Context) ([]model.AiLLMModel, error) {
	var rows []model.AiLLMModel
	err := r.dbq(ctx).Order("is_default DESC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledLLMModels(ctx context.Context) ([]model.AiLLMModel, error) {
	var rows []model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ?", true).Order("is_default DESC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) CountEnabledLLMModels(ctx context.Context) (int64, error) {
	return r.countWhere(ctx, &model.AiLLMModel{}, "enabled = ?", true)
}

func (r *AiRepository) GetLLMModelByID(ctx context.Context, id uint) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateLLMModel(ctx context.Context, row *model.AiLLMModel, clearDefault bool) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if clearDefault {
			if err := repo.dbq(ctx).Model(&model.AiLLMModel{}).Where("is_default = ?", true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return repo.dbq(ctx).Create(row).Error
	})
}

func (r *AiRepository) UpdateLLMModel(ctx context.Context, id uint, updates map[string]any, clearOtherDefault bool) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if clearOtherDefault {
			if err := repo.dbq(ctx).Model(&model.AiLLMModel{}).Where("id <> ? AND is_default = ?", id, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return repo.dbq(ctx).Model(&model.AiLLMModel{}).Where("id = ?", id).Updates(updates).Error
	})
}

func (r *AiRepository) DeleteLLMModel(ctx context.Context, id uint) error {
	_, err := r.deleteByID(ctx, &model.AiLLMModel{}, id)
	return err
}

func (r *AiRepository) SetDefaultLLMModel(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if err := repo.dbq(ctx).Model(&model.AiLLMModel{}).Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return repo.dbq(ctx).Model(&model.AiLLMModel{}).Where("id = ?", id).
			Updates(map[string]any{"is_default": true, "enabled": true}).Error
	})
}

func (r *AiRepository) FindLLMModelByName(ctx context.Context, name string) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ? AND name = ?", true, name).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) FindLLMModelByProvider(ctx context.Context, provider string) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ? AND provider = ?", true, provider).
		Order("is_default DESC, id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) FindDefaultLLMModel(ctx context.Context) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ? AND is_default = ?", true, true).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) FindFirstEnabledLLMModel(ctx context.Context) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ?", true).Order("id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) FindEmbeddingModel(ctx context.Context) (*model.AiLLMModel, error) {
	var row model.AiLLMModel
	err := r.dbq(ctx).Where("enabled = ? AND model_type = ?", true, "embedding").
		Order("is_default DESC, id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
