package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServerAccessGrantRepository struct {
	db *gorm.DB
}

func NewServerAccessGrantRepository(db *gorm.DB) ServerAccessGrantRepo {
	return &ServerAccessGrantRepository{db: db}
}

func (r *ServerAccessGrantRepository) Get(
	ctx context.Context,
	projectID, serverID uint,
	principalKind, principalRef string,
) (*model.ServerAccessGrant, error) {
	var g model.ServerAccessGrant
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND server_id = ? AND principal_kind = ? AND principal_ref = ?",
			projectID, serverID, principalKind, principalRef).
		First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *ServerAccessGrantRepository) ListVisibleServerIDs(
	ctx context.Context,
	projectID uint,
	principalKind, principalRef string,
) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.ServerAccessGrant{}).
		Where("project_id = ? AND principal_kind = ? AND principal_ref = ? AND (can_view = ? OR can_exec = ? OR can_manage = ?)",
			projectID, principalKind, principalRef, true, true, true).
		Pluck("server_id", &ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

func (r *ServerAccessGrantRepository) List(ctx context.Context, p ServerAccessGrantListParams) ([]model.ServerAccessGrant, error) {
	q := r.db.WithContext(ctx).Model(&model.ServerAccessGrant{}).Where("project_id = ?", p.ProjectID)
	if p.PrincipalKind != "" && p.PrincipalRef != "" {
		q = q.Where("principal_kind = ? AND principal_ref = ?", p.PrincipalKind, p.PrincipalRef)
	}
	if p.ServerID > 0 {
		q = q.Where("server_id = ?", p.ServerID)
	}
	var rows []model.ServerAccessGrant
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ServerAccessGrantRepository) Upsert(ctx context.Context, row *model.ServerAccessGrant) error {
	if row == nil {
		return nil
	}
	remark := strings.TrimSpace(row.Remark)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "project_id"}, {Name: "server_id"},
			{Name: "principal_kind"}, {Name: "principal_ref"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"can_view":   row.CanView,
			"can_exec":   row.CanExec,
			"can_manage": row.CanManage,
			"remark":     remark,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(row).Error
}

func (r *ServerAccessGrantRepository) Delete(ctx context.Context, projectID, grantID uint) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND project_id = ?", grantID, projectID).
		Delete(&model.ServerAccessGrant{})
	return res.RowsAffected, res.Error
}
