// mysqldump 执行链路：远端导出 + 上传对象存储。
package mysqlbackup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/mysqlbackup"
)

func (s *MysqlBackupService) runMysqldumpUpload(ctx context.Context, inst *model.MysqlBackupInstance, pw string, job *model.MysqlBackupJob, target mysqlbackup.DumpTarget, logPrefix string) error {
	sshCli, sv, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()
	s.logBackupPhase(job.ID, "ssh_connected", "server_id", inst.ServerID)

	workDir, err := mysqlbackup.NormalizeMysqldumpWorkDir(inst.MysqldumpWorkDir)
	if err != nil {
		return err
	}
	optIDs, err := mysqlbackup.ParseMysqldumpOptionIDs(inst.MysqldumpOptions)
	if err != nil {
		return err
	}
	dumpFlags, err := mysqlbackup.FormatMysqldumpFlags(optIDs, inst.MysqldumpExtraArgs)
	if err != nil {
		return err
	}

	startedAt := time.Now().In(mysqlbackup.BackupNameLocation())
	basename, err := s.backupArtifactBasename(ctx, inst, startedAt)
	if err != nil {
		return err
	}
	remotePath := filepath.ToSlash(filepath.Join(workDir, basename+".sql.gz"))
	logPath := filepath.ToSlash(filepath.Join(workDir, basename+".log"))
	job.RemotePath = remotePath
	job.BackupMode = model.MysqlBackupModeMysqldump
	_ = s.backupRepo.PatchJob(ctx, job.ID, map[string]any{
		"remote_path": remotePath,
		"backup_mode": model.MysqlBackupModeMysqldump,
	})

	dumpTarget := mysqlbackup.FormatDumpArgsShell(target, shellQuote)
	connectArgs, connectLog := mysqlbackup.FormatMysqldumpConnectArgs(inst.MysqlSocket, inst.MysqlHost, inst.MysqlPort, inst.MysqlUser, shellQuote)

	dumpCmd := mysqlbackup.BuildMysqldumpRemoteScript(mysqlbackup.MysqldumpRemoteScriptParams{
		WorkDir: workDir, Basename: basename,
		MySQLPass: shellQuote(pw), DumpFlags: dumpFlags, DumpTarget: dumpTarget,
		DumpTargetLabel: target.ObjectLabel, MysqldumpBin: inst.MysqldumpBin,
		ConnectArgs: connectArgs, ConnectLog: connectLog, ShellQuote: shellQuote,
	})
	stopPoll := s.startPollBackupJobLog(ctx, job.ID, sshCli, logPath, logPrefix)
	defer stopPoll()

	s.logBackupPhase(job.ID, "mysqldump_start",
		"work_dir", workDir,
		"basename", basename,
		"remote_sql_gz", remotePath,
		"flags", dumpFlags,
	)
	res, err := sshCli.Exec(ctx, dumpCmd, 65536)
	job.LogExcerpt = mysqlbackup.TruncateLog(logPrefix + strings.TrimSpace(res.Stdout+res.Stderr))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(res.Stderr+res.Stdout))
	}
	s.logBackupPhase(job.ID, "mysqldump_done", "exit_code", res.ExitCode, "ssh_duration_ms", res.Duration.Milliseconds())

	if !inst.UploadToMinio {
		s.logBackupPhase(job.ID, "skip_minio_upload", "remote_sql_gz", remotePath, "remote_log", logPath)
		job.CheckOK = true
		_ = sv
		return nil
	}

	minioCli, err := s.newObjectStore(ctx)
	if err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "minio_client_ready",
		"endpoint", minioCli.Endpoint(),
		"bucket", minioCli.Bucket(),
	)

	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err := sshCli.WaitRemoteFile(waitCtx, remotePath, 1024, 30*time.Minute); err != nil {
		return err
	}

	localPath := filepath.Join(os.TempDir(), basename+".sql.gz")
	defer os.Remove(localPath)
	if err := sshCli.DownloadFile(waitCtx, remotePath, localPath); err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "local_dump_ready", "local_path", localPath)

	objectKey := mysqlbackup.BuildMinioObjectKey(basename, "sql.gz")
	s.logBackupPhase(job.ID, "minio_upload_start", "object_key", objectKey)
	size, err := minioCli.UploadFile(ctx, objectKey, localPath, "application/gzip")
	if err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "minio_upload_done", "bytes", size, "object_key", objectKey)
	job.MinioBucket = minioCli.Bucket()
	job.MinioObject = objectKey
	job.FileSize = size
	job.CheckOK = true
	_ = sv
	return nil
}
