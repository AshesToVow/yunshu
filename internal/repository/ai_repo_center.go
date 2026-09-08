package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// --- Tool ---

func (r *AiRepository) ListTools(ctx context.Context) ([]model.AiToolDef, error) {
	var rows []model.AiToolDef
	err := r.dbq(ctx).Order("module ASC, name ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledScriptTools(ctx context.Context) ([]model.AiToolDef, error) {
	var rows []model.AiToolDef
	err := r.dbq(ctx).Where("enabled = ? AND runtime = ?", true, "script").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetToolByID(ctx context.Context, id uint) (*model.AiToolDef, error) {
	var row model.AiToolDef
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) GetToolByName(ctx context.Context, name string) (*model.AiToolDef, error) {
	var row model.AiToolDef
	err := r.dbq(ctx).Where("name = ?", name).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateTool(ctx context.Context, row *model.AiToolDef) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateTool(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiToolDef{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteToolRow(ctx context.Context, row *model.AiToolDef) error {
	return r.dbq(ctx).Delete(row).Error
}

func (r *AiRepository) UpdateToolEnabled(ctx context.Context, id uint, enabled bool) error {
	return r.dbq(ctx).Model(&model.AiToolDef{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (r *AiRepository) CountToolsByRuntime(ctx context.Context, runtime string) (int64, error) {
	return r.countWhere(ctx, &model.AiToolDef{}, "runtime = ?", runtime)
}

func (r *AiRepository) CountAllTools(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiToolDef{})
}

func (r *AiRepository) CountToolsByName(ctx context.Context, name string) (int64, error) {
	return r.countWhere(ctx, &model.AiToolDef{}, "name = ?", name)
}

// --- SOP ---

func (r *AiRepository) ListSOPs(ctx context.Context, limit int) ([]model.AiSOP, error) {
	var rows []model.AiSOP
	q := r.dbq(ctx).Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledSOPs(ctx context.Context, limit int) ([]model.AiSOP, error) {
	var rows []model.AiSOP
	q := r.dbq(ctx).Where("enabled = ?", true)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetSOPByID(ctx context.Context, id uint) (*model.AiSOP, error) {
	var row model.AiSOP
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateSOP(ctx context.Context, row *model.AiSOP) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateSOP(ctx context.Context, id uint, updates map[string]any) error {
	if err := r.dbq(ctx).Model(&model.AiSOP{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (r *AiRepository) DeleteSOP(ctx context.Context, id uint) error {
	n, err := r.deleteByID(ctx, &model.AiSOP{}, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AiRepository) CountSOPs(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiSOP{})
}

func (r *AiRepository) CountSOPByCode(ctx context.Context, code string) (int64, error) {
	return r.countWhere(ctx, &model.AiSOP{}, "code = ?", code)
}

// --- Incident Case ---

func (r *AiRepository) ListIncidentCases(ctx context.Context, limit int) ([]model.AiIncidentCase, error) {
	var rows []model.AiIncidentCase
	q := r.dbq(ctx).Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledIncidentCases(ctx context.Context, limit int) ([]model.AiIncidentCase, error) {
	var rows []model.AiIncidentCase
	q := r.dbq(ctx).Where("enabled = ?", true)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetIncidentCaseByID(ctx context.Context, id uint) (*model.AiIncidentCase, error) {
	var row model.AiIncidentCase
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateIncidentCase(ctx context.Context, row *model.AiIncidentCase) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateIncidentCase(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiIncidentCase{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteIncidentCase(ctx context.Context, id uint) error {
	n, err := r.deleteByID(ctx, &model.AiIncidentCase{}, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AiRepository) CountIncidentCases(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiIncidentCase{})
}

func (r *AiRepository) CountIncidentCaseByCaseID(ctx context.Context, caseID string) (int64, error) {
	return r.countWhere(ctx, &model.AiIncidentCase{}, "case_id = ?", caseID)
}

// --- Knowledge Base ---

func (r *AiRepository) ListKnowledgeBases(ctx context.Context) ([]model.AiKnowledgeBase, error) {
	var rows []model.AiKnowledgeBase
	err := r.dbq(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetKnowledgeBaseByID(ctx context.Context, id uint) (*model.AiKnowledgeBase, error) {
	var row model.AiKnowledgeBase
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) GetKnowledgeBaseByCode(ctx context.Context, code string) (*model.AiKnowledgeBase, error) {
	var row model.AiKnowledgeBase
	err := r.dbq(ctx).Where("code = ?", code).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateKnowledgeBase(ctx context.Context, row *model.AiKnowledgeBase) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateKnowledgeBase(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiKnowledgeBase{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteKnowledgeBaseCascade(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		var docs []model.AiKbDocument
		_ = repo.dbq(ctx).Where("kb_id = ?", id).Find(&docs).Error
		for _, d := range docs {
			_ = repo.dbq(ctx).Where("document_id = ?", d.ID).Delete(&model.AiKbChunk{}).Error
		}
		_ = repo.dbq(ctx).Where("kb_id = ?", id).Delete(&model.AiKbDocument{}).Error
		_ = repo.dbq(ctx).Where("kb_id = ?", id).Delete(&model.AiKbChunk{}).Error
		res := repo.dbq(ctx).Delete(&model.AiKnowledgeBase{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *AiRepository) CountKnowledgeBases(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiKnowledgeBase{})
}

// --- KB Document ---

func (r *AiRepository) ListKBDocuments(ctx context.Context, kbID uint, limit int) ([]model.AiKbDocument, error) {
	var rows []model.AiKbDocument
	q := r.dbq(ctx).Order("id DESC")
	if kbID > 0 {
		q = q.Where("kb_id = ?", kbID)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledKBDocuments(ctx context.Context) ([]model.AiKbDocument, error) {
	var rows []model.AiKbDocument
	err := r.dbq(ctx).Where("enabled = ?", true).Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetKBDocumentByID(ctx context.Context, id uint) (*model.AiKbDocument, error) {
	var row model.AiKbDocument
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateKBDocument(ctx context.Context, row *model.AiKbDocument) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateKBDocument(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiKbDocument{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteKBDocumentCascade(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		_ = repo.dbq(ctx).Where("document_id = ?", id).Delete(&model.AiKbChunk{}).Error
		res := repo.dbq(ctx).Delete(&model.AiKbDocument{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *AiRepository) DeleteKBDocumentChunks(ctx context.Context, documentID uint) error {
	return r.dbq(ctx).Where("document_id = ?", documentID).Delete(&model.AiKbChunk{}).Error
}

func (r *AiRepository) CountKBDocuments(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiKbDocument{})
}

func (r *AiRepository) CountKBDocumentBySource(ctx context.Context, kbID uint, source string) (int64, error) {
	return r.countWhere(ctx, &model.AiKbDocument{}, "kb_id = ? AND source = ?", kbID, source)
}

// --- Eval ---

func (r *AiRepository) ListEvalCases(ctx context.Context) ([]model.AiEvalCase, error) {
	var rows []model.AiEvalCase
	err := r.dbq(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *AiRepository) ListEnabledEvalCases(ctx context.Context) ([]model.AiEvalCase, error) {
	var rows []model.AiEvalCase
	err := r.dbq(ctx).Where("enabled = ?", true).Find(&rows).Error
	return rows, err
}

func (r *AiRepository) GetEvalCaseByID(ctx context.Context, id uint) (*model.AiEvalCase, error) {
	var row model.AiEvalCase
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) CreateEvalCase(ctx context.Context, row *model.AiEvalCase) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) UpdateEvalCase(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiEvalCase{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteEvalCase(ctx context.Context, id uint) error {
	n, err := r.deleteByID(ctx, &model.AiEvalCase{}, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AiRepository) CountEvalCases(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiEvalCase{})
}

func (r *AiRepository) CountEvalCaseByCaseCode(ctx context.Context, caseCode string) (int64, error) {
	return r.countWhere(ctx, &model.AiEvalCase{}, "case_code = ?", caseCode)
}

func (r *AiRepository) CreateEvalRun(ctx context.Context, run *model.AiEvalRun) error {
	return r.dbq(ctx).Create(run).Error
}

func (r *AiRepository) SaveEvalRun(ctx context.Context, run *model.AiEvalRun) error {
	return r.dbq(ctx).Save(run).Error
}

func (r *AiRepository) CreateEvalResult(ctx context.Context, result *model.AiEvalResult) error {
	return r.dbq(ctx).Create(result).Error
}
