package mysqlbackup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/mysqlbackup"
	"yunshu/internal/pkg/objectstore"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/sshserver"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

	"crypto/cipher"

	"gorm.io/gorm"
)

// ObjectStoreFactory 创建 MinIO 客户端；由装配层注入，避免 Service 直拿 *gorm.DB 读字典。
type ObjectStoreFactory func(ctx context.Context) (*objectstore.Client, error)

// SchedulerConfigResolver 解析备份调度字典配置。
type SchedulerConfigResolver func(ctx context.Context) dictconfig.MysqlBackupSchedulerConfig

type MysqlBackupService struct {
	backupRepo   interfaces.MysqlBackupRepository
	serverRepo   interfaces.ServerRepository
	projectRepo  interfaces.ProjectRepository
	userRepo     interfaces.UserRepository
	newObjectStore ObjectStoreFactory
	resolveSched   SchedulerConfigResolver
	aead         cipher.AEAD
	mailer       mailer.Sender
	appName      string
	schedMu      sync.Mutex
	schedRunning map[uint]bool
	jobCancels   sync.Map // jobID -> context.CancelFunc
}

func NewMysqlBackupService(
	backupRepo interfaces.MysqlBackupRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userRepo interfaces.UserRepository,
	newObjectStore ObjectStoreFactory,
	resolveSched SchedulerConfigResolver,
	encryptionKey string,
	emailSender mailer.Sender,
	appName string,
) (*MysqlBackupService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	if newObjectStore == nil {
		newObjectStore = func(context.Context) (*objectstore.Client, error) {
			return nil, fmt.Errorf("object store factory not configured")
		}
	}
	if resolveSched == nil {
		resolveSched = func(context.Context) dictconfig.MysqlBackupSchedulerConfig {
			return dictconfig.ResolveMysqlBackupSchedulerConfig(context.Background(), nil, dictconfig.DefaultMysqlBackupSchedulerDictTypes())
		}
	}
	return &MysqlBackupService{
		backupRepo:     backupRepo,
		serverRepo:     serverRepo,
		projectRepo:    projectRepo,
		userRepo:       userRepo,
		newObjectStore: newObjectStore,
		resolveSched:   resolveSched,
		aead:           aead,
		mailer:         emailSender,
		appName:        strings.TrimSpace(appName),
		schedRunning:   make(map[uint]bool),
	}, nil
}

type MysqlBackupInstanceItem struct {
	ID                 uint     `json:"id"`
	ProjectID          uint     `json:"project_id"`
	ServerID           uint     `json:"server_id"`
	ServerName         string   `json:"server_name,omitempty"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	MysqlHost          string   `json:"mysql_host"`
	MysqlPort          int      `json:"mysql_port"`
	MysqlSocket        string   `json:"mysql_socket"`
	MysqlUser          string   `json:"mysql_user"`
	BackupMode         string   `json:"backup_mode"`
	BackupScope        string   `json:"backup_scope"`
	DatabaseName       string   `json:"database_name"`
	TableName          string   `json:"table_name"`
	DatabaseNames      string   `json:"database_names"`
	RemoteDataDir      string   `json:"remote_data_dir"`
	RemoteLogDir       string   `json:"remote_log_dir"`
	MysqlDataDir       string   `json:"mysql_datadir"`
	UploadToMinio      bool     `json:"upload_to_minio"`
	MysqldumpWorkDir   string   `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string `json:"mysqldump_options"`
	MysqldumpExtraArgs string   `json:"mysqldump_extra_args"`
	MysqldumpBin       string   `json:"mysqldump_bin"`
	XtrabackupTool     string   `json:"xtrabackup_tool"`
	XtrabackupBin      string   `json:"xtrabackup_bin"`
	InnobackupexBin    string   `json:"innobackupex_bin"`
	ScheduleEnabled    bool     `json:"schedule_enabled"`
	CronSpec           string   `json:"cron_spec"`
	LastScheduledAt    string   `json:"last_scheduled_at,omitempty"`
	NotifyEnabled      bool     `json:"notify_enabled"`
	NotifyUserIDs      []uint   `json:"notify_user_ids"`
	NotifyUsers        []MysqlBackupNotifyUserItem `json:"notify_users,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
}

type MysqlBackupInstanceUpsertRequest struct {
	ProjectID          uint     `json:"project_id"`
	ServerID           uint     `json:"server_id" binding:"required"`
	Name               string   `json:"name" binding:"required,max=128"`
	Enabled            *bool    `json:"enabled"`
	MysqlHost          string   `json:"mysql_host"`
	MysqlPort          int      `json:"mysql_port"`
	MysqlSocket        string   `json:"mysql_socket"`
	MysqlUser          string   `json:"mysql_user" binding:"required"`
	MysqlPassword      string   `json:"mysql_password"`
	BackupMode         string   `json:"backup_mode"`
	BackupScope        string   `json:"backup_scope"`
	DatabaseName       string   `json:"database_name"`
	TableName          string   `json:"table_name"`
	DatabaseNames      string   `json:"database_names"`
	RemoteDataDir      string   `json:"remote_data_dir"`
	RemoteLogDir       string   `json:"remote_log_dir"`
	MysqlDataDir       string   `json:"mysql_datadir"`
	UploadToMinio      *bool    `json:"upload_to_minio"`
	MysqldumpWorkDir   string   `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string `json:"mysqldump_options"`
	MysqldumpExtraArgs string   `json:"mysqldump_extra_args"`
	MysqldumpBin       string   `json:"mysqldump_bin"`
	XtrabackupTool     string   `json:"xtrabackup_tool"`
	XtrabackupBin      string   `json:"xtrabackup_bin"`
	InnobackupexBin    string   `json:"innobackupex_bin"`
	ScheduleEnabled    *bool    `json:"schedule_enabled"`
	CronSpec           string   `json:"cron_spec"`
	NotifyEnabled      *bool    `json:"notify_enabled"`
	NotifyUserIDs      []uint   `json:"notify_user_ids"`
}

