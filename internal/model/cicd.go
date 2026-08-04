package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	CicdServiceTypeFrontend = "frontend"
	CicdServiceTypeBackend  = "backend"
	CicdServiceTypeMicro    = "microservice"

	CicdDeployKindRegular   = "regular"
	CicdDeployKindContainer = "container"

	CicdRefTypeBranch = "branch"
	CicdRefTypeTag    = "tag"

	// DefaultNodeToolName Jenkins Global Tool Configuration 中 Node 安装名称的默认值。
	DefaultNodeToolName = "node24"

	CicdRunStatusPending   = "pending"
	CicdRunStatusRunning   = "running"
	CicdRunStatusSuccess   = "success"
	CicdRunStatusFailure   = "failure"
	CicdRunStatusAborted   = "aborted"
	CicdRunStatusCancelled = "cancelled"

	// CD 发布审批流
	CicdRunStatusPendingApproval  = "pending_approval"  // 待审核
	CicdRunStatusPendingExecution = "pending_execution" // 待执行（已通过审核）
	CicdRunStatusRejected         = "rejected"          // 已驳回

	CicdReleaseTypePodUpdate      = "pod_update"
	CicdReleaseTypeServiceOnline  = "service_online"
	CicdReleaseTypeArtifactDeploy = "artifact_deploy"

	// CD 发布操作类型（京雀式：CI 仅构建，发布按操作类型区分）
	CicdReleaseOpFrontendOnline   = "frontend_online"   // 服务上线
	CicdReleaseOpFrontendRollback = "frontend_rollback" // 服务回滚
	CicdReleaseOpBackendInitial   = "backend_initial"   // 服务初次部署
	CicdReleaseOpBackendUpdate    = "backend_update"    // 服务更新
	CicdReleaseOpContainerRollback = "container_rollback" // 容器回滚

	CicdPublishModeBuildOnly        = "仅构建"
	CicdPublishModeArtifactDeploy   = "制品发布"
)

// CicdService 应用服务（CI/CD 发布单元，对应 Jenkins 一个 Job）。
type CicdService struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ProjectID    uint           `json:"project_id" gorm:"not null;index:idx_cicd_svc_proj;uniqueIndex:uk_cicd_svc_proj_identifier,priority:1"`
	Identifier   string         `json:"identifier" gorm:"size:128;not null;uniqueIndex:uk_cicd_svc_proj_identifier,priority:2;comment:唯一标识符"`
	Name         string         `json:"name" gorm:"size:128;not null;comment:应用名称"`
	ServiceType  string         `json:"service_type" gorm:"size:32;not null;default:'backend';comment:frontend|backend|microservice"`
	Owner        string         `json:"owner" gorm:"size:64;comment:负责人"`
	ProductLine  string         `json:"product_line" gorm:"size:128;comment:产品线"`
	Remark       string         `json:"remark" gorm:"size:512"`
	Status       int            `json:"status" gorm:"not null;default:1;comment:1启用 0停用"`
	JenkinsJob   string         `json:"jenkins_job" gorm:"size:256;comment:Jenkins Job 名（不含 folder）"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdService) TableName() string { return "cicd_services" }

// CicdCiConfig 服务的 CI 打包配置（一服务一条）。
type CicdCiConfig struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ServiceID        uint           `json:"service_id" gorm:"not null;uniqueIndex;comment:所属服务"`
	GitURL           string         `json:"git_url" gorm:"size:512;not null;comment:业务仓库 SSH 地址"`
	RefType          string         `json:"ref_type" gorm:"size:16;not null;default:'branch';comment:branch|tag"`
	RefName          string         `json:"ref_name" gorm:"size:128;not null;default:'main';comment:默认分支或 tag"`
	BuildType        string         `json:"build_type" gorm:"size:32;not null;comment:npm|yarn|mvn|gradle|python|golang"`
	BuildShell       string         `json:"build_shell" gorm:"size:512;comment:构建命令参数"`
	BuildPath        string         `json:"build_path" gorm:"size:256;comment:制品目录 target/build/dist"`
	ProjectName      string         `json:"project_name" gorm:"size:128;comment:后端 JAR 命名"`
	Version          string         `json:"version" gorm:"size:64;comment:版本号"`
	NodeVersion      string         `json:"node_version" gorm:"size:32;comment:前端 Node 版本"`
	NpmInstallMode   string         `json:"npm_install_mode" gorm:"size:16;default:'install';comment:install|ci|skip"`
	CleanNpmCache    bool           `json:"clean_npm_cache" gorm:"not null;default:false"`
	CleanNodeModules bool           `json:"clean_node_modules" gorm:"not null;default:false"`
	JavaToolName     string         `json:"java_tool_name" gorm:"size:64;default:'jdk8'"`
	ServerPort       string         `json:"server_port" gorm:"size:16;comment:后端服务端口"`
	PackConfigPaths  string         `json:"pack_config_paths" gorm:"size:512;comment:golang 随包配置目录"`
	Description      string         `json:"description" gorm:"size:512"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdCiConfig) TableName() string { return "cicd_ci_configs" }

