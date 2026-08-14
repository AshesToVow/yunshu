package cmdb

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/pkg/sshserver"

	"gorm.io/gorm"
)

const defaultMaxTransferFileMB = 50

// ServerFileListQuery 远端目录列表。
type ServerFileListQuery struct {
	ProjectID uint   `form:"-"`
	ServerID  uint   `form:"-"`
	Path      string `form:"path"`
}

// ServerFilePathQuery 远端文件路径。
type ServerFilePathQuery struct {
	ProjectID uint   `form:"-"`
	ServerID  uint   `form:"-"`
	Path      string `form:"path" binding:"required"`
}

func (s *Service) maxTransferBytes(ctx context.Context) int64 {
	mb := defaultMaxTransferFileMB
	if v, ok := dictconfig.FetchEnabledDictValue(ctx, s.db, "cmdb_max_transfer_file_mb"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			mb = n
		}
	}
	return int64(mb) * 1024 * 1024
}

func (s *Service) MaxTransferFileMB(ctx context.Context) int {
	return int(s.maxTransferBytes(ctx) / (1024 * 1024))
}

func validateRemotePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/", nil
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if strings.Contains(p, "\x00") {
		return "", constants.ErrBadRequestWithMsg("非法路径")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	clean := path.Clean(p)
	if clean == "." {
		clean = "/"
	}
	if strings.Contains(clean, "..") {
		return "", constants.ErrBadRequestWithMsg("路径不允许包含 ..")
	}
	return clean, nil
}

func (s *Service) dialServerSSH(ctx context.Context, projectID, serverID uint) (*sshclient.Client, error) {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrLogSourceServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "dialServerSSH", err)
	}
	if sv.ProjectID != projectID {
		return nil, constants.ErrServerNotInCurrentProject
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "dialServerSSH", err)
	}
	sshCfg, err := sshserver.DecryptCredentialToSSHConfig(ctx, s.aead, "cmdb", *sv, *cred)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "dialServerSSH", err)
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cli, err := sshclient.Dial(cctx, sshCfg)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	return cli, nil
}

// ListServerFiles 列出服务器远端目录。
func (s *Service) ListServerFiles(ctx context.Context, q ServerFileListQuery) ([]sshclient.RemoteFileEntry, error) {
	remote, err := validateRemotePath(q.Path)
	if err != nil {
		return nil, err
	}
	cli, err := s.dialServerSSH(ctx, q.ProjectID, q.ServerID)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	list, err := cli.ListDir(ctx, remote)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("列出远端目录失败: " + err.Error())
	}
	return list, nil
}

// UploadServerFile 上传文件到服务器。
func (s *Service) UploadServerFile(ctx context.Context, projectID, serverID uint, remoteDir, filename string, r io.Reader, sizeHint int64) error {
	dir, err := validateRemotePath(remoteDir)
	if err != nil {
		return err
	}
	filename = path.Base(strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/"))
	if filename == "" || filename == "." || filename == "/" {
		return constants.ErrBadRequestWithMsg("文件名无效")
	}
	maxBytes := s.maxTransferBytes(ctx)
	if sizeHint > 0 && sizeHint > maxBytes {
		return constants.ErrBadRequestWithMsg("文件超过上限 " + strconv.FormatInt(maxBytes/(1024*1024), 10) + "MB（字典 cmdb_max_transfer_file_mb）")
	}
	remotePath := path.Join(dir, filename)
	cli, err := s.dialServerSSH(ctx, projectID, serverID)
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.UploadReader(ctx, remotePath, r, maxBytes, 0o644)
	if err != nil {
		return constants.ErrBadRequestWithMsg("上传失败: " + err.Error())
	}
	return nil
}

// DownloadServerFile 下载服务器文件到 w。
func (s *Service) DownloadServerFile(ctx context.Context, projectID, serverID uint, remotePath string, w io.Writer) (string, error) {
	remote, err := validateRemotePath(remotePath)
	if err != nil {
		return "", err
	}
	if remote == "/" {
		return "", constants.ErrBadRequestWithMsg("请指定文件路径")
	}
	cli, err := s.dialServerSSH(ctx, projectID, serverID)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	maxBytes := s.maxTransferBytes(ctx)
	if _, err := cli.DownloadTo(ctx, remote, w, maxBytes); err != nil {
		return "", constants.ErrBadRequestWithMsg("下载失败: " + err.Error())
	}
	return path.Base(remote), nil
}

// DeleteServerFile 删除远端文件（非目录）。
func (s *Service) DeleteServerFile(ctx context.Context, projectID, serverID uint, remotePath string) error {
	remote, err := validateRemotePath(remotePath)
	if err != nil {
		return err
	}
	if remote == "/" {
		return constants.ErrBadRequestWithMsg("不允许删除根目录")
	}
	cli, err := s.dialServerSSH(ctx, projectID, serverID)
	if err != nil {
		return err
	}
	defer cli.Close()
	if err := cli.RemoveRemoteFile(remote); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return constants.ErrNotFoundWithMsg("文件不存在")
		}
		return constants.ErrBadRequestWithMsg("删除失败: " + err.Error())
	}
	return nil
}
