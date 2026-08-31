package k8s

// 本文件仅存放 Pod 相关的入参 / 出参 DTO（含诊断结构），行为实现见：
// k8s_pod_service.go（CRUD/编排）、k8s_pod_exec.go、k8s_pod_logs.go、
// k8s_pod_files.go、k8s_pod_build.go、k8s_pod_diagnose.go。

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

type PodListQuery = ClusterNamespaceOptionalKeywordQuery

type PodLogsQuery struct {
	ClusterID    uint   `form:"cluster_id" binding:"required"`
	Namespace    string `form:"namespace" binding:"required"`
	Name         string `form:"name" binding:"required"`
	Container    string `form:"container"`
	TailLines    int64  `form:"tail_lines"`
	Follow       bool   `form:"follow"`
	Previous     bool   `form:"previous"`
	SinceSeconds int64  `form:"since_seconds"`
	SinceTime    string `form:"since_time"`
	Timestamps   bool   `form:"timestamps"`
	Keyword      string `form:"keyword"`
	StartTime    string `form:"start_time"`
	EndTime      string `form:"end_time"`
}

type PodDiagnoseQuery = PodDetailQuery

type PodDiagnoseHint struct {
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Action string `json:"action,omitempty"`
}

type PodDiagnoseContainerIssue struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Reason       string `json:"reason,omitempty"`
	Message      string `json:"message,omitempty"`
	RestartCount int32  `json:"restart_count"`
	LogSnippet   string `json:"log_snippet,omitempty"`
}

type PodDiagnoseResult struct {
	Summary    string                      `json:"summary"`
	Phase      string                      `json:"phase"`
	Ready      bool                        `json:"ready"`
	NodeName   string                      `json:"node_name"`
	Hints      []PodDiagnoseHint           `json:"hints"`
	Events     []PodEventItem              `json:"events"`
	Containers []PodDiagnoseContainerIssue `json:"containers"`
}

type PodFileQuery struct {
	ClusterID uint   `json:"cluster_id" form:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" form:"namespace" binding:"required"`
	Name      string `json:"name" form:"name" binding:"required"`
	Container string `json:"container" form:"container"`
	Path      string `json:"path" form:"path"`
}

type PodExecRequest struct {
	ClusterNamespaceNameCommandRequest
	Container string `json:"container"`
}

type PodDeleteRequest = ClusterNamespaceNameRequest

type PodDetailQuery = ClusterNamespaceNameQuery
type PodEventQuery = ClusterNamespaceNameQuery
type PodRestartRequest = ClusterNamespaceNameRequest
type PodCreateYAMLRequest = ClusterManifestApplyRequest

type PodCreateSimpleRequest struct {
	ClusterNamespaceNameRequest
	Image   string `json:"image" binding:"required"`
	Command string `json:"command"`

	ContainerName   string            `json:"container_name"`
	ImagePullPolicy string            `json:"image_pull_policy"`
	RestartPolicy   string            `json:"restart_policy"`
	Port            int32             `json:"port"`
	Env             map[string]string `json:"env"`
	Labels          map[string]string `json:"labels"`

	RequestsCPU    string `json:"requests_cpu"`
	RequestsMemory string `json:"requests_memory"`
	LimitsCPU      string `json:"limits_cpu"`
	LimitsMemory   string `json:"limits_memory"`

	Tolerations       []PodCreateSimpleToleration `json:"tolerations"`
	NodeSelector      map[string]string           `json:"node_selector"`
	PriorityClassName string                      `json:"priority_class_name"`
	Affinity          *corev1.Affinity            `json:"affinity"`
}

type PodCreateSimpleToleration struct {
	Key               string `json:"key"`
	Operator          string `json:"operator"`
	Value             string `json:"value"`
	Effect            string `json:"effect"`
	TolerationSeconds *int64 `json:"toleration_seconds"`
}

type PodFileItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	ModTime     string `json:"mod_time"`
}

type PodItem struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Phase        string    `json:"phase"`
	NodeName     string    `json:"node_name"`
	Ready        bool      `json:"ready"`
	PodIP        string    `json:"pod_ip"`
	HostIP       string    `json:"host_ip"`
	QOSClass     string    `json:"qos_class"`
	RestartCount int32     `json:"restart_count"`
	Images       []string  `json:"images"`
	StartTime    time.Time `json:"start_time"`

	HostNetwork     bool    `json:"host_network"`
	ContainersText  string  `json:"containers_text,omitempty"`
	ResourceText    string  `json:"resource_text,omitempty"`
	CPUUsage        string  `json:"cpu_usage,omitempty"`
	MemUsage        string  `json:"mem_usage,omitempty"`
	CPUPctRequest   float64 `json:"cpu_pct_request,omitempty"`
	CPUPctLimit     float64 `json:"cpu_pct_limit,omitempty"`
	CPUPctNodeAlloc float64 `json:"cpu_pct_node_alloc,omitempty"`
	MemPctRequest   float64 `json:"mem_pct_request,omitempty"`
	MemPctLimit     float64 `json:"mem_pct_limit,omitempty"`
	MemPctNodeAlloc float64 `json:"mem_pct_node_alloc,omitempty"`
	LabelCount      int     `json:"label_count,omitempty"`
	AnnotationCount int     `json:"annotation_count,omitempty"`
}

type PodContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restart_count"`
	State        string `json:"state"`
}

type PodDetail struct {
	Name              string                `json:"name"`
	Namespace         string                `json:"namespace"`
	UID               string                `json:"uid"`
	Phase             string                `json:"phase"`
	NodeName          string                `json:"node_name"`
	ServiceAccount    string                `json:"service_account"`
	PodIP             string                `json:"pod_ip"`
	HostIP            string                `json:"host_ip"`
	QOSClass          string                `json:"qos_class"`
	Labels            map[string]string     `json:"labels"`
	Annotations       map[string]string     `json:"annotations"`
	Containers        []PodContainerInfo    `json:"containers"`
	InitContainers    []PodContainerInfo    `json:"init_containers"`
	Conditions        []corev1.PodCondition `json:"conditions"`
	Volumes           []corev1.Volume       `json:"volumes"`
	Tolerations       []corev1.Toleration   `json:"tolerations"`
	NodeSelector      map[string]string     `json:"node_selector"`
	PriorityClassName string                `json:"priority_class_name"`
	Affinity          *corev1.Affinity      `json:"affinity"`
	StartTime         time.Time             `json:"start_time"`
	CreationTime      time.Time             `json:"creation_time"`
}

type PodEventItem struct {
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
}

// ExecTerminalSize 前端终端窗口尺寸（用于 ExecTTYStream 的 resize 队列）。
type ExecTerminalSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}
