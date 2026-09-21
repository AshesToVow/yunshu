package esmgmt

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// ReindexRequest 提交 reindex。
type ReindexRequest struct {
	ConnectionID uint            `json:"connection_id"`
	SourceIndex  string          `json:"source_index"`
	DestIndex    string          `json:"dest_index"`
	Query        json.RawMessage `json:"query"`
}

func (s *Service) CreateReindex(ctx context.Context, req ReindexRequest, actor *auth.CurrentUser) (*model.EsmgmtReindexJob, error) {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return nil, err
	}
	source, err := validateIndexRef(req.SourceIndex, true)
	if err != nil {
		return nil, err
	}
	dest, err := validateIndexRef(req.DestIndex, false)
	if err != nil {
		return nil, err
	}
	if source == dest {
		return nil, constants.ErrBadRequestWithMsg("源索引与目标索引不能相同")
	}
	if err := guardPlatformIndex(dest, actor); err != nil {
		return nil, err
	}
	var query json.RawMessage
	if q := bytesTrim(req.Query); len(q) > 0 && string(q) != "null" {
		if !json.Valid(q) || q[0] != '{' {
			return nil, constants.ErrBadRequestWithMsg("query 须为 JSON 对象")
		}
		if strings.Contains(strings.ToLower(string(q)), "painless") {
			return nil, constants.ErrBadRequestWithMsg("不允许脚本查询")
		}
		query = q
	}
	connID := req.ConnectionID
	if connID == 0 {
		def, err := s.repo.GetDefaultConnection(ctx)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("请指定 connection_id")
		}
		connID = def.ID
	}
	n, err := s.repo.CountRunningReindexJobs(ctx, connID, source, dest)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, constants.ErrBadRequestWithMsg("相同源/目标已有进行中的 reindex")
	}
	body := map[string]any{
		"source": map[string]any{"index": source},
		"dest":   map[string]any{"index": dest},
	}
	if len(query) > 0 {
		var q any
		if err := json.Unmarshal(query, &q); err != nil {
			return nil, constants.ErrBadRequestWithMsg("query 无法解析")
		}
		body["source"].(map[string]any)["query"] = q
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	cli, err := s.resolveClient(ctx, connID)
	if err != nil {
		return nil, err
	}
	taskID, err := cli.StartReindex(ctx, payload)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("提交 reindex 失败: " + err.Error())
	}
	if !safeTaskID(taskID) {
		return nil, constants.ErrBadRequestWithMsg("ES 返回的 task id 非法")
	}
	job := &model.EsmgmtReindexJob{
		ConnectionID: connID,
		SourceIndex:  source,
		DestIndex:    dest,
		TaskID:       taskID,
		Status:       "running",
		Phase:        "submitted",
		CreatedBy:    actorID(actor),
	}
	if err := s.repo.CreateReindexJob(ctx, job); err != nil {
		return nil, err
	}
	go s.watchReindex(job.ID)
	return job, nil
}

func (s *Service) ListReindexJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtReindexJob, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.repo.ListReindexJobs(ctx, connectionID, limit)
}

func (s *Service) GetReindexJob(ctx context.Context, id uint) (*model.EsmgmtReindexJob, error) {
	if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("任务 ID 无效")
	}
	job, err := s.repo.GetReindexJob(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	return job, nil
}

func (s *Service) CancelReindex(ctx context.Context, id uint, actor *auth.CurrentUser) error {
	job, err := s.GetReindexJob(ctx, id)
	if err != nil {
		return err
	}
	if err := s.assertConnectionWrite(ctx, job.ConnectionID, actor); err != nil {
		return err
	}
	if job.Status != "pending" && job.Status != "running" {
		return constants.ErrBadRequestWithMsg("任务已结束")
	}
	if job.TaskID != "" && safeTaskID(job.TaskID) {
		cli, err := s.resolveClient(ctx, job.ConnectionID)
		if err != nil {
			return err
		}
		if err := cli.CancelTask(ctx, job.TaskID); err != nil {
			return constants.ErrBadRequestWithMsg("取消失败: " + err.Error())
		}
	}
	return s.repo.UpdateReindexJobFields(ctx, id, map[string]any{
		"status": "cancelled",
		"phase":  "cancelled",
	})
}

func (s *Service) watchReindex(jobID uint) {
	defer func() {
		if r := recover(); r != nil {
			_ = s.repo.UpdateReindexJobFields(context.Background(), jobID, map[string]any{
				"status": "failed", "phase": "panic", "error_message": "job panic",
			})
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = s.repo.UpdateReindexJobFields(context.Background(), jobID, map[string]any{
				"status": "failed", "phase": "timeout", "error_message": "轮询超时",
			})
			return
		case <-ticker.C:
			if done := s.pollReindex(ctx, jobID); done {
				return
			}
		}
	}
}

func (s *Service) pollReindex(ctx context.Context, jobID uint) bool {
	job, err := s.repo.GetReindexJob(ctx, jobID)
	if err != nil {
		return true
	}
	if job.Status == "cancelled" || job.Status == "success" || job.Status == "failed" {
		return true
	}
	if job.TaskID == "" || !safeTaskID(job.TaskID) {
		_ = s.repo.UpdateReindexJobFields(ctx, jobID, map[string]any{
			"status": "failed", "phase": "invalid_task", "error_message": "task id 无效",
		})
		return true
	}
	cli, err := s.resolveClient(ctx, job.ConnectionID)
	if err != nil {
		return false
	}
	raw, err := cli.GetTask(ctx, job.TaskID)
	if err != nil {
		return false
	}
	var wrap struct {
		Completed bool `json:"completed"`
		Error     any  `json:"error"`
		Task      struct {
			Status struct {
				Total   int `json:"total"`
				Created int `json:"created"`
			} `json:"status"`
		} `json:"task"`
		Response struct {
			Total   int `json:"total"`
			Created int `json:"created"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return false
	}
	total := wrap.Task.Status.Total
	created := wrap.Task.Status.Created
	if wrap.Completed {
		total = wrap.Response.Total
		created = wrap.Response.Created
	}
	fields := map[string]any{
		"phase":        "running",
		"total":        total,
		"created_docs": created,
	}
	if wrap.Completed {
		if wrap.Error != nil {
			msg, _ := json.Marshal(wrap.Error)
			fields["status"] = "failed"
			fields["phase"] = "error"
			fields["error_message"] = truncateMsg(string(msg))
		} else {
			fields["status"] = "success"
			fields["phase"] = "done"
			fields["error_message"] = ""
		}
		_ = s.repo.UpdateReindexJobFields(ctx, jobID, fields)
		return true
	}
	fields["status"] = "running"
	_ = s.repo.UpdateReindexJobFields(ctx, jobID, fields)
	return false
}

func safeTaskID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 || strings.Contains(id, "/") {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == ':':
		default:
			return false
		}
	}
	return true
}

func truncateMsg(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}
