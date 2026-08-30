// 备份产物：对象存储命名（前缀/basename）、最新产物定位、远端备份校验与预签名下载。
package mysqlbackup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/mysqlbackup"

	"gorm.io/gorm"
)

func (s *MysqlBackupService) findLatestBackupArtifact(ctx context.Context, inst *model.MysqlBackupInstance) (*mysqlbackup.BackupArtifact, error) {
	prefix, err := s.backupArtifactNamePrefix(ctx, inst)
	if err != nil {
		return nil, err
	}
	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return nil, err
	}
	defer sshCli.Close()
	script := mysqlbackup.BuildFindLatestBackupScript(inst.RemoteDataDir, inst.RemoteLogDir, prefix, 30)
	res, err := sshCli.Exec(ctx, script, 16384)
	if err != nil && !strings.Contains(res.Stdout+res.Stderr, "NOT_FOUND") {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHExecFailedPrefix + err.Error())
	}
	artifact := mysqlbackup.ParseFindLatestBackupOutput(strings.TrimSpace(res.Stdout+"\n"+res.Stderr), inst.MysqlPort)
	if artifact.OK {
		artifact.Message = fmt.Sprintf("找到有效备份: %s", artifact.BackupFile)
	} else {
		artifact.Message = fmt.Sprintf("未找到匹配前缀 %q 的有效备份（*.tar.gz 且日志末行含 %s）", prefix, mysqlbackup.BackupCompletedMarker)
	}
	return &artifact, nil
}

func (s *MysqlBackupService) CheckRemoteBackup(ctx context.Context, projectID, instanceID uint, dayOffset int) (*mysqlbackup.RemoteCheckResult, error) {
	inst, _, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if !model.IsXtrabackupBackupMode(inst.BackupMode) {
		return nil, constants.ErrBadRequestWithMsg("该实例不是 xtrabackup 模式")
	}
	_ = dayOffset
	artifact, err := s.findLatestBackupArtifact(ctx, inst)
	if err != nil {
		return nil, err
	}
	return &mysqlbackup.RemoteCheckResult{
		BackupFile:   artifact.BackupFile,
		LogFile:      artifact.LogFile,
		LogCompleted: artifact.OK,
		OK:           artifact.OK,
		Message:      artifact.Message,
	}, nil
}

func (s *MysqlBackupService) backupArtifactNamePrefix(ctx context.Context, inst *model.MysqlBackupInstance) (string, error) {
	projectName := fmt.Sprintf("project_%d", inst.ProjectID)
	if proj, err := s.projectRepo.GetByID(ctx, inst.ProjectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}
	return mysqlbackup.BuildArtifactNamePrefix(projectName, inst.MysqlHost, inst.MysqlPort), nil
}

func (s *MysqlBackupService) PresignDownload(ctx context.Context, projectID, jobID uint) (string, error) {
	job, err := s.backupRepo.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", constants.ErrNotFound
		}
		return "", err
	}
	if job.ProjectID != projectID || job.Status != "success" {
		return "", constants.ErrBadRequestWithMsg("任务未完成")
	}
	if strings.TrimSpace(job.MinioObject) == "" {
		return "", constants.ErrBadRequestWithMsg("该任务未上传 MinIO，请查看日志中的远端路径")
	}
	cli, err := s.newObjectStore(ctx)
	if err != nil {
		return "", err
	}
	return cli.PresignedGetURL(ctx, job.MinioObject, 15*time.Minute)
}

func (s *MysqlBackupService) backupArtifactBasename(ctx context.Context, inst *model.MysqlBackupInstance, at time.Time) (string, error) {
	projectName := fmt.Sprintf("project_%d", inst.ProjectID)
	if proj, err := s.projectRepo.GetByID(ctx, inst.ProjectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}
	return mysqlbackup.BuildArtifactBasename(projectName, inst.MysqlHost, inst.MysqlPort, at), nil
}