type MysqlBackupInstanceListQuery struct {
	ProjectID uint `form:"project_id"`
	Page      int  `form:"page"`
	PageSize  int  `form:"page_size"`
}

type MysqlBackupJobListQuery struct {
	ProjectID  uint `form:"project_id"`
	InstanceID uint `form:"instance_id"`
	Page       int  `form:"page"`
	PageSize   int  `form:"page_size"`
}

func (s *MysqlBackupService) ListInstances(ctx context.Context, q MysqlBackupInstanceListQuery) (*pagination.Result[MysqlBackupInstanceItem], error) {
	list, total, err := s.backupRepo.ListInstances(ctx, repository.MysqlBackupInstanceListParams{
		ProjectID: q.ProjectID, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "mysql.backup", "ListInstances", err)
	}
	out := make([]MysqlBackupInstanceItem, 0, len(list))
	for _, inst := range list {
		out = append(out, s.toInstanceItem(ctx, inst))
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	return &pagination.Result[MysqlBackupInstanceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *MysqlBackupService) UpsertInstance(ctx context.Context, id uint, req MysqlBackupInstanceUpsertRequest, actor *auth.CurrentUser) (*MysqlBackupInstanceItem, error) {
	if err := s.ensureServerInProject(ctx, req.ProjectID, req.ServerID); err != nil {
		return nil, err
	}
	mode := strings.TrimSpace(req.BackupMode)
	if mode == "" {
		mode = model.MysqlBackupModeMysqldump
	}
	if mode != model.MysqlBackupModeMysqldump && !model.IsXtrabackupBackupMode(mode) {
		return nil, constants.ErrBadRequestWithMsg("backup_mode 须为 mysqldump 或 xtrabackup")
	}
	if model.IsXtrabackupBackupMode(mode) {
		mode = model.MysqlBackupModeXtrabackup
	}

	var inst *model.MysqlBackupInstance
	if id > 0 {
		existing, err := s.backupRepo.GetInstanceInProject(ctx, req.ProjectID, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFound
			}
			return nil, bizerrors.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
		inst = existing
	} else {
		inst = &model.MysqlBackupInstance{ProjectID: req.ProjectID}
	}

	inst.ServerID = req.ServerID
	inst.Name = strings.TrimSpace(req.Name)
	if req.Enabled != nil {
		inst.Enabled = *req.Enabled
	} else if id == 0 {
		inst.Enabled = true
	}
	inst.MysqlHost = strings.TrimSpace(req.MysqlHost)
	if inst.MysqlHost == "" {
		inst.MysqlHost = "127.0.0.1"
	}
	if req.MysqlPort > 0 {
		inst.MysqlPort = req.MysqlPort
	} else if inst.MysqlPort <= 0 {
		inst.MysqlPort = 3306
	}
	inst.MysqlUser = strings.TrimSpace(req.MysqlUser)
	socket, err := mysqlbackup.NormalizeMysqlSocket(req.MysqlSocket)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqlSocket = socket
	inst.BackupMode = mode
	scope := strings.TrimSpace(req.BackupScope)
	if scope == "" {
		scope = model.MysqlBackupScopeAll
	}
	if mode == model.MysqlBackupModeMysqldump {
		if err := validateMysqlBackupScope(scope, req.DatabaseName, req.TableName, req.DatabaseNames); err != nil {
			return nil, err
		}
		inst.BackupScope = scope
		inst.DatabaseName = strings.TrimSpace(req.DatabaseName)
		inst.BackupTable = strings.TrimSpace(req.TableName)
	} else {
		inst.BackupScope = model.MysqlBackupScopeAll
		if strings.TrimSpace(req.RemoteDataDir) == "" || strings.TrimSpace(req.RemoteLogDir) == "" {
			return nil, constants.ErrBadRequestWithMsg("xtrabackup 模式须填写 remote_data_dir 与 remote_log_dir")
		}
		if strings.TrimSpace(req.MysqlDataDir) == "" {
			return nil, constants.ErrBadRequestWithMsg("xtrabackup 模式须填写 mysql_datadir（宿主机 MySQL 数据目录，Docker 常为 /export/mysql_data）")
		}
	}
	inst.DatabaseNames = strings.TrimSpace(req.DatabaseNames)
	inst.RemoteDataDir = strings.TrimSpace(req.RemoteDataDir)
	inst.RemoteLogDir = strings.TrimSpace(req.RemoteLogDir)
	inst.MysqlDataDir = strings.TrimSpace(req.MysqlDataDir)
	if req.UploadToMinio != nil {
		inst.UploadToMinio = *req.UploadToMinio
	} else if id == 0 {
		inst.UploadToMinio = true
	}

	workDir, err := mysqlbackup.NormalizeMysqldumpWorkDir(req.MysqldumpWorkDir)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqldumpWorkDir = workDir
	if err := mysqlbackup.ValidateBackupPathIsolation(workDir, inst.RemoteDataDir, inst.RemoteLogDir); err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	optionsJSON := marshalMysqldumpOptionIDs(req.MysqldumpOptions)
	optIDs, err := mysqlbackup.ParseMysqldumpOptionIDs(optionsJSON)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	if _, err := mysqlbackup.FormatMysqldumpFlags(optIDs, req.MysqldumpExtraArgs); err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqldumpOptions = optionsJSON
	inst.MysqldumpExtraArgs = strings.TrimSpace(req.MysqldumpExtraArgs)
	dumpBin, err := mysqlbackup.NormalizeMysqldumpBin(req.MysqldumpBin)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqldumpBin = dumpBin
	tool, err := mysqlbackup.NormalizeXtrabackupTool(req.XtrabackupTool)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.XtrabackupTool = tool
	xbBin, err := mysqlbackup.NormalizeXtrabackupBin(req.XtrabackupBin)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.XtrabackupBin = xbBin
	ibBin, err := mysqlbackup.NormalizeInnobackupexBin(req.InnobackupexBin)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.InnobackupexBin = ibBin

	if req.ScheduleEnabled != nil {
		inst.ScheduleEnabled = *req.ScheduleEnabled
	}
	cronSpec := strings.TrimSpace(req.CronSpec)
	if cronSpec != "" || (req.ScheduleEnabled != nil && *req.ScheduleEnabled) {
		if err := ValidateMysqlBackupCronSpec(cronSpec); err != nil {
			return nil, err
		}
		inst.CronSpec = cronSpec
	} else if id == 0 {
		inst.CronSpec = ""
	}
	if inst.ScheduleEnabled && strings.TrimSpace(inst.CronSpec) == "" {
		return nil, constants.ErrBadRequestWithMsg("启用定时备份时必须填写 cron_spec（Cron 表达式）")
	}

	if req.NotifyEnabled != nil {
		inst.NotifyEnabled = *req.NotifyEnabled
	} else if id == 0 {
		inst.NotifyEnabled = true
	}
	if req.NotifyUserIDs != nil {
		ids := dedupeNotifyUserIDs(req.NotifyUserIDs)
		if inst.NotifyEnabled && len(ids) == 0 {
			return nil, constants.ErrBadRequestWithMsg("开启备份通知时须至少指定一名接收用户")
		}
		inst.NotifyUserIDs = marshalNotifyUserIDs(ids)
	} else if id == 0 {
		if actor != nil && actor.ID > 0 {
			inst.NotifyUserIDs = marshalNotifyUserIDs([]uint{actor.ID})
		} else {
			inst.NotifyUserIDs = "[]"
			inst.NotifyEnabled = false
		}
	}

	if pw := strings.TrimSpace(req.MysqlPassword); pw != "" {
		enc, err := cryptox.EncryptString(s.aead, pw)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
		inst.EncPassword = enc
	} else if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("新建实例须填写 mysql_password")
	}

	if id > 0 {
		if err := s.backupRepo.UpdateInstance(ctx, inst); err != nil {
			return nil, bizerrors.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
	} else {
		if err := s.backupRepo.CreateInstance(ctx, inst); err != nil {
			return nil, bizerrors.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
	}
	item := s.toInstanceItem(ctx, *inst)
	return &item, nil
}

func (s *MysqlBackupService) DeleteInstance(ctx context.Context, projectID, id uint) error {
	if _, err := s.backupRepo.GetInstanceInProject(ctx, projectID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return bizerrors.Pass(ctx, "mysql.backup", "DeleteInstance", err)
	}
	return s.backupRepo.DeleteInstance(ctx, id)
}

func (s *MysqlBackupService) PingInstance(ctx context.Context, projectID, instanceID uint) (bool, string, error) {
	inst, pw, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return false, "", err
	}
	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return false, "", err
	}
	defer sshCli.Close()
	script := mysqlbackup.BuildMysqlPingRemoteScript(
		inst.MysqlSocket, inst.MysqlHost, inst.MysqlPort, inst.MysqlUser, pw, inst.MysqldumpBin, shellQuote,
	)
	res, err := sshCli.Exec(ctx, script, 4096)
	out := strings.TrimSpace(res.Stdout + "\n" + res.Stderr)
	if err != nil || !strings.Contains(out, "status=1i") {
		if out == "" {
			connectLog := mysqlbackup.FormatMysqldumpConnectLog(inst.MysqlSocket, inst.MysqlHost, inst.MysqlPort, inst.MysqlUser)
			if err != nil {
				out = fmt.Sprintf("mysqlping,%s status=0i error=%s", connectLog, err.Error())
			} else {
				out = fmt.Sprintf("mysqlping,%s status=0i", connectLog)
			}
		}
		return false, out, nil
	}
	return true, out, nil
}

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

	go s.runBackupJobAsync(job.ID, projectID, instanceID, trigger)
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

func validateMysqlBackupScope(scope, dbName, tableName, databaseNames string) error {
	scope = strings.TrimSpace(scope)
	switch scope {
	case model.MysqlBackupScopeTable:
		if strings.TrimSpace(dbName) == "" || strings.TrimSpace(tableName) == "" {
			return constants.ErrBadRequestWithMsg("单表备份须填写 database_name 与 table_name")
		}
	case model.MysqlBackupScopeDatabase:
		if strings.TrimSpace(dbName) == "" && strings.TrimSpace(databaseNames) == "" {
			return constants.ErrBadRequestWithMsg("单库备份须填写 database_name 或 database_names")
		}
	case model.MysqlBackupScopeAll, "":
		return nil
	default:
		return constants.ErrBadRequestWithMsg("backup_scope 须为 all、database 或 table")
	}
	return nil
}

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

func (s *MysqlBackupService) backupArtifactNamePrefix(ctx context.Context, inst *model.MysqlBackupInstance) (string, error) {
	projectName := fmt.Sprintf("project_%d", inst.ProjectID)
	if proj, err := s.projectRepo.GetByID(ctx, inst.ProjectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}
	return mysqlbackup.BuildArtifactNamePrefix(projectName, inst.MysqlHost, inst.MysqlPort), nil
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

func (s *MysqlBackupService) startPollBackupJobLog(ctx context.Context, jobID uint, sshCli *sshclient.Client, logPath, prefix string) context.CancelFunc {
	pollCtx, cancel := context.WithCancel(ctx)
	go func() {
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
	}()
	return cancel
}

func (s *MysqlBackupService) decryptInstancePassword(inst *model.MysqlBackupInstance) (string, error) {
	if inst == nil || inst.EncPassword == "" {
		return "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码，无法执行备份")
	}
	return cryptox.DecryptString(s.aead, inst.EncPassword)
}

func (s *MysqlBackupService) loadInstanceSecrets(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupInstance, string, error) {
	inst, err := s.backupRepo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", constants.ErrNotFound
		}
		return nil, "", bizerrors.Pass(ctx, "mysql.backup", "loadInstanceSecrets", err)
	}
	if inst.EncPassword == "" {
		return nil, "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码")
	}
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, "", bizerrors.Pass(ctx, "mysql.backup", "loadInstanceSecrets", err)
	}
	return inst, pw, nil
}

