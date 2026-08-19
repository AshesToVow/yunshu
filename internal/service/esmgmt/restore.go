package esmgmt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/esclient"
)

// RestoreIndexRequest 从备份恢复。
type RestoreIndexRequest struct {
	BackupJobID    uint   `json:"backup_job_id"`
	ConnectionID   uint   `json:"connection_id"` // 0 则用备份任务的连接
	TargetIndex         string `json:"target_index"`  // 空则用源索引名
	DeleteExisting      bool   `json:"delete_existing"`
	ConfirmTargetIndex  string `json:"confirm_target_index"`
}

// CreateIndexRestore 创建恢复任务：分词/settings → mapping → 数据。
func (s *Service) CreateIndexRestore(ctx context.Context, req RestoreIndexRequest, actor *auth.CurrentUser) (*model.EsmgmtRestoreJob, error) {
	if err := s.assertDestructiveRestore(ctx, req, actor); err != nil {
		return nil, err
	}
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil && req.ConnectionID != 0 {
		return nil, err
	}
	if req.BackupJobID == 0 {
		return nil, constants.ErrBadRequestWithMsg("backup_job_id 无效")
	}
	backup, err := s.GetBackupJob(ctx, req.BackupJobID)
	if err != nil {
		return nil, err
	}
	if backup.Status != "success" {
		return nil, constants.ErrBadRequestWithMsg("仅可恢复成功的备份任务")
	}
	if strings.TrimSpace(backup.AnalysisObject) == "" ||
		strings.TrimSpace(backup.MappingObject) == "" ||
		strings.TrimSpace(backup.DataObject) == "" {
		return nil, constants.ErrBadRequestWithMsg("备份产物不完整")
	}
	target := strings.TrimSpace(req.TargetIndex)
	if target == "" {
		target = backup.IndexName
	}
	if strings.HasPrefix(target, ".") {
		return nil, constants.ErrBadRequestWithMsg("禁止恢复到系统索引")
	}
	if strings.HasPrefix(target, "yunshu-") && !isSuperAdmin(actor) {
		return nil, constants.ErrForbiddenWithMsg("恢复到 yunshu-* 索引须超级管理员")
	}
	connID := req.ConnectionID
	if connID == 0 {
		connID = backup.ConnectionID
	}
	if err := s.assertConnectionWrite(ctx, connID, actor); err != nil {
		return nil, err
	}
	if s.newObjectStore == nil {
		return nil, constants.ErrBadRequestWithMsg("对象存储未配置")
	}
	if _, err := s.newObjectStore(ctx); err != nil {
		return nil, constants.ErrBadRequestWithMsg("MinIO 不可用: " + err.Error())
	}
	job := &model.EsmgmtRestoreJob{
		BackupJobID:    backup.ID,
		ConnectionID:   connID,
		SourceIndex:    backup.IndexName,
		TargetIndex:    target,
		DeleteExisting: req.DeleteExisting,
		Status:         "pending",
		Phase:          "queued",
		CreatedBy:      actorID(actor),
	}
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}
	go s.runRestoreJob(job.ID)
	return job, nil
}

func (s *Service) ListRestoreJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtRestoreJob, error) {
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
	var list []model.EsmgmtRestoreJob
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Service) GetRestoreJob(ctx context.Context, id uint) (*model.EsmgmtRestoreJob, error) {
	if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("任务 ID 无效")
	}
	var job model.EsmgmtRestoreJob
	if err := s.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Service) runRestoreJob(jobID uint) {
	defer func() {
		if r := recover(); r != nil {
			_ = s.db.WithContext(context.Background()).Model(&model.EsmgmtRestoreJob{}).
				Where("id = ?", jobID).
				Updates(map[string]any{"status": "failed", "phase": "panic", "error_message": "job panic"}).Error
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	var job model.EsmgmtRestoreJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return
	}
	_ = s.db.WithContext(ctx).Model(&job).Updates(map[string]any{
		"status": "running",
		"phase":  "download",
	}).Error

	fail := func(phase string, err error) {
		msg := err.Error()
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		_ = s.db.WithContext(context.Background()).Model(&model.EsmgmtRestoreJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":        "failed",
				"phase":         phase,
				"error_message": msg,
			}).Error
	}

	var backup model.EsmgmtBackupJob
	if err := s.db.WithContext(ctx).First(&backup, job.BackupJobID).Error; err != nil {
		fail("download", err)
		return
	}
	store, err := s.newObjectStore(ctx)
	if err != nil {
		fail("download", err)
		return
	}
	analysisBytes, err := store.GetBytes(ctx, store.RelativeKey(backup.AnalysisObject))
	if err != nil {
		fail("download", fmt.Errorf("analysis: %w", err))
		return
	}
	mappingBytes, err := store.GetBytes(ctx, store.RelativeKey(backup.MappingObject))
	if err != nil {
		fail("download", fmt.Errorf("mapping: %w", err))
		return
	}
	dataBytes, err := store.GetBytes(ctx, store.RelativeKey(backup.DataObject))
	if err != nil {
		fail("download", fmt.Errorf("data: %w", err))
		return
	}

	cli, err := s.resolveClient(ctx, job.ConnectionID)
	if err != nil {
		fail("resolve", err)
		return
	}

	_ = s.db.WithContext(ctx).Model(&job).Update("phase", "create_index").Error
	exists, err := cli.IndexExists(ctx, job.TargetIndex)
	if err != nil {
		fail("create_index", err)
		return
	}
	if exists {
		if !job.DeleteExisting {
			fail("create_index", fmt.Errorf("目标索引已存在，请传 delete_existing=true"))
			return
		}
		if err := cli.DeleteIndex(ctx, job.TargetIndex); err != nil {
			fail("create_index", err)
			return
		}
	}

	createBody, err := buildRestoreCreateBody(analysisBytes, mappingBytes, job.SourceIndex)
	if err != nil {
		fail("create_index", err)
		return
	}
	if err := cli.CreateIndex(ctx, job.TargetIndex, createBody); err != nil {
		fail("create_index", err)
		return
	}

	_ = s.db.WithContext(ctx).Model(&job).Update("phase", "data").Error
	docCount, err := bulkRestoreDocs(ctx, cli, job.TargetIndex, dataBytes)
	if err != nil {
		fail("data", err)
		return
	}

	_ = s.db.WithContext(context.Background()).Model(&model.EsmgmtRestoreJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        "success",
			"phase":         "done",
			"doc_count":     docCount,
			"error_message": "",
		}).Error
}

