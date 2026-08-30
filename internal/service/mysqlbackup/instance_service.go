// 备份实例维度的读写：列表、Upsert、删除、连通性探测、备份范围校验与实例凭据解密。
// 任务编排见 job_runner.go；对象存储命名见 artifact.go。
package mysqlbackup

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/mysqlbackup"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

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

	passwordTouched := false
	if pw := strings.TrimSpace(req.MysqlPassword); pw != "" {
		enc, err := cryptox.EncryptString(s.aead, pw)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
		inst.EncPassword = enc
		passwordTouched = true
	} else if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("新建实例须填写 mysql_password")
	}

	if id > 0 {
		// 未提交新密码时禁止 Save 覆盖 EncPassword（避免表单空串/零值把库中密文清空）。
		if err := s.backupRepo.UpdateInstance(ctx, inst, passwordTouched); err != nil {
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
	if strings.TrimSpace(inst.EncPassword) == "" {
		return nil, "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码，请编辑实例并保存密码后再试")
	}
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, "", constants.ErrBadRequestWithMsg("MySQL 密码解密失败，请重新编辑实例并填写密码后保存")
	}
	return inst, pw, nil
}

func (s *MysqlBackupService) ensureServerInProject(ctx context.Context, projectID, serverID uint) error {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrServerNotFound
		}
		return bizerrors.Pass(ctx, "mysql.backup", "ensureServerInProject", err)
	}
	if sv.ProjectID != projectID {
		return constants.ErrServerNotInCurrentProject
	}
	return nil
}
