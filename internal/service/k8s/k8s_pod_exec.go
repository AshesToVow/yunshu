package k8s

// Pod 命令执行相关：一次性 Exec、容器名解析、交互式 TTY 流。

import (
	"context"
	"io"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	kom "github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// Exec 执行对应的业务逻辑。
func (s *K8sPodService) Exec(ctx context.Context, req PodExecRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	container, err := s.resolveExecContainer(ctx, k, req.Namespace, req.Name, req.Container)
	if err != nil {
		return "", err
	}
	cmd := strings.Fields(strings.TrimSpace(req.Command))
	if len(cmd) == 0 {
		return "", constants.ErrBadRequestWithMsg(constants.ErrMsgeddb4f63e4c7)
	}

	var out []byte
	err = k.WithContext(ctx).Namespace(req.Namespace).Name(req.Name).Ctl().Pod().ContainerName(container).Command(cmd[0], cmd[1:]...).Execute(&out).Error
	if err != nil {
		return "", bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmt1493bc1ea07a)
	}
	return string(out), nil
}

func (s *K8sPodService) resolveExecContainer(ctx context.Context, k *kom.Kubectl, namespace, podName, container string) (string, error) {
	ns := strings.TrimSpace(namespace)
	name := strings.TrimSpace(podName)
	want := strings.TrimSpace(container)
	if ns == "" || name == "" {
		return "", constants.ErrBadRequestWithMsg(constants.ErrMsge278df185255)
	}
	var pod corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).Name(name).Get(&pod).Error; err != nil {
		return "", bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmtc52b9130d74c)
	}
	if want != "" {
		for _, c := range pod.Spec.Containers {
			if c.Name == want {
				return want, nil
			}
		}
		for _, c := range pod.Spec.InitContainers {
			if c.Name == want {
				return want, nil
			}
		}
		for _, c := range pod.Spec.EphemeralContainers {
			if c.Name == want {
				return want, nil
			}
		}
		return "", constants.ErrBadRequestWithMsg("指定容器不存在: " + want)
	}
	if len(pod.Spec.Containers) == 0 {
		return "", constants.ErrBadRequestWithMsg("Pod 无可用容器")
	}
	return pod.Spec.Containers[0].Name, nil
}

// ExecTTYStream opens an interactive TTY exec stream to the container.
// 保留 client-go 直连：需要 remotecommand.TerminalSizeQueue 来支持前端终端窗口 resize，
// 当前 kom StreamExecute 不暴露该能力（且默认 TTY=false），因此这里采用最小原生实现。
func (s *K8sPodService) ExecTTYStream(
	ctx context.Context,
	clusterID uint,
	namespace string,
	podName string,
	container string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	sizeQueue remotecommand.TerminalSizeQueue,
) error {
	_, restCfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(podName) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsge278df185255)
	}

	resolvedContainer, err := s.resolveExecContainer(ctx, k, namespace, podName, container)
	if err != nil {
		return err
	}

	req := k.Client().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(strings.TrimSpace(podName)).
		Namespace(strings.TrimSpace(namespace)).
		SubResource("exec")

	cmd := []string{"sh", "-l"}
	execOpts := &corev1.PodExecOptions{
		Container: resolvedContainer,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(execOpts, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", req.URL())
	if err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               true,
		TerminalSizeQueue: sizeQueue,
	})
}
