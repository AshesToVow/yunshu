package cmdb

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/pkg/sshserver"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// ListServers 分页列出项目下的服务器。
func (s *Service) ListServers(ctx context.Context, q ServerListQuery) (*pagination.Result[ServerItem], error) {
	if err := s.ensureDefaultServerGroups(ctx, q.ProjectID); err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListServers", err)
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	// If cloud_account_id provided, resolve to its group_id for server filtering.
	groupID := q.GroupID
	if q.CloudAccountID != nil {
		acc, err := s.cloudAccountRepo.GetByID(ctx, *q.CloudAccountID)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "ListServers", err)
		}
		if acc.ProjectID != q.ProjectID {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg053a6a395b16)
		}
		gid := acc.GroupID
		groupID = &gid
	}

	var visibleIDs []uint
	unrestricted, ids, err := s.visibleServerScope(ctx, q.ProjectID, q.Actor)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListServers", err)
	}
	if !unrestricted {
		visibleIDs = ids
	}

	list, total, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID:  q.ProjectID,
		Keyword:    strings.TrimSpace(q.Keyword),
		GroupID:    groupID,
		SourceType: strings.TrimSpace(q.SourceType),
		Provider:   strings.TrimSpace(q.Provider),
		ServerIDs:  visibleIDs,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListServers", err)
	}
	out := make([]ServerItem, 0, len(list))
	for _, it := range list {
		out = append(out, toServerItem(it))
	}
	return &pagination.Result[ServerItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// UpsertServer 创建或更新服务器及可选凭据。
func (s *Service) UpsertServer(ctx context.Context, req ServerUpsertRequest) (*ServerItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, req.ProjectID); err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
	}
	if req.Port <= 0 {
		req.Port = 22
	}
	osType := strings.TrimSpace(req.OSType)
	if osType == "" {
		osType = "linux"
	}
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	var sv *model.Server
	var err error
	if req.ID != nil && *req.ID > 0 {
		sv, err = s.serverRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrServerNotFound
			}
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
		}
	} else {
		sv = &model.Server{}
	}

	sv.ProjectID = req.ProjectID
	sv.GroupID = req.GroupID
	sv.Name = strings.TrimSpace(req.Name)
	sv.Host = strings.TrimSpace(req.Host)
	sv.Port = req.Port
	sv.OSType = osType
	sv.Tags = strings.TrimSpace(req.Tags)
	sv.Status = status
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = model.ServerGroupCategorySelfHosted
	}
	sv.SourceType = sourceType
	sv.Provider = strings.TrimSpace(req.Provider)
	sv.CloudInstanceID = strings.TrimSpace(req.CloudInstanceID)
	sv.CloudRegion = strings.TrimSpace(req.CloudRegion)

	if sv.ID == 0 {
		if err := s.serverRepo.Create(ctx, sv); err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
		}
	} else {
		if err := s.serverRepo.Save(ctx, sv); err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
		}
	}

	// credential optional: only upsert when username provided
	if strings.TrimSpace(req.Username) != "" && strings.TrimSpace(req.AuthType) != "" {
		var existingCred *model.ServerCredential
		if sv.ID > 0 {
			c, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
				}
			} else {
				existingCred = c
			}
		}
		cred, err := s.buildCredentialForSave(ctx, *sv, req, existingCred)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
		}
		if err := s.serverRepo.UpsertCredential(ctx, cred); err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServer", err)
		}
	}

	item := toServerItem(*sv)
	return &item, nil
}

