package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertRuleAssigneeRepository struct {
	db *gorm.DB
}

func NewAlertRuleAssigneeRepository(db *gorm.DB) AlertRuleAssigneeRepo {
	return &AlertRuleAssigneeRepository{db: db}
}

func (r *AlertRuleAssigneeRepository) ListByRule(ctx context.Context, ruleID uint) ([]model.AlertRuleAssignee, error) {
	var list []model.AlertRuleAssignee
	err := r.db.WithContext(ctx).Where("monitor_rule_id = ?", ruleID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AlertRuleAssigneeRepository) GetPrimaryByRule(ctx context.Context, ruleID uint) (*model.AlertRuleAssignee, error) {
	var row model.AlertRuleAssignee
	err := r.db.WithContext(ctx).Where("monitor_rule_id = ?", ruleID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertRuleAssigneeRepository) Create(ctx context.Context, row *model.AlertRuleAssignee) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertRuleAssigneeRepository) Save(ctx context.Context, row *model.AlertRuleAssignee) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertRuleAssigneeRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertRuleAssignee{}, id)
	return res.RowsAffected, res.Error
}

func (r *AlertRuleAssigneeRepository) ListAll(ctx context.Context) ([]model.AlertRuleAssignee, error) {
	var list []model.AlertRuleAssignee
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (r *AlertRuleAssigneeRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.AlertRuleAssignee{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AlertRuleAssigneeRepository) ListProjectMemberUserIDsByDepartments(ctx context.Context, projectID uint, deptIDs []uint) ([]uint, error) {
	if projectID == 0 || len(deptIDs) == 0 {
		return nil, nil
	}
	var uids []uint
	err := r.db.WithContext(ctx).Table("project_members AS pm").
		Select("DISTINCT pm.user_id").
		Joins("INNER JOIN users u ON u.id = pm.user_id AND u.deleted_at IS NULL").
		Where("pm.project_id = ? AND pm.deleted_at IS NULL AND u.department_id IN ?", projectID, deptIDs).
		Pluck("pm.user_id", &uids).Error
	return uids, err
}

var _ AlertRuleAssigneeRepo = (*AlertRuleAssigneeRepository)(nil)
