package esmgmt

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/objectstore"
)

// ObjectStoreFactory 从字典解析 MinIO 客户端。
type ObjectStoreFactory func(ctx context.Context) (*objectstore.Client, error)

const (
	BackupTriggerManual    = "manual"
	BackupTriggerScheduled = "scheduled"
)

// BackupIndexRequest 触发索引备份。
type BackupIndexRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Index        string `json:"index"`
	MaxDocs      int    `json:"max_docs"`
}

// CreateIndexBackup 创建备份任务并异步执行：分词(settings.analysis) → mapping → 数据 → MinIO。
func (s *Service) CreateIndexBackup(ctx context.Context, req BackupIndexRequest, createdBy uint) (*model.EsmgmtBackupJob, error) {
	return s.enqueueBackup(ctx, req, BackupTriggerManual, createdBy)
}

func (s *Service) enqueueBackup(ctx context.Context, req BackupIndexRequest, trigger string, createdBy uint) (*model.EsmgmtBackupJob, error) {
	index := strings.TrimSpace(req.Index)
	if index == "" {
		return nil, constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	if strings.HasPrefix(index, ".") {
		return nil, constants.ErrBadRequestWithMsg("禁止备份系统索引")
	}
	if s.newObjectStore == nil {
		return nil, constants.ErrBadRequestWithMsg("对象存储未配置")
	}
	if _, err := s.newObjectStore(ctx); err != nil {
		return nil, constants.ErrBadRequestWithMsg("MinIO 不可用: " + err.Error())
	}
	if trigger == "" {
		trigger = BackupTriggerManual
	}
	job := &model.EsmgmtBackupJob{
		ConnectionID: req.ConnectionID,
		IndexName:    index,
		Trigger:      trigger,
		Status:       "pending",
		Phase:        "queued",
		CreatedBy:    createdBy,
	}
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}
	go s.runBackupJob(job.ID, req.MaxDocs)
	return job, nil
}

func (s *Service) ListBackupJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtBackupJob, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := s.db.WithContext(ctx).Order("id desc").Limit(limit)
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtBackupJob
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Service) GetBackupJob(ctx context.Context, id uint) (*model.EsmgmtBackupJob, error) {
	if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("任务 ID 无效")
	}
	var job model.EsmgmtBackupJob
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// BackupDownloadResult 预签名下载。
type BackupDownloadResult struct {
	URL       string `json:"url"`
	Artifact  string `json:"artifact"`
	ObjectKey string `json:"object_key"`
	ExpiresIn int    `json:"expires_in_sec"`
}

// PresignBackupDownload 生成备份产物临时下载链接。
func (s *Service) PresignBackupDownload(ctx context.Context, jobID uint, artifact string) (*BackupDownloadResult, error) {
	job, err := s.GetBackupJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "success" {
		return nil, constants.ErrBadRequestWithMsg("任务未成功完成，无法下载")
	}
	artifact = strings.ToLower(strings.TrimSpace(artifact))
	if artifact == "" {
		artifact = "zip"
	}
	var stored string
	switch artifact {
	case "zip":
		stored = job.MinioObject
	case "analysis":
		stored = job.AnalysisObject
	case "mapping":
		stored = job.MappingObject
	case "data":
		stored = job.DataObject
	default:
		return nil, constants.ErrBadRequestWithMsg("artifact 须为 zip|analysis|mapping|data")
	}
	if strings.TrimSpace(stored) == "" {
		return nil, constants.ErrBadRequestWithMsg("该产物对象键为空")
	}
	store, err := s.newObjectStore(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("MinIO 不可用: " + err.Error())
	}
	rel := store.RelativeKey(stored)
	const expiry = 15 * time.Minute
	url, err := store.PresignedGetURL(ctx, rel, expiry)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("生成下载链接失败: " + err.Error())
	}
	return &BackupDownloadResult{
		URL:       url,
		Artifact:  artifact,
		ObjectKey: store.FullKey(rel),
		ExpiresIn: int(expiry.Seconds()),
	}, nil
}

