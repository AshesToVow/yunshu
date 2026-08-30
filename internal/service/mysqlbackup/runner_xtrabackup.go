// xtrabackup 执行链路：远端物理备份 + 上传对象存储。
package mysqlbackup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/mysqlbackup"
)

func (s *MysqlBackupService) runXtrabackupUpload(ctx context.Context, inst *model.MysqlBackupInstance, pw string, job *model.MysqlBackupJob) error {
	sshCli, sv, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()
	s.logBackupPhase(job.ID, "ssh_connected", "server_id", inst.ServerID)

	dataDir := strings.TrimSuffix(strings.TrimSpace(inst.RemoteDataDir), "/")
	logDir := strings.TrimSuffix(strings.TrimSpace(inst.RemoteLogDir), "/")
	if dataDir == "" || logDir == "" {
		return constants.ErrBadRequestWithMsg("xtrabackup 须配置 remote_data_dir 与 remote_log_dir")
	}

	startedAt := time.Now().In(mysqlbackup.BackupNameLocation())
	basename, err := s.backupArtifactBasename(ctx, inst, startedAt)
	if err != nil {
		return err
	}
	remoteArchive := filepath.ToSlash(filepath.Join(dataDir, basename+".tar.gz"))
	logPath := filepath.ToSlash(filepath.Join(logDir, basename+".log"))
	job.RemotePath = remoteArchive
	job.BackupMode = model.MysqlBackupExecXtrabackup

	cliConnect, xbConnect, connectLog := mysqlbackup.FormatMysqlConnectShell(inst.MysqlSocket, inst.MysqlHost, inst.MysqlPort, inst.MysqlUser, shellQuote)

	script := mysqlbackup.BuildXtrabackupRemoteScript(mysqlbackup.XtrabackupRemoteScriptParams{
		DataDir: dataDir, LogDir: logDir, Basename: basename,
		MySQLPass: shellQuote(pw), MySQLDir: inst.MysqlDataDir, Parallel: 4,
		ToolPref: inst.XtrabackupTool, XtrabackupBin: inst.XtrabackupBin, InnobackupexBin: inst.InnobackupexBin,
		ConnectLog: connectLog, CLIConnect: cliConnect, XBConnect: xbConnect, ShellQuote: shellQuote,
	})
	stopPoll := s.startPollBackupJobLog(ctx, job.ID, sshCli, logPath, "")
	defer stopPoll()

	s.logBackupPhase(job.ID, "xtrabackup_start",
		"data_dir", dataDir,
		"basename", basename,
		"archive", remoteArchive,
	)
	// maxBytes=0：日志走远端 $LOG + 轮询 tail，避免 xtrabackup 巨量 stdout 塞满 SSH 导致 tee/tar 卡死。
	res, err := sshCli.Exec(ctx, script, 0)
	if err != nil {
		return err
	}
	tail, _ := s.tailRemoteFile(ctx, sshCli, logPath, 120)
	job.LogExcerpt = mysqlbackup.TruncateLog(strings.TrimSpace(tail))
	execTool := mysqlbackup.DetectPhysicalBackupExecTool(tail)
	if execTool == mysqlbackup.XtrabackupToolInnobackupex {
		job.BackupMode = model.MysqlBackupExecInnobackupex
	} else {
		job.BackupMode = model.MysqlBackupExecXtrabackup
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%s failed (exit=%d): %s", execTool, res.ExitCode, job.LogExcerpt)
	}
	if !strings.Contains(tail, mysqlbackup.BackupCompletedMarker) {
		return fmt.Errorf("%s finished without completion marker in log: %s", execTool, mysqlbackup.TruncateLog(tail))
	}
	archSize, sizeErr := sshCli.RemoteFileSize(remoteArchive)
	if sizeErr != nil {
		return fmt.Errorf("backup archive not found after xtrabackup: %w", sizeErr)
	}
	if err := mysqlbackup.ValidateArchiveSize(archSize, remoteArchive); err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "xtrabackup_done", "exit_code", res.ExitCode, "archive_bytes", archSize, "ssh_duration_ms", res.Duration.Milliseconds())

	if !inst.UploadToMinio {
		job.CheckOK = true
		s.logBackupPhase(job.ID, "skip_minio_upload", "remote_archive", remoteArchive, "remote_log", logPath)
		_ = sv
		return nil
	}

	minioCli, err := s.newObjectStore(ctx)
	if err != nil {
		return err
	}
	dlCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	localPath := filepath.Join(os.TempDir(), basename+".tar.gz")
	defer os.Remove(localPath)
	if err := sshCli.DownloadFile(dlCtx, remoteArchive, localPath); err != nil {
		return err
	}
	objectKey := mysqlbackup.BuildMinioObjectKey(basename, "tar.gz")
	s.logBackupPhase(job.ID, "minio_upload_start", "object_key", objectKey)
	size, err := minioCli.UploadFile(ctx, objectKey, localPath, "application/gzip")
	if err != nil {
		return err
	}
	job.MinioBucket = minioCli.Bucket()
	job.MinioObject = objectKey
	job.FileSize = size
	job.CheckOK = true
	_ = sv
	return nil
}
