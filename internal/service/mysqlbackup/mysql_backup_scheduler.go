package mysqlbackup

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/cronutil"
)

const defaultMysqlBackupInnerTick = "*/30 * * * * *"

// ValidateMysqlBackupCronSpec 校验实例 Cron 表达式。
func ValidateMysqlBackupCronSpec(spec string) error {
	return cronutil.ValidateSpec(spec, "cron_spec")
}

// RunMysqlBackupScheduler 启动定时备份 Worker（字典 mysql_backup_scheduler_* 控制开关与节拍）。
func (s *MysqlBackupService) RunMysqlBackupScheduler(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	log := mysqlBackupLog()
	cfg := dictconfig.ResolveMysqlBackupSchedulerConfig(ctx, s.db, dictconfig.DefaultMysqlBackupSchedulerDictTypes())
	if !cfg.Enabled {
		log.Infow("MySQL backup scheduler disabled by dict")
		return
	}
	spec := strings.TrimSpace(cfg.TickSpec)
	if spec == "" {
		spec = defaultMysqlBackupInnerTick
	}
	log.Infow("Started MySQL backup scheduler", "tick_spec", spec)
	cronutil.RunWorker(ctx, spec, func() {
		if err := s.tickScheduledBackups(ctx); err != nil {
			log.Warnw("MySQL backup scheduler tick failed", "error", err)
		}
	}, "")
}

func (s *MysqlBackupService) tickScheduledBackups(ctx context.Context) error {
	if s.aead == nil {
		return nil
	}
	list, err := s.backupRepo.ListScheduleEnabledInstances(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range list {
		inst := &list[i]
		cronSpec := strings.TrimSpace(inst.CronSpec)
		if cronSpec == "" {
			continue
		}
		if !cronutil.ShouldRunWithDayAnchor(cronSpec, inst.LastScheduledAt, now) {
			continue
		}
		running, err := s.backupRepo.HasRunningJob(ctx, inst.ID)
		if err != nil {
			continue
		}
		if running {
			continue
		}
		s.runScheduledInstance(ctx, inst, now)
	}
	return nil
}

func (s *MysqlBackupService) runScheduledInstance(ctx context.Context, inst *model.MysqlBackupInstance, now time.Time) {
	s.schedMu.Lock()
	if s.schedRunning[inst.ID] {
		s.schedMu.Unlock()
		return
	}
	s.schedRunning[inst.ID] = true
	s.schedMu.Unlock()

	defer func() {
		s.schedMu.Lock()
		delete(s.schedRunning, inst.ID)
		s.schedMu.Unlock()
	}()

	_ = s.backupRepo.TouchLastScheduledAt(ctx, inst.ID, now)
	_, _ = s.enqueueBackup(ctx, inst.ProjectID, inst.ID, model.MysqlBackupTriggerScheduled)
}