func (s *Service) runBackupJob(jobID uint, maxDocs int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var job model.EsmgmtBackupJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return
	}
	_ = s.db.WithContext(ctx).Model(&job).Updates(map[string]any{
		"status": "running",
		"phase":  "analysis",
	}).Error

	fail := func(phase string, err error) {
		msg := err.Error()
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		_ = s.db.WithContext(context.Background()).Model(&model.EsmgmtBackupJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":        "failed",
				"phase":         phase,
				"error_message": msg,
			}).Error
	}

	cli, err := s.resolveClient(ctx, job.ConnectionID)
	if err != nil {
		fail("resolve", err)
		return
	}

	settingsRaw, err := cli.GetIndexSettings(ctx, job.IndexName)
	if err != nil {
		fail("analysis", err)
		return
	}
	analysisPayload := extractAnalysis(settingsRaw, job.IndexName)
	analysisBytes, err := json.MarshalIndent(analysisPayload, "", "  ")
	if err != nil {
		fail("analysis", err)
		return
	}

	_ = s.db.WithContext(ctx).Model(&job).Update("phase", "mapping").Error
	mappingRaw, err := cli.GetIndexMapping(ctx, job.IndexName)
	if err != nil {
		fail("mapping", err)
		return
	}
	mappingBytes, err := json.MarshalIndent(mappingRaw, "", "  ")
	if err != nil {
		fail("mapping", err)
		return
	}

	_ = s.db.WithContext(ctx).Model(&job).Update("phase", "data").Error
	hits, err := cli.ScrollAll(ctx, job.IndexName, maxDocs)
	if err != nil {
		fail("data", err)
		return
	}
	var dataBuf bytes.Buffer
	for _, h := range hits {
		line, mErr := json.Marshal(map[string]any{
			"_id":     h.ID,
			"_source": h.Source,
		})
		if mErr != nil {
			fail("data", mErr)
			return
		}
		dataBuf.Write(line)
		dataBuf.WriteByte('\n')
	}

	_ = s.db.WithContext(ctx).Model(&job).Update("phase", "upload").Error
	store, err := s.newObjectStore(ctx)
	if err != nil {
		fail("upload", err)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	base := fmt.Sprintf("esmgmt-backups/%s/%s", sanitizeObjectName(job.IndexName), ts)
	analysisKey := base + "/01-analysis.json"
	mappingKey := base + "/02-mapping.json"
	dataKey := base + "/03-data.ndjson"
	manifestKey := base + "/manifest.json"
	zipKey := base + "/backup.zip"

	manifest := map[string]any{
		"index":         job.IndexName,
		"connection_id": job.ConnectionID,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"doc_count":     len(hits),
		"order":         []string{"01-analysis.json", "02-mapping.json", "03-data.ndjson"},
		"note":          "elasticdump-like: analysis/settings first, then mapping, then documents",
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")

	if err := store.PutBytes(ctx, analysisKey, analysisBytes, "application/json"); err != nil {
		fail("upload", err)
		return
	}
	if err := store.PutBytes(ctx, mappingKey, mappingBytes, "application/json"); err != nil {
		fail("upload", err)
		return
	}
	if err := store.PutBytes(ctx, dataKey, dataBuf.Bytes(), "application/x-ndjson"); err != nil {
		fail("upload", err)
		return
	}
	if err := store.PutBytes(ctx, manifestKey, manifestBytes, "application/json"); err != nil {
		fail("upload", err)
		return
	}

	zipBytes, err := buildBackupZip(analysisBytes, mappingBytes, dataBuf.Bytes(), manifestBytes)
	if err != nil {
		fail("upload", err)
		return
	}
	if err := store.PutBytes(ctx, zipKey, zipBytes, "application/zip"); err != nil {
		fail("upload", err)
		return
	}

	_ = s.db.WithContext(context.Background()).Model(&model.EsmgmtBackupJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":          "success",
			"phase":           "done",
			"doc_count":       len(hits),
			"minio_bucket":    store.Bucket(),
			"minio_object":    zipKey,
			"analysis_object": analysisKey,
			"mapping_object":  mappingKey,
			"data_object":     dataKey,
			"error_message":   "",
		}).Error
}

func extractAnalysis(settingsRoot map[string]any, indexName string) map[string]any {
	out := map[string]any{
		"index": indexName,
	}
	if settingsRoot == nil {
		return out
	}
	var idxObj map[string]any
	if v, ok := settingsRoot[indexName].(map[string]any); ok {
		idxObj = v
	} else {
		for _, v := range settingsRoot {
			if m, ok := v.(map[string]any); ok {
				idxObj = m
				break
			}
		}
	}
	if idxObj == nil {
		out["settings"] = settingsRoot
		return out
	}
	settings, _ := idxObj["settings"].(map[string]any)
	if settings == nil {
		out["settings"] = idxObj
		return out
	}
	indexSettings, _ := settings["index"].(map[string]any)
	analysis := map[string]any{}
	if indexSettings != nil {
		if a, ok := indexSettings["analysis"]; ok {
			analysis = map[string]any{"analysis": a}
		}
	}
	out["analysis"] = analysis
	out["index_settings"] = settings
	return out
}

func buildBackupZip(analysis, mapping, data, manifest []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := []struct {
		name string
		body []byte
	}{
		{"01-analysis.json", analysis},
		{"02-mapping.json", mapping},
		{"03-data.ndjson", data},
		{"manifest.json", manifest},
	}
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(f.body); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sanitizeObjectName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "..", "_")
	out := replacer.Replace(name)
	if out == "" {
		return "index"
	}
	return out
}
