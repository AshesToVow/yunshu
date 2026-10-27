package k8s

// Pod 日志读取：一次性拉取与 follow 流式推送。

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sutil"
)

// GetLogs 获取相关的业务逻辑。
func (s *K8sPodService) GetLogs(ctx context.Context, query PodLogsQuery) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return "", err
	}
	opts, err := buildPodLogOptions(query, false)
	if err != nil {
		return "", err
	}
	var stream io.ReadCloser
	err = k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		GetLogs(&stream, opts).Error
	if err != nil {
		return "", bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmtd868fafc39ca)
	}
	if stream == nil {
		return "", constants.ErrInternalWithMsg(constants.ErrMsgf53aee0dab26)
	}
	defer stream.Close()
	// 未指定 tail 时日志可能极大，限制单次读取上限，避免一次性把整段日志读进内存。
	const maxPodLogBytes = 32 << 20
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, io.LimitReader(stream, maxPodLogBytes+1)); err != nil {
		return "", bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmt8e15ae24a3a1)
	}
	truncated := buf.Len() > maxPodLogBytes
	raw := buf.Bytes()
	if truncated {
		raw = raw[:maxPodLogBytes]
		if idx := bytes.LastIndexByte(raw, '\n'); idx > 0 {
			raw = raw[:idx+1]
		}
	}
	text := strings.ToValidUTF8(string(raw), "")
	filtered := k8sutil.FilterLogLines(text, query.Keyword, query.StartTime, query.EndTime)
	if truncated {
		filtered += "\n... 日志内容超过 32 MiB，仅展示前 32 MiB，可使用 tail 或时间范围缩小查询"
	}
	return filtered, nil
}

// StreamLogs 执行对应的业务逻辑。
func (s *K8sPodService) StreamLogs(ctx context.Context, query PodLogsQuery, onLine func(string) error) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	opts, err := buildPodLogOptions(query, true)
	if err != nil {
		return err
	}
	var stream io.ReadCloser
	err = k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		GetLogs(&stream, opts).Error
	if err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	if stream == nil {
		return constants.ErrInternalWithMsg(constants.ErrMsgf59e8e01bf0d)
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, e := reader.ReadString('\n')
			if line != "" {
				if cbErr := onLine(line); cbErr != nil {
					return cbErr
				}
			}
			if e != nil {
				if e == io.EOF {
					return nil
				}
				return e
			}
		}
	}
}
