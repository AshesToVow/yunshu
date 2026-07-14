package logplatform

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunshu/internal/config"
	"yunshu/internal/service/k8s"
)

const (
	deployModeBinary = "binary"
	deployModeK8s    = "k8s"

	defaultLoggieK8sNamespace = "loggie"
	defaultLoggieDaemonSet    = "loggie"
	defaultLoggieSinkName     = "yunshu-es"
)

type LoggieK8sBundle struct {
	NamespaceYAML        string `json:"namespace_yaml"`
	SinkYAML             string `json:"sink_yaml"`
	ClusterLogConfigYAML string `json:"cluster_log_config_yaml"`
	CombinedManifest     string `json:"combined_manifest"`
	Namespace            string `json:"namespace"`
	DaemonSetName        string `json:"daemonset_name"`
	SinkName             string `json:"sink_name"`
	ClusterLogConfigName string `json:"cluster_log_config_name"`
	ClusterID            uint   `json:"cluster_id"`
}

func normalizeDeployMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case deployModeK8s, "kubernetes", "pod", "daemonset":
		return deployModeK8s
	default:
		return deployModeBinary
	}
}

func defaultK8sNamespace(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return defaultLoggieK8sNamespace
	}
	return ns
}

func defaultK8sDaemonSet(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultLoggieDaemonSet
	}
	return name
}

func clusterLogConfigName(projectID uint) string {
	return fmt.Sprintf("yunshu-p%d-container-logs", projectID)
}

