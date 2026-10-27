package repository

import (
	"context"
	"errors"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

// ErrRegistrationNotPending 表示申请已被其他审核人处理，条件更新未命中。
var ErrRegistrationNotPending = errors.New("registration request is not pending")

// RegistrationRequestRepository 用户自助注册申请的持久化访问。
type RegistrationRequestRepository struct {
	db *gorm.DB
}

// RegistrationRequestListParams 注册申请列表查询条件。
type RegistrationRequestListParams struct {
	Keyword  string
	Status   *int
	Page     int
	PageSize int
}

func NewRegistrationRequestRepository(db *gorm.DB) RegistrationRequestRepo {
	return &RegistrationRequestRepository{db: db}
}

func (r *RegistrationRequestRepository) Create(ctx context.Context, req *model.RegistrationRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *RegistrationRequestRepository) GetByID(ctx context.Context, id uint) (*model.RegistrationRequest, error) {
	var req model.RegistrationRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *RegistrationRequestRepository) List(ctx context.Context, params RegistrationRequestListParams) ([]model.RegistrationRequest, int64, error) {
	var list []model.RegistrationRequest
	query := r.db.WithContext(ctx).Model(&model.RegistrationRequest{})

	if params.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?",
			"%"+params.Keyword+"%", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	page, pageSize := pagination.Normalize(params.Page, params.PageSize)
	total, err := listWithPagination(query, page, pageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *RegistrationRequestRepository) UpdateStatus(ctx context.Context, id uint, status model.RegistrationRequestStatus, reviewerID uint, comment string) error {
	res := r.db.WithContext(ctx).Model(&model.RegistrationRequest{}).
		Where("id = ? AND status = ?", id, model.RegistrationPending).
		Updates(reviewFields(status, reviewerID, comment))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRegistrationNotPending
	}
	return nil
}

// ApproveWithUser 在同一事务内把申请置为通过并创建用户，
// 避免出现“申请已通过但用户未创建”或并发重复审核产生两个用户。
func (r *RegistrationRequestRepository) ApproveWithUser(ctx context.Context, id uint, reviewerID uint, comment string, user *model.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.RegistrationRequest{}).
			Where("id = ? AND status = ?", id, model.RegistrationPending).
			Updates(reviewFields(model.RegistrationApproved, reviewerID, comment))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRegistrationNotPending
		}
		return tx.Create(user).Error
	})
}

func reviewFields(status model.RegistrationRequestStatus, reviewerID uint, comment string) map[string]interface{} {
	return map[string]interface{}{
		"status":         status,
		"reviewer_id":    reviewerID,
		"review_comment": comment,
		"reviewed_at":    time.Now(),
	}
}

func (r *RegistrationRequestRepository) CountPending(ctx context.Context, username, email string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RegistrationRequest{}).
		Where("(username = ? OR email = ?) AND status = ?", username, email, model.RegistrationPending).
		Count(&count).Error
	return count, err
}
