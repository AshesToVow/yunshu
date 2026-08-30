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
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, stream); err != nil {
		return "", bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmt8e15ae24a3a1)
	}
	return k8sutil.FilterLogLines(buf.String(), query.Keyword, query.StartTime, query.EndTime), nil
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