func (s *Service) buildCredentialForSave(ctx context.Context, server model.Server, req ServerUpsertRequest, existing *model.ServerCredential) (*model.ServerCredential, error) {
	authType := strings.ToLower(strings.TrimSpace(req.AuthType))
	if authType == "" {
		authType = "password"
	}
	cred := &model.ServerCredential{ServerID: server.ID, AuthType: authType, Username: strings.TrimSpace(req.Username), KeyVersion: 1}
	if ul := strings.TrimSpace(req.UsernameDictLabel); ul != "" {
		cred.UsernameDictLabel = &ul
	}
	if pl := strings.TrimSpace(req.PasswordDictLabel); pl != "" {
		cred.PasswordDictLabel = &pl
	}
	switch authType {
	case "password":
		if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
			enc, err := cryptox.EncryptString(s.aead, *req.Password)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "cmdb", "buildCredentialForSave", err)
			}
			cred.EncPassword = &enc
		} else if existing != nil && existing.EncPassword != nil && strings.TrimSpace(*existing.EncPassword) != "" {
			cred.EncPassword = existing.EncPassword
		} else {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg3aa7a9e035a4)
		}
	case "key":
		if req.PrivateKey != nil && strings.TrimSpace(*req.PrivateKey) != "" {
			encKey, err := cryptox.EncryptString(s.aead, *req.PrivateKey)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "cmdb", "buildCredentialForSave", err)
			}
			cred.EncPrivateKey = &encKey
		} else if existing != nil && existing.EncPrivateKey != nil && strings.TrimSpace(*existing.EncPrivateKey) != "" {
			cred.EncPrivateKey = existing.EncPrivateKey
		} else {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg6bd217c30983)
		}
		if req.Passphrase != nil && strings.TrimSpace(*req.Passphrase) != "" {
			encPP, err := cryptox.EncryptString(s.aead, *req.Passphrase)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "cmdb", "buildCredentialForSave", err)
			}
			cred.EncPassphrase = &encPP
		} else if existing != nil {
			cred.EncPassphrase = existing.EncPassphrase
		}
	default:
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgb6ada5b863ef)
	}
	return cred, nil
}

// DeleteServer 按 ID 删除服务器。
func (s *Service) DeleteServer(ctx context.Context, id uint) error {
	return s.serverRepo.DeleteByID(ctx, id)
}

// GetServer 获取服务器详情（含凭据摘要）。
func (s *Service) GetServer(ctx context.Context, id uint) (*ServerDetailItem, error) {
	sv, err := s.serverRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "GetServer", err)
	}
	base := toServerItem(*sv)
	out := &ServerDetailItem{ServerItem: base}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return out, nil
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "GetServer", err)
	}
	out.AuthType = cred.AuthType
	out.Username = cred.Username
	out.PasswordSet = cred.EncPassword != nil && strings.TrimSpace(*cred.EncPassword) != ""
	out.PrivateKeySet = cred.EncPrivateKey != nil && strings.TrimSpace(*cred.EncPrivateKey) != ""
	out.UsernameDictLabel = cloneStrPtr(cred.UsernameDictLabel)
	out.PasswordDictLabel = cloneStrPtr(cred.PasswordDictLabel)
	return out, nil
}

func cloneStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	cp := v
	return &cp
}

// ExecServerCommand 在目标服务器上执行单次 SSH 命令。
func (s *Service) ExecServerCommand(ctx context.Context, req ServerExecRequest) (*ServerExecResult, error) {
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "ExecServerCommand", err)
	}
	if sv.ProjectID != req.ProjectID {
		return nil, constants.ErrServerNotInCurrentProject
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "ExecServerCommand", err)
	}

	sshCfg, err := sshserver.DecryptCredentialToSSHConfig(ctx, s.aead, "cmdb", *sv, *cred)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ExecServerCommand", err)
	}

	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	if timeoutSec > 120 {
		timeoutSec = 120
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cli, err := sshclient.Dial(cctx, sshCfg)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	defer cli.Close()

	cmd := strings.TrimSpace(req.Command)
	result, err := cli.Exec(cctx, cmd, 256*1024)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHExecFailedPrefix + err.Error())
	}
	return &ServerExecResult{
		ServerID:   sv.ID,
		Command:    cmd,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMS: result.Duration.Milliseconds(),
		Truncated:  result.Truncated,
	}, nil
}

// StreamServerTerminal 建立交互式 SSH 终端流。
func (s *Service) StreamServerTerminal(ctx context.Context, projectID, serverID uint, stdin io.Reader, stdout, stderr io.Writer, sizes <-chan sshclient.TerminalSize) error {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrServerNotFound
		}
		return bizerrors.Pass(ctx, "cmdb", "StreamServerTerminal", err)
	}
	if sv.ProjectID != projectID {
		return constants.ErrServerNotInCurrentProject
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
		}
		return bizerrors.Pass(ctx, "cmdb", "StreamServerTerminal", err)
	}
	sshCfg, err := sshserver.DecryptCredentialToSSHConfig(ctx, s.aead, "cmdb", *sv, *cred)
	if err != nil {
		return bizerrors.Pass(ctx, "cmdb", "StreamServerTerminal", err)
	}
	cli, err := sshclient.Dial(ctx, sshCfg)
	if err != nil {
		return constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	defer cli.Close()
	return cli.ShellStream(ctx, stdin, stdout, stderr, sizes)
}

