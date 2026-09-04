package logplatform

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

const savedQueryKindLog = "log"

// LogSavedQueryUpsert 收藏当前日志检索条件。
type LogSavedQueryUpsert struct {
	Name      string         `json:"name"`
	ProjectID uint           `json:"project_id"`
	Query     map[string]any `json:"query"` // 表单条件 JSON
	Remark    string         `json:"remark"`
}

// LogSavedQueryService 日志查询收藏（复用 platform_saved_queries，kind=log）。
type LogSavedQueryService struct {
	db *gorm.DB
}

func NewLogSavedQueryService(db *gorm.DB) *LogSavedQueryService {
	return &LogSavedQueryService{db: db}
}

// SavedQueries 复用 ClusterLogService 的 DB。
func (s *ClusterLogService) SavedQueries() *LogSavedQueryService {
	return NewLogSavedQueryService(s.db)
}

func (s *LogSavedQueryService) List(ctx context.Context, userID, projectID uint) ([]model.PlatformSavedQuery, error) {
	q := s.db.WithContext(ctx).Where("user_id = ? AND kind = ?", userID, savedQueryKindLog)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	var list []model.PlatformSavedQuery
	if err := q.Order("updated_at DESC").Limit(100).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *LogSavedQueryService) Create(ctx context.Context, userID uint, req LogSavedQueryUpsert) (*model.PlatformSavedQuery, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("名称不能为空")
	}
	if req.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	raw, err := json.Marshal(req.Query)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil, constants.ErrBadRequestWithMsg("query 无效")
	}
	row := model.PlatformSavedQuery{
		UserID:    userID,
		Name:      name,
		Query:     string(raw),
		Kind:      savedQueryKindLog,
		ProjectID: req.ProjectID,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *LogSavedQueryService) Delete(ctx context.Context, userID, id uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND kind = ?", id, userID, savedQueryKindLog).
		Delete(&model.PlatformSavedQuery{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("收藏不存在")
	}
	return nil
}
