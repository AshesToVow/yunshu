package sshclient

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"github.com/pkg/sftp"
)

// UploadBytes 通过 SFTP 写入远端文件。
func (c *Client) UploadBytes(ctx context.Context, remotePath string, data []byte, perm os.FileMode) error {
	if c == nil || c.client == nil {
		return io.ErrClosedPipe
	}
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	dir := path.Dir(remotePath)
	if dir != "" && dir != "." && dir != "/" {
		if err := sftpClient.MkdirAll(dir); err != nil {
			return err
		}
	}
	dst, err := sftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	done := make(chan error, 1)
	safeGo(func() {
		_, copyErr := io.Copy(dst, bytes.NewReader(data))
		if copyErr == nil && perm != 0 {
			copyErr = sftpClient.Chmod(remotePath, perm)
		}
		done <- copyErr
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
