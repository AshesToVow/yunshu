package esmgmt

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"
)

const defaultEsmgmtBackupInnerTick = "*/30 * * * * *"

// ScheduleUpsertRequest 创建/更新定时备份。
type ScheduleUpsertRequest struct {
	ConnectionID uint   `json:"connection_id"`
	IndexName    string `json:"index_name"`
	Enabled      *bool  `json:"enabled"`
	CronSpec     string `json:"cron_spec"`
	MaxDocs      *int   `json:"max_docs"`
	Remark       string `json:"remark"`
	ClearRemark  bool   `json:"clear_remark"`
}

// ValidateEsmgmtBackupCronSpec 校验 Cron。
func ValidateEsmgmtBackupCronSpec(spec string) error {
	return cronutil.ValidateSpec(spec, "cron_spec")
}

// CreateSchedule 新建定时备份规则。
func (s *Service) CreateSchedule(ctx context.Context, req ScheduleUpsertRequest, actor *auth.CurrentUser) (*model.EsmgmtBackupSchedule, error) {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return nil, err
	}
	index := strings.TrimSpace(req.IndexName)
	cronSpec := strings.TrimSpace(req.CronSpec)
	if index == "" {
		return nil, constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	if strings.HasPrefix(index, ".") {
		return nil, constants.ErrBadRequestWithMsg("禁止调度系统索引")
	}
	if cronSpec == "" {
		return nil, constants.ErrBadRequestWithMsg("cron_spec 不能为空")
	}
	if err := ValidateEsmgmtBackupCronSpec(cronSpec); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.EsmgmtBackupSchedule{
		ConnectionID: req.ConnectionID,
		IndexName:    index,
		Enabled:      enabled,
		CronSpec:     cronSpec,
		Remark:       strings.TrimSpace(req.Remark),
		CreatedBy:    actorID(actor),
	}
	if req.MaxDocs != nil {
		row.MaxDocs = *req.MaxDocs
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateSchedule 更新定时备份规则。
func (s *Service) UpdateSchedule(ctx context.Context, id uint, req ScheduleUpsertRequest, actor *auth.CurrentUser) (*model.EsmgmtBackupSchedule, error) {
	if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("调度 ID 无效")
	}
	var row model.EsmgmtBackupSchedule
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	if req.ConnectionID > 0 {
		row.ConnectionID = req.ConnectionID
	}
	if name := strings.TrimSpace(req.IndexName); name != "" {
		if strings.HasPrefix(name, ".") {
			return nil, constants.ErrBadRequestWithMsg("禁止调度系统索引")
		}
		row.IndexName = name
	}
	if spec := strings.TrimSpace(req.CronSpec); spec != "" {
		if err := ValidateEsmgmtBackupCronSpec(spec); err != nil {
			return nil, err
		}
		row.CronSpec = spec
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.MaxDocs != nil {
		row.MaxDocs = *req.MaxDocs
	}
	if req.ClearRemark || strings.TrimSpace(req.Remark) != "" {
		row.Remark = strings.TrimSpace(req.Remark)
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) DeleteSchedule(ctx context.Context, id uint) error {
	if id == 0 {
		return constants.ErrBadRequestWithMsg("调度 ID 无效")
	}
	res := s.db.WithContext(ctx).Delete(&model.EsmgmtBackupSchedule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) ListSchedules(ctx context.Context, connectionID uint) ([]model.EsmgmtBackupSchedule, error) {
	q := s.db.WithContext(ctx).Order("id desc")
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtBackupSchedule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// RunBackupScheduler 启动定时备份 Worker（字典 esmgmt_backup_scheduler_*）。
func (s *Service) RunBackupScheduler(ctx context.Context) {
	if s == nil || s.resolveSched == nil {
		return
	}
	cfg := s.resolveSched(ctx)
	if !cfg.Enabled {
		slog.Info("esmgmt backup scheduler disabled by dict")
		return
	}
	spec := strings.TrimSpace(cfg.TickSpec)
	if spec == "" {
		spec = defaultEsmgmtBackupInnerTick
	}
	slog.Info("started esmgmt backup scheduler", "tick_spec", spec)
	cronutil.RunWorker(ctx, spec, func() {
		if err := s.tickScheduledBackups(ctx); err != nil {
			slog.Warn("esmgmt backup scheduler tick failed", "error", err)
		}
	}, defaultEsmgmtBackupInnerTick)
}

func (s *Service) tickScheduledBackups(ctx context.Context) error {
	var list []model.EsmgmtBackupSchedule
	if err := s.db.WithContext(ctx).
		Where("enabled = ? AND cron_spec <> ?", true, "").
		Find(&list).Error; err != nil {
		return err
	}
	now := time.Now()
	for i := range list {
		sch := &list[i]
		if !cronutil.ShouldRunWithDayAnchor(sch.CronSpec, sch.LastScheduledAt, now) {
			continue
		}
		var running int64
		_ = s.db.WithContext(ctx).Model(&model.EsmgmtBackupJob{}).
			Where("connection_id = ? AND index_name = ? AND status IN ?", sch.ConnectionID, sch.IndexName, []string{"pending", "running"}).
			Count(&running).Error
		if running > 0 {
			continue
		}
		s.runScheduled(ctx, sch, now)
	}
	return nil
}

func (s *Service) runScheduled(ctx context.Context, sch *model.EsmgmtBackupSchedule, now time.Time) {
	s.schedMu.Lock()
	if s.schedRunning[sch.ID] {
		s.schedMu.Unlock()
		return
	}
	s.schedRunning[sch.ID] = true
	s.schedMu.Unlock()

	defer func() {
		s.schedMu.Lock()
		delete(s.schedRunning, sch.ID)
		s.schedMu.Unlock()
	}()

	_ = s.db.WithContext(ctx).Model(&model.EsmgmtBackupSchedule{}).
		Where("id = ?", sch.ID).
		Update("last_scheduled_at", now).Error

	_, _ = s.enqueueBackup(ctx, BackupIndexRequest{
		ConnectionID: sch.ConnectionID,
		Index:        sch.IndexName,
		MaxDocs:      sch.MaxDocs,
	}, BackupTriggerScheduled, sch.CreatedBy)
}
