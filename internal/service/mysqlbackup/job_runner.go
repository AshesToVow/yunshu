// 备份任务编排：入队、异步执行、收尾、任务列表与停止/删除，含远端进程尽力 kill。
// 两条执行链路本体见 runner_mysqldump.go / runner_xtrabackup.go。
package mysqlbackup

import (
	"context"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/pkg/mysqlbackup"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

func (s *MysqlBackupService) RunBackup(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupJob, error) {
	return s.enqueueBackup(ctx, projectID, instanceID, model.MysqlBackupTriggerManual)
}

func (s *MysqlBackupService) enqueueBackup(ctx context.Context, projectID, instanceID uint, trigger string) (*model.MysqlBackupJob, error) {
	n, _ := s.backupRepo.FailStaleRunningJobs(ctx, 2*time.Hour)
	if n > 0 {
		mysqlBackupLog().Warn("Marked stale MySQL backup jobs as failed", "count", n)
	}
	inst, _, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if !inst.Enabled {
		return nil, constants.ErrBadRequestWithMsg("备份实例已停用")
	}
	running, err := s.backupRepo.HasRunningJob(ctx, instanceID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "mysql.backup", "enqueueBackup", err)
	}
	if running {
		return nil, constants.ErrBadRequestWithMsg("该实例已有进行中的备份任务")
	}

	target := mysqlbackup.BuildDumpTarget(inst)
	now := time.Now()
	job := &model.MysqlBackupJob{
		InstanceID:   inst.ID,
		ProjectID:    projectID,
		Status:       "running",
		BackupMode:   inst.BackupMode,
		TriggerType:  trigger,
		BackupScope:  target.Scope,
		DatabaseName: target.Database,
		BackupTable:  target.Table,
		StartedAt:    &now,
	}
	if err := s.backupRepo.CreateJob(ctx, job); err != nil {
		return nil, bizerrors.Pass(ctx, "mysql.backup", "enqueueBackup", err)
	}

	// 任务级 goroutine：不纳入进程 Wait（备份可能长达数小时），但需 panic 兜底避免整进程崩溃。
	lifecycle.GoDetached("mysqlbackup.run-job", func() {
		s.runBackupJobAsync(job.ID, projectID, instanceID, trigger)
	})
	return job, nil
}

const mysqlBackupJobTimeout = 35 * time.Minute

const mysqlXtrabackupJobTimeout = 2 * time.Hour

func (s *MysqlBackupService) runBackupJobAsync(jobID, projectID, instanceID uint, trigger string) {
	defer s.jobCancels.Delete(jobID)

	timeout := mysqlBackupJobTimeout
	if inst, err := s.backupRepo.GetInstanceInProject(context.Background(), projectID, instanceID); err == nil && model.IsXtrabackupBackupMode(inst.BackupMode) {
		timeout = mysqlXtrabackupJobTimeout
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), timeout)
	s.jobCancels.Store(jobID, cancel)
	defer cancel()
	s.finishBackupJob(jobCtx, jobID, projectID, instanceID, trigger)
}

func (s *MysqlBackupService) finishBackupJob(ctx context.Context, jobID, projectID, instanceID uint, trigger string) {
	started := time.Now()
	defer func() {
		s.tryNotifyBackupJobEmail(context.Background(), jobID, projectID, instanceID, trigger, time.Since(started))
	}()
	job, err := s.backupRepo.GetJob(ctx, jobID)
	if err != nil {
		return
	}
	inst, pw, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		fin := time.Now()
		job.FinishedAt = &fin
		_ = s.backupRepo.UpdateJob(ctx, job)
		s.logBackupJobDone(jobID, 0, "", trigger, "failed", time.Since(started), err)
		return
	}
	s.logBackupJobBegin(jobID, inst, trigger)
	target := mysqlbackup.BuildDumpTarget(inst)

	var runErr error
	switch {
	case model.IsXtrabackupBackupMode(inst.BackupMode):
		runErr = s.runXtrabackupUpload(ctx, inst, pw, job)
	default:
		runErr = s.runMysqldumpUpload(ctx, inst, pw, job, target, "")
	}

	fin := time.Now()
	job.FinishedAt = &fin
	if s.jobWasCancelled(ctx, jobID) {
		s.logBackupJobDone(jobID, inst.ID, inst.Name, trigger, "cancelled", time.Since(started), nil)
		return
	}
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			job.Status = "cancelled"
			if strings.TrimSpace(job.ErrorMessage) == "" {
				job.ErrorMessage = "任务已取消"
			}
		} else {
			job.Status = "failed"
			job.ErrorMessage = runErr.Error()
		}
	} else {
		job.Status = "success"
	}
	_ = s.backupRepo.UpdateJob(ctx, job)
	s.logBackupJobDone(jobID, inst.ID, inst.Name, trigger, job.Status, time.Since(started), runErr,
		"minio_object", job.MinioObject,
		"file_size", job.FileSize,
		"remote_path", job.RemotePath,
	)
}