func buildRestoreCreateBody(analysisBytes, mappingBytes []byte, sourceIndex string) (map[string]any, error) {
	var analysisRoot map[string]any
	if err := json.Unmarshal(analysisBytes, &analysisRoot); err != nil {
		return nil, fmt.Errorf("parse analysis: %w", err)
	}
	var mappingRoot map[string]any
	if err := json.Unmarshal(mappingBytes, &mappingRoot); err != nil {
		return nil, fmt.Errorf("parse mapping: %w", err)
	}

	settings := map[string]any{}
	if idxSettings, ok := analysisRoot["index_settings"].(map[string]any); ok {
		settings = sanitizeIndexSettings(idxSettings)
	} else if a, ok := analysisRoot["analysis"].(map[string]any); ok {
		settings = map[string]any{"index": a}
	}

	mappings := extractMappings(mappingRoot, sourceIndex)
	body := map[string]any{}
	if len(settings) > 0 {
		body["settings"] = settings
	}
	if mappings != nil {
		body["mappings"] = mappings
	}
	return body, nil
}

func extractMappings(mappingRoot map[string]any, sourceIndex string) any {
	if mappingRoot == nil {
		return nil
	}
	pick := func(m map[string]any) any {
		if mm, ok := m["mappings"]; ok {
			return mm
		}
		return m
	}
	if v, ok := mappingRoot[sourceIndex].(map[string]any); ok {
		return pick(v)
	}
	for _, v := range mappingRoot {
		if m, ok := v.(map[string]any); ok {
			return pick(m)
		}
	}
	if mm, ok := mappingRoot["mappings"]; ok {
		return mm
	}
	return mappingRoot
}

var readonlySettingKeys = map[string]struct{}{
	"uuid": {}, "creation_date": {}, "version": {}, "provided_name": {},
	"routing": {}, "resize": {}, "blocks": {},
}

func sanitizeIndexSettings(settings map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range settings {
		if k == "index" {
			if idx, ok := v.(map[string]any); ok {
				clean := map[string]any{}
				for ik, iv := range idx {
					if _, skip := readonlySettingKeys[ik]; skip {
						continue
					}
					if strings.HasPrefix(ik, "routing.") || strings.HasPrefix(ik, "version.") {
						continue
					}
					clean[ik] = iv
				}
				if len(clean) > 0 {
					out["index"] = clean
				}
			}
			continue
		}
		out[k] = v
	}
	return out
}

func bulkRestoreDocs(ctx context.Context, cli *esclient.Client, index string, data []byte) (int, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return 0, nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	// 单行文档可能较大
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 8*1024*1024)

	var batch bytes.Buffer
	count := 0
	flush := func() error {
		if batch.Len() == 0 {
			return nil
		}
		res, err := cli.Bulk(ctx, batch.Bytes())
		batch.Reset()
		if err != nil {
			return err
		}
		if res != nil && res.Failed > 0 {
			return fmt.Errorf("bulk partial failure: failed=%d first=%s", res.Failed, res.FirstError)
		}
		return nil
	}

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var doc struct {
			ID     string         `json:"_id"`
			Source map[string]any `json:"_source"`
		}
		if err := json.Unmarshal(line, &doc); err != nil {
			return count, fmt.Errorf("parse ndjson: %w", err)
		}
		meta := map[string]any{"index": map[string]any{"_index": index}}
		if id := strings.TrimSpace(doc.ID); id != "" {
			meta["index"].(map[string]any)["_id"] = id
		}
		metaLine, _ := json.Marshal(meta)
		srcLine, err := json.Marshal(doc.Source)
		if err != nil {
			return count, err
		}
		batch.Write(metaLine)
		batch.WriteByte('\n')
		batch.Write(srcLine)
		batch.WriteByte('\n')
		count++
		if count%500 == 0 {
			if err := flush(); err != nil {
				return count, err
			}
		}
	}
	if err := sc.Err(); err != nil {
		return count, err
	}
	if err := flush(); err != nil {
		return count, err
	}
	return count, nil
}