// NodeToolName 返回 Jenkins Job 参数 nodeToolName（与 Global Tool 名称一致）。
func NodeToolNameFromConfig(c *CicdCiConfig) string {
	if c == nil {
		return DefaultNodeToolName
	}
	if v := strings.TrimSpace(c.NodeVersion); v != "" {
		return v
	}
	return DefaultNodeToolName
}

// CicdDeployConfig 发布配置（常规 SSH 或容器化，可多条按环境）。
type CicdDeployConfig struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	ServiceID           uint           `json:"service_id" gorm:"not null;index"`
	Name                string         `json:"name" gorm:"size:128;not null;comment:配置名称"`
	DeployKind          string         `json:"deploy_kind" gorm:"size:32;not null;default:'regular';comment:regular|container"`
	Tenv                string         `json:"tenv" gorm:"size:16;not null;default:'dev';comment:dev|test|prod"`
	AuditEnabled        bool           `json:"audit_enabled" gorm:"not null;default:false;comment:发布审核"`
	Importance          string         `json:"importance" gorm:"size:32;comment:重要级别"`
	DestPath            string         `json:"dest_path" gorm:"size:512;comment:部署路径"`
	ServerIDsJSON       string         `json:"server_ids_json" gorm:"type:text;comment:CMDB 服务器 ID 列表 JSON"`
	DeployUser          string         `json:"deploy_user" gorm:"size:64;default:'root'"`
	DeployGroup         string         `json:"deploy_group" gorm:"size:64;default:'root'"`
	ArtifactRetainCount int            `json:"artifact_retain_count" gorm:"not null;default:10"`
	RunUser             string         `json:"run_user" gorm:"size:64;default:'app'"`
	StartScriptType     string         `json:"start_script_type" gorm:"size:32;default:'脚本模板'"`
	CustomScriptContent string         `json:"custom_script_content" gorm:"type:text"`
	CleanDeployDir      bool           `json:"clean_deploy_dir" gorm:"not null;default:false"`
	JVMOpts             string         `json:"jvm_opts" gorm:"type:text"`
	ServerPort          int            `json:"server_port" gorm:"default:8080"`
	DeployMethod        string         `json:"deploy_method" gorm:"size:16;default:'kubectl';comment:kubectl|helm"`
	DeployAction        string         `json:"deploy_action" gorm:"size:32;default:'服务更新'"`
	DeployConfigType    string         `json:"deploy_config_type" gorm:"size:64;default:'使用deployment模板'"`
	DeployConfigTemplate string        `json:"deploy_config_template" gorm:"size:64;default:'基础模板'"`
	K8sNamespace        string         `json:"k8s_namespace" gorm:"size:128"`
	K8sClusterID        *uint          `json:"k8s_cluster_id,omitempty"`
	ImageName           string         `json:"image_name" gorm:"size:128"`
	ImageTag            string         `json:"image_tag" gorm:"size:128"`
	Replicas            int            `json:"replicas" gorm:"default:1"`
	ContainerPort       int            `json:"container_port" gorm:"default:8080"`
	Status              int            `json:"status" gorm:"not null;default:1"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdDeployConfig) TableName() string { return "cicd_deploy_configs" }

// CicdBuildRun CI 打包记录。
type CicdBuildRun struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	ProjectID         uint           `json:"project_id" gorm:"not null;index:idx_cicd_build_proj"`
	ServiceID         uint           `json:"service_id" gorm:"not null;index"`
	BuildNumber       int            `json:"build_number" gorm:"not null;default:0;index"`
	BranchName        string         `json:"branch_name" gorm:"size:128"`
	PublishMode       string         `json:"publish_mode" gorm:"size:32;not null;default:'仅构建'"`
	Tenv              string         `json:"tenv" gorm:"size:16"`
	BuildResult       string         `json:"build_result" gorm:"size:32;not null;default:'pending'"`
	BuilderUserID     *uint          `json:"builder_user_id,omitempty"`
	BuilderName       string         `json:"builder_name" gorm:"size:64"`
	Version           string         `json:"version" gorm:"size:64"`
	PackagePath       string         `json:"package_path" gorm:"size:512;comment:MinIO 路径"`
	ImageAddress      string         `json:"image_address" gorm:"size:512"`
	DownloadURL       string         `json:"download_url" gorm:"size:1024"`
	SecurityScanPass  *bool          `json:"security_scan_pass,omitempty"`
	JenkinsQueueID    int64          `json:"jenkins_queue_id"`
	JenkinsBuildURL   string         `json:"jenkins_build_url" gorm:"size:512"`
	ParamsJSON        string         `json:"params_json" gorm:"type:text"`
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	FinishedAt        *time.Time     `json:"finished_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdBuildRun) TableName() string { return "cicd_build_runs" }

