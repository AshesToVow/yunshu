package dbmgmt

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/sshserver"

	"crypto/cipher"

	"golang.org/x/crypto/ssh"
)

func dialServerSSH(ctx context.Context, aead cipher.AEAD, repo sshserver.CredentialReader, serverID uint) (*ssh.Client, *model.Server, error) {
	cli, sv, err := sshserver.DialServer(ctx, aead, "dbmgmt", repo, serverID)
	if err != nil {
		return nil, nil, err
	}
	if cli == nil || cli.SSHClient() == nil {
		return nil, sv, err
	}
	return cli.SSHClient(), sv, nil
}
