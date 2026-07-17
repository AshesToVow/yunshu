package mysqlbackup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
)

func (s *MysqlBackupService) tryNotifyBackupJobEmail(ctx context.Context, jobID, projectID, instanceID uint, trigger string, duration time.Duration) {
	if s == nil || s.mailer == nil || !s.mailer.Enabled() {
		return
	}

	job, err := s.backupRepo.GetJob(ctx, jobID)
	if err != nil || job == nil {
		return
	}
	if job.Status == "running" {
		return
	}

	inst, err := s.backupRepo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil || inst == nil || !inst.NotifyEnabled {
		return
	}
	emails := s.resolveNotifyEmails(ctx, parseNotifyUserIDs(inst.NotifyUserIDs))
	if len(emails) == 0 {
		return
	}

	projectName := fmt.Sprintf("project_%d", projectID)
	if proj, err := s.projectRepo.GetByID(ctx, projectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}

	instanceName := strings.TrimSpace(inst.Name)
	if instanceName == "" {
		instanceName = fmt.Sprintf("instance_%d", instanceID)
	}
	backupMode := strings.TrimSpace(job.BackupMode)
	if backupMode == "" {
		backupMode = inst.BackupMode
	}

	subject, body := buildBackupNotifyMail(s.appName, projectName, instanceName, job, trigger, backupMode, duration)
	for _, to := range emails {
		if err := s.mailer.Send(ctx, to, subject, body); err != nil {
			mysqlBackupLog().Warn("MySQL backup notify email failed",
				"job_id", jobID,
				"to", to,
				"status", job.Status,
				"error", err,
			)
		}
	}
}

func buildBackupNotifyMail(appName, projectName, instanceName string, job *model.MysqlBackupJob, trigger, backupMode string, duration time.Duration) (string, string) {
	statusLabel := backupJobStatusLabel(job.Status)
	triggerLabel := backupTriggerLabel(trigger)
	modeLabel := backupModeLabel(backupMode)

	app := strings.TrimSpace(appName)
	if app == "" {
		app = "yunshu"
	}
	subject := fmt.Sprintf("[%s] MySQL备份%s - %s / %s", app, statusLabel, projectName, instanceName)

	var b strings.Builder
	fmt.Fprintf(&b, "MySQL 备份任务已结束\n\n")
	fmt.Fprintf(&b, "状态：%s\n", statusLabel)
	fmt.Fprintf(&b, "项目：%s\n", projectName)
	fmt.Fprintf(&b, "实例：%s\n", instanceName)
	fmt.Fprintf(&b, "任务 ID：%d\n", job.ID)
	fmt.Fprintf(&b, "触发方式：%s\n", triggerLabel)
	fmt.Fprintf(&b, "备份方式：%s\n", modeLabel)
	if job.BackupScope != "" {
		fmt.Fprintf(&b, "备份范围：%s\n", backupScopeLabel(job))
	}
	if duration > 0 {
		fmt.Fprintf(&b, "耗时：%s\n", duration.Round(time.Second))
	}
	if job.StartedAt != nil {
		fmt.Fprintf(&b, "开始时间：%s\n", job.StartedAt.In(mysqlBackupMailLocation()).Format("2006-01-02 15:04:05 MST"))
	}
	if job.FinishedAt != nil {
		fmt.Fprintf(&b, "结束时间：%s\n", job.FinishedAt.In(mysqlBackupMailLocation()).Format("2006-01-02 15:04:05 MST"))
	}
	if strings.TrimSpace(job.RemotePath) != "" {
		fmt.Fprintf(&b, "远端路径：%s\n", job.RemotePath)
	}
	if strings.TrimSpace(job.MinioObject) != "" {
		fmt.Fprintf(&b, "MinIO 对象：%s\n", job.MinioObject)
	}
	if job.FileSize > 0 {
		fmt.Fprintf(&b, "文件大小：%.2f MiB\n", float64(job.FileSize)/1024/1024)
	}
	if strings.TrimSpace(job.ErrorMessage) != "" {
		fmt.Fprintf(&b, "\n错误信息：\n%s\n", strings.TrimSpace(job.ErrorMessage))
	}
	fmt.Fprintf(&b, "\n请在控制台「MySQL 备份 → 备份记录」查看详情与日志。\n")
	return subject, b.String()
}

func mysqlBackupMailLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func backupJobStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "success":
		return "成功"
	case "failed":
		return "失败"
	case "cancelled":
		return "已取消"
	default:
		return status
	}
}

func backupTriggerLabel(trigger string) string {
	switch strings.TrimSpace(trigger) {
	case model.MysqlBackupTriggerScheduled:
		return "定时"
	case model.MysqlBackupTriggerManual:
		return "手动"
	default:
		if trigger == "" {
			return "手动"
		}
		return trigger
	}
}

func backupModeLabel(mode string) string {
	switch strings.TrimSpace(mode) {
	case model.MysqlBackupModeMysqldump:
		return "mysqldump（逻辑备份）"
	case model.MysqlBackupExecInnobackupex:
		return "innobackupex（物理备份）"
	case model.MysqlBackupModeXtrabackup, model.MysqlBackupModeRemoteCheck:
		return "xtrabackup（物理备份）"
	default:
		if mode == "" {
			return "未知"
		}
		return mode
	}
}

func backupScopeLabel(job *model.MysqlBackupJob) string {
	if job == nil {
		return "-"
	}
	switch strings.TrimSpace(job.BackupScope) {
	case model.MysqlBackupScopeTable:
		return fmt.Sprintf("单表 %s.%s", job.DatabaseName, job.BackupTable)
	case model.MysqlBackupScopeDatabase:
		if strings.TrimSpace(job.DatabaseName) != "" {
			return "单库 " + job.DatabaseName
		}
		return "单库"
	case model.MysqlBackupScopeAll:
		return "全部库"
	default:
		return job.BackupScope
	}
}
