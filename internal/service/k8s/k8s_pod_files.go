package k8s

// Pod 容器内文件操作：列目录、下载、删除、上传。
// 路径一律经 k8sutil.ValidatePodContainerPath 校验，上传/下载受 maxPodUploadBytes /
// maxPodDownloadBytes（见 k8s_runtime_credential.go）限制。

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sutil"
)

// ListFiles 查询列表相关的业务逻辑。
func (s *K8sPodService) ListFiles(ctx context.Context, query PodFileQuery) ([]PodFileItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(query.Path)
	if path == "" {
		path = "/"
	}
	if err := k8sutil.ValidatePodContainerPath(path); err != nil {
		return nil, err
	}
	files, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		ListAllFiles(path)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmt85e9441edd38)
	}
	out := make([]PodFileItem, 0, len(files))
	for _, f := range files {
		if f == nil {
			continue
		}
		out = append(out, PodFileItem{
			Name:        f.Name,
			Path:        f.Path,
			Type:        f.Type,
			IsDir:       f.IsDir,
			Size:        f.Size,
			Permissions: f.Permissions,
			Owner:       f.Owner,
			Group:       f.Group,
			ModTime:     f.ModTime,
		})
	}
	return out, nil
}

// ReadFile 执行对应的业务逻辑。
func (s *K8sPodService) ReadFile(ctx context.Context, query PodFileQuery) ([]byte, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(query.Path)
	if err := k8sutil.ValidatePodContainerPath(path); err != nil {
		return nil, err
	}
	data, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		DownloadFile(path)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmt2b7dfae8ff2c)
	}
	if len(data) > maxPodDownloadBytes {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("文件超过 %d MiB 下载上限", maxPodDownloadBytes>>20))
	}
	return data, nil
}

// DeleteFile 删除相关的业务逻辑。
func (s *K8sPodService) DeleteFile(ctx context.Context, query PodFileQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(query.Path)
	if err := k8sutil.ValidatePodContainerPath(path); err != nil {
		return err
	}
	if _, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		DeleteFile(path); err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}

// UploadFile 执行对应的业务逻辑。
func (s *K8sPodService) UploadFile(ctx context.Context, query PodFileQuery, filename string, r io.Reader) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	if err := assertK8sWritable(ctx, cluster, "upload", query.Namespace); err != nil {
		return err
	}
	dest := strings.TrimSpace(query.Path)
	if dest == "" {
		dest = "/tmp"
	}
	if err := k8sutil.ValidatePodContainerPath(dest); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "pod-upload-*"+filepath.Ext(filename))
	if err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	limited := io.LimitReader(r, maxPodUploadBytes+1)
	n, err := io.Copy(tmp, limited)
	if err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	if n > maxPodUploadBytes {
		return constants.ErrBadRequestWithMsg("上传文件超过 32MiB 上限")
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	if err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		UploadFile(dest, tmp); err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}
