package dbmgmt

import (
	"context"
	"fmt"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/sshserver"

	"crypto/cipher"

	"golang.org/x/crypto/ssh"
)

func dialServerSSH(ctx context.Context, aead cipher.AEAD, repo sshserver.CredentialReader, serverID uint) (*ssh.Client, *model.Server, error) {
	sv, err := repo.GetByID(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	cred, err := repo.GetCredentialByServerID(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := sshserver.DecryptCredentialToSSHConfig(ctx, aead, "dbmgmt", *sv, *cred)
	if err != nil {
		return nil, nil, err
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, port)
	dialer := &ssh.ClientConfig{
		User:            cfg.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // 与 sshclient 一致，内网跳板
		Timeout:         15 * time.Second,
	}
	switch cfg.AuthType {
	case "password":
		dialer.Auth = []ssh.AuthMethod{ssh.Password(cfg.Password)}
	default:
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err != nil {
			return nil, nil, err
		}
		dialer.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}
	cli, err := ssh.Dial("tcp", addr, dialer)
	if err != nil {
		return nil, nil, err
	}
	return cli, sv, nil
}
