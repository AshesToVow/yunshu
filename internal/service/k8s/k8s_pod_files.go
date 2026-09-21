// Pod 容器内文件操作：列目录、下载、删除、上传。
// 路径一律经 k8sutil.ValidatePodContainerPath 校验，上传/下载受 maxPodUploadBytes /
// maxPodDownloadBytes（见 k8s_runtime_credential.go）限制。

package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
)

// ListFiles 查询列表相关的业务逻辑。
func (s *K8sPodService) ListFiles(ctx context.Context, query PodFileQuery) ([]PodFileItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	dirPath := strings.TrimSpace(query.Path)
	if dirPath == "" {
		dirPath = "/"
	}
	if err := k8sutil.ValidatePodContainerPath(dirPath); err != nil {
		return nil, err
	}
	container, err := s.resolveExecContainer(ctx, k, query.Namespace, query.Name, query.Container)
	if err != nil {
		return nil, err
	}
	files, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(container).
		ListAllFiles(dirPath)
	if err != nil {
		return nil, mapPodFileError(err, "列出目录")
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
	filePath := strings.TrimSpace(query.Path)
	if err := k8sutil.ValidatePodContainerPath(filePath); err != nil {
		return nil, err
	}
	container, err := s.resolveExecContainer(ctx, k, query.Namespace, query.Name, query.Container)
	if err != nil {
		return nil, err
	}
	podCtl := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(container)
	if err := rejectOversizedPodFile(podCtl.ListAllFiles, filePath, maxPodDownloadBytes); err != nil {
		return nil, err
	}
	data, err := podCtl.DownloadFile(filePath)
	if err != nil {
		return nil, mapPodFileError(err, "下载文件")
	}
	if len(data) > maxPodDownloadBytes {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("文件超过 %d MiB 下载上限", maxPodDownloadBytes>>20))
	}
	return data, nil
}

// DeleteFile 删除相关的业务逻辑。
func (s *K8sPodService) DeleteFile(ctx context.Context, query PodFileQuery) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	if err := assertK8sWritable(ctx, cluster, "exec", query.Namespace); err != nil {
		return err
	}
	filePath := strings.TrimSpace(query.Path)
	if err := k8sutil.ValidatePodContainerPath(filePath); err != nil {
		return err
	}
	container, err := s.resolveExecContainer(ctx, k, query.Namespace, query.Name, query.Container)
	if err != nil {
		return err
	}
	if _, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(container).
		DeleteFile(filePath); err != nil {
		return mapPodFileError(err, "删除文件")
	}
	return nil
}

// UploadFile 执行对应的业务逻辑。
func (s *K8sPodService) UploadFile(ctx context.Context, query PodFileQuery, filename string, r io.Reader) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	if err := assertK8sWritable(ctx, cluster, "exec", query.Namespace); err != nil {
		return err
	}
	dest := strings.TrimSpace(query.Path)
	if dest == "" {
		dest = "/tmp"
	}
	if err := k8sutil.ValidatePodContainerPath(dest); err != nil {
		return err
	}
	container, err := s.resolveExecContainer(ctx, k, query.Namespace, query.Name, query.Container)
	if err != nil {
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
		ContainerName(container).
		UploadFile(dest, tmp); err != nil {
		return mapPodFileError(err, "上传文件")
	}
	return nil
}

// rejectOversizedPodFile 下载前用父目录 ls 预判大小，避免超大文件 tar 整包进内存。
// 列目录失败时放行，仍由 DownloadFile 后的字节上限兜底。
func rejectOversizedPodFile(listFn func(string) ([]*kom.FileInfo, error), filePath string, maxBytes int64) error {
	base := path.Base(filePath)
	if base == "." || base == "/" || base == "" {
		return nil
	}
	parent := path.Dir(filePath)
	if parent == "." || parent == "" {
		parent = "/"
	}
	files, err := listFn(parent)
	if err != nil {
		return nil
	}
	for _, f := range files {
		if f == nil || f.IsDir {
			continue
		}
		if f.Name == base || path.Base(f.Path) == base {
			if f.Size > maxBytes {
				return constants.ErrBadRequestWithMsg(fmt.Sprintf("文件超过 %d MiB 下载上限", maxBytes>>20))
			}
			return nil
		}
	}
	return nil
}

// mapPodFileError 将容器内 exec/ls/tar 失败转为可读业务错误，避免一律 500。
func mapPodFileError(err error, action string) error {
	if err == nil {
		return nil
	}
	if mapped := k8sMapAPIError(err); mapped != err {
		if _, ok := bizerrors.As(mapped); ok {
			return mapped
		}
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "executable file not found"),
		strings.Contains(msg, "command not found"),
		strings.Contains(msg, "no such file or directory") &&
			(strings.Contains(msg, "ls") || strings.Contains(msg, "tar") || strings.Contains(msg, "rm")):
		return constants.ErrBadRequestWithMsg(
			action + "失败：容器内无 ls/shell（常见于 distroless、scratch、部分系统组件镜像如 kube-proxy），无法使用文件管理。请换用带基础工具的业务 Pod，或使用 kubectl debug。",
		)
	case strings.Contains(msg, "a container name must be specified"):
		return constants.ErrBadRequestWithMsg(action + "失败：该 Pod 有多个容器，请指定容器名")
	case strings.Contains(msg, "container not found"):
		return constants.ErrBadRequestWithMsg(action + "失败：指定容器不存在")
	}
	detail := strings.TrimSpace(err.Error())
	if len(detail) > 280 {
		detail = detail[:280] + "…"
	}
	return constants.ErrBadRequestWithMsg(action + "失败: " + detail)
}
