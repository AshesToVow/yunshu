// 备份任务结构化日志（开始/结束/阶段）与远端日志文件轮询。
package mysqlbackup

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/pkg/mysqlbackup"
	"yunshu/internal/pkg/sshclient"
)

func (s *MysqlBackupService) logBackupJobBegin(jobID uint, inst *model.MysqlBackupInstance, trigger string) {
	if inst == nil {
		return
	}
	mysqlBackupLog().Info("Started MySQL backup job",
		"job_id", jobID,
		"instance_id", inst.ID,
		"project_id", inst.ProjectID,
		"instance_name", inst.Name,
		"backup_mode", inst.BackupMode,
		"trigger", trigger,
		"mysql_user", inst.MysqlUser,
		"mysql_host", inst.MysqlHost,
		"mysql_port", inst.MysqlPort,
	)
}

func (s *MysqlBackupService) logBackupJobDone(jobID, instanceID uint, instanceName, trigger, status string, dur time.Duration, runErr error, extra ...any) {
	attrs := []any{
		"job_id", jobID,
		"instance_id", instanceID,
		"instance_name", instanceName,
		"trigger", trigger,
		"status", status,
		"duration_ms", dur.Milliseconds(),
	}
	attrs = append(attrs, extra...)
	if runErr != nil {
		mysqlBackupLog().Error("Failed to finish MySQL backup job", append(attrs, "error", runErr)...)
		return
	}
	mysqlBackupLog().Info("Finished MySQL backup job", attrs...)
}

func (s *MysqlBackupService) logBackupPhase(jobID uint, phase string, attrs ...any) {
	base := []any{"job_id", jobID, "phase", phase}
	mysqlBackupLog().Info("MySQL backup job phase", append(base, attrs...)...)
}

func (s *MysqlBackupService) startPollBackupJobLog(ctx context.Context, jobID uint, sshCli *sshclient.Client, logPath, prefix string) context.CancelFunc {
	pollCtx, cancel := context.WithCancel(ctx)
	lifecycle.GoDetached("mysqlbackup.poll-job-log", func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				tail, err := s.tailRemoteFile(pollCtx, sshCli, logPath, 100)
				if err != nil {
					continue
				}
				excerpt := mysqlbackup.TruncateLog(prefix + strings.TrimSpace(tail))
				_ = s.backupRepo.PatchJob(pollCtx, jobID, map[string]any{"log_excerpt": excerpt})
			}
		}
	})
	return cancel
}