// BuildK8sLoggieBundle 生成 Namespace + Sink + ClusterLogConfig 清单。
// requirePodLabel=true 时仅采集带 yunshu.project_id=<projectID> 的 Pod；
// false（默认）则采集集群内全部 Pod stdout，并固定写入 project_id。
func BuildK8sLoggieBundle(projectID, clusterID uint, ns, dsName string, esCfg config.ElasticsearchConfig, requirePodLabel bool) LoggieK8sBundle {
	ns = defaultK8sNamespace(ns)
	dsName = defaultK8sDaemonSet(dsName)
	esCfg = esCfg.Normalized()
	indexPattern := strings.TrimSpace(esCfg.IndexPattern)
	if indexPattern == "" {
		indexPattern = "yunshu-logs-*"
	}
	indexSink := strings.TrimSuffix(indexPattern, "*") + "${+YYYY.MM.DD}"

	var hosts []string
	for _, h := range esCfg.Addresses {
		h = strings.TrimSpace(h)
		if h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		hosts = []string{"http://elasticsearch.logging.svc:9200"}
	}

	var hostsYAML strings.Builder
	for _, h := range hosts {
		hostsYAML.WriteString("      - ")
		hostsYAML.WriteString(h)
		hostsYAML.WriteByte('\n')
	}

	authBlock := ""
	if u := strings.TrimSpace(esCfg.Username); u != "" {
		authBlock += fmt.Sprintf("    username: %q\n", u)
	}
	if p := strings.TrimSpace(esCfg.Password); p != "" {
		authBlock += fmt.Sprintf("    password: %q\n", p)
	}

	clcName := clusterLogConfigName(projectID)
	sinkName := defaultLoggieSinkName
	projectKey := fmt.Sprintf("%d", projectID)

	nsYAML := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    app.kubernetes.io/name: loggie
    yunshu.project_id: %q
`, ns, projectKey)

	sinkYAML := fmt.Sprintf(`apiVersion: loggie.io/v1beta1
kind: Sink
metadata:
  name: %s
  namespace: %s
  labels:
    yunshu.project_id: %q
spec:
  sink: |
    type: elasticsearch
    hosts:
%s    index: %s
%s`, sinkName, ns, projectKey, hostsYAML.String(), indexSink, authBlock)

	// project_id 固定写入 Yunshu 项目 ID。
	// Loggie labelSelector 为 map（非 matchExpressions）；缺标签时会 matches no pods。
	selectorBlock := "    type: pod\n"
	if requirePodLabel {
		selectorBlock += fmt.Sprintf(`    labelSelector:
      "yunshu.project_id": %q
`, projectKey)
	}
	clcYAML := fmt.Sprintf(`apiVersion: loggie.io/v1beta1
kind: ClusterLogConfig
metadata:
  name: %s
  labels:
    yunshu.project_id: %q
spec:
  selector:
%s  pipeline:
    sources: |
      - type: file
        name: container-logs
        containerName: "*"
        paths:
          - /var/log/pods/*/*/*.log
        addonMeta: true
    sinkRef: %s
    interceptors: |
      - type: addK8sMeta
        addFields:
          project_id: %q
          server_id: "${pod.labels.yunshu\\.server_id}"
          service_id: "${pod.labels.yunshu\\.service_id}"
          log_source_id: "${pod.labels.yunshu\\.log_source_id}"
          namespace: "${namespace}"
          pod: "${pod.name}"
          container: "${container.name}"
      - type: transformer
        actions:
          - action: copy(state.filename, file_path)
            ignoreError: true
          - action: regex(body)
            pattern: '^(?P<ts>\\S+)\\s+(?P<stream>stdout|stderr)\\s+(?P<flag>\\S)\\s+(?P<message>.*)$'
            ignoreError: true
`, clcName, projectKey, selectorBlock, sinkName, projectKey)

	combined := strings.TrimSpace(nsYAML) + "\n---\n" + strings.TrimSpace(sinkYAML) + "\n---\n" + strings.TrimSpace(clcYAML) + "\n"

	return LoggieK8sBundle{
		NamespaceYAML:        nsYAML,
		SinkYAML:             sinkYAML,
		ClusterLogConfigYAML: clcYAML,
		CombinedManifest:     combined,
		Namespace:            ns,
		DaemonSetName:        dsName,
		SinkName:             sinkName,
		ClusterLogConfigName: clcName,
		ClusterID:            clusterID,
	}
}

func (s *LoggieAgentService) applyK8sLoggieManifest(ctx context.Context, clusterID uint, manifest string) error {
	if s.k8sRuntime == nil {
		return fmt.Errorf("K8s 运行时未配置，无法下发 Loggie CR")
	}
	if clusterID == 0 {
		return fmt.Errorf("cluster_id 必填")
	}
	_, k, err := s.k8sRuntime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return err
	}
	dyn := k8s.NewDynamicResourceService(s.k8sRuntime)
	if err := dyn.ApplyManifest(ctx, k, manifest, nil); err != nil {
		parts := strings.Split(manifest, "\n---\n")
		var last error
		okAny := false
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if err := dyn.ApplyManifest(ctx, k, part, nil); err != nil {
				last = err
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "already exists") {
					okAny = true
					continue
				}
				return err
			}
			okAny = true
		}
		if !okAny && last != nil {
			return last
		}
	}
	return nil
}

func (s *LoggieAgentService) restartLoggieDaemonSet(ctx context.Context, clusterID uint, namespace, name string) error {
	if s.k8sWorkload == nil {
		return fmt.Errorf("K8s Workload 服务未配置，无法重启 Loggie DaemonSet")
	}
	return s.k8sWorkload.DaemonSetRestart(ctx, k8s.NamespacedDetailQuery{
		ClusterID: clusterID,
		Namespace: defaultK8sNamespace(namespace),
		Name:      defaultK8sDaemonSet(name),
	})
}

func (s *LoggieAgentService) probeK8sLoggieDaemonSet(ctx context.Context, clusterID uint, namespace, name string) (ready, desired int32, err error) {
	if s.k8sRuntime == nil {
		return 0, 0, fmt.Errorf("K8s 运行时未配置")
	}
	_, k, err := s.k8sRuntime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return 0, 0, err
	}
	dyn := k8s.NewDynamicResourceService(s.k8sRuntime)
	gvk, _ := dyn.GVKByKind("DaemonSet")
	u, err := dyn.GetByGVK(ctx, k, gvk, defaultK8sNamespace(namespace), defaultK8sDaemonSet(name))
	if err != nil {
		return 0, 0, err
	}
	ready64, _, _ := unstructured.NestedInt64(u.Object, "status", "numberReady")
	desired64, _, _ := unstructured.NestedInt64(u.Object, "status", "desiredNumberScheduled")
	return int32(ready64), int32(desired64), nil
}
