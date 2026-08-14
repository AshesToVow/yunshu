package logplatform

import (
	"fmt"
	"strconv"
	"strings"
)

// ClusterLogManifestInput DaemonSet 清单渲染入参。
type ClusterLogManifestInput struct {
	ProjectID     uint
	ClusterID     uint
	Namespace     string
	Image         string
	SystemYAML    string
	PipelinesYAML string
}

// RenderClusterLogManifest 生成 Namespace + SA + ConfigMap + DaemonSet YAML（按项目隔离资源名）。
func RenderClusterLogManifest(in ClusterLogManifestInput) string {
	ns := strings.TrimSpace(in.Namespace)
	if ns == "" {
		ns = defaultClusterLogNamespace
	}
	image := strings.TrimSpace(in.Image)
	if image == "" {
		image = "ghcr.io/loggie-io/loggie:v1.7.1"
	}
	app := ClusterLoggieAppName(in.ProjectID)
	cm := ClusterLoggieConfigMapName(in.ProjectID)
	dataPath := fmt.Sprintf("/var/lib/%s", app)
	system := indentYAMLBlock(in.SystemYAML, 4)
	pipelines := indentYAMLBlock(in.PipelinesYAML, 4)
	pid := strconv.FormatUint(uint64(in.ProjectID), 10)
	cid := strconv.FormatUint(uint64(in.ClusterID), 10)
	labels := fmt.Sprintf("app: %s\n    yunshu.io/project-id: %q\n    yunshu.io/cluster-id: %q", app, pid, cid)

	return fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    %s
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
  labels:
    %s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
  labels:
    %s
data:
  loggie.yml: |
%s
  pipelines.yml: |
%s
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: %s
  namespace: %s
  labels:
    %s
spec:
  selector:
    matchLabels:
      app: %s
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: %s
        yunshu.io/project-id: %q
        yunshu.io/cluster-id: %q
    spec:
      serviceAccountName: %s
      tolerations:
        - operator: Exists
      containers:
        - name: loggie
          image: %s
          imagePullPolicy: IfNotPresent
          args:
            - -config.system=/etc/loggie/loggie.yml
            - -config.pipeline=/etc/loggie/pipelines.yml
          ports:
            - name: monitor
              containerPort: 9196
              protocol: TCP
          resources:
            requests:
              cpu: 50m
              memory: 128Mi
            limits:
              cpu: "1"
              memory: 512Mi
          volumeMounts:
            - name: config
              mountPath: /etc/loggie
              readOnly: true
            - name: varlogpods
              mountPath: /var/log/pods
              readOnly: true
            - name: varlogcontainers
              mountPath: /var/log/containers
              readOnly: true
            - name: data
              mountPath: /data/loggie
      volumes:
        - name: config
          configMap:
            name: %s
        - name: varlogpods
          hostPath:
            path: /var/log/pods
            type: DirectoryOrCreate
        - name: varlogcontainers
          hostPath:
            path: /var/log/containers
            type: DirectoryOrCreate
        - name: data
          hostPath:
            path: %s
            type: DirectoryOrCreate
`,
		ns, labels,
		app, ns, labels,
		cm, ns, labels, system, pipelines,
		app, ns, labels,
		app,
		app, pid, cid,
		app,
		image,
		cm,
		dataPath,
	)
}

func indentYAMLBlock(raw string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var b strings.Builder
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}
		b.WriteString(pad)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