func (s *MysqlBackupService) ListJobs(ctx context.Context, q MysqlBackupJobListQuery) (*pagination.Result[model.MysqlBackupJob], error) {
	list, total, err := s.backupRepo.ListJobs(ctx, repository.MysqlBackupJobListParams{
		ProjectID: q.ProjectID, InstanceID: q.InstanceID, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "mysql.backup", "ListJobs", err)
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	return &pagination.Result[model.MysqlBackupJob]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

// StopJob 停止进行中的备份任务（释放实例锁，best-effort 终止远端进程）。
func (s *MysqlBackupService) StopJob(ctx context.Context, projectID, jobID uint) (*model.MysqlBackupJob, error) {
	job, err := s.backupRepo.GetJobInProject(ctx, projectID, jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "mysql.backup", "StopJob", err)
	}
	if job.Status != "running" {
		return nil, constants.ErrBadRequestWithMsg("仅进行中的任务可停止")
	}
	if v, ok := s.jobCancels.Load(jobID); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
	if inst, err := s.backupRepo.GetInstanceInProject(ctx, projectID, job.InstanceID); err == nil && inst != nil {
		s.killRemoteBackupBestEffort(ctx, inst, job)
	}
	now := time.Now()
	msg := "用户手动停止"
	if err := s.backupRepo.PatchJob(ctx, jobID, map[string]any{
		"status":        "cancelled",
		"error_message": msg,
		"finished_at":   now,
	}); err != nil {
		return nil, bizerrors.Pass(ctx, "mysql.backup", "StopJob", err)
	}
	job.Status = "cancelled"
	job.ErrorMessage = msg
	job.FinishedAt = &now
	mysqlBackupLog().Info("MySQL backup job stopped by user", "job_id", jobID, "instance_id", job.InstanceID)
	return job, nil
}

// DeleteJob 删除备份记录（进行中的任务须先停止）。
func (s *MysqlBackupService) DeleteJob(ctx context.Context, projectID, jobID uint) error {
	job, err := s.backupRepo.GetJobInProject(ctx, projectID, jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return bizerrors.Pass(ctx, "mysql.backup", "DeleteJob", err)
	}
	if job.Status == "running" {
		return constants.ErrBadRequestWithMsg("进行中的任务请先停止后再删除")
	}
	if err := s.backupRepo.DeleteJob(ctx, projectID, jobID); err != nil {
		return bizerrors.Pass(ctx, "mysql.backup", "DeleteJob", err)
	}
	return nil
}

func (s *MysqlBackupService) jobWasCancelled(ctx context.Context, jobID uint) bool {
	job, err := s.backupRepo.GetJob(ctx, jobID)
	return err == nil && job != nil && job.Status == "cancelled"
}

func (s *MysqlBackupService) killRemoteBackupBestEffort(ctx context.Context, inst *model.MysqlBackupInstance, job *model.MysqlBackupJob) {
	if inst == nil || job == nil {
		return
	}
	remotePath := strings.TrimSpace(job.RemotePath)
	if remotePath == "" {
		return
	}
	script := mysqlbackup.BuildKillBackupByArtifactScript(remotePath)
	if script == "" {
		return
	}
	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return
	}
	defer sshCli.Close()
	killCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, _ = sshCli.Exec(killCtx, script, 4096)
}