// CicdReleaseRun CD 发布/工单记录。
type CicdReleaseRun struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	ProjectID          uint           `json:"project_id" gorm:"not null;index:idx_cicd_release_proj"`
	ServiceID          uint           `json:"service_id" gorm:"not null;index"`
	DeployConfigID     *uint          `json:"deploy_config_id,omitempty"`
	Title              string         `json:"title" gorm:"size:256;not null"`
	ReleaseKind        string         `json:"release_kind" gorm:"size:32;not null;default:'regular'"`
	ReleaseType        string         `json:"release_type" gorm:"size:32;comment:frontend_online|frontend_rollback|backend_initial|backend_update|pod_update|service_online"`
	Tenv               string         `json:"tenv" gorm:"size:16"`
	Status             string         `json:"status" gorm:"size:32;not null;default:'pending'"`
	CurrentStageKey    string         `json:"current_stage_key" gorm:"size:32;comment:当前待审批节点"`
	SubmitterUserID    *uint          `json:"submitter_user_id,omitempty"`
	SubmitterName      string         `json:"submitter_name" gorm:"size:64"`
	ImageAddress       string         `json:"image_address" gorm:"size:512"`
	ArtifactName       string         `json:"artifact_name" gorm:"size:256;comment:MinIO 制品名"`
	AuditEnabled       bool           `json:"audit_enabled" gorm:"not null;default:false;comment:提交时是否走审批"`
	RequestJSON        string         `json:"request_json" gorm:"type:text;comment:原始发布请求 JSON"`
	ReviewerUserID     *uint          `json:"reviewer_user_id,omitempty"`
	ReviewerName       string         `json:"reviewer_name" gorm:"size:64"`
	ReviewComment      string         `json:"review_comment" gorm:"size:512"`
	ReviewedAt         *time.Time     `json:"reviewed_at,omitempty"`
	JenkinsBuildNumber int            `json:"jenkins_build_number"`
	JenkinsBuildURL    string         `json:"jenkins_build_url" gorm:"size:512"`
	JenkinsQueueURL    string         `json:"jenkins_queue_url" gorm:"size:512;comment:Jenkins 队列项路径，构建号未落库前用于异步补偿"`
	ParamsJSON         string         `json:"params_json" gorm:"type:text"`
	VerifyStatus       string         `json:"verify_status" gorm:"size:32;comment:发布后验证状态 passed|failed|partial"`
	VerifyJSON         string         `json:"verify_json" gorm:"type:text;comment:验证结果 JSON"`
	VerifiedAt         *time.Time     `json:"verified_at,omitempty"`
	StartedAt          *time.Time     `json:"started_at,omitempty"`
	FinishedAt         *time.Time     `json:"finished_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdReleaseRun) TableName() string { return "cicd_release_runs" }
