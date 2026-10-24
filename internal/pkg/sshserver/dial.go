package sshserver

import (
	"context"
	"crypto/cipher"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/sshclient"

	"gorm.io/gorm"
)

// CredentialReader 读取服务器 SSH 凭据（ServerRepository 子集）。
type CredentialReader interface {
	GetByID(ctx context.Context, id uint) (*model.Server, error)
	GetCredentialByServerID(ctx context.Context, serverID uint) (*model.ServerCredential, error)
}

// DecryptCredentialToSSHConfig 解密 ServerCredential 为 sshclient.Config。
func DecryptCredentialToSSHConfig(ctx context.Context, aead cipher.AEAD, domain string, sv model.Server, cred model.ServerCredential) (sshclient.Config, error) {
	cfg := sshclient.Config{
		Host:     sv.Host,
		Port:     sv.Port,
		Username: cred.Username,
	}
	switch strings.ToLower(strings.TrimSpace(cred.AuthType)) {
	case "password":
		if cred.EncPassword == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg666b6d7186e5)
		}
		pw, err := cryptox.DecryptString(aead, *cred.EncPassword)
		if err != nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHCredentialDecryptFailed)
		}
		cfg.AuthType = sshclient.AuthPassword
		cfg.Password = pw
	case "key":
		if cred.EncPrivateKey == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg298c7d5f0d54)
		}
		pk, err := cryptox.DecryptString(aead, *cred.EncPrivateKey)
		if err != nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHCredentialDecryptFailed)
		}
		cfg.AuthType = sshclient.AuthKey
		cfg.PrivateKey = pk
		if cred.EncPassphrase != nil {
			pp, err := cryptox.DecryptString(aead, *cred.EncPassphrase)
			if err != nil {
				return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHCredentialDecryptFailed)
			}
			cfg.Passphrase = pp
		}
	default:
		return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsge9e731f82ff9)
	}
	return cfg, nil
}

// DialServer 连接目标服务器 SSH。
func DialServer(ctx context.Context, aead cipher.AEAD, domain string, repo CredentialReader, serverID uint) (*sshclient.Client, *model.Server, error) {
	sv, err := repo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, constants.ErrServerNotFound
		}
		return nil, nil, bizerrors.Pass(ctx, domain, "DialServer", err)
	}
	cred, err := repo.GetCredentialByServerID(ctx, serverID)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
	}
	cfg, err := DecryptCredentialToSSHConfig(ctx, aead, domain, *sv, *cred)
	if err != nil {
		return nil, nil, err
	}
	cli, err := sshclient.Dial(ctx, cfg)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	return cli, sv, nil
}
