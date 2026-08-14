package sshclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// RemoteFileEntry 远端目录项。
type RemoteFileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
}

// ListDir 列出远端目录。
func (c *Client) ListDir(ctx context.Context, remoteDir string) ([]RemoteFileEntry, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("ssh client not connected")
	}
	remoteDir = normalizeRemotePath(remoteDir)
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return nil, err
	}
	defer sftpClient.Close()

	done := make(chan struct {
		list []RemoteFileEntry
		err  error
	}, 1)
	safeGo(func() {
		infos, listErr := sftpClient.ReadDir(remoteDir)
		if listErr != nil {
			done <- struct {
				list []RemoteFileEntry
				err  error
			}{err: listErr}
			return
		}
		out := make([]RemoteFileEntry, 0, len(infos))
		for _, info := range infos {
			name := info.Name()
			out = append(out, RemoteFileEntry{
				Name:    name,
				Path:    path.Join(remoteDir, name),
				IsDir:   info.IsDir(),
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().UTC().Format(time.RFC3339),
			})
		}
		done <- struct {
			list []RemoteFileEntry
			err  error
		}{list: out}
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-done:
		return res.list, res.err
	}
}

// UploadReader 通过 SFTP 流式上传，最多写入 maxBytes（<=0 不限制）。
func (c *Client) UploadReader(ctx context.Context, remotePath string, r io.Reader, maxBytes int64, perm os.FileMode) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("ssh client not connected")
	}
	remotePath = normalizeRemotePath(remotePath)
	if remotePath == "" || remotePath == "/" {
		return 0, fmt.Errorf("invalid remote path")
	}
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()

	dir := path.Dir(remotePath)
	if dir != "" && dir != "." && dir != "/" {
		if err := sftpClient.MkdirAll(dir); err != nil {
			return 0, err
		}
	}
	dst, err := sftpClient.Create(remotePath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	src := io.Reader(r)
	if maxBytes > 0 {
		src = io.LimitReader(r, maxBytes+1)
	}
	done := make(chan struct {
		n   int64
		err error
	}, 1)
	safeGo(func() {
		n, copyErr := io.Copy(dst, src)
		if copyErr == nil && maxBytes > 0 && n > maxBytes {
			_ = sftpClient.Remove(remotePath)
			done <- struct {
				n   int64
				err error
			}{n: n, err: fmt.Errorf("file exceeds max size %d bytes", maxBytes)}
			return
		}
		if copyErr == nil && perm != 0 {
			copyErr = sftpClient.Chmod(remotePath, perm)
		}
		done <- struct {
			n   int64
			err error
		}{n: n, err: copyErr}
	})
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-done:
		return res.n, res.err
	}
}

// DownloadTo 将远端文件流式写入 w，返回字节数；超过 maxBytes 则报错。
func (c *Client) DownloadTo(ctx context.Context, remotePath string, w io.Writer, maxBytes int64) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("ssh client not connected")
	}
	remotePath = normalizeRemotePath(remotePath)
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()

	st, err := sftpClient.Stat(remotePath)
	if err != nil {
		return 0, err
	}
	if st.IsDir() {
		return 0, fmt.Errorf("path is a directory")
	}
	if maxBytes > 0 && st.Size() > maxBytes {
		return 0, fmt.Errorf("file exceeds max size %d bytes", maxBytes)
	}
	src, err := sftpClient.Open(remotePath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	done := make(chan struct {
		n   int64
		err error
	}, 1)
	safeGo(func() {
		n, copyErr := io.Copy(w, src)
		done <- struct {
			n   int64
			err error
		}{n: n, err: copyErr}
	})
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-done:
		return res.n, res.err
	}
}

func normalizeRemotePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	clean := path.Clean(p)
	if clean == "." {
		return "/"
	}
	return clean
}
