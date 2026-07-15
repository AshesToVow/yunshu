package logplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/esclient"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

const globalRetentionProjectID uint = 0

type LogRetentionService struct {
	es   *ElasticsearchProvider
	repo interfaces.LogRetentionRepository
}

func NewLogRetentionService(es *ElasticsearchProvider, repo interfaces.LogRetentionRepository) *LogRetentionService {
	return &LogRetentionService{es: es, repo: repo}
}

type LogRetentionItem struct {
	ID              uint    `json:"id"`
	ProjectID       uint    `json:"project_id"`
	ServerID        uint    `json:"server_id"`
	RetentionDays   int     `json:"retention_days"`
	MaxStoreBytes   int64   `json:"max_store_bytes"`
	MaxIndexCount   int     `json:"max_index_count"`
	Enabled         bool    `json:"enabled"`
	IndexPattern    string  `json:"index_pattern,omitempty"`
	Remark          *string `json:"remark,omitempty"`
	Source          string  `json:"source,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

type LogRetentionUpsertRequest struct {
	ServerID        *uint   `json:"server_id"`
	RetentionDays   int     `json:"retention_days" binding:"required,min=1,max=3650"`
	MaxStoreBytes   int64   `json:"max_store_bytes"`
	MaxIndexCount   int     `json:"max_index_count"`
	Enabled         bool    `json:"enabled"`
	IndexPattern    *string `json:"index_pattern"`
	Remark          *string `json:"remark"`
}

type LogRetentionCleanupResult struct {
	DeletedIndices   []string `json:"deleted_indices"`
	DeletedDocuments int64    `json:"deleted_documents"`
	Message          string   `json:"message"`
}

type ESStorageStats struct {
	IndexPattern  string `json:"index_pattern"`
	IndexCount    int    `json:"index_count"`
	DocumentCount int64  `json:"document_count"`
	StoreBytes    int64  `json:"store_bytes"`
	StoreHuman    string `json:"store_human"`
}

func (s *LogRetentionService) List(ctx context.Context) ([]LogRetentionItem, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logretention", "List", err)
	}
	out := make([]LogRetentionItem, 0, len(list))
	for _, it := range list {
		out = append(out, toRetentionItem(it, retentionSource(it.ProjectID, it.ServerID)))
	}
	return out, nil
}

func (s *LogRetentionService) GetGlobal(ctx context.Context) (LogRetentionItem, error) {
	return s.getByScope(ctx, globalRetentionProjectID, 0)
}

func (s *LogRetentionService) UpsertGlobal(ctx context.Context, req LogRetentionUpsertRequest) (LogRetentionItem, error) {
	return s.upsert(ctx, globalRetentionProjectID, 0, req)
}

func (s *LogRetentionService) GetProject(ctx context.Context, projectID uint) (LogRetentionItem, error) {
	if projectID == 0 {
		return LogRetentionItem{}, constants.ErrProjectIDRequired
	}
	it, err := s.repo.GetByScope(ctx, projectID, 0)
	if err == nil {
		return toRetentionItem(*it, "project"), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return LogRetentionItem{}, bizerrors.Pass(ctx, "logretention", "GetProject", err)
	}
	global, err := s.GetGlobal(ctx)
	if err == nil {
		global.Source = "global"
		return global, nil
	}
	return s.defaultRetentionItem(ctx), nil
}

func (s *LogRetentionService) UpsertProject(ctx context.Context, projectID uint, req LogRetentionUpsertRequest) (LogRetentionItem, error) {
	if projectID == 0 {
		return LogRetentionItem{}, constants.ErrProjectIDRequired
	}
	serverID := uint(0)
	if req.ServerID != nil {
		serverID = *req.ServerID
	}
	return s.upsert(ctx, projectID, serverID, req)
}

func (s *LogRetentionService) DeleteProjectOverride(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrProjectIDRequired
	}
	return s.repo.DeleteByScope(ctx, projectID, 0)
}

func (s *LogRetentionService) StorageStats(ctx context.Context) (*ESStorageStats, error) {
	if s.es == nil {
		return nil, constants.ErrBadRequestWithMsg("Elasticsearch 未配置")
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	indices, err := cli.CatIndices(ctx, cfg.IndexPattern)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logretention", "StorageStats", err)
	}
	var docs, bytes int64
	for _, idx := range indices {
		docs += idx.DocsCount
		bytes += idx.StoreBytes
	}
	return &ESStorageStats{
		IndexPattern:  cfg.IndexPattern,
		IndexCount:    len(indices),
		DocumentCount: docs,
		StoreBytes:    bytes,
		StoreHuman:    humanBytes(bytes),
	}, nil
}

func (s *LogRetentionService) RunCleanup(ctx context.Context) (*LogRetentionCleanupResult, error) {
	if s.es == nil {
		return &LogRetentionCleanupResult{Message: "Elasticsearch 未配置，跳过清理"}, nil
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return &LogRetentionCleanupResult{Message: err.Error()}, nil
	}
	policies, err := s.repo.List(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logretention", "RunCleanup", err)
	}
	if len(policies) == 0 {
		policies = []model.LogRetentionPolicy{s.defaultPolicyModel(ctx)}
	}
	res := &LogRetentionCleanupResult{}
	now := time.Now().UTC()
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		pattern := strings.TrimSpace(p.IndexPattern)
		if pattern == "" {
			if p.ServerID > 0 {
				pattern = AgentIndexPattern(p.ServerID)
			} else {
				pattern = GlobalAgentIndexPattern()
			}
		}
		part, err := s.cleanupPolicy(ctx, cli, cfg, pattern, p, now)
		if err != nil {
			return nil, err
		}
		res.DeletedIndices = append(res.DeletedIndices, part.DeletedIndices...)
		res.DeletedDocuments += part.DeletedDocuments
	}
	if len(res.DeletedIndices) == 0 && res.DeletedDocuments == 0 {
		res.Message = "无过期日志需要清理"
	} else {
		res.Message = fmt.Sprintf("已删除 %d 个索引、%d 条文档", len(res.DeletedIndices), res.DeletedDocuments)
	}
	return res, nil
}

func (s *LogRetentionService) cleanupPolicy(ctx context.Context, cli *esclient.Client, cfg config.ElasticsearchConfig, pattern string, p model.LogRetentionPolicy, now time.Time) (*LogRetentionCleanupResult, error) {
	retentionDays := p.RetentionDays
	if retentionDays <= 0 {
		retentionDays = cfg.DefaultRetentionDays
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	indices, err := cli.CatIndices(ctx, pattern)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logretention", "cleanupPolicy", err)
	}
	res := &LogRetentionCleanupResult{}
	dated := 0
	for _, idx := range indices {
		dt, ok := esclient.ParseIndexDate(idx.Name)
		if !ok {
			continue
		}
		dated++
		if dt.Before(cutoff) {
			if err := cli.DeleteIndex(ctx, idx.Name); err != nil {
				return nil, bizerrors.Pass(ctx, "logretention", "DeleteIndex", err)
			}
			res.DeletedIndices = append(res.DeletedIndices, idx.Name)
		}
	}
	if p.MaxIndexCount > 0 && len(indices) > p.MaxIndexCount {
		sortable := make([]esclient.IndexInfo, 0, len(indices))
		for _, idx := range indices {
			if dt, ok := esclient.ParseIndexDate(idx.Name); ok {
				_ = dt
				sortable = append(sortable, idx)
			}
		}
		excess := len(sortable) - p.MaxIndexCount
		for i := 0; i < excess && i < len(sortable); i++ {
			name := sortable[i].Name
			if err := cli.DeleteIndex(ctx, name); err != nil {
				return nil, err
			}
			res.DeletedIndices = append(res.DeletedIndices, name)
		}
	}
	if p.MaxStoreBytes > 0 {
		var total int64
		for _, idx := range indices {
			total += idx.StoreBytes
		}
		if total > p.MaxStoreBytes {
			for _, idx := range indices {
				if total <= p.MaxStoreBytes {
					break
				}
				if err := cli.DeleteIndex(ctx, idx.Name); err != nil {
					return nil, err
				}
				total -= idx.StoreBytes
				res.DeletedIndices = append(res.DeletedIndices, idx.Name)
			}
		}
	}
	if dated == 0 && len(indices) > 0 {
		query := map[string]any{
			"range": map[string]any{cfg.TimestampField: map[string]any{"lt": cutoff.Format(time.RFC3339)}},
		}
		if p.ProjectID > 0 {
			query = map[string]any{
				"bool": map[string]any{
					"filter": []map[string]any{
						{"term": map[string]any{"project_id": fmt.Sprintf("%d", p.ProjectID)}},
						query,
					},
				},
			}
		}
		if p.ServerID > 0 {
			query = map[string]any{
				"bool": map[string]any{
					"filter": []map[string]any{
						{"term": map[string]any{"server_id": fmt.Sprintf("%d", p.ServerID)}},
						query,
					},
				},
			}
		}
		deleted, err := cli.DeleteByQuery(ctx, pattern, map[string]any{"query": query})
		if err != nil {
			return nil, bizerrors.Pass(ctx, "logretention", "DeleteByQuery", err)
		}
		res.DeletedDocuments += deleted
	}
	return res, nil
}

func (s *LogRetentionService) getByScope(ctx context.Context, projectID, serverID uint) (LogRetentionItem, error) {
	it, err := s.repo.GetByScope(ctx, projectID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if projectID == globalRetentionProjectID {
				return s.defaultRetentionItem(ctx), nil
			}
			return LogRetentionItem{}, err
		}
		return LogRetentionItem{}, bizerrors.Pass(ctx, "logretention", "getByScope", err)
	}
	return toRetentionItem(*it, retentionSource(it.ProjectID, it.ServerID)), nil
}

func (s *LogRetentionService) upsert(ctx context.Context, projectID, serverID uint, req LogRetentionUpsertRequest) (LogRetentionItem, error) {
	cfg := config.ElasticsearchConfig{DefaultRetentionDays: 30}.Normalized()
	if s.es != nil {
		cfg, _ = s.es.Resolve(ctx)
	}
	days := req.RetentionDays
	if days <= 0 {
		days = cfg.DefaultRetentionDays
	}
	if days > 3650 {
		days = 3650
	}
	it, err := s.repo.GetByScope(ctx, projectID, serverID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return LogRetentionItem{}, bizerrors.Pass(ctx, "logretention", "upsert", err)
		}
		it = &model.LogRetentionPolicy{ProjectID: projectID, ServerID: serverID}
	}
	it.RetentionDays = days
	it.MaxStoreBytes = req.MaxStoreBytes
	it.MaxIndexCount = req.MaxIndexCount
	it.Enabled = req.Enabled
	if req.IndexPattern != nil {
		it.IndexPattern = strings.TrimSpace(*req.IndexPattern)
	}
	it.Remark = req.Remark
	if err := s.repo.Save(ctx, it); err != nil {
		return LogRetentionItem{}, bizerrors.Pass(ctx, "logretention", "upsert", err)
	}
	return toRetentionItem(*it, retentionSource(projectID, serverID)), nil
}

func (s *LogRetentionService) defaultRetentionItem(ctx context.Context) LogRetentionItem {
	if s.es == nil {
		return LogRetentionItem{RetentionDays: 30, Enabled: true, Source: "default"}
	}
	return toRetentionItem(s.defaultPolicyModel(ctx), "default")
}

func (s *LogRetentionService) defaultPolicyModel(ctx context.Context) model.LogRetentionPolicy {
	if s.es == nil {
		return model.LogRetentionPolicy{ProjectID: globalRetentionProjectID, RetentionDays: 30, Enabled: true}
	}
	cfg, _ := s.es.Resolve(ctx)
	return model.LogRetentionPolicy{
		ProjectID:     globalRetentionProjectID,
		RetentionDays: cfg.DefaultRetentionDays,
		Enabled:       true,
		IndexPattern:  cfg.IndexPattern,
	}
}

func toRetentionItem(it model.LogRetentionPolicy, source string) LogRetentionItem {
	return LogRetentionItem{
		ID:              it.ID,
		ProjectID:       it.ProjectID,
		ServerID:        it.ServerID,
		RetentionDays:   it.RetentionDays,
		MaxStoreBytes:   it.MaxStoreBytes,
		MaxIndexCount:   it.MaxIndexCount,
		Enabled:         it.Enabled,
		IndexPattern:    strings.TrimSpace(it.IndexPattern),
		Remark:          it.Remark,
		Source:          source,
		UpdatedAt:       it.UpdatedAt.Format(time.RFC3339),
	}
}

func retentionSource(projectID, serverID uint) string {
	if projectID == 0 {
		return "global"
	}
	if serverID > 0 {
		return "server"
	}
	return "project"
}

func projectSource(projectID uint) string {
	return retentionSource(projectID, 0)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	exp := 0
	val := float64(n)
	for val >= unit && exp < 4 {
		val /= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp-1]
	return fmt.Sprintf("%.1f %s", val, suffix)
}
