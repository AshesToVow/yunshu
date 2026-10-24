package logplatform

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
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
	repo interfaces.LogSavedQueryRepository
}

func NewLogSavedQueryService(repo interfaces.LogSavedQueryRepository) *LogSavedQueryService {
	return &LogSavedQueryService{repo: repo}
}

// SavedQueries 复用 ClusterLogService 注入的收藏 Repo。
func (s *ClusterLogService) SavedQueries() *LogSavedQueryService {
	return NewLogSavedQueryService(s.savedQueryRepo)
}

func (s *LogSavedQueryService) List(ctx context.Context, userID, projectID uint) ([]model.PlatformSavedQuery, error) {
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
	return s.repo.List(ctx, userID, projectID)
}

func (s *LogSavedQueryService) Create(ctx context.Context, userID uint, req LogSavedQueryUpsert) (*model.PlatformSavedQuery, error) {
	if s.repo == nil {
		return nil, constants.ErrBadRequestWithMsg("数据库不可用")
	}
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
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *LogSavedQueryService) Delete(ctx context.Context, userID, id uint) error {
	if s.repo == nil {
		return constants.ErrBadRequestWithMsg("数据库不可用")
	}
	n, err := s.repo.DeleteByUser(ctx, userID, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg("收藏不存在")
	}
	return nil
}