func (s *MysqlBackupService) dialServer(ctx context.Context, serverID uint) (*sshclient.Client, *model.Server, error) {
	return sshserver.DialServer(ctx, s.aead, "mysql.backup", s.serverRepo, serverID)
}

func (s *MysqlBackupService) ensureServerInProject(ctx context.Context, projectID, serverID uint) error {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrLogSourceServerNotFound
		}
		return bizerrors.Pass(ctx, "mysql.backup", "ensureServerInProject", err)
	}
	if sv.ProjectID != projectID {
		return constants.ErrServerNotInCurrentProject
	}
	return nil
}

func (s *MysqlBackupService) toInstanceItem(ctx context.Context, inst model.MysqlBackupInstance) MysqlBackupInstanceItem {
	item := MysqlBackupInstanceItem{
		ID: inst.ID, ProjectID: inst.ProjectID, ServerID: inst.ServerID,
		Name: inst.Name, Enabled: inst.Enabled, MysqlHost: inst.MysqlHost, MysqlPort: inst.MysqlPort,
		MysqlSocket: inst.MysqlSocket, MysqlUser: inst.MysqlUser, BackupMode: inst.BackupMode,
		BackupScope: inst.BackupScope, DatabaseName: inst.DatabaseName, TableName: inst.BackupTable,
		DatabaseNames: inst.DatabaseNames, RemoteDataDir: inst.RemoteDataDir, RemoteLogDir: inst.RemoteLogDir,
		MysqlDataDir:  inst.MysqlDataDir,
		UploadToMinio: inst.UploadToMinio, MysqldumpWorkDir: inst.MysqldumpWorkDir,
		MysqldumpExtraArgs: inst.MysqldumpExtraArgs, MysqldumpBin: inst.MysqldumpBin,
		XtrabackupTool: inst.XtrabackupTool, XtrabackupBin: inst.XtrabackupBin, InnobackupexBin: inst.InnobackupexBin,
		ScheduleEnabled: inst.ScheduleEnabled, CronSpec: inst.CronSpec,
		NotifyEnabled: inst.NotifyEnabled,
	}
	notifyIDs := parseNotifyUserIDs(inst.NotifyUserIDs)
	item.NotifyUserIDs = notifyIDs
	item.NotifyUsers = s.resolveNotifyUserBriefs(ctx, notifyIDs)
	item.MysqldumpOptions = parseMysqldumpOptionsForAPI(inst.MysqldumpOptions)
	if inst.LastScheduledAt != nil && !inst.LastScheduledAt.IsZero() {
		item.LastScheduledAt = inst.LastScheduledAt.Format(time.RFC3339)
	}
	if sv, err := s.serverRepo.GetByID(ctx, inst.ServerID); err == nil {
		item.ServerName = sv.Name
	}
	if !inst.CreatedAt.IsZero() {
		item.CreatedAt = inst.CreatedAt.Format(time.RFC3339)
	}
	if !inst.UpdatedAt.IsZero() {
		item.UpdatedAt = inst.UpdatedAt.Format(time.RFC3339)
	}
	return item
}

func (s *MysqlBackupService) ListMysqldumpOptions() []mysqlbackup.MysqldumpOption {
	return mysqlbackup.MysqldumpOptionCatalog
}

func marshalMysqldumpOptionIDs(ids []string) string {
	if len(ids) == 0 {
		bs, _ := json.Marshal(mysqlbackup.DefaultMysqldumpOptionIDs())
		return string(bs)
	}
	bs, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(bs)
}

func parseMysqldumpOptionsForAPI(raw string) []string {
	ids, err := mysqlbackup.ParseMysqldumpOptionIDs(raw)
	if err != nil {
		return mysqlbackup.DefaultMysqldumpOptionIDs()
	}
	return ids
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (s *MysqlBackupService) tailRemoteFile(ctx context.Context, sshCli *sshclient.Client, path string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	script := fmt.Sprintf(`tail -n %d %q 2>/dev/null || true`, lines, path)
	res, err := sshCli.Exec(ctx, script, 65536)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout + res.Stderr), nil
}
